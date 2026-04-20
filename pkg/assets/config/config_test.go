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
	if cfg.Paths.SourceDir != filepath.Join(dir, "src") {
		t.Errorf("SourceDir = %q, want %q", cfg.Paths.SourceDir, filepath.Join(dir, "src"))
	}
	if cfg.Paths.PublicDir != filepath.Join(dir, "dist") {
		t.Errorf("PublicDir = %q, want %q", cfg.Paths.PublicDir, filepath.Join(dir, "dist"))
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
	if cfg.Paths.SourceRoot() != filepath.Join(dir, defaultSourceDir) {
		t.Errorf("SourceRoot = %q, want %q", cfg.Paths.SourceRoot(), filepath.Join(dir, defaultSourceDir))
	}
	if cfg.Paths.PublicRoot() != filepath.Join(dir, defaultPublicDir) {
		t.Errorf("PublicRoot = %q, want %q", cfg.Paths.PublicRoot(), filepath.Join(dir, defaultPublicDir))
	}
	if cfg.Paths.URLPrefix() != defaultPublicPrefix {
		t.Errorf("URLPrefix = %q, want %q", cfg.Paths.URLPrefix(), defaultPublicPrefix)
	}
}

func TestLoad_resolves_relative_to_yaml_dir(t *testing.T) {
	parent := t.TempDir()
	sub := filepath.Join(parent, "subdir")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `
paths:
  source_dir: assets
  public_dir: out
`
	yamlPath := filepath.Join(sub, ".assets.yaml")
	if err := os.WriteFile(yamlPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(parent); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Paths.SourceDir != filepath.Join(sub, "assets") {
		t.Errorf("SourceDir = %q, want %q", cfg.Paths.SourceDir, filepath.Join(sub, "assets"))
	}
	if cfg.Paths.PublicDir != filepath.Join(sub, "out") {
		t.Errorf("PublicDir = %q, want %q", cfg.Paths.PublicDir, filepath.Join(sub, "out"))
	}
}

func TestLoad_absolute_paths_unchanged(t *testing.T) {
	dir := t.TempDir()
	absSource := filepath.Join(dir, "abs-source")
	absPublic := filepath.Join(dir, "abs-public")
	content := "paths:\n  source_dir: " + absSource + "\n  public_dir: " + absPublic + "\n"
	yamlPath := filepath.Join(dir, ".assets.yaml")
	if err := os.WriteFile(yamlPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Paths.SourceDir != absSource {
		t.Errorf("SourceDir = %q, want %q", cfg.Paths.SourceDir, absSource)
	}
	if cfg.Paths.PublicDir != absPublic {
		t.Errorf("PublicDir = %q, want %q", cfg.Paths.PublicDir, absPublic)
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
