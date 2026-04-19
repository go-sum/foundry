package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_valid(t *testing.T) {
	dir := t.TempDir()
	content := `
paths:
  source_dir: src
  public_dir: dist
  public_prefix: /assets
js:
  downloads:
    - name: htmx
      version: "2.0.4"
      url: "https://example.com/htmx.min.js"
      target: js/htmx.min.js
`
	path := filepath.Join(dir, ".assets.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Paths.SourceDir != "src" {
		t.Errorf("SourceDir = %q, want %q", cfg.Paths.SourceDir, "src")
	}
	if cfg.Paths.PublicDir != "dist" {
		t.Errorf("PublicDir = %q, want %q", cfg.Paths.PublicDir, "dist")
	}
	if cfg.Paths.PublicPrefix != "/assets" {
		t.Errorf("PublicPrefix = %q, want %q", cfg.Paths.PublicPrefix, "/assets")
	}
	if len(cfg.JS.Downloads) != 1 {
		t.Fatalf("JS.Downloads len = %d, want 1", len(cfg.JS.Downloads))
	}
	if cfg.JS.Downloads[0].Name != "htmx" {
		t.Errorf("JS.Downloads[0].Name = %q, want %q", cfg.JS.Downloads[0].Name, "htmx")
	}
}

func TestLoad_missing(t *testing.T) {
	_, err := Load("/nonexistent/.assets.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_defaults(t *testing.T) {
	dir := t.TempDir()
	content := `paths: {}`
	path := filepath.Join(dir, ".assets.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Paths.SourceRoot() != defaultSourceDir {
		t.Errorf("SourceRoot = %q, want %q", cfg.Paths.SourceRoot(), defaultSourceDir)
	}
	if cfg.Paths.PublicRoot() != defaultPublicDir {
		t.Errorf("PublicRoot = %q, want %q", cfg.Paths.PublicRoot(), defaultPublicDir)
	}
	if cfg.Paths.URLPrefix() != defaultPublicPrefix {
		t.Errorf("URLPrefix = %q, want %q", cfg.Paths.URLPrefix(), defaultPublicPrefix)
	}
}

func TestPaths_URLPrefix(t *testing.T) {
	tests := []struct {
		prefix string
		want   string
	}{
		{prefix: "/public", want: "/public"},
		{prefix: "/public/", want: "/public"},
		{prefix: "public", want: "/public"},
		{prefix: "/", want: "/"},
		{prefix: "", want: "/public"},
		{prefix: "  /assets  ", want: "/assets"},
	}
	for _, tt := range tests {
		p := Paths{PublicPrefix: tt.prefix}
		p = p.withDefaults()
		if got := p.URLPrefix(); got != tt.want {
			t.Errorf("URLPrefix(%q) = %q, want %q", tt.prefix, got, tt.want)
		}
	}
}

func TestPaths_PublicURL(t *testing.T) {
	tests := []struct {
		prefix string
		rel    string
		want   string
	}{
		{prefix: "/public", rel: "js/app.js", want: "/public/js/app.js"},
		{prefix: "/public", rel: "/js/app.js", want: "/public/js/app.js"},
		{prefix: "/public", rel: "", want: "/public"},
		{prefix: "/assets", rel: "css/app.css", want: "/assets/css/app.css"},
	}
	for _, tt := range tests {
		p := Paths{PublicPrefix: tt.prefix}.withDefaults()
		if got := p.PublicURL(tt.rel); got != tt.want {
			t.Errorf("PublicURL(%q, %q) = %q, want %q", tt.prefix, tt.rel, got, tt.want)
		}
	}
}
