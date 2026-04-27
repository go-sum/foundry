// Package home implements the home page feature.
package home

import (
	"context"

	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/page"
	"github.com/go-sum/foundry/pkg/web"
)

// Checker describes a service whose health is shown on the home page.
type Checker struct {
	Name string
	Fn   func(ctx context.Context) error
}

// Handler serves the home page.
type Handler struct {
	checkers []Checker
	reqOpts  []view.RequestOption
}

// NewHandler creates a new home Handler.
func NewHandler(checkers []Checker, opts ...view.RequestOption) *Handler {
	return &Handler{checkers: checkers, reqOpts: opts}
}

// Show renders the home page with live health status for each configured service.
func (h *Handler) Show(c *web.Context) (web.Response, error) {
	var statuses []page.ServiceStatus
	for _, ch := range h.checkers {
		statuses = append(statuses, page.ServiceStatus{
			Name:    ch.Name,
			Healthy: ch.Fn(c.Context()) == nil,
		})
	}
	vr := view.NewRequest(c, h.reqOpts...)
	return view.Render(vr, page.HomePage(vr, statuses), page.HomeContent(vr, statuses))
}
