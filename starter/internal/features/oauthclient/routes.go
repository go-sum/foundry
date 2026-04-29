package oauthclient

import "github.com/go-sum/foundry/pkg/web/router"

const (
	RouteConnect  = "auth.connect"
	RouteCallback = "auth.callback"
)

// Routes returns the route nodes owned by the first-party OAuth client.
func Routes(h *Handler) []router.Node {
	return []router.Node{
		router.GET("/auth/connect", RouteConnect, h.Connect),
		router.GET("/auth/callback", RouteCallback, h.Callback),
	}
}
