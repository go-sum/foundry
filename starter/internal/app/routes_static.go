package app

import (
	"fmt"
	"os"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/static"
)

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
				c.Request.URL.Path = "/" + c.Param("rest")
				return staticH(c)
			}),
		),
	)
	return nil
}
