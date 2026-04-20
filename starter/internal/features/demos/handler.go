package demos

import (
	"github.com/go-sum/componentry/showcase"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
)

// Handler serves the component showcase page.
type Handler struct {
	rt      *router.Router
	reqOpts []view.RequestOption
}

// NewHandler creates a new demos Handler.
func NewHandler(rt *router.Router, opts ...view.RequestOption) *Handler {
	return &Handler{rt: rt, reqOpts: opts}
}

// Show renders the component showcase page.
func (h *Handler) Show(c *web.Context) (web.Response, error) {
	vr := view.NewRequest(c, h.reqOpts...)
	return view.Render(vr, vr.Page("Component Showcase", showcase.Showcase()), nil)
}
