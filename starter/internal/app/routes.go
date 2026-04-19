package app

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	starterconfig "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/internal/features/demos"
	"github.com/go-sum/foundry/internal/features/hello"
	"github.com/go-sum/foundry/internal/features/home"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web"
	"github.com/go-sum/web/compress"
	"github.com/go-sum/web/etag"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/secure"
	"github.com/go-sum/web/session"
	"github.com/go-sum/web/site"
	"github.com/go-sum/web/static"
)

// RegisterRoutes registers all application routes on the router.
func RegisterRoutes(rt *router.Router, sec Security, assets static.AssetsConfig, s *site.Site) error {
	if err := registerStaticRoutes(rt, assets); err != nil {
		return err
	}
	registerPublicRoutes(rt, sec, s)
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

	prefix := assets.URLPrefix
	router.Register(rt,
		router.GroupNode(prefix,
			router.GET("/{rest...}", "static.assets", func(c *web.Context) (web.Response, error) {
				stripped := "/" + c.Param("rest")
				c.Request.URL = &url.URL{Path: stripped}
				return staticH(c)
			}),
		),
	)
	return nil
}

func registerPublicRoutes(rt *router.Router, sec Security, s *site.Site) {
	navOpt := view.WithNavConfig(starterconfig.DefaultNav())
	homeH := home.NewHandler(rt.Routes, navOpt)
	helloH := hello.NewHandler(rt.Routes, navOpt)
	demosH := demos.NewHandler(rt.Routes, navOpt)
	metaH := site.NewHandlers(s, rt,
		site.RobotsConfig{DefaultAllow: true},
		site.SitemapConfig{
			Routes: []site.RouteEntry{
				{Name: "home.show"},
				{Name: "hello.greeting"},
				{Name: "demos.showcase"},
			},
			DefaultChangeFreq: "weekly",
		},
	)

	router.Register(rt,
		router.Layout(
			router.Use(
				web.WithMaxBody(8<<20),
				web.MethodOverride(web.MethodOverrideConfig{}),
				compress.Middleware(compress.Config{}),
				etag.Middleware(etag.Config{}),
				secure.OriginGuard(secure.OriginGuardConfig{TrustedOrigins: sec.Origins}),
			),
			router.GET("/robots.txt", "meta.robots", metaH.RobotsTxt),
			router.GET("/sitemap.xml", "meta.sitemap", metaH.SitemapXML),
			router.GET("/healthz", "health.check", func(_ *web.Context) (web.Response, error) {
				return web.Text(http.StatusOK, "ok"), nil
			}),
			router.GET("/", "home.show", homeH.Show),
			router.GET("/hello/greeting", "hello.greeting", helloH.Greeting),
			router.GET("/hello/{name}", "hello.show", helloH.Show),

			router.GroupNode("/demos",
				router.GET("/", "demos.showcase", demosH.Show),
			),

			router.GroupNode("/account",
				router.Use(session.Guard(session.DefaultGuardConfig())),
			),
		),
	)
}
