package app

import (
	"fmt"
	"log/slog"

	config "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/internal/features/home"
	"github.com/go-sum/foundry/internal/features/oauthclient"
	"github.com/go-sum/foundry/pkg/auth/provider"
	authweb "github.com/go-sum/foundry/pkg/auth/web"
	"github.com/go-sum/foundry/pkg/docs"
	"github.com/go-sum/foundry/pkg/showcase"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/health"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/site"
)

type RouteDeps struct {
	HealthHandler web.Handler
	Public        PublicRouteDeps
	API           APIRouteDeps
}

type PublicRouteDeps struct {
	Middleware     []web.Middleware
	CSRFMiddleware web.Middleware
	MetaNodes      []router.Node
	FeatureNodes   []router.Node
	PackageNodes   []router.Node
}

type APIRouteDeps struct {
	Middleware          []web.Middleware
	PublicNodes         []router.Node
	ProtectedMiddleware []web.Middleware
	ProtectedNodes      []router.Node
}

func buildRouteDeps(rt *router.Router, cfg RouteConfig, sec Security, svc Services, publicDir string, s *site.Site, pres Presentation) (RouteDeps, error) {
	cfg = applyRouteDefaults(cfg)

	metaHandlers := site.NewHandlers(s, rt,
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
	authRateLimit, err := authRateLimitMiddleware(sec, svc)
	if err != nil {
		return RouteDeps{}, fmt.Errorf("auth rate limit middleware: %w", err)
	}

	protectedMiddleware := []web.Middleware{secure.CSRF(sec.CSRF)}
	if authRateLimit != nil {
		protectedMiddleware = append(protectedMiddleware, authRateLimit)
	}

	tokenPath := provider.DefaultRouteConfig().Token.Pattern
	if svc.OAuthProvider != nil {
		tokenPath = svc.OAuthProvider.RouteConfig().Token.Pattern
	}

	publicPage := pageRenderer(pres.ViewOpts)
	return RouteDeps{
		HealthHandler: health.Handler(healthCheckers(svc)...),
		Public: PublicRouteDeps{
			Middleware:     contentMiddleware(sec),
			CSRFMiddleware: secure.CSRF(sec.CSRF),
			MetaNodes:      site.Routes(metaHandlers),
			FeatureNodes: []router.Node{
				router.GET(cfg.Home.Pattern, cfg.Home.Name, home.NewHandler(homeServiceChecks(svc), pres.ViewOpts...).Show),
				router.GET(cfg.ContactForm.Pattern, cfg.ContactForm.Name, contactForm),
				router.POST(cfg.ContactSubmit.Pattern, cfg.ContactSubmit.Name, contactSubmit),
			},
			PackageNodes: showcase.Routes(showcase.Config{
				Icons: pres.Icons,
				DB:    svc.DBPool,
				KV:    svc.KVStore,
				Page:  publicPage,
			}),
		},
		API: APIRouteDeps{
			Middleware:          apiMiddleware(sec, tokenPath),
			PublicNodes:         append(protectedDocs(svc.Auth, publicDir), routesFrom(svc.OAuthProvider, provider.PublicRoutes)...),
			ProtectedMiddleware: protectedMiddleware,
			ProtectedNodes:      authNodes(cfg, svc),
		},
	}, nil
}

func authRateLimitMiddleware(sec Security, svc Services) (web.Middleware, error) {
	if svc.RateLimiter == nil {
		return nil, nil
	}
	return ratelimit.Middleware(ratelimit.MiddlewareConfig{
		Limiter:    svc.RateLimiter,
		Profile:    config.RateLimitRoutesAuth,
		KeyFunc:    sec.RateLimitKey,
		FailClosed: true,
		OnError: func(err error, c *web.Context) {
			slog.ErrorContext(c.Context(), "auth rate limit store error",
				"error", err,
				"request_id", web.RequestID(c),
			)
		},
	})
}

func authNodes(cfg RouteConfig, svc Services) []router.Node {
	return router.Nodes(
		routesFrom(svc.Auth, authweb.Routes),
		routesFrom(svc.OAuthProvider, provider.ProtectedRoutes),
		oauthClientNodes(cfg, svc.OAuthClient),
	)
}

func oauthClientNodes(cfg RouteConfig, h *oauthclient.Handler) []router.Node {
	if h == nil {
		return nil
	}
	return []router.Node{
		router.GET(cfg.OAuthConnect.Pattern, cfg.OAuthConnect.Name, h.Connect),
		router.GET(cfg.OAuthCallback.Pattern, cfg.OAuthCallback.Name, h.Callback),
	}
}

func protectedDocs(auth *authweb.Module, publicDir string) []router.Node {
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
