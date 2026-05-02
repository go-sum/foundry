package app

import (
	"cmp"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/static"
)

// RouteSpec pairs a URL pattern with its named route.
type RouteSpec struct {
	Pattern string
	Name    string
}

// RouteConfig holds all starter-owned route specs.
// Package-owned routes (auth, provider, docs, showcase) use their own RouteConfig types.
type RouteConfig struct {
	Health        RouteSpec
	Home          RouteSpec
	ContactForm   RouteSpec
	ContactSubmit RouteSpec
	OAuthConnect  RouteSpec
	OAuthCallback RouteSpec
}

// DefaultRouteConfig returns conventional patterns for all starter-owned routes.
func DefaultRouteConfig() RouteConfig { return applyRouteDefaults(RouteConfig{}) }

// applyRouteDefaults fills any zero-value RouteSpec with its default.
// cmp.Or returns the first non-zero comparable value, so a fully-zero
// RouteConfig transparently adopts all defaults, and a partial override
// keeps only the fields that were set.
func applyRouteDefaults(r RouteConfig) RouteConfig {
	return RouteConfig{
		Health:        cmp.Or(r.Health, RouteSpec{"/healthz", "health.check"}),
		Home:          cmp.Or(r.Home, RouteSpec{"/", "home.show"}),
		ContactForm:   cmp.Or(r.ContactForm, RouteSpec{"/contact", "contact.form"}),
		ContactSubmit: cmp.Or(r.ContactSubmit, RouteSpec{"/contact", "contact.submit"}),
		OAuthConnect:  cmp.Or(r.OAuthConnect, RouteSpec{"/auth/connect", "auth.connect"}),
		OAuthCallback: cmp.Or(r.OAuthCallback, RouteSpec{"/auth/callback", "auth.callback"}),
	}
}

// RegisterRoutes registers the assembled application route trees on the router.
func RegisterRoutes(rt *router.Router, cfg RouteConfig, assets static.AssetsConfig, deps RouteDeps) error {
	cfg = applyRouteDefaults(cfg)
	if err := registerStaticRoutes(rt, assets); err != nil {
		return err
	}
	router.Register(rt, router.Nodes(
		HealthRoutes(cfg, deps),
		PublicRoutes(deps),
		APIRoutes(deps),
	)...)
	return nil
}

// HealthRoutes returns the health-check route (no middleware applied).
func HealthRoutes(cfg RouteConfig, deps RouteDeps) []router.Node {
	return []router.Node{
		router.GET(cfg.Health.Pattern, cfg.Health.Name, deps.HealthHandler),
	}
}

// PublicRoutes returns browser-facing routes wrapped in the assembled middleware tree.
func PublicRoutes(deps RouteDeps) []router.Node {
	return []router.Node{
		router.Layout(router.Nodes(
			[]router.Node{router.Use(deps.Public.Middleware...)},
			[]router.Node{router.Use(deps.Public.CSRFMiddleware)},
			deps.Public.MetaNodes,
			deps.Public.FeatureNodes,
			deps.Public.PackageNodes,
		)...),
	}
}

// APIRoutes returns the assembled API route tree.
func APIRoutes(deps RouteDeps) []router.Node {
	return []router.Node{
		router.Layout(router.Nodes(
			[]router.Node{router.Use(deps.API.Middleware...)},
			deps.API.PublicNodes,
			[]router.Node{router.Layout(router.Nodes(
				[]router.Node{router.Use(deps.API.ProtectedMiddleware...)},
				deps.API.ProtectedNodes,
			)...)},
		)...),
	}
}
