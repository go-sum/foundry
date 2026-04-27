// Package home implements the home page feature.
package home

import (
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/page"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
)

// Handler serves the home page.
type Handler struct {
	rt      *router.Router
	reqOpts []view.RequestOption
}

// NewHandler creates a new home Handler.
func NewHandler(rt *router.Router, opts ...view.RequestOption) *Handler {
	return &Handler{rt: rt, reqOpts: opts}
}

// Show renders the home page with HTMX dual-mode support.
func (h *Handler) Show(c *web.Context) (web.Response, error) {
	helloURL := h.rt.MustReverse("hello.show", map[string]string{"name": "World"})
	vr := view.NewRequest(c, h.reqOpts...)
	return view.Render(vr, page.HomePage(vr, helloURL), page.HomeContent(vr, helloURL))
}
