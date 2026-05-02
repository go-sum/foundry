// Package app is the composition root for the starter application.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/go-sum/foundry/pkg/componentry/icons"
	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/queue"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/serve"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/go-sum/foundry/pkg/web/site"
	"github.com/go-sum/foundry/pkg/web/validate"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
	"github.com/jackc/pgx/v5/pgxpool"

	config "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/internal/features/contact"
	"github.com/go-sum/foundry/internal/features/oauthclient"
)

// App is the assembled application.
type App struct {
	Runtime
	Security
	Services
	router       *router.Router
	sessionStore session.Store
}

type stoppableSessionStore interface {
	Stop()
}

// Option configures the application at construction time.
type Option func(*appOptions)

type appOptions struct {
	sessionStoreFactory func() session.Store
	kvStoreFactory      func(context.Context, Runtime) (kv.Store, error)
}

// WithSessionStoreFactory overrides the factory used to create the test-only
// in-memory session store. Intended for tests that need to observe store
// lifecycle events or deliberately avoid cookie/KV storage.
func WithSessionStoreFactory(f func() session.Store) Option {
	return func(o *appOptions) { o.sessionStoreFactory = f }
}

// WithKVStoreFactory overrides the shared KV dependency used by the app.
// Intended for tests that need deterministic startup and shutdown behavior.
func WithKVStoreFactory(f func(context.Context, Runtime) (kv.Store, error)) Option {
	return func(o *appOptions) { o.kvStoreFactory = f }
}

// Runtime holds cross-cutting infrastructure dependencies.
type Runtime struct {
	Config *config.Config
	Logger *slog.Logger
	Tracer trace.Tracer
}

// Security holds resolved security middleware configurations.
type Security struct {
	CSRF         secure.CSRFConfig
	Headers      secure.HeadersConfig
	CSP          secure.CSPNonceConfig
	Origins      []string
	AllowedHosts []string
	ServerOrigin string
	RateLimitKey ratelimit.KeyFunc
	Session      session.Config
}

// Services holds application-level service instances.
type Services struct {
	DBPool      *pgxpool.Pool
	KVStore     kv.Store
	RateLimiter *ratelimit.Limiter
	Queue       *queue.Dispatcher
	Processor   *queue.Processor
	EmailSender email.Sender
	Contact     *contact.Module
	Auth        *authn.Module
	// OAuthProvider is the built-in OAuth 2.0 Authorization Server.
	OAuthProvider *provider.ProviderModule
	// OAuthClient is the first-party OAuth 2.1 client handler.
	OAuthClient    *oauthclient.Handler
	SchemaRegistry *db.Registry
}

// Presentation consolidates view-layer dependencies assembled at the composition root.
type Presentation struct {
	ViewOpts []viewstate.RequestOption
	Icons    *icons.Registry
}

// Close shuts down background services and releases resources.
func (s Services) Close() error {
	var errs []error
	if s.Processor != nil {
		if err := s.Processor.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("processor: %w", err))
		}
	}
	if s.RateLimiter != nil {
		if err := s.RateLimiter.Close(); err != nil {
			errs = append(errs, fmt.Errorf("rate limiter: %w", err))
		}
	}
	if s.KVStore != nil {
		if err := s.KVStore.Close(); err != nil {
			errs = append(errs, fmt.Errorf("kv: %w", err))
		}
	}
	if s.DBPool != nil {
		s.DBPool.Close()
	}
	return errors.Join(errs...)
}

// Close shuts down background services and stops any stoppable session store.
func (a *App) Close() error {
	var errs []error
	if err := a.Services.Close(); err != nil {
		errs = append(errs, fmt.Errorf("services: %w", err))
	}
	if store, ok := a.sessionStore.(stoppableSessionStore); ok {
		store.Stop()
	}
	return errors.Join(errs...)
}

// New builds and wires the complete application. Returns an error on any
// configuration or infrastructure failure — never calls os.Exit.
func New(ctx context.Context, opts ...Option) (_ *App, err error) {
	var o appOptions
	for _, opt := range opts {
		opt(&o)
	}
	runtime, err := provideRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("runtime: %w", err)
	}

	manifest, iconReg, err := provideAssets(runtime.Config)
	if err != nil {
		return nil, fmt.Errorf("assets: %w", err)
	}

	routing := router.New()

	val := validate.New()

	routes := DefaultRouteConfig()

	pres := Presentation{
		ViewOpts: []viewstate.RequestOption{
			viewstate.WithIconRegistry(iconReg),
			viewstate.WithPathFunc(manifest.Path),
			config.DefaultNav(routing, routes.OAuthConnect.Name),
		},
		Icons: iconReg,
	}

	sharedKV, err := provideKVStore(ctx, runtime, o.kvStoreFactory)
	if err != nil {
		return nil, fmt.Errorf("kv: %w", err)
	}

	rlCfg := runtime.Config.RateLimit
	limiter, err := ratelimit.NewLimiter(rlCfg.Store.WithKVStore(sharedKV), rlCfg.Profiles(), runtime.Logger)
	if err != nil {
		if sharedKV != nil {
			_ = sharedKV.Close()
		}
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	security, store, err := provideSecurity(ctx, runtime, sharedKV, o.sessionStoreFactory)
	if err != nil {
		_ = limiter.Close()
		if sharedKV != nil {
			_ = sharedKV.Close()
		}
		return nil, fmt.Errorf("security: %w", err)
	}

	app := &App{
		Runtime:      runtime,
		Security:     security,
		router:       routing,
		sessionStore: store,
		Services: Services{
			KVStore:     sharedKV,
			RateLimiter: limiter,
		},
	}
	defer func() {
		if err == nil {
			return
		}
		if closeErr := app.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("cleanup: %w", closeErr))
		}
	}()

	coreMw, err := coreMiddleware(routing, runtime, security)
	if err != nil {
		return nil, fmt.Errorf("middleware: %w", err)
	}
	routing.Use(coreMw...)

	services, err := provideServices(ctx, runtime, security, routing, pres, sharedKV, limiter, val)
	if err != nil {
		return nil, fmt.Errorf("services: %w", err)
	}
	app.Services = services

	s := site.New(runtime.Config.Site)
	if err := RegisterRoutes(routing, routes, security, services, runtime.Config.Assets, runtime.Config.PublicDir, s, pres); err != nil {
		return nil, fmt.Errorf("routes: %w", err)
	}
	routing.Freeze()

	return app, nil
}

// Run starts the HTTP server, waits for ctx to be cancelled, then gracefully
// shuts down within the configured shutdown timeout.
func (a *App) Run(ctx context.Context) error {
	return serve.ListenAndServe(ctx, a.router.Serve, a.Config.Server)
}
