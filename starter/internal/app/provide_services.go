package app

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/notification"
	"github.com/go-sum/foundry/pkg/notification/notifylog"
	"github.com/go-sum/foundry/pkg/queue"
	"github.com/go-sum/foundry/pkg/queue/pgstore"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"

	config "github.com/go-sum/foundry/config"
	appdb "github.com/go-sum/foundry/db"
	"github.com/go-sum/foundry/internal/features/contact"
	"github.com/go-sum/foundry/internal/features/oauthclient"
)

func provideServices(ctx context.Context, runtime Runtime, _ Security, rt *router.Router, pres Presentation, kvStore kv.Store, val validate.Validator) (Services, error) {
	if runtime.Config.Env == config.Testing {
		return Services{KVStore: kvStore}, nil
	}

	var pool *pgxpool.Pool
	if err := cfgpkg.ConnectWithRetry(ctx, "db", runtime.Logger, 3, func() error {
		var err error
		pool, err = db.ConnectDSN(ctx, cfgpkg.ExpandSecret("DATABASE_URL"),
			db.WithProductionDefaults(),
			db.WithSlowQueryLogger(runtime.Logger, 500*time.Millisecond),
		)
		return err
	}); err != nil {
		return Services{}, fmt.Errorf("services: db: %w", err)
	}
	db.LogPoolStats(ctx, pool, runtime.Logger, 60*time.Second)

	schemaReg, err := db.LoadRegistryFromYAML(appdb.ConfigYAML, appdb.SchemaFiles,
		db.WithResolver(appdb.ExternalSchemas()),
	)
	if err != nil {
		return Services{}, fmt.Errorf("services: schema registry: %w", err)
	}
	if err := db.VerifyFingerprint(ctx, pool, schemaReg.Fingerprint()); err != nil {
		pool.Close()
		return Services{}, fmt.Errorf("services: schema not ready (run 'task db:migrate'): %w", err)
	}

	if kvStore == nil {
		pool.Close()
		return Services{}, fmt.Errorf("services: kv: missing shared store")
	}

	qStore := pgstore.New(pool)
	qDispatcher := queue.NewDispatcher(qStore, queue.WithDispatcherLogger(runtime.Logger))

	notifier := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelLog: notifylog.New(runtime.Logger),
	}, runtime.Logger)

	contactMod := contact.NewModule(contact.ModuleConfig{
		Pool:        pool,
		KV:          kvStore,
		Queue:       qDispatcher,
		Notifier:    notifier,
		Router:      rt,
		Validator:   val,
		Service: contact.ServiceConfig{
			RateLimit:  runtime.Config.Contact.RateLimit,
			RateWindow: runtime.Config.Contact.RateWindow,
			QueueName:  contact.QueueName,
		},
		Worker: contact.WorkerConfig{
			SendTo:   runtime.Config.Contact.SendTo,
			SendFrom: runtime.Config.Contact.SendFrom,
		},
		ViewOpts: pres.ViewOpts,
		Logger:   runtime.Logger,
	})

	authMod, oauthProvider, err := provideAuth(runtime.Config, runtime.Logger, pool, kvStore, rt, pres.ViewOpts, val)
	if err != nil {
		pool.Close()
		return Services{}, fmt.Errorf("services: %w", err)
	}

	oauthClientH := oauthclient.New(runtime.Config.Auth.FirstPartyClientConfig())

	processor := queue.NewProcessor(qStore, queue.WithLogger(runtime.Logger))
	processor.Register(contact.QueueName, contactMod.QueueHandler,
		queue.WithWorkers(2),
		queue.WithMaxAttempts(5),
		queue.WithTimeout(30*time.Second),
	)
	processor.Start(ctx)

	return Services{
		DBPool:         pool,
		KVStore:        kvStore,
		Queue:          qDispatcher,
		Processor:      processor,
		Notifier:       notifier,
		Contact:        contactMod,
		Auth:           authMod,
		OAuthProvider:  oauthProvider,
		OAuthClient:    oauthClientH,
		SchemaRegistry: schemaReg,
	}, nil
}
