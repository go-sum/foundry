package provider

import (
	"cmp"

	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/router"
)

// RouteSpec pairs a URL pattern with its router name. Both fields are required;
// a zero-value RouteSpec is replaced by the default during module construction.
type RouteSpec struct {
	Pattern string
	Name    string
}

// RouteConfig holds the URL pattern and route name for every OAuth 2.0 provider
// endpoint. Pass a zero-value RouteConfig (or omit the field entirely) to use
// DefaultRouteConfig. Populate individual fields to override specific endpoints.
type RouteConfig struct {
	Discovery     RouteSpec
	Token         RouteSpec
	Userinfo      RouteSpec
	Authorize     RouteSpec
	AuthorizePost RouteSpec
}

// DefaultRouteConfig returns the conventional OAuth 2.0 route patterns.
func DefaultRouteConfig() RouteConfig { return applyRouteDefaults(RouteConfig{}) }

// applyRouteDefaults fills any zero-value RouteSpec with its default.
// cmp.Or returns the first non-zero comparable value, so a fully-zero
// RouteConfig transparently adopts all defaults, and a partial override
// keeps only the fields that were set.
func applyRouteDefaults(r RouteConfig) RouteConfig {
	return RouteConfig{
		Discovery:     cmp.Or(r.Discovery, RouteSpec{"/.well-known/openid-configuration", "oauth.discovery"}),
		Token:         cmp.Or(r.Token, RouteSpec{"/oauth/token", "oauth.token"}),
		Userinfo:      cmp.Or(r.Userinfo, RouteSpec{"/oauth/userinfo", "oauth.userinfo"}),
		Authorize:     cmp.Or(r.Authorize, RouteSpec{"/oauth/authorize", "oauth.authorize"}),
		AuthorizePost: cmp.Or(r.AuthorizePost, RouteSpec{"/oauth/authorize", "oauth.authorize.post"}),
	}
}

// PublicRoutes returns the public API routes for the OAuth 2.0 provider module.
func PublicRoutes(m *ProviderModule) []router.Node {
	r := m.routes
	return []router.Node{
		router.GET(r.Discovery.Pattern, r.Discovery.Name, m.discoveryHandler.Serve),
		router.POST(r.Token.Pattern, r.Token.Name, m.tokenHandler.Exchange),
		router.GET(r.Userinfo.Pattern, r.Userinfo.Name, m.userinfoHandler.Serve),
	}
}

// ProtectedRoutes returns the browser-facing routes that require the user to
// be logged in and remain inside the CSRF-protected route branch.
func ProtectedRoutes(m *ProviderModule) []router.Node {
	r := m.routes
	return []router.Node{
		router.Layout(
			router.Use(authn.RequireAuth(m.signinPath)),
			router.GET(r.Authorize.Pattern, r.Authorize.Name, m.authorizeHandler.Show),
			router.POST(r.AuthorizePost.Pattern, r.AuthorizePost.Name, m.authorizeHandler.Submit),
		),
	}
}

// Routes returns the full declarative route tree for the OAuth 2.0 provider module.
// The caller registers the returned nodes via router.Register(rt, provider.Routes(m)...).
func Routes(m *ProviderModule) []router.Node {
	return router.Nodes(
		PublicRoutes(m),
		ProtectedRoutes(m),
	)
}
