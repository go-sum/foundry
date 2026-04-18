// Package app provides an optional convenience assembler that wires a Router,
// ErrorBoundary, access logging, and request IDs together. It delegates server
// lifecycle to serve.ListenAndServeGracefully.
//
// Users who need full control over middleware ordering, multiple routers, or
// custom server configuration should compose pkg/web packages manually instead.
package app

import (
	"context"
	"log/slog"

	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/serve"
)

// Config configures the App assembler.
type Config struct {
	// Server configures the underlying HTTP server.
	// Uses serve.DefaultServerConfig() when Addr is empty.
	Server serve.ServerConfig

	// Logger is used for error boundary and access log.
	// Defaults to slog.Default() when nil.
	Logger *slog.Logger

	// ErrorRenderer renders errors as HTML for browser clients.
	// When nil, all errors are rendered as application/problem+json.
	ErrorRenderer web.ErrorRenderer

	// Boundary is additional configuration for the ErrorBoundary middleware.
	// Logger and Renderer are overridden from Config.Logger and Config.ErrorRenderer
	// if not explicitly set in Boundary.
	Boundary web.BoundaryConfig

	// SecureDefaults controls whether router.New() (secure defaults on) or
	// router.NewWithoutSecureDefaults() is used. Defaults to true.
	SecureDefaults bool
}

// App is the assembled application. Its Router field is public so callers
// can register routes and add middleware before calling Run.
type App struct {
	// Router is the route registry. Register routes and middleware on it
	// before calling Run.
	Router *router.Router

	// Logger is the logger used by the app's internal components.
	Logger *slog.Logger

	cfg Config
}

// New creates an App with the given config and installs standard middleware:
// WithRequestID → ErrorBoundary → AccessLogMiddleware (outermost to innermost).
// Secure defaults (security headers, CSP nonce) are installed by the router
// unless cfg.SecureDefaults is explicitly false.
func New(cfg Config) *App {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Server.Addr == "" {
		cfg.Server = serve.DefaultServerConfig()
	}

	// Wire logger and error renderer into boundary config if not already set.
	if cfg.Boundary.Logger == nil {
		cfg.Boundary.Logger = cfg.Logger
	}
	if cfg.Boundary.Renderer == nil && cfg.ErrorRenderer != nil {
		cfg.Boundary.Renderer = cfg.ErrorRenderer
	}

	var rt *router.Router
	if !cfg.SecureDefaults {
		rt = router.NewWithoutSecureDefaults()
	} else {
		rt = router.New()
	}

	rt.Use(
		web.WithRequestID(),
		web.ErrorBoundary(cfg.Boundary),
		serve.AccessLogMiddleware(),
	)

	return &App{
		Router: rt,
		Logger: cfg.Logger,
		cfg:    cfg,
	}
}

// Run freezes the router, starts the HTTP server, and blocks until ctx is
// canceled. It then gracefully shuts down within the configured timeout.
// Callers should pass a context canceled by signal.NotifyContext.
func (a *App) Run(ctx context.Context) error {
	a.Router.Freeze()
	return serve.ListenAndServeGracefully(ctx, a.Router.Serve, a.cfg.Server)
}
