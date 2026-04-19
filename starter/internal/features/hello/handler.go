// Package hello implements the hello greeting feature.
package hello

import (
	"cmp"
	"unicode"

	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/page"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
	"github.com/go-sum/web/router"
)

// Handler serves the hello greeting page.
type Handler struct {
	getRoutes func() []router.Route
	reqOpts   []view.RequestOption
}

// NewHandler creates a new hello Handler.
func NewHandler(getRoutes func() []router.Route, opts ...view.RequestOption) *Handler {
	return &Handler{getRoutes: getRoutes, reqOpts: opts}
}

// Greeting renders just the greeting fragment for HTMX partial swaps.
// It reads the name from the query parameter sent by hx-include.
func (h *Handler) Greeting(c *web.Context) (web.Response, error) {
	name := cmp.Or(c.URL().Query().Get("name"), "World")
	return render.Fragment(page.HelloPartial(name))
}

// Show renders the hello page for the named route parameter.
func (h *Handler) Show(c *web.Context) (web.Response, error) {
	name := cmp.Or(c.Param("name"), "World")
	if !isValidName(name) {
		return web.Response{}, web.ErrBadRequest("name must be 1–64 letters only")
	}
	vr := view.NewRequest(c, h.getRoutes(), h.reqOpts...)
	return view.Render(vr, page.HelloPage(vr, name), page.HelloPartial(name))
}

func isValidName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
