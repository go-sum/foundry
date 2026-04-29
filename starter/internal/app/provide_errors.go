package app

import (
	"strings"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/otelweb"
	"github.com/go-sum/foundry/pkg/web/router"

	config "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/errorpage"
)

func provideErrorBoundary(runtime Runtime, routing *router.Router) web.Middleware {
	return web.ErrorBoundary(web.BoundaryConfig{
		Renderer:     &appErrorRenderer{rt: routing},
		Logger:       runtime.Logger,
		CaptureStack: runtime.Config.Env == config.Production,
		OnError:      otelweb.MakeOnError(),
		Op: func(c *web.Context) string {
			return c.Method() + " " + c.URL().Path
		},
		Subsystem: func(*web.Context) string { return "http" },
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
	rt *router.Router
}

// RenderError renders the error as an HTML response, choosing full-page or
// HTMX fragment mode based on the request.
func (r *appErrorRenderer) RenderError(c *web.Context, e *web.Error) web.Response {
	vr := view.NewRequest(c)
	full := errorpage.ErrorPage(vr, e)
	partial := errorpage.ErrorContent(e)
	return view.RenderWithStatus(vr, e.Status, full, partial)
}
