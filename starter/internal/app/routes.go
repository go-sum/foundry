package app

import (
	"cmp"

	"github.com/go-sum/foundry/internal/features/home"
	"github.com/go-sum/foundry/internal/features/oauthclient"
	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/docs"
	"github.com/go-sum/foundry/pkg/showcase"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/health"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/site"
	"github.com/go-sum/foundry/pkg/web/static"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"

	g "maragu.dev/gomponents"
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

// RegisterRoutes registers all application routes on the router.
// cfg is the route configuration; pass DefaultRouteConfig() for conventional paths.
func RegisterRoutes(rt *router.Router, cfg RouteConfig, sec Security, svc Services, assets static.AssetsConfig, publicDir string, s *site.Site, pres Presentation) error {
	cfg = applyRouteDefaults(cfg)
	if err := registerStaticRoutes(rt, assets); err != nil {
		return err
	}
	router.Register(rt, router.Nodes(
		HealthRoutes(cfg, svc),
		PublicRoutes(rt, cfg, sec, svc, s, pres),
		APIRoutes(cfg, sec, svc, publicDir),
	)...)
	return nil
}

// HealthRoutes returns the health-check route (no middleware applied).
func HealthRoutes(cfg RouteConfig, svc Services) []router.Node {
	return []router.Node{
		router.GET(cfg.Health.Pattern, cfg.Health.Name, health.Handler(healthCheckers(svc)...)),
	}
}

// PublicRoutes returns browser-facing routes wrapped in content + CSRF middleware.
func PublicRoutes(rt *router.Router, cfg RouteConfig, sec Security, svc Services, s *site.Site, pres Presentation) []router.Node {
	metaH := site.NewHandlers(s, rt,
		site.RobotsConfig{DefaultAllow: true},
		site.SitemapConfig{
			Routes: []site.RouteEntry{
				{Name: cfg.Home.Name},
				{Name: "demos.showcase"},
				{Name: cfg.ContactForm.Name},
			},
			DefaultChangeFreq: "weekly",
		},
	)

	contactForm, contactSubmit := contactHandlers(svc)

	return []router.Node{
		router.Layout(router.Nodes(
			[]router.Node{router.Use(contentMiddleware(sec)...)},
			[]router.Node{router.Use(secure.CSRF(sec.CSRF))},
			site.Routes(metaH),
			[]router.Node{
				router.GET(cfg.Home.Pattern, cfg.Home.Name, home.NewHandler(homeServiceChecks(svc), pres.ViewOpts...).Show),
				router.GET(cfg.ContactForm.Pattern, cfg.ContactForm.Name, contactForm),
				router.POST(cfg.ContactSubmit.Pattern, cfg.ContactSubmit.Name, contactSubmit),
			},
			showcase.Routes(showcase.Config{
				Icons: pres.Icons,
				DB:    svc.DBPool,
				KV:    svc.KVStore,
				Page: func(c *web.Context, title string, content g.Node) (web.Response, error) {
					vr := viewstate.NewRequest(c, pres.ViewOpts...)
					return viewstate.Render(vr, vr.Page(title, content), nil)
				},
			}),
		)...),
	}
}

// APIRoutes returns the API-middleware tree: docs, OAuth public endpoints, and CSRF-protected auth.
func APIRoutes(cfg RouteConfig, sec Security, svc Services, publicDir string) []router.Node {
	oauthCfg := provider.DefaultRouteConfig()
	if svc.OAuthProvider != nil {
		oauthCfg = svc.OAuthProvider.RouteConfig()
	}
	return []router.Node{
		router.Layout(router.Nodes(
			[]router.Node{router.Use(apiMiddleware(sec, oauthCfg.Token.Pattern)...)},
			protectedDocs(svc.Auth, publicDir),
			routesFrom(svc.OAuthProvider, provider.PublicRoutes),
			[]router.Node{router.Layout(router.Nodes(
				[]router.Node{router.Use(secure.CSRF(sec.CSRF))},
				AuthRoutes(cfg, svc),
			)...)},
		)...),
	}
}

// AuthRoutes returns CSRF-protected routes: auth, OAuth provider, OAuth client.
func AuthRoutes(cfg RouteConfig, svc Services) []router.Node {
	return router.Nodes(
		routesFrom(svc.Auth, authn.Routes),
		routesFrom(svc.OAuthProvider, provider.ProtectedRoutes),
		OAuthClientRoutes(cfg, svc.OAuthClient),
	)
}

// OAuthClientRoutes returns routes for the first-party OAuth client handler.
func OAuthClientRoutes(cfg RouteConfig, h *oauthclient.Handler) []router.Node {
	if h == nil {
		return nil
	}
	return []router.Node{
		router.GET(cfg.OAuthConnect.Pattern, cfg.OAuthConnect.Name, h.Connect),
		router.GET(cfg.OAuthCallback.Pattern, cfg.OAuthCallback.Name, h.Callback),
	}
}

func protectedDocs(auth *authn.Module, publicDir string) []router.Node {
	routes := docs.Routes(docs.DefaultConfig(publicDir))
	if auth == nil {
		return routes
	}
	return []router.Node{router.Scope(auth.RequireAuth(), routes...)}
}

func routesFrom[T any](dep *T, fn func(*T) []router.Node) []router.Node {
	if dep == nil {
		return nil
	}
	return fn(dep)
}

