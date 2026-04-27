// Package app is the composition root for the starter application.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/go-sum/foundry/pkg/componentry/icons"
	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/notification"
	"github.com/go-sum/foundry/pkg/queue"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/htmx"
	"github.com/go-sum/foundry/pkg/web/otelweb"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/serve"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/go-sum/foundry/pkg/web/site"
	"github.com/jackc/pgx/v5/pgxpool"

	config "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/internal/features/contact"
	"github.com/go-sum/foundry/internal/features/oauthclient"
	"github.com/go-sum/foundry/internal/view"
)

// App is the assembled application.
type App struct {
	Runtime
	Security
	Services
	router       *router.Router
	sessionStore session.Store
}

// Runtime holds cross-cutting infrastructure dependencies.
type Runtime struct {
	Config *config.Config
	Logger *slog.Logger
	Tracer trace.Tracer
}

// Security holds resolved security middleware configurations.
type Security struct {
	CSRF    secure.CSRFConfig
	Headers secure.HeadersConfig
	CSP     secure.CSPNonceConfig
	Origins []string
	Session session.Config
}

// Services holds application-level service instances.
type Services struct {
	DBPool         *pgxpool.Pool
	KVStore        kv.Store
	Queue          *queue.Dispatcher
	Processor      *queue.Processor
	Notifier       *notification.Dispatcher
	Contact        *contact.Module
	Auth           *auth.Module
	// OAuthProvider is the built-in OAuth 2.0 Authorization Server.
	OAuthProvider  *provider.ProviderModule
	// OAuthClient is the first-party OAuth 2.1 client handler.
	OAuthClient    *oauthclient.Handler
	SchemaRegistry *db.Registry
}

// Presentation consolidates view-layer dependencies assembled at the composition root.
type Presentation struct {
	ViewOpts []view.RequestOption
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

// New builds and wires the complete application. Returns an error on any
// configuration or infrastructure failure — never calls os.Exit.
func New(ctx context.Context) (*App, error) {
	runtime, err := provideRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("runtime: %w", err)
	}

	manifest, iconReg := provideAssets(runtime.Config)

	routing := router.New()

	pres := Presentation{
		ViewOpts: []view.RequestOption{
			config.DefaultNav(routing),
			view.WithPathFunc(manifest.Path),
			view.WithIconRegistry(iconReg),
		},
		Icons: iconReg,
	}

	security, store, err := provideSecurity(ctx, runtime)
	if err != nil {
		return nil, fmt.Errorf("security: %w", err)
	}

	routing.Use(
		web.AsyncContext(),
		otelweb.Middleware(runtime.Tracer),
		web.WithRequestID(),
		provideErrorBoundary(runtime, routing),
		serve.AccessLogMiddleware(runtime.Logger),
		secure.Headers(security.Headers),
		secure.CSPNonce(security.CSP),
		session.Middleware(security.Session),
		secure.CSRF(security.CSRF),
		htmx.VaryMiddleware(),
		auth.LoadSession(),
	)

	services, err := provideServices(ctx, runtime, security, routing, pres)
	if err != nil {
		return nil, fmt.Errorf("services: %w", err)
	}

	s := site.New(runtime.Config.Site)
	if err := RegisterRoutes(routing, security, services, runtime.Config.Assets, runtime.Config.PublicDir, s, pres); err != nil {
		return nil, fmt.Errorf("routes: %w", err)
	}
	routing.Freeze()

	return &App{
		Runtime:      runtime,
		Security:     security,
		Services:     services,
		router:       routing,
		sessionStore: store,
	}, nil
}

// Run starts the HTTP server, waits for ctx to be cancelled, then gracefully
// shuts down within the configured shutdown timeout.
func (a *App) Run(ctx context.Context) error {
	return serve.ListenAndServe(ctx, a.router.Serve, a.Config.Server)
}
