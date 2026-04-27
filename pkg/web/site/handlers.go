package site

import (
	"cmp"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
)

const (
	robotsCacheControl  = "public, max-age=86400"
	sitemapCacheControl = "public, max-age=3600"
)

// Handlers serves /robots.txt and /sitemap.xml.
type Handlers struct {
	site       *Site
	router     *router.Router
	robotsCfg  RobotsConfig
	sitemapCfg SitemapConfig
}

// NewHandlers constructs a Handlers from a Site, Router, and config values.
func NewHandlers(s *Site, rt *router.Router, robots RobotsConfig, sitemap SitemapConfig) *Handlers {
	return &Handlers{
		site:       s,
		router:     rt,
		robotsCfg:  robots,
		sitemapCfg: sitemap,
	}
}

// RobotsTxt generates and serves /robots.txt.
// The sitemap URL is derived from the site's AbsoluteURL.
// Cache-Control: public, max-age=86400 (24 hours).
func (h *Handlers) RobotsTxt(c *web.Context) (web.Response, error) {
	robotsCfg := h.robotsCfg
	if robotsCfg.SitemapURL == "" {
		robotsCfg.SitemapURL = h.site.AbsoluteURL("/sitemap.xml")
	}

	content := BuildRobots(robotsCfg)

	cc := cmp.Or(h.robotsCfg.CacheControl, robotsCacheControl)

	resp := web.Text(http.StatusOK, content)
	resp.Headers.Set("Cache-Control", cc)
	return resp, nil
}

// SitemapXML generates and serves /sitemap.xml from named routes and
// static pages declared in SitemapConfig.
// Cache-Control: public, max-age=3600 (1 hour).
func (h *Handlers) SitemapXML(c *web.Context) (web.Response, error) {
	entries := h.resolveEntries()

	data, err := BuildSitemap(entries)
	if err != nil {
		return web.Response{}, fmt.Errorf("sitemap.xml: %w", err)
	}

	cc := cmp.Or(h.sitemapCfg.CacheControl, sitemapCacheControl)

	resp := web.XML(http.StatusOK, data)
	resp.Headers.Set("Cache-Control", cc)
	return resp, nil
}

// resolveEntries assembles the full entry list from config:
//  1. Named routes resolved from the router (parameterized routes skipped).
//  2. Static pages with explicit paths prepended with the site base URL.
//
// Per-entry changefreq and priority fall back to SitemapConfig defaults.
func (h *Handlers) resolveEntries() []Entry {
	cfg := h.sitemapCfg
	routes := h.router.Routes()

	var entries []Entry

	for _, r := range cfg.Routes {
		var matched *router.Route
		for i := range routes {
			if routes[i].Name == r.Name {
				matched = &routes[i]
				break
			}
		}
		if matched == nil {
			continue
		}
		if matched.Method != http.MethodGet {
			continue
		}
		if strings.Contains(matched.Pattern, "{") {
			continue
		}

		changefreq := cmp.Or(r.ChangeFreq, cfg.DefaultChangeFreq)

		entries = append(entries, Entry{
			Loc:        h.site.AbsoluteURL(matched.Pattern),
			ChangeFreq: changefreq,
			Priority:   resolvePriority(r.Priority, cfg.DefaultPriority),
		})
	}

	for _, sp := range cfg.StaticPages {
		changefreq := cmp.Or(sp.ChangeFreq, cfg.DefaultChangeFreq)

		entries = append(entries, Entry{
			Loc:        h.site.AbsoluteURL(sp.Path),
			ChangeFreq: changefreq,
			Priority:   resolvePriority(sp.Priority, cfg.DefaultPriority),
		})
	}

	return entries
}

// resolvePriority returns the entry's explicit priority when set, the default
// when non-zero, or nil (omit <priority> from XML) when both are unset.
func resolvePriority(entry *float64, defaultPriority float64) *float64 {
	if entry != nil {
		return entry
	}
	if defaultPriority != 0 {
		return &defaultPriority
	}
	return nil
}
