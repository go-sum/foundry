package docs

import "github.com/go-sum/web/router"

// Routes returns the router nodes for the documentation handler.
func Routes(cfg Config) []router.Node {
	h := NewHandler(cfg)
	return []router.Node{
		router.GroupNode(cfg.BasePath,
			router.GET("", "docs.index", h.Serve),
			router.GET("/{path...}", "docs.show", h.Serve),
		),
	}
}
