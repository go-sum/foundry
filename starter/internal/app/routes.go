package app

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-sum/foundry/internal/features/home"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/go-sum/foundry/pkg/docs"
	"github.com/go-sum/foundry/pkg/showcase/componentry"
	showcasedb "github.com/go-sum/foundry/pkg/showcase/db"
	showcasekv "github.com/go-sum/foundry/pkg/showcase/kv"
	showcasequeue "github.com/go-sum/foundry/pkg/showcase/queue"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/compress"
	"github.com/go-sum/foundry/pkg/web/etag"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/site"
	"github.com/go-sum/foundry/pkg/web/static"
	"github.com/jackc/pgx/v5/pgxpool"
	g "maragu.dev/gomponents"
)

// RegisterRoutes registers all application routes on the router.
func RegisterRoutes(rt *router.Router, sec Security, svc Services, assets static.AssetsConfig, publicDir string, s *site.Site, pres Presentation) error {
	if err := registerStaticRoutes(rt, assets); err != nil {
		return err
	}
	router.Register(rt, docs.Routes(docs.DefaultConfig(publicDir))...)
	if err := registerPublicRoutes(rt, sec, svc, s, pres); err != nil {
		return fmt.Errorf("public routes: %w", err)
	}
	return nil
}

func registerStaticRoutes(rt *router.Router, assets static.AssetsConfig) error {
	root, err := os.OpenRoot(assets.PublicDir)
	if err != nil {
		return fmt.Errorf("static: cannot open public dir: %w", err)
	}

	rawH := static.Handler(root, static.Options{
		Precompressed: true,
	})
	staticH := static.VersionedCacheControl("v")(rawH)

	router.Register(rt,
		router.Group(assets.URLPrefix,
			router.GET("/{rest...}", "static.assets", func(c *web.Context) (web.Response, error) {
				stripped := "/" + c.Param("rest")
				c.Request.URL.Path = stripped
				return staticH(c)
			}),
		),
	)
	return nil
}

type healthChecker interface {
	Check(ctx context.Context) error
}

type dbHealthChecker struct {
	pool *pgxpool.Pool
}

func (d *dbHealthChecker) Check(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func healthHandler(checkers ...healthChecker) web.Handler {
	return func(c *web.Context) (web.Response, error) {
		for _, ch := range checkers {
			if err := ch.Check(c.Context()); err != nil {
				return web.Response{}, web.ErrUnavailable("database unhealthy", err)
			}
		}
		return web.Text(http.StatusOK, "ok"), nil
	}
}

func unavailableHandler(feature string) web.Handler {
	return func(_ *web.Context) (web.Response, error) {
		return web.Response{}, web.ErrUnavailable(feature+" feature unavailable", nil)
	}
}

// Route name constants for the first-party OAuth client.
const (
	RouteOAuthConnect  = "auth.connect"
	RouteOAuthCallback = "auth.callback"
)

func registerPublicRoutes(rt *router.Router, sec Security, svc Services, s *site.Site, pres Presentation) error {
	var serviceChecks []home.Checker
	if svc.DBPool != nil {
		serviceChecks = append(serviceChecks, home.Checker{
			Name: "Database",
			Fn:   func(ctx context.Context) error { return svc.DBPool.Ping(ctx) },
		})
	}
	if svc.KVStore != nil {
		serviceChecks = append(serviceChecks, home.Checker{
			Name: "KV Store",
			Fn:   svc.KVStore.Ping,
		})
	}
	homeH := home.NewHandler(serviceChecks, pres.ViewOpts...)
	metaH := site.NewHandlers(s, rt,
		site.RobotsConfig{DefaultAllow: true},
		site.SitemapConfig{
			Routes: []site.RouteEntry{
				{Name: "home.show"},
				{Name: "demos.showcase"},
				{Name: "contact.form"},
			},
			DefaultChangeFreq: "weekly",
		},
	)

	var contactForm, contactSubmit web.Handler
	if svc.Contact != nil && svc.Contact.Handler != nil {
		contactForm = svc.Contact.Handler.Form
		contactSubmit = svc.Contact.Handler.Submit
	} else {
		contactForm = unavailableHandler("contact")
		contactSubmit = unavailableHandler("contact")
	}

	var checkers []healthChecker
	if svc.DBPool != nil {
		checkers = append(checkers, &dbHealthChecker{pool: svc.DBPool})
	}

	router.Register(rt,
		router.GET("/healthz", "health.check", healthHandler(checkers...)),
		router.Layout(router.Nodes(
			[]router.Node{
				router.Use(
					web.WithMaxBody(8<<20),
					web.MethodOverride(web.MethodOverrideConfig{}),
					compress.Middleware(compress.Config{}),
					etag.Middleware(etag.Config{}),
					secure.OriginGuard(secure.OriginGuardConfig{TrustedOrigins: sec.Origins}),
				),
				router.GET("/robots.txt", "meta.robots", metaH.RobotsTxt),
				router.GET("/sitemap.xml", "meta.sitemap", metaH.SitemapXML),
				router.GET("/", "home.show", homeH.Show),
				router.GET("/contact", "contact.form", contactForm),
				router.POST("/contact", "contact.submit", contactSubmit),
				router.Group("/account",
					router.Use(auth.RequireAuth(router.NewResolver(rt).Path(RouteOAuthConnect))),
				),
			},
			showcaseNodes(svc, pres),
		)...),
	)

	// Auth routes (identity provider — signin/signup/verify/signout).
	if svc.Auth != nil {
		router.Register(rt, auth.Routes(svc.Auth)...)
	}

	// OAuth Provider routes (Authorization Server — authorize/token/userinfo/discovery).
	if svc.OAuthProvider != nil {
		router.Register(rt, provider.Routes(svc.OAuthProvider)...)
	}

	// First-party OAuth client routes (connect + callback).
	if svc.OAuthClient != nil {
		router.Register(rt,
			router.GET("/auth/connect", RouteOAuthConnect, svc.OAuthClient.Connect),
			router.GET("/auth/callback", RouteOAuthCallback, svc.OAuthClient.Callback),
		)
	}

	return nil
}

func showcaseNodes(svc Services, pres Presentation) []router.Node {
	pageFn := func(c *web.Context, title string, content g.Node) (web.Response, error) {
		vr := view.NewRequest(c, pres.ViewOpts...)
		return view.Render(vr, vr.Page(title, content), nil)
	}

	showcaseCfg := componentry.DefaultConfig()
	showcaseCfg.Icons = pres.Icons
	showcaseCfg.Page = pageFn
	nodes := componentry.Routes(showcaseCfg)

	if svc.DBPool != nil {
		dbCfg := showcasedb.DefaultConfig()
		dbCfg.Pool = svc.DBPool
		dbCfg.Page = pageFn
		nodes = append(nodes, showcasedb.Routes(dbCfg)...)

		queueCfg := showcasequeue.DefaultConfig()
		queueCfg.Pool = svc.DBPool
		queueCfg.Page = pageFn
		nodes = append(nodes, showcasequeue.Routes(queueCfg)...)
	}

	if svc.KVStore != nil {
		kvCfg := showcasekv.DefaultConfig()
		kvCfg.Store = svc.KVStore
		kvCfg.Page = pageFn
		nodes = append(nodes, showcasekv.Routes(kvCfg)...)
	}

	return nodes
}
