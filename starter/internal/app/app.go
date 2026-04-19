// Package app is the composition root for the starter application.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/go-sum/web"
	"github.com/go-sum/web/htmx"
	"github.com/go-sum/web/otelweb"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/secure"
	"github.com/go-sum/web/serve"
	"github.com/go-sum/web/session"
	"github.com/go-sum/web/site"

	config "github.com/go-sum/foundry/config"
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
	CSRF        secure.CSRFConfig
	Headers     secure.HeadersConfig
	CSP         secure.CSPNonceConfig
	Origins     []string
	Session     session.Config
}

// Services is a placeholder for application services
type Services struct{}

// New builds and wires the complete application. Returns an error on any
// configuration or infrastructure failure — never calls os.Exit.
func New(ctx context.Context) (*App, error) {
	runtime, err := provideRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("runtime: %w", err)
	}

	provideAssets(runtime.Config)

	security, store, err := provideSecurity(ctx, runtime)
	if err != nil {
		return nil, fmt.Errorf("security: %w", err)
	}

	services, err := provideServices(ctx, runtime, security)
	if err != nil {
		return nil, fmt.Errorf("services: %w", err)
	}

	routing := router.New()
	routing.Use(
		web.AsyncContext(),
		otelweb.Middleware(runtime.Tracer),
		web.WithRequestID(),
		provideErrorBoundary(runtime, routing),
		serve.AccessLogMiddleware(),
		secure.Headers(security.Headers),
		secure.CSPNonce(security.CSP),
		session.Middleware(security.Session),
		secure.CSRF(security.CSRF),
		htmx.VaryMiddleware(),
	)

	s := site.New(runtime.Config.Site)
	if err := RegisterRoutes(routing, security, runtime.Config.Assets, s); err != nil {
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
	return serve.ListenAndServeGracefully(ctx, a.router.Serve, a.Runtime.Config.Server)
}
