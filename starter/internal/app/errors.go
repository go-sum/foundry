package app

import (
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/errorpage"
	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
)

// appErrorRenderer implements web.ErrorRenderer for the starter application.
// It renders error pages using the application's view layer. It never exposes
// cause text or internal detail to the client.
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
