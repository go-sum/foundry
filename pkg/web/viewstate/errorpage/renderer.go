// Package errorpage provides full-page and partial error views plus a
// web.ErrorRenderer implementation ready for use in web.BoundaryConfig.
package errorpage

import (
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/viewstate"
)

// NewErrorRenderer returns a web.ErrorRenderer that renders errors as HTML
// using the viewstate request machinery. Full-page requests get ErrorPage;
// HTMX partial requests get ErrorContent as a fragment.
func NewErrorRenderer() web.ErrorRenderer {
	return &errorRenderer{}
}

type errorRenderer struct{}

func (r *errorRenderer) RenderError(c *web.Context, e *web.Error) web.Response {
	vr := viewstate.NewRequest(c)
	full := ErrorPage(vr, e)
	partial := ErrorContent(e)
	return viewstate.RenderWithStatus(vr, e.Status, full, partial)
}
