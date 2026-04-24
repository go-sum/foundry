package app

import (
	"fmt"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sum/db"
	"github.com/go-sum/docs"
	"github.com/go-sum/foundry/config"
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
func RegisterRoutes(rt *router.Router, sec Security, svc Services, assets static.AssetsConfig, publicDir string, s *site.Site) error {
	if err := registerStaticRoutes(rt, assets); err != nil {
		return err
	}
	router.Register(rt, docs.Routes(docs.DefaultConfig(publicDir))...)
	if err := registerPublicRoutes(rt, sec, svc, s); err != nil {
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

	prefix := assets.URLPrefix
	router.Register(rt,
		router.GroupNode(prefix,
			router.GET("/{rest...}", "static.assets", func(c *web.Context) (web.Response, error) {
				stripped := "/" + c.Param("rest")
				c.Request.URL.Path = stripped
				return staticH(c)
			}),
		),
	)
	return nil
}

func healthHandler(pool *pgxpool.Pool, tables []string) web.Handler {
	return func(c *web.Context) (web.Response, error) {
		if pool != nil {
			if err := db.Health(c.Context(), pool, tables...); err != nil {
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

func registerPublicRoutes(rt *router.Router, sec Security, svc Services, s *site.Site) error {
	navOpt := view.WithNavConfig(config.DefaultNav())
	homeH := home.NewHandler(rt, navOpt)
	helloH := hello.NewHandler(rt, navOpt)
	demosH := demos.NewHandler(rt, navOpt)
	metaH := site.NewHandlers(s, rt,
		site.RobotsConfig{DefaultAllow: true},
		site.SitemapConfig{
			Routes: []site.RouteEntry{
				{Name: "home.show"},
				{Name: "hello.greeting"},
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

	var healthTables []string
	if svc.SchemaRegistry != nil {
		healthTables = svc.SchemaRegistry.HealthTables()
	}
	router.Register(rt,
		router.GET("/healthz", "health.check", healthHandler(svc.DBPool, healthTables)),
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
			router.GET("/", "home.show", homeH.Show),
			router.GET("/hello/greeting", "hello.greeting", helloH.Greeting),
			router.GET("/hello/{name}", "hello.show", helloH.Show),

			router.GroupNode("/componentry",
				router.GET("/_components", "demos.showcase", demosH.Show),
				router.GET("/demo/search", "demos.search", demosH.Search),
				router.GET("/demo/validate", "demos.validate", demosH.Validate),
				router.GET("/demo/paginate", "demos.paginate", demosH.Paginate),
				router.GET("/demo/region", "demos.region", demosH.Region),
				router.GET("/demo/region/{id}", "demos.region-by-id", demosH.Region),
				router.GET("/demo/oob-toast", "demos.oob-toast", demosH.OOBToast),
			),

			router.GET("/contact", "contact.form", contactForm),
			router.POST("/contact", "contact.submit", contactSubmit),

			router.GroupNode("/account",
				router.Use(session.Guard(session.DefaultGuardConfig())),
			),
		),
	)
	return nil
}
