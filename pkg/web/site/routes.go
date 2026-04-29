package site

import "github.com/go-sum/foundry/pkg/web/router"

const (
	RouteRobots  = "meta.robots"
	RouteSitemap = "meta.sitemap"
)

// Routes returns the route nodes owned by site metadata handlers.
func Routes(h *Handlers) []router.Node {
	return []router.Node{
		router.GET("/robots.txt", RouteRobots, h.RobotsTxt),
		router.GET("/sitemap.xml", RouteSitemap, h.SitemapXML),
	}
}
