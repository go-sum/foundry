package componentry

import (
	"github.com/go-sum/web"
	"github.com/go-sum/web/router"

	g "maragu.dev/gomponents"
)

// PageFunc renders a full HTML page. The starter provides this from its view
// layer; the package stays decoupled from starter/internal/view.
type PageFunc func(c *web.Context, title string, content g.Node) (web.Response, error)

// Config holds configuration for the showcase handler.
type Config struct {
	BasePath string
	Page     PageFunc // required for the Show (full-page) handler
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{BasePath: "/showcase/componentry"}
}

// Routes returns the router nodes for the showcase handler.
func Routes(cfg Config) []router.Node {
	h := newHandler(cfg)
	return []router.Node{
		router.GroupNode(cfg.BasePath,
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
