// Package app is the composition root for the starter application.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/go-sum/web"
	"github.com/go-sum/web/htmx"
	"github.com/go-sum/web/otelweb"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/secure"
	"github.com/go-sum/web/serve"
	"github.com/go-sum/web/session"

	config "github.com/go-sum/foundry/config"
)

// App is the assembled application.
type App struct {
	Runtime
	Security
	Services
	router       *router.Router
	sessionStore *session.MemoryStore
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
	CSPTemplate string
	Origins     []string
	Session     session.Config
}

// Services is a placeholder for application service dependencies (populated from Stage B onward).
type Services struct{}

// New builds and wires the complete application. Returns an error on any
// configuration or infrastructure failure — never calls os.Exit.
func New(ctx context.Context) (*App, error) {
	runtime, err := provideRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("runtime: %w", err)
	}

	security, store, err := provideSecurity(ctx, runtime)
	if err != nil {
		return nil, fmt.Errorf("security: %w", err)
	}

	services, err := provideServices(ctx, runtime, security)
	if err != nil {
		return nil, fmt.Errorf("services: %w", err)
	}

	routing := router.NewWithoutSecureDefaults()
	routing.Use(
		otelweb.Middleware(runtime.Tracer),
		web.WithRequestID(),
		provideErrorBoundary(runtime, routing),
		serve.AccessLogMiddleware(),
		secure.Headers(security.Headers),
		secure.CSPNonce(secure.CSPNonceConfig{CSPTemplate: security.CSPTemplate}),
		session.Middleware(security.Session),
		secure.CSRF(security.CSRF),
		htmx.VaryMiddleware(),
	)

	if err := RegisterRoutes(routing, security, runtime.Config.Assets); err != nil {
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

// Handler returns the application's root handler.
func (a *App) Handler() web.Handler {
	return func(c *web.Context) (web.Response, error) {
		return a.router.Serve(c)
	}
}

// Run starts the HTTP server, waits for ctx to be cancelled, then gracefully
// shuts down within the configured shutdown timeout.
func (a *App) Run(ctx context.Context) error {
	srv := serve.NewServer(a.Handler(), a.Runtime.Config.Server)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	<-ctx.Done()

	sctx, cancel := context.WithTimeout(context.Background(), a.Runtime.Config.Server.ShutdownTimeout)
	defer cancel()

	if err := serve.Shutdown(sctx, srv); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	if err, ok := <-errCh; ok {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
