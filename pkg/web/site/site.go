// Package site provides site-identity configuration and meta-endpoint
// handlers (robots.txt, sitemap.xml).
package site

import (
	"net/url"
	"strings"
)

// BuildAllowedHosts returns the list of hostnames the server should accept.
// It extracts the hostname from baseURL (if valid) and appends any
// comma-separated entries from extra. The config layer passes raw env values;
// all assembly logic lives here.
func BuildAllowedHosts(baseURL, extra string) []string {
	var hosts []string
	if baseURL != "" {
		u, err := url.Parse(baseURL)
		if err == nil && u.Host != "" {
			hosts = append(hosts, u.Hostname())
		}
	}
	for _, h := range strings.Split(extra, ",") {
		if h = strings.TrimSpace(h); h != "" {
			hosts = append(hosts, h)
		}
	}
	return hosts
}

// Config holds site-identity configuration.
type Config struct {
	BaseURL         string   `validate:"required,url"`
	OriginAllowlist []string
	// AllowedHosts lists hostnames the server accepts. Built by
	// BuildAllowedHosts from the BaseURL hostname and additional entries.
	// Environment overlays may append further hosts (e.g. localhost for
	// dev proxies).
	AllowedHosts []string
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
