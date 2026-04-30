package app

import (
	"github.com/go-sum/foundry/internal/features/contact"
	"github.com/go-sum/foundry/internal/features/home"
	"github.com/go-sum/foundry/internal/features/oauthclient"
	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/docs"
	"github.com/go-sum/foundry/pkg/showcase"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/health"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/site"
	"github.com/go-sum/foundry/pkg/web/static"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"

	g "maragu.dev/gomponents"
)

// RegisterRoutes registers all application routes on the router.
func RegisterRoutes(rt *router.Router, sec Security, svc Services, assets static.AssetsConfig, publicDir string, s *site.Site, pres Presentation) error {
	if err := registerStaticRoutes(rt, assets); err != nil {
		return err
	}

	router.Register(rt, router.Nodes(
		starterRouteTree(rt, sec, svc, s, pres),
		packageOwnedRouteTree(sec, svc, publicDir),
	)...)
	return nil
}

func starterRouteTree(rt *router.Router, sec Security, svc Services, s *site.Site, pres Presentation) []router.Node {
	metaH := site.NewHandlers(s, rt,
		site.RobotsConfig{DefaultAllow: true},
		site.SitemapConfig{
			Routes: []site.RouteEntry{
				{Name: home.RouteShow},
				{Name: "demos.showcase"},
				{Name: contact.RouteForm},
			},
			DefaultChangeFreq: "weekly",
		},
	)

	homeH := home.NewHandler(homeServiceChecks(svc), pres.ViewOpts...)
	contactForm, contactSubmit := contactHandlers(svc)

	return []router.Node{
		router.GET("/healthz", "health.check", health.Handler(healthCheckers(svc)...)),
		router.Layout(router.Nodes(
			[]router.Node{router.Use(contentMiddleware(sec)...)},
			site.Routes(metaH),
			home.Routes(homeH),
			contact.RoutesWithHandlers(contactForm, contactSubmit),
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

func packageOwnedRouteTree(sec Security, svc Services, publicDir string) []router.Node {
	return []router.Node{router.Layout(router.Nodes(
		[]router.Node{router.Use(apiMiddleware(sec)...)},
		protectedDocs(svc.Auth, publicDir),
		authRoutes(svc),
	)...)}
}

func protectedDocs(auth *authn.Module, publicDir string) []router.Node {
	routes := docs.Routes(docs.DefaultConfig(publicDir))
	if auth == nil {
		return routes
	}
	return []router.Node{router.Scope(auth.RequireAuth(), routes...)}
}

func authRoutes(svc Services) []router.Node {
	return router.Nodes(
		routesFrom(svc.Auth, authn.Routes),
		routesFrom(svc.OAuthProvider, provider.Routes),
		routesFrom(svc.OAuthClient, oauthclient.Routes),
	)
}

func routesFrom[T any](dep *T, fn func(*T) []router.Node) []router.Node {
	if dep == nil {
		return nil
	}
	return fn(dep)
}
