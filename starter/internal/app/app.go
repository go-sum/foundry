// Package app is the composition root for the starter application.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"go.opentelemetry.io/otel/trace"

	"github.com/go-sum/foundry/pkg/auth/provider"
	authweb "github.com/go-sum/foundry/pkg/auth/web"
	"github.com/go-sum/foundry/pkg/componentry/icons"
	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/queue"
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
	closer       cfgpkg.Closer
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
	webServicesFactory  func(context.Context, Runtime, Security, *router.Router, Presentation, kv.Store, *ratelimit.Limiter, validate.Validator) (Services, error)
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

// WithWebServicesFactory overrides the web-only services assembly. Intended for
// tests that need deterministic startup without external infrastructure.
func WithWebServicesFactory(f func(context.Context, Runtime, Security, *router.Router, Presentation, kv.Store, *ratelimit.Limiter, validate.Validator) (Services, error)) Option {
	return func(o *appOptions) { o.webServicesFactory = f }
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
	// Infrastructure
	DBPool         *pgxpool.Pool
	KVStore        kv.Store
	RateLimiter    *ratelimit.Limiter
	QueueStore     queue.Store
	Queue          *queue.Dispatcher
	SchemaRegistry *db.Registry
	EmailSender    email.Sender

	// Application modules
	Contact       *contact.Module
	Auth          *authweb.Module
	OAuthProvider *provider.ProviderModule
	OAuthClient   *oauthclient.Handler
}

// Presentation consolidates view-layer dependencies assembled at the composition root.
type Presentation struct {
	ViewOpts []viewstate.RequestOption
	Icons    *icons.Registry
}

// ProviderContext holds infrastructure dependencies shared across module providers.
type ProviderContext struct {
	Runtime   Runtime
	Pool      *pgxpool.Pool
	KVStore   kv.Store
	Router    *router.Router
	Validator validate.Validator
	ViewOpts  []viewstate.RequestOption
}

// Close shuts down background services and releases resources.
// Resources registered via closer are shut down in LIFO order.
// Session store and KV store are handled directly to support App structs
// constructed outside of New (e.g. in tests).
func (a *App) Close() error {
	var errs []error
	if store, ok := a.sessionStore.(stoppableSessionStore); ok {
		store.Stop()
	}
	if err := a.closer.Close(); err != nil {
		errs = append(errs, err)
	}
	if a.Services.KVStore != nil {
		if err := a.Services.KVStore.Close(); err != nil {
			errs = append(errs, fmt.Errorf("kv: %w", err))
		}
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
			primaryNav(routing, routes.OAuthConnect.Name),
		},
		Icons: iconReg,
	}

	sharedKV, err := provideKVStore(ctx, runtime, o.kvStoreFactory)
	if err != nil {
		return nil, fmt.Errorf("kv: %w", err)
	}

	rlCfg := runtime.Config.RateLimit
	limiter, err := ratelimit.NewLimiter(rlCfg.Store.WithKVStore(sharedKV), rlCfg.Policies, runtime.Logger)
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
	app.closer.Add("rate-limiter", limiter.Close)
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

	webServicesFactory := o.webServicesFactory
	if webServicesFactory == nil {
		webServicesFactory = provideWebServices
	}
	services, err := webServicesFactory(ctx, runtime, security, routing, pres, sharedKV, limiter, val)
	if err != nil {
		return nil, fmt.Errorf("services: %w", err)
	}
	if services.KVStore == nil {
		services.KVStore = sharedKV
	}
	if services.RateLimiter == nil {
		services.RateLimiter = limiter
	}
	app.Services = services
	if services.DBPool != nil {
		app.closer.Add("db", func() error { services.DBPool.Close(); return nil })
	}
	if services.Queue != nil {
		app.closer.Add("queue-dispatcher", services.Queue.Close)
	}

	s := site.New(runtime.Config.Site)
	publicDir := filepath.Dir(runtime.Config.Assets.PublicDir)
	routeDeps, err := buildRouteDeps(routing, routes, security, services, publicDir, s, pres)
	if err != nil {
		return nil, fmt.Errorf("route deps: %w", err)
	}
	if err := RegisterRoutes(routing, routes, runtime.Config.Assets, routeDeps); err != nil {
		return nil, fmt.Errorf("routes: %w", err)
	}
	routing.Freeze()

	return app, nil
}

// Run starts the HTTP server, waits for ctx to be cancelled, then gracefully
// shuts down within the configured shutdown timeout.
func (a *App) Run(ctx context.Context) error {
	return serve.ListenAndServe(ctx, a.router.Serve, a.Config.Web.Server)
}
