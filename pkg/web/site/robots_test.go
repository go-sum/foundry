package site_test

import (
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/web/site"
)

func TestBuildRobots(t *testing.T) {
	tests := []struct {
		name        string
		cfg         site.RobotsConfig
		wantLines   []string
		noWantLines []string
	}{
		{
			name: "DefaultAllow false blocks all crawlers",
			cfg:  site.RobotsConfig{DefaultAllow: false},
			wantLines: []string{
				"User-agent: *",
				"Disallow: /",
			},
			noWantLines: []string{
				"Disallow: /components",
				"Disallow: /admin",
				"Disallow: /profile",
				"Disallow: /signin",
				"Disallow: /signup",
				"Disallow: /health",
			},
		},
		{
			name: "DefaultAllow true with empty DisallowPaths uses DefaultDisallowPaths",
			cfg:  site.RobotsConfig{DefaultAllow: true, DisallowPaths: nil},
			wantLines: []string{
				"User-agent: *",
				"Disallow: /components",
				"Disallow: /admin",
				"Disallow: /profile",
				"Disallow: /signin",
				"Disallow: /signup",
				"Disallow: /health",
			},
			// "Disallow: /\n" is the exact whole-path block-all directive.
			// Must not be confused with "Disallow: /components" etc.
			noWantLines: []string{
				"Disallow: /\n",
			},
		},
		{
			name: "DefaultAllow true with custom paths uses only custom paths",
			cfg: site.RobotsConfig{
				DefaultAllow:  true,
				DisallowPaths: []string{"/private", "/api"},
			},
			wantLines: []string{
				"User-agent: *",
				"Disallow: /private",
				"Disallow: /api",
			},
			noWantLines: []string{
				"Disallow: /components",
				"Disallow: /admin",
				"Disallow: /profile",
				"Disallow: /signin",
				"Disallow: /signup",
				"Disallow: /health",
			},
		},
		{
			name: "SitemapURL set appends Sitemap directive",
			cfg: site.RobotsConfig{
				DefaultAllow: true,
				SitemapURL:   "https://example.com/sitemap.xml",
			},
			wantLines: []string{
				"User-agent: *",
				"Sitemap: https://example.com/sitemap.xml",
			},
			noWantLines: nil,
		},
		{
			name: "SitemapURL empty omits Sitemap directive",
			cfg: site.RobotsConfig{
				DefaultAllow: true,
				SitemapURL:   "",
			},
			wantLines: []string{
				"User-agent: *",
			},
			noWantLines: []string{
				"Sitemap:",
			},
		},
		{
			name: "User-agent always present regardless of DefaultAllow false",
			cfg:  site.RobotsConfig{DefaultAllow: false},
			wantLines: []string{
				"User-agent: *",
			},
			noWantLines: nil,
		},
		{
			name: "User-agent always present regardless of DefaultAllow true",
			cfg:  site.RobotsConfig{DefaultAllow: true},
			wantLines: []string{
				"User-agent: *",
			},
			noWantLines: nil,
		},
		{
			name: "DefaultAllow false with SitemapURL appends Sitemap directive",
			cfg: site.RobotsConfig{
				DefaultAllow: false,
				SitemapURL:   "https://example.com/sitemap.xml",
			},
			wantLines: []string{
				"User-agent: *",
				"Disallow: /",
				"Sitemap: https://example.com/sitemap.xml",
			},
			noWantLines: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := site.BuildRobots(tt.cfg)

			for _, line := range tt.wantLines {
				if !strings.Contains(got, line) {
					t.Errorf("BuildRobots() output missing expected line %q\nGot:\n%s", line, got)
				}
			}
			for _, line := range tt.noWantLines {
				if strings.Contains(got, line) {
					t.Errorf("BuildRobots() output should not contain line %q\nGot:\n%s", line, got)
				}
			}
		})
	}
}

func TestDefaultDisallowPaths(t *testing.T) {
	want := []string{
		"/components",
		"/admin",
		"/profile",
		"/signin",
		"/signup",
		"/health",
	}

	if len(site.DefaultDisallowPaths) != len(want) {
		t.Fatalf("DefaultDisallowPaths len = %d, want %d", len(site.DefaultDisallowPaths), len(want))
	}
	for i, p := range want {
		if site.DefaultDisallowPaths[i] != p {
			t.Errorf("DefaultDisallowPaths[%d] = %q, want %q", i, site.DefaultDisallowPaths[i], p)
		}
	}
}
