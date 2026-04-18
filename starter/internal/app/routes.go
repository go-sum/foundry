package app

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/go-sum/foundry/internal/features/hello"
	"github.com/go-sum/foundry/internal/features/home"
	"github.com/go-sum/web"
	"github.com/go-sum/web/compress"
	"github.com/go-sum/web/etag"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/secure"
	"github.com/go-sum/web/session"
	"github.com/go-sum/web/static"
)

// RegisterRoutes registers all application routes on the router.
func RegisterRoutes(rt *router.Router, sec Security, assets static.AssetsConfig) error {
	if err := registerStaticRoutes(rt, assets); err != nil {
		return err
	}
	registerPublicRoutes(rt, sec)
	return nil
}

func registerStaticRoutes(rt *router.Router, assets static.AssetsConfig) error {
	root, err := os.OpenRoot(assets.PublicDir)
	if err != nil {
		return fmt.Errorf("static: cannot open public dir: %w", err)
	}

	staticH := static.Handler(root, static.Options{
		CacheControl:  "public, max-age=31536000, immutable",
		Precompressed: true,
	})

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

func registerPublicRoutes(rt *router.Router, sec Security) {
	homeH := home.NewHandler(rt.Routes)
	helloH := hello.NewHandler(rt.Routes)

	router.Register(rt,
		router.Layout(
			router.Use(
				web.WithMaxBody(8<<20),
				web.MethodOverride(web.MethodOverrideConfig{}),
				compress.Middleware(compress.Config{}),
				etag.Middleware(etag.Config{}),
				secure.OriginGuard(secure.OriginGuardConfig{TrustedOrigins: sec.Origins}),
			),
			router.GET("/healthz", "health.check", func(_ *web.Context) (web.Response, error) {
				return web.Text(http.StatusOK, "ok"), nil
			}),
			router.GET("/", "home.show", homeH.Show),
			router.GET("/hello/greeting", "hello.greeting", helloH.Greeting),
			router.GET("/hello/{name}", "hello.show", helloH.Show),

			router.GroupNode("/account",
				router.Use(session.Guard(session.DefaultGuardConfig())),
			),
		),
	)
}
