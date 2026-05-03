// Package common assembles the shared infrastructure services (database,
// queue, email) that are consumed by both the web process and the worker.
package common

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/queue"
	"github.com/go-sum/foundry/pkg/queue/pgstore"

	appdb "github.com/go-sum/foundry/db"
)

// Services holds the shared infrastructure instances assembled by Provide.
type Services struct {
	DBPool         *pgxpool.Pool
	SchemaRegistry *db.Registry
	EmailSender    email.Sender
	QueueStore     *pgstore.Store
	Queue          *queue.Dispatcher
}

// Provide connects to all shared infrastructure services and returns them.
// The caller is responsible for closing DBPool on error or shutdown.
func Provide(ctx context.Context, logger *slog.Logger, dbDSN string, emailCfg email.Config) (Services, error) {
	pool, schemaReg, err := connectDatabase(ctx, logger, dbDSN)
	if err != nil {
		return Services{}, err
	}
	qStore, qDispatcher := provideQueue(pool, logger)
	emailSender, err := provideEmailSender(emailCfg, logger)
	if err != nil {
		pool.Close()
		return Services{}, fmt.Errorf("services: email: %w", err)
	}
	return Services{
		DBPool:         pool,
		SchemaRegistry: schemaReg,
		EmailSender:    emailSender,
		QueueStore:     qStore,
		Queue:          qDispatcher,
	}, nil
}

func connectDatabase(ctx context.Context, logger *slog.Logger, dsn string) (*pgxpool.Pool, *db.Registry, error) {
	var pool *pgxpool.Pool
	if err := cfgpkg.ConnectWithRetry(ctx, "db", logger, 3, func() error {
		var err error
		pool, err = db.ConnectDSN(ctx, dsn,
			db.WithProductionDefaults(),
			db.WithSlowQueryLogger(logger, 500*time.Millisecond),
		)
		return err
	}); err != nil {
		return nil, nil, fmt.Errorf("services: db: %w", err)
	}
	db.LogPoolStats(ctx, pool, logger, 60*time.Second)

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
