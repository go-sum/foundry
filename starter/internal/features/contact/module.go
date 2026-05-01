package contact

import (
	"log/slog"

	coredb "github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/queue"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
)

// Module bundles the contact feature's handler and queue worker.
type Module struct {
	Handler      *Handler
	QueueName    string
	QueueHandler queue.HandlerFunc

	svc          Service
	val          validate.Validator
	clientIPFunc ratelimit.KeyFunc
}

// ModuleConfig holds all dependencies needed to wire the contact feature.
type ModuleConfig struct {
	Pool         coredb.DBTX
	RateLimiter  *ratelimit.Limiter
	Queue        *queue.Dispatcher
	EmailSender  email.Sender
	Router       *router.Router
	Validator    validate.Validator
	Service      ServiceConfig
	Worker       WorkerConfig
	ViewOpts     []viewstate.RequestOption
	Logger       *slog.Logger
	ClientIPFunc ratelimit.KeyFunc
}

// NewModule wires the contact feature module.
func NewModule(cfg ModuleConfig) *Module {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	repo := NewRepository(cfg.Pool)
	svc := NewService(repo, cfg.RateLimiter, cfg.Queue, cfg.Service, logger)
	worker := NewNotifyHandler(cfg.EmailSender, cfg.Worker)

	m := &Module{
		QueueName:    QueueName,
		QueueHandler: worker,
		svc:          svc,
		val:          cfg.Validator,
		clientIPFunc: cfg.ClientIPFunc,
	}
	if cfg.Router != nil {
		m.Handler = NewHandler(cfg.Router, svc, cfg.Validator, cfg.ClientIPFunc, cfg.ViewOpts...)
	}
	return m
}

// NewHandler creates a contact Handler using the module's already-wired service.
// Call this after the router is available when NewModule was called without one.
func (m *Module) NewHandler(rt *router.Router, opts ...viewstate.RequestOption) *Handler {
	return NewHandler(rt, m.svc, m.val, m.clientIPFunc, opts...)
}
