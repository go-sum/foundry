package contact

import (
	"log/slog"

	"github.com/go-sum/kv"
	"github.com/go-sum/notification"
	"github.com/go-sum/queue"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/validate"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sum/foundry/internal/view"
)

// Module bundles the contact feature's handler and queue worker.
type Module struct {
	Handler      *Handler
	QueueName    string
	QueueHandler queue.HandlerFunc

	svc Service
	val validate.Validator
}

// ModuleConfig holds all dependencies needed to wire the contact feature.
type ModuleConfig struct {
	Pool      *pgxpool.Pool
	KV        kv.Store
	Queue     *queue.Dispatcher
	Notifier  *notification.Dispatcher
	Router    *router.Router
	Validator validate.Validator
	Service   ServiceConfig
	Worker    WorkerConfig
	ViewOpts  []view.RequestOption
	Logger    *slog.Logger
}

// NewModule wires the contact feature module.
func NewModule(cfg ModuleConfig) *Module {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	repo := NewRepository(cfg.Pool)
	svc := NewService(repo, cfg.KV, cfg.Queue, cfg.Service, logger)
	worker := NewNotifyHandler(cfg.Notifier, cfg.Worker)

	m := &Module{
		QueueName:    QueueName,
		QueueHandler: worker,
		svc:          svc,
		val:          cfg.Validator,
	}
	if cfg.Router != nil {
		m.Handler = NewHandler(cfg.Router, svc, cfg.Validator, cfg.ViewOpts...)
	}
	return m
}

// NewHandler creates a contact Handler using the module's already-wired service.
// Call this after the router is available when NewModule was called without one.
func (m *Module) NewHandler(rt *router.Router, opts ...view.RequestOption) *Handler {
	return NewHandler(rt, m.svc, m.val, opts...)
}
