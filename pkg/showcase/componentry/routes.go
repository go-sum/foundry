package componentry

import (
	"github.com/go-sum/componentry/icons"
	"github.com/go-sum/showcase"
	"github.com/go-sum/web/router"
)

// Config holds configuration for the showcase handler.
type Config struct {
	BasePath string
	Icons    *icons.Registry
	Page     showcase.PageFunc
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{BasePath: "/showcase/componentry"}
}

// Routes returns the router nodes for the showcase handler.
func Routes(cfg Config) []router.Node {
	h := newHandler(cfg)
	return []router.Node{
		router.Group(cfg.BasePath,
			router.GET("/components", "demos.showcase", h.Show),
			router.GET("/demo/search", "demos.search", h.Search),
			router.GET("/demo/validate", "demos.validate", h.Validate),
			router.GET("/demo/paginate", "demos.paginate", h.Paginate),
			router.GET("/demo/region", "demos.region", h.Region),
			router.GET("/demo/region/{id}", "demos.region-by-id", h.Region),
			router.GET("/demo/oob-toast", "demos.oob-toast", h.OOBToast),
		),
	}
}
