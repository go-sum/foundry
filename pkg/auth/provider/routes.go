package provider

import (
	auth "github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web/router"
)

// Route name constants.
const (
	RouteAuthorize     = "oauth.authorize"
	RouteAuthorizePost = "oauth.authorize.post"
	RouteToken         = "oauth.token"
	RouteUserinfo      = "oauth.userinfo"
	RouteDiscovery     = "oauth.discovery"
)

// Routes returns the declarative route tree for the OAuth 2.0 provider module.
// The caller registers the returned nodes via router.Register(rt, provider.Routes(m)...).
func Routes(m *ProviderModule) []router.Node {
	return []router.Node{
		// OIDC discovery endpoint (public, no auth required).
		router.GET("/.well-known/openid-configuration", RouteDiscovery, m.discoveryHandler.Serve),

		// Authorization endpoint — requires the user to be logged in.
		router.Group("/oauth",
			router.Use(auth.RequireAuth(m.signinPath)),
			router.GET("/authorize", RouteAuthorize, m.authorizeHandler.Show),
			router.POST("/authorize", RouteAuthorizePost, m.authorizeHandler.Submit),
		),

		// Token endpoint — public API (no session, no CSRF).
		router.POST("/oauth/token", RouteToken, m.tokenHandler.Exchange),

		// Userinfo endpoint — public API, bearer-token authenticated.
		router.GET("/oauth/userinfo", RouteUserinfo, m.userinfoHandler.Serve),
	}
}
