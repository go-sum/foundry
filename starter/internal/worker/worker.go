// Package worker assembles and runs the background-job worker process.
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	config "github.com/go-sum/foundry/config"
	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runtime holds cross-cutting infrastructure for the background worker.
// It is intentionally leaner than app.Runtime — it holds only what the worker needs.
type Runtime struct {
	Config *config.WorkerConfig
	Logger *slog.Logger
}

// Processor is the minimal lifecycle contract required by the worker runtime.
// *queue.Processor satisfies this interface.
type Processor interface {
	Start(context.Context)
	Ping(context.Context) error
	Stop() error
}

// Services holds worker-only service instances.
type Services struct {
	DBPool    *pgxpool.Pool
	Processor Processor
}

// Worker is the assembled background-job runtime.
type Worker struct {
	Runtime  Runtime
	Services Services
	closer   cfgpkg.Closer
}

// Option configures the Worker at construction time.
type Option func(*options)

type options struct {
	servicesFactory func(context.Context, Runtime) (Services, error)
}

// WithServicesFactory overrides the worker services assembly.
// Intended for tests that need deterministic worker startup without external infrastructure.
func WithServicesFactory(f func(context.Context, Runtime) (Services, error)) Option {
	return func(o *options) { o.servicesFactory = f }
}

// New builds and wires the background worker runtime.
func New(ctx context.Context, opts ...Option) (_ *Worker, err error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	runtime, err := provideRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("runtime: %w", err)
	}

	servicesFactory := o.servicesFactory
	if servicesFactory == nil {
		servicesFactory = provideServices
	}

	services, err := servicesFactory(ctx, runtime)
	if err != nil {
		return nil, fmt.Errorf("services: %w", err)
	}

	w := &Worker{
		Runtime:  runtime,
		Services: services,
	}
	if services.Processor != nil {
		w.closer.Add("processor", services.Processor.Stop)
	}
	if services.DBPool != nil {
		w.closer.Add("db", func() error { services.DBPool.Close(); return nil })
	}
	defer func() {
		if err == nil {
			return
		}
		if closeErr := w.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("cleanup: %w", closeErr))
		}
	}()

	return w, nil
}

// Run starts the worker processor and blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	if w.Services.Processor == nil {
		return fmt.Errorf("worker: processor unavailable")
	}
	w.Services.Processor.Start(ctx)
	<-ctx.Done()
	return nil
}

// Check verifies worker dependency reachability without starting the processor.
func (w *Worker) Check(ctx context.Context) error {
	if w.Services.Processor == nil {
		return fmt.Errorf("worker: processor unavailable")
	}
	return w.Services.Processor.Ping(ctx)
}

// Close shuts down worker resources in LIFO order.
func (w *Worker) Close() error {
	if err := w.closer.Close(); err != nil {
		return err
	}
	return nil
}
