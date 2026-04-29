package contact

import (
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
)

const (
	RouteForm   = "contact.form"
	RouteSubmit = "contact.submit"
)

// Routes returns the route nodes owned by the contact feature.
func Routes(h *Handler) []router.Node {
	return RoutesWithHandlers(h.Form, h.Submit)
}

// RoutesWithHandlers returns the contact routes for the given handlers.
func RoutesWithHandlers(form, submit web.Handler) []router.Node {
	return []router.Node{
		router.GET("/contact", RouteForm, form),
		router.POST("/contact", RouteSubmit, submit),
	}
}
