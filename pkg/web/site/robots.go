package site

import (
	"fmt"
	"strings"
)

// DefaultDisallowPaths is the list of paths disallowed when DefaultAllow is
// true and DisallowPaths is empty. These are internal surfaces or auth
// mutation endpoints that have no SEO value.
var DefaultDisallowPaths = []string{
	"/_components",
	"/admin",
	"/profile",
	"/signin",
	"/signup",
	"/health",
}

// RobotsConfig controls what BuildRobots emits.
type RobotsConfig struct {
	// DefaultAllow true means all crawlers are allowed by default; specific
	// paths in DisallowPaths are excluded.
	// DefaultAllow false emits "Disallow: /" to block all crawlers.
	DefaultAllow bool

	// DisallowPaths is the list of paths to disallow when DefaultAllow is
	// true. When nil or empty, DefaultDisallowPaths is used instead.
	DisallowPaths []string

	// SitemapURL is the absolute URL of the sitemap
	// (e.g. https://example.com/sitemap.xml). When non-empty, a
	// "Sitemap:" directive is appended to the output.
	// When empty, the handler derives the URL from the site's base URL
	// as "<base>/sitemap.xml".
	SitemapURL string

	// CacheControl is the Cache-Control header value served with /robots.txt.
	// Empty string uses the handler default ("public, max-age=86400").
	CacheControl string
}

// BuildRobots generates a robots.txt document from cfg.
// The output is always valid robots.txt — an empty config produces a
// permissive file that allows all crawlers without any disallow rules.
func BuildRobots(cfg RobotsConfig) string {
	var b strings.Builder
	b.WriteString("User-agent: *\n")

	if !cfg.DefaultAllow {
		b.WriteString("Disallow: /\n")
	} else {
		paths := cfg.DisallowPaths
		if len(paths) == 0 {
			paths = DefaultDisallowPaths
		}
		for _, p := range paths {
			fmt.Fprintf(&b, "Disallow: %s\n", p)
		}
	}

	if cfg.SitemapURL != "" {
		fmt.Fprintf(&b, "\nSitemap: %s\n", cfg.SitemapURL)
	}

	return b.String()
}
