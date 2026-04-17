// Package home implements the home page feature.
package home

import (
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/page"
	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
)

// Handler serves the home page.
type Handler struct {
	getRoutes func() []router.Route
}

// NewHandler creates a new home Handler.
func NewHandler(getRoutes func() []router.Route) *Handler {
	return &Handler{getRoutes: getRoutes}
}

// Show renders the home page with HTMX dual-mode support.
func (h *Handler) Show(c *web.Context) (web.Response, error) {
	vr := view.NewRequest(c, h.getRoutes())
	return view.Render(vr, page.HomePage(vr), page.HomeContent())
}
