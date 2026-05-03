package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/queue"
	"github.com/go-sum/foundry/pkg/queue/pgstore"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"

	config "github.com/go-sum/foundry/config"
	appdb "github.com/go-sum/foundry/db"
	"github.com/go-sum/foundry/internal/features/contact"
	"github.com/go-sum/foundry/internal/features/oauthclient"
)

func provideServices(ctx context.Context, runtime Runtime, sec Security, rt *router.Router, pres Presentation, kvStore kv.Store, limiter *ratelimit.Limiter, val validate.Validator) (Services, error) {
	if runtime.Config.Env == config.Testing {
		return Services{KVStore: kvStore, RateLimiter: limiter}, nil
	}

	pool, schemaReg, err := provideDatabase(ctx, runtime)
	if err != nil {
		return Services{}, err
	}

	if kvStore == nil {
		pool.Close()
		return Services{}, fmt.Errorf("services: kv: missing shared store")
	}

	pc := ProviderContext{
		Runtime:   runtime,
		Pool:      pool,
		KVStore:   kvStore,
		Router:    rt,
		Validator: val,
		ViewOpts:  pres.ViewOpts,
	}

	qStore, qDispatcher := provideQueue(pool, runtime.Logger)
	emailSender, err := provideEmailSender(runtime.Config.App.Email, runtime.Logger)
	if err != nil {
		pool.Close()
		return Services{}, fmt.Errorf("services: email: %w", err)
	}

	contactMod := provideContactModule(pc, limiter, qDispatcher, emailSender, sec.RateLimitKey, runtime.Config.App.Contact)

	authMod, oauthProvider, err := provideAuth(pc, sec, emailSender)
	if err != nil {
		pool.Close()
		return Services{}, fmt.Errorf("services: %w", err)
	}

	oauthClientH := oauthclient.New(runtime.Config.Auth.OAuthClient)
	processor := startBackgroundWorkers(ctx, qStore, contactMod, runtime.Logger)

	return Services{
		DBPool:         pool,
		KVStore:        kvStore,
		RateLimiter:    limiter,
		Queue:          qDispatcher,
		Processor:      processor,
		EmailSender:    emailSender,
		Contact:        contactMod,
		Auth:           authMod,
		OAuthProvider:  oauthProvider,
		OAuthClient:    oauthClientH,
		SchemaRegistry: schemaReg,
	}, nil
}

func provideDatabase(ctx context.Context, runtime Runtime) (*pgxpool.Pool, *db.Registry, error) {
	var pool *pgxpool.Pool
	if err := cfgpkg.ConnectWithRetry(ctx, "db", runtime.Logger, 3, func() error {
		var err error
		pool, err = db.ConnectDSN(ctx, runtime.Config.DB.DSN,
			db.WithProductionDefaults(),
			db.WithSlowQueryLogger(runtime.Logger, 500*time.Millisecond),
		)
		return err
	}); err != nil {
		return nil, nil, fmt.Errorf("services: db: %w", err)
	}
	db.LogPoolStats(ctx, pool, runtime.Logger, 60*time.Second)

	schemaReg, err := db.LoadRegistryFromYAML(appdb.ConfigYAML, appdb.SchemaFiles,
		db.WithResolver(appdb.ExternalSchemas()),
	)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("services: schema registry: %w", err)
	}
	if err := db.VerifyFingerprint(ctx, pool, schemaReg.Fingerprint()); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("services: schema not ready (run 'task db:migrate'): %w", err)
	}
	return pool, schemaReg, nil
}

func provideQueue(pool *pgxpool.Pool, logger *slog.Logger) (*pgstore.Store, *queue.Dispatcher) {
	qStore := pgstore.New(pool)
	return qStore, queue.NewDispatcher(qStore, queue.WithDispatcherLogger(logger))
}

func provideEmailSender(cfg email.Config, logger *slog.Logger) (email.Sender, error) {
	return email.New(cfg, logger)
}

func provideContactModule(pc ProviderContext, limiter *ratelimit.Limiter, qDispatcher *queue.Dispatcher, emailSender email.Sender, rateLimitKey ratelimit.KeyFunc, cfg config.ContactConfig) *contact.Module {
	return contact.NewModule(contact.ModuleConfig{
		Pool:         pc.Pool,
		RateLimiter:  limiter,
		Queue:        qDispatcher,
		EmailSender:  emailSender,
		Router:       pc.Router,
		Validator:    pc.Validator,
		ClientIPFunc: rateLimitKey,
		Service: contact.ServiceConfig{
			RateLimitProfile: config.RateLimitContactSubmitEmail,
			QueueName:        contact.QueueName,
		},
		Worker: contact.WorkerConfig{
			SendTo:   cfg.SendTo,
			SendFrom: cfg.SendFrom,
		},
		ViewOpts: pc.ViewOpts,
		Logger:   pc.Runtime.Logger,
	})
}

func startBackgroundWorkers(ctx context.Context, qStore *pgstore.Store, contactMod *contact.Module, logger *slog.Logger) *queue.Processor {
	processor := queue.NewProcessor(qStore, queue.WithLogger(logger))
	processor.Register(contact.QueueName, contactMod.QueueHandler,
		queue.WithWorkers(2),
		queue.WithMaxAttempts(5),
		queue.WithTimeout(30*time.Second),
	)
	processor.Start(ctx)
	return processor
}
