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

	browserNodes := []router.Node{
		router.Use(contentMiddleware(sec)...),
	}
	browserNodes = append(browserNodes, site.Routes(metaH)...)
	browserNodes = append(browserNodes, home.Routes(homeH)...)
	browserNodes = append(browserNodes, contact.RoutesWithHandlers(contactForm, contactSubmit)...)
	browserNodes = append(browserNodes, showcase.Routes(showcase.Config{
		Icons: pres.Icons,
		DB:    svc.DBPool,
		KV:    svc.KVStore,
		Page: func(c *web.Context, title string, content g.Node) (web.Response, error) {
			vr := viewstate.NewRequest(c, pres.ViewOpts...)
			return viewstate.Render(vr, vr.Page(title, content), nil)
		},
	})...)

	return []router.Node{
		router.GET("/healthz", "health.check", health.Handler(healthCheckers(svc)...)),
		router.Layout(browserNodes...),
	}
}

func packageOwnedRouteTree(sec Security, svc Services, publicDir string) []router.Node {
	nodes := []router.Node{
		router.Use(apiMiddleware(sec)...),
	}
	nodes = append(nodes, docs.Routes(docs.DefaultConfig(publicDir))...)
	if svc.Auth != nil {
		nodes = append(nodes, authn.Routes(svc.Auth)...)
	}
	if svc.OAuthProvider != nil {
		nodes = append(nodes, provider.Routes(svc.OAuthProvider)...)
	}
	if svc.OAuthClient != nil {
		nodes = append(nodes, oauthclient.Routes(svc.OAuthClient)...)
	}
	return []router.Node{
		router.Layout(nodes...),
	}
}
