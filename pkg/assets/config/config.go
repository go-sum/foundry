package config

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	DefaultConfigPath   = ".assets.yaml"
	defaultSourceDir    = "static"
	defaultPublicDir    = "public"
	defaultPublicPrefix = "/public"
)

type Config struct {
	Paths   Paths                   `yaml:"paths"`
	JS      JSConfig                `yaml:"js"`
	CSS     []CSSConfig             `yaml:"css"`
	Sprites map[string]SpriteConfig `yaml:"sprites"`
	Fonts   FontConfig              `yaml:"fonts"`
}

type Paths struct {
	SourceDir    string `yaml:"source_dir"`
	PublicDir    string `yaml:"public_dir"`
	PublicPrefix string `yaml:"public_prefix"`
}

type JSConfig struct {
	Downloads []JSDownload `yaml:"downloads"`
	Bundles   []JSBundle   `yaml:"bundles"`
	Minify    []JSMinify   `yaml:"minify"`
}

type JSMinify struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

type JSDownload struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	URL     string `yaml:"url"`
	Target  string `yaml:"target"`
}

type JSBundle struct {
	Entry  string `yaml:"entry"`
	Target string `yaml:"target"`
}

type CSSConfig struct {
	Tool   string `yaml:"tool"`
	Input  string `yaml:"input"`
	Output string `yaml:"output"`
}

type FontConfig struct {
	Downloads []FontDownload `yaml:"downloads"`
}

type FontDownload struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	URL     string `yaml:"url"`
	Target  string `yaml:"target"`
}

type SpriteConfig struct {
	Enabled bool            `yaml:"enabled"`
	Target  string          `yaml:"target"`
	Sources []SourcesConfig `yaml:"sources"`
}

type SourcesConfig struct {
	Path  string   `yaml:"path"`
	Files []string `yaml:"files"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	cfg.Paths = cfg.Paths.withDefaults()
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving config path %s: %w", path, err)
	}
	baseDir := filepath.Dir(absPath)
	if !filepath.IsAbs(cfg.Paths.SourceDir) {
		cfg.Paths.SourceDir = filepath.Join(baseDir, cfg.Paths.SourceDir)
	}
	if !filepath.IsAbs(cfg.Paths.PublicDir) {
		cfg.Paths.PublicDir = filepath.Join(baseDir, cfg.Paths.PublicDir)
	}
	cfg.normalize()
	return &cfg, nil
}

func (p Paths) SourceRoot() string { return cleanPath(cmp.Or(p.SourceDir, defaultSourceDir)) }
func (p Paths) PublicRoot() string { return cleanPath(cmp.Or(p.PublicDir, defaultPublicDir)) }

func (p Paths) URLPrefix() string {
	prefix := strings.TrimSpace(p.PublicPrefix)
	prefix = cmp.Or(prefix, defaultPublicPrefix)
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if prefix != "/" {
		prefix = strings.TrimRight(prefix, "/")
	}
	return prefix
}

func (p Paths) PublicURL(rel string) string {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "/")
	if rel == "" {
		return p.URLPrefix()
	}
	return p.URLPrefix() + "/" + rel
}

func (p Paths) withDefaults() Paths {
	return Paths{
		SourceDir:    cmp.Or(strings.TrimSpace(p.SourceDir), defaultSourceDir),
		PublicDir:    cmp.Or(strings.TrimSpace(p.PublicDir), defaultPublicDir),
		PublicPrefix: cmp.Or(strings.TrimSpace(p.PublicPrefix), defaultPublicPrefix),
	}
}

func (cfg *Config) normalize() {
	for i, dl := range cfg.JS.Downloads {
		cfg.JS.Downloads[i].Target = resolvePublicPath(cfg.Paths, dl.Target)
	}
	for i, bundle := range cfg.JS.Bundles {
		cfg.JS.Bundles[i].Entry = resolveSourcePath(cfg.Paths, bundle.Entry)
		cfg.JS.Bundles[i].Target = resolvePublicPath(cfg.Paths, bundle.Target)
	}
	for i, m := range cfg.JS.Minify {
		cfg.JS.Minify[i].Source = resolveSourcePath(cfg.Paths, m.Source)
		cfg.JS.Minify[i].Target = resolveSourcePath(cfg.Paths, m.Target)
	}
	for i, entry := range cfg.CSS {
		cfg.CSS[i].Input = resolveSourcePath(cfg.Paths, entry.Input)
		cfg.CSS[i].Output = resolvePublicPath(cfg.Paths, entry.Output)
	}
	for i, dl := range cfg.Fonts.Downloads {
		cfg.Fonts.Downloads[i].Target = resolvePublicPath(cfg.Paths, dl.Target)
	}
	normalized := make(map[string]SpriteConfig, len(cfg.Sprites))
	for name, sprite := range cfg.Sprites {
		sprite.Target = resolvePublicPath(cfg.Paths, sprite.Target)
		for j, src := range sprite.Sources {
			sprite.Sources[j].Path = resolveSpriteSourcePath(cfg.Paths, src.Path)
		}
		normalized[name] = sprite
	}
	cfg.Sprites = normalized
}

func resolveSourcePath(p Paths, rel string) string {
	return resolveLocalPath(p.SourceRoot(), rel)
}

func resolvePublicPath(p Paths, rel string) string {
	return resolveLocalPath(p.PublicRoot(), rel)
}

func resolveSpriteSourcePath(p Paths, path string) string {
	if isRemotePath(path) {
		return path
	}
	if strings.HasPrefix(path, "file://") {
		return path
	}
	return resolveLocalPath(p.SourceRoot(), path)
}

func resolveLocalPath(base, rel string) string {
	if rel == "" || isRemotePath(rel) {
		return rel
	}
	if filepath.IsAbs(rel) {
		return cleanPath(rel)
	}
	return cleanPath(filepath.Join(base, rel))
}

func cleanPath(p string) string {
	return filepath.Clean(filepath.FromSlash(p))
}

func isRemotePath(p string) bool {
	return strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://")
}
