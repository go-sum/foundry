package site_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/site"
)

// newTestSite creates a Site with the given base URL for use in tests.
func newTestSite(baseURL string) *site.Site {
	if baseURL == "" {
		baseURL = "https://example.com"
	}
	return site.New(site.Config{BaseURL: baseURL})
}

// newTestRouter creates a router with optional routes registered.
// Each route is registered via Handle(method, pattern, name, handler).
func newTestRouter(routes ...struct {
	method, pattern, name string
}) *router.Router {
	rt := router.New()
	noop := func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}
	for _, r := range routes {
		rt.Handle(r.method, r.pattern, r.name, noop)
	}
	return rt
}

// newTestContext creates a *web.Context for the given path.
func newTestContext(path string) *web.Context {
	u := &url.URL{Path: path}
	req := web.NewRequest(http.MethodGet, u)
	return web.NewContext(context.Background(), req)
}

// readBody reads all bytes from a web.Response body and returns them as a string.
func readBody(t *testing.T, resp web.Response) string {
	t.Helper()
	if resp.Body == nil {
		return ""
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	return string(b)
}

// --- RobotsTxt tests ---

func TestRobotsTxt(t *testing.T) {
	tests := []struct {
		name            string
		cfg             site.RobotsConfig
		wantStatus      int
		wantContentType string
		wantCacheCtrl   string
		wantBodyLines   []string
		noWantBodyLines []string
	}{
		{
			name:            "default config returns 200 with standard headers",
			cfg:             site.RobotsConfig{DefaultAllow: true},
			wantStatus:      http.StatusOK,
			wantContentType: "text/plain; charset=UTF-8",
			wantCacheCtrl:   "public, max-age=86400",
			wantBodyLines: []string{
				"User-agent: *",
				"Sitemap: https://example.com/sitemap.xml",
			},
		},
		{
			name:          "DefaultAllow false emits Disallow: /",
			cfg:           site.RobotsConfig{DefaultAllow: false},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "public, max-age=86400",
			wantBodyLines: []string{
				"User-agent: *",
				"Disallow: /",
			},
		},
		{
			name: "custom CacheControl overrides default",
			cfg: site.RobotsConfig{
				DefaultAllow: true,
				CacheControl: "public, max-age=3600",
			},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "public, max-age=3600",
		},
		{
			name: "custom SitemapURL used verbatim instead of auto-derived",
			cfg: site.RobotsConfig{
				DefaultAllow: true,
				SitemapURL:   "https://custom.example.com/sitemap.xml",
			},
			wantStatus: http.StatusOK,
			wantBodyLines: []string{
				"Sitemap: https://custom.example.com/sitemap.xml",
			},
		},
		{
			name: "auto-derived sitemap URL uses base URL",
			cfg:  site.RobotsConfig{DefaultAllow: true},
			wantBodyLines: []string{
				"Sitemap: https://example.com/sitemap.xml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSite("https://example.com")
			rt := newTestRouter()
			h := site.NewHandlers(s, rt, tt.cfg, site.SitemapConfig{})
			c := newTestContext("/robots.txt")

			resp, err := h.RobotsTxt(c)
			if err != nil {
				t.Fatalf("RobotsTxt() error = %v", err)
			}

			if tt.wantStatus != 0 && resp.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", resp.Status, tt.wantStatus)
			}

			if tt.wantContentType != "" {
				got := resp.Headers.Get("Content-Type")
				if got != tt.wantContentType {
					t.Errorf("Content-Type = %q, want %q", got, tt.wantContentType)
				}
			}

			if tt.wantCacheCtrl != "" {
				got := resp.Headers.Get("Cache-Control")
				if got != tt.wantCacheCtrl {
					t.Errorf("Cache-Control = %q, want %q", got, tt.wantCacheCtrl)
				}
			}

			body := readBody(t, resp)

			for _, line := range tt.wantBodyLines {
				if !strings.Contains(body, line) {
					t.Errorf("body missing %q\nGot:\n%s", line, body)
				}
			}
			for _, line := range tt.noWantBodyLines {
				if strings.Contains(body, line) {
					t.Errorf("body should not contain %q\nGot:\n%s", line, body)
				}
			}
		})
	}
}

func TestRobotsTxt_SitemapURLCallerValuePreserved(t *testing.T) {
	// When cfg.SitemapURL is set, the handler must not override it.
	s := newTestSite("https://mysite.example.org")
	rt := newTestRouter()
	cfg := site.RobotsConfig{
		DefaultAllow: true,
		SitemapURL:   "https://cdn.example.com/sitemap.xml",
	}
	h := site.NewHandlers(s, rt, cfg, site.SitemapConfig{})
	c := newTestContext("/robots.txt")

	resp, err := h.RobotsTxt(c)
	if err != nil {
		t.Fatalf("RobotsTxt() error = %v", err)
	}

	body := readBody(t, resp)

	if !strings.Contains(body, "Sitemap: https://cdn.example.com/sitemap.xml") {
		t.Errorf("expected caller-supplied sitemap URL, got body:\n%s", body)
	}
	if strings.Contains(body, "https://mysite.example.org/sitemap.xml") {
		t.Errorf("handler must not override caller-supplied SitemapURL, got body:\n%s", body)
	}
}

// --- SitemapXML tests ---

func TestSitemapXML(t *testing.T) {
	tests := []struct {
		name            string
		sitemapCfg      site.SitemapConfig
		routes          []struct{ method, pattern, name string }
		wantStatus      int
		wantContentType string
		wantCacheCtrl   string
		wantBodyContains []string
		noBodyContains  []string
	}{
		{
			name:            "empty config returns 200 with valid XML",
			sitemapCfg:      site.SitemapConfig{},
			wantStatus:      http.StatusOK,
			wantContentType: "application/xml; charset=UTF-8",
			wantCacheCtrl:   "public, max-age=3600",
			wantBodyContains: []string{
				`<?xml version="1.0" encoding="UTF-8"?>`,
				`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`,
			},
		},
		{
			name: "named GET route resolved to absolute URL",
			routes: []struct{ method, pattern, name string }{
				{http.MethodGet, "/", "home.show"},
			},
			sitemapCfg: site.SitemapConfig{
				Routes: []site.RouteEntry{
					{Name: "home.show"},
				},
			},
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"<loc>https://example.com/</loc>",
			},
		},
		{
			name: "parameterized route skipped",
			routes: []struct{ method, pattern, name string }{
				{http.MethodGet, "/users/{id}", "user.show"},
			},
			sitemapCfg: site.SitemapConfig{
				Routes: []site.RouteEntry{
					{Name: "user.show"},
				},
			},
			wantStatus: http.StatusOK,
			noBodyContains: []string{
				"<loc>https://example.com/users/",
			},
		},
		{
			name: "non-GET route skipped",
			routes: []struct{ method, pattern, name string }{
				{http.MethodPost, "/submit", "form.submit"},
			},
			sitemapCfg: site.SitemapConfig{
				Routes: []site.RouteEntry{
					{Name: "form.submit"},
				},
			},
			wantStatus: http.StatusOK,
			noBodyContains: []string{
				"<loc>https://example.com/submit</loc>",
			},
		},
		{
			name: "unknown route name in SitemapConfig silently skipped",
			sitemapCfg: site.SitemapConfig{
				Routes: []site.RouteEntry{
					{Name: "does.not.exist"},
				},
			},
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`,
			},
		},
		{
			name: "static page included as absolute URL",
			sitemapCfg: site.SitemapConfig{
				StaticPages: []site.StaticEntry{
					{Path: "/about"},
				},
			},
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"<loc>https://example.com/about</loc>",
			},
		},
		{
			name: "default changefreq applied to route entry without explicit changefreq",
			routes: []struct{ method, pattern, name string }{
				{http.MethodGet, "/news", "news.list"},
			},
			sitemapCfg: site.SitemapConfig{
				Routes: []site.RouteEntry{
					{Name: "news.list"},
				},
				DefaultChangeFreq: "daily",
			},
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"<changefreq>daily</changefreq>",
			},
		},
		{
			name: "explicit changefreq on route overrides default",
			routes: []struct{ method, pattern, name string }{
				{http.MethodGet, "/news", "news.list"},
			},
			sitemapCfg: site.SitemapConfig{
				Routes: []site.RouteEntry{
					{Name: "news.list", ChangeFreq: "weekly"},
				},
				DefaultChangeFreq: "daily",
			},
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"<changefreq>weekly</changefreq>",
			},
		},
		{
			name: "default priority applied when RouteEntry.Priority is nil and DefaultPriority > 0",
			routes: []struct{ method, pattern, name string }{
				{http.MethodGet, "/", "home.show"},
			},
			sitemapCfg: site.SitemapConfig{
				Routes: []site.RouteEntry{
					{Name: "home.show"},
				},
				DefaultPriority: 0.9,
			},
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"<priority>0.9</priority>",
			},
		},
		{
			name: "custom CacheControl overrides default",
			sitemapCfg: site.SitemapConfig{
				CacheControl: "public, max-age=7200",
			},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "public, max-age=7200",
		},
		{
			name: "multiple static pages all present",
			sitemapCfg: site.SitemapConfig{
				StaticPages: []site.StaticEntry{
					{Path: "/about"},
					{Path: "/contact"},
					{Path: "/pricing"},
				},
			},
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"<loc>https://example.com/about</loc>",
				"<loc>https://example.com/contact</loc>",
				"<loc>https://example.com/pricing</loc>",
			},
		},
		{
			name: "GET route with no param included, parameterized route skipped",
			routes: []struct{ method, pattern, name string }{
				{http.MethodGet, "/blog", "blog.list"},
				{http.MethodGet, "/blog/{slug}", "blog.show"},
			},
			sitemapCfg: site.SitemapConfig{
				Routes: []site.RouteEntry{
					{Name: "blog.list"},
					{Name: "blog.show"},
				},
			},
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"<loc>https://example.com/blog</loc>",
			},
			noBodyContains: []string{
				"<loc>https://example.com/blog/{slug}</loc>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSite("https://example.com")
			rt := newTestRouter(tt.routes...)
			h := site.NewHandlers(s, rt, site.RobotsConfig{DefaultAllow: true}, tt.sitemapCfg)
			c := newTestContext("/sitemap.xml")

			resp, err := h.SitemapXML(c)
			if err != nil {
				t.Fatalf("SitemapXML() error = %v", err)
			}

			if tt.wantStatus != 0 && resp.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", resp.Status, tt.wantStatus)
			}

			if tt.wantContentType != "" {
				got := resp.Headers.Get("Content-Type")
				if got != tt.wantContentType {
					t.Errorf("Content-Type = %q, want %q", got, tt.wantContentType)
				}
			}

			if tt.wantCacheCtrl != "" {
				got := resp.Headers.Get("Cache-Control")
				if got != tt.wantCacheCtrl {
					t.Errorf("Cache-Control = %q, want %q", got, tt.wantCacheCtrl)
				}
			}

			body := readBody(t, resp)

			for _, want := range tt.wantBodyContains {
				if !strings.Contains(body, want) {
					t.Errorf("body missing %q\nGot:\n%s", want, body)
				}
			}
			for _, noWant := range tt.noBodyContains {
				if strings.Contains(body, noWant) {
					t.Errorf("body should not contain %q\nGot:\n%s", noWant, body)
				}
			}
		})
	}
}

func TestNewHandlers_NilContextDoesNotPanic(t *testing.T) {
	s := newTestSite("https://example.com")
	rt := newTestRouter()
	h := site.NewHandlers(s, rt, site.RobotsConfig{DefaultAllow: true}, site.SitemapConfig{})

	// Handlers must not panic when called with a nil context.
	// Both RobotsTxt and SitemapXML accept *web.Context — passing nil is a
	// boundary condition that should not crash.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("handler panicked with nil context: %v", r)
		}
	}()

	_, _ = h.RobotsTxt(nil)
}

func TestSitemapXML_ContentTypeIsXML(t *testing.T) {
	s := newTestSite("https://example.com")
	rt := newTestRouter()
	h := site.NewHandlers(s, rt, site.RobotsConfig{}, site.SitemapConfig{})
	c := newTestContext("/sitemap.xml")

	resp, err := h.SitemapXML(c)
	if err != nil {
		t.Fatalf("SitemapXML() error = %v", err)
	}

	if got, want := resp.Headers.Get("Content-Type"), "application/xml; charset=UTF-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

func TestRobotsTxt_ContentTypeIsPlainText(t *testing.T) {
	s := newTestSite("https://example.com")
	rt := newTestRouter()
	h := site.NewHandlers(s, rt, site.RobotsConfig{DefaultAllow: true}, site.SitemapConfig{})
	c := newTestContext("/robots.txt")

	resp, err := h.RobotsTxt(c)
	if err != nil {
		t.Fatalf("RobotsTxt() error = %v", err)
	}

	if got, want := resp.Headers.Get("Content-Type"), "text/plain; charset=UTF-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

func TestRobotsTxt_DefaultCacheControl(t *testing.T) {
	s := newTestSite("https://example.com")
	rt := newTestRouter()
	// Empty CacheControl in config → handler uses default.
	h := site.NewHandlers(s, rt, site.RobotsConfig{DefaultAllow: true, CacheControl: ""}, site.SitemapConfig{})
	c := newTestContext("/robots.txt")

	resp, err := h.RobotsTxt(c)
	if err != nil {
		t.Fatalf("RobotsTxt() error = %v", err)
	}

	if got, want := resp.Headers.Get("Cache-Control"), "public, max-age=86400"; got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}

func TestSitemapXML_DefaultCacheControl(t *testing.T) {
	s := newTestSite("https://example.com")
	rt := newTestRouter()
	// Empty CacheControl in config → handler uses default.
	h := site.NewHandlers(s, rt, site.RobotsConfig{}, site.SitemapConfig{CacheControl: ""})
	c := newTestContext("/sitemap.xml")

	resp, err := h.SitemapXML(c)
	if err != nil {
		t.Fatalf("SitemapXML() error = %v", err)
	}

	if got, want := resp.Headers.Get("Cache-Control"), "public, max-age=3600"; got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}
