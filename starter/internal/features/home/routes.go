package home

import "github.com/go-sum/foundry/pkg/web/router"

const RouteShow = "home.show"

// Routes returns the route nodes owned by the home feature.
func Routes(h *Handler) []router.Node {
	return []router.Node{
		router.GET("/", RouteShow, h.Show),
	}
}
