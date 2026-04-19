// Package site provides site-identity configuration and meta-endpoint
// handlers (robots.txt, sitemap.xml).
package site

import (
	"net/url"
	"strings"
)

// Config holds site-identity configuration.
type Config struct {
	BaseURL         string   `validate:"required,url"`
	OriginAllowlist []string
}

// DefaultConfig returns an empty site config. BaseURL must come from env.
func DefaultConfig() Config { return Config{} }

// Site provides site-identity helpers built from a Config.
type Site struct {
	cfg    Config
	origin string // cached scheme+host
}

// New creates a Site from cfg. Panics if cfg.BaseURL is not a valid URL.
func New(cfg Config) *Site {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Host == "" {
		panic("web/site: invalid BaseURL: " + cfg.BaseURL)
	}
	return &Site{
		cfg:    cfg,
		origin: u.Scheme + "://" + u.Host,
	}
}

// Origin returns the scheme+host of the base URL (e.g., "https://example.com").
func (s *Site) Origin() string { return s.origin }

// AbsoluteURL resolves path relative to the base URL.
// Leading slashes on path are normalized.
func (s *Site) AbsoluteURL(path string) string {
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	p := "/" + strings.TrimLeft(path, "/")
	return base + p
}

// IsAllowedOrigin reports whether origin matches the site's own origin or
// appears in cfg.OriginAllowlist.
func (s *Site) IsAllowedOrigin(origin string) bool {
	if origin == s.origin {
		return true
	}
	for _, o := range s.cfg.OriginAllowlist {
		if o == origin {
			return true
		}
	}
	return false
}
