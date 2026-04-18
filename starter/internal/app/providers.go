package app

import (
	"context"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-sum/web"
	"github.com/go-sum/web/otelweb"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/session"

	config "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/errorpage"
)

func provideRuntime(_ context.Context) (Runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return Runtime{}, err
	}
	return Runtime{
		Config: cfg,
		Logger: slog.Default(),
		Tracer: noop.NewTracerProvider().Tracer("app"),
	}, nil
}

func provideSecurity(_ context.Context, runtime Runtime) (Security, *session.MemoryStore, error) {
	cfg := runtime.Config

	origins := make([]string, 0, 1+len(cfg.Site.OriginAllowlist))
	if cfg.Site.BaseURL != "" {
		origins = append(origins, cfg.Site.BaseURL)
	}
	origins = append(origins, cfg.Site.OriginAllowlist...)

	sessCfg, store := provideSession(runtime)

	sec := Security{
		CSRF:        cfg.CSRF,
		Headers:     cfg.Headers,
		CSPTemplate: cfg.CSP.CSPTemplate,
		Origins:     origins,
		Session:     sessCfg,
	}
	return sec, store, nil
}

func provideSession(runtime Runtime) (session.Config, *session.MemoryStore) {
	store := session.NewMemoryStore()
	return session.NewConfig(runtime.Config.Session, store), store
}

func provideServices(_ context.Context, _ Runtime, _ Security) (Services, error) {
	return Services{}, nil
}

func provideErrorBoundary(runtime Runtime, routing *router.Router) web.Middleware {
	return web.ErrorBoundary(web.BoundaryConfig{
		Renderer:     &appErrorRenderer{getRoutes: routing.Routes},
		Logger:       runtime.Logger,
		CaptureStack: runtime.Config.Env == config.Production,
		OnError:      otelweb.MakeOnError(),
		Op: func(c *web.Context) string {
			return c.Method() + " " + c.URL().Path
		},
		Subsystem: func(c *web.Context) string { return "http" },
		TraceID:   otelweb.ExtractTraceID(),
		SpanID:    otelweb.ExtractSpanID(),
		DedupeKey: func(c *web.Context) string {
			parts := strings.SplitN(c.URL().Path, "/", 3)
			if len(parts) > 1 {
				return c.Method() + "|" + parts[1]
			}
			return c.Method() + "|" + c.URL().Path
		},
	})
}

// appErrorRenderer implements web.ErrorRenderer for the starter application.
type appErrorRenderer struct {
	getRoutes func() []router.Route
}

// RenderError renders the error as an HTML response, choosing full-page or
// HTMX fragment mode based on the request.
func (r *appErrorRenderer) RenderError(c *web.Context, e *web.Error) web.Response {
	vr := view.NewRequest(c, r.getRoutes())
	full := errorpage.ErrorPage(vr, e)
	partial := errorpage.ErrorContent(e)
	return view.RenderWithStatus(vr, e.Status, full, partial)
}
