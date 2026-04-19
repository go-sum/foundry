package build

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-sum/assets/config"
	"golang.org/x/sync/errgroup"
)

// SpriteOptions configures sprite generation.
type SpriteOptions struct {
	Name   string // empty = all enabled sprites
	DryRun bool
}

var (
	reViewBox     = regexp.MustCompile(`(?i)viewBox=["']([^"']+)["']`)
	reOuterSVG    = regexp.MustCompile(`(?si)<svg[^>]*>(.*)</svg>`)
	reOuterSVGTag = regexp.MustCompile(`(?si)^<svg([^>]*)>`)
	reScript      = regexp.MustCompile(`(?si)<script[^>]*>.*?</script>`)
	reEventAttr   = regexp.MustCompile(`(?i)\son\w+="[^"]*"`)
	rePresAttr    = regexp.MustCompile(`(?i)\b(fill|stroke|stroke-width|stroke-linecap|stroke-linejoin|stroke-dasharray|stroke-miterlimit|fill-rule|clip-rule)="([^"]*)"`)
)

func BuildSprites(cfg *config.Config, opts SpriteOptions, client *http.Client, out io.Writer) error {
	built, totalIcons := 0, 0
	for name, sprite := range cfg.Sprites {
		if !sprite.Enabled {
			continue
		}
		if opts.Name != "" && name != opts.Name {
			continue
		}
		if err := BuildSprite(name, sprite, client, opts.DryRun, out); err != nil {
			return err
		}
		built++
		for _, src := range sprite.Sources {
			totalIcons += len(src.Files)
		}
	}
	fmt.Fprintf(out, "Built %d sprite(s), %d total icons\n", built, totalIcons)
	return nil
}

func BuildSprite(name string, cfg config.SpriteConfig, client *http.Client, dryRun bool, out io.Writer) error {
	if !dryRun && allRemoteSources(cfg.Sources) {
		if _, err := os.Stat(cfg.Target); err == nil {
			fmt.Fprintf(out, "  ↷ %s: target exists, skipping (delete to force rebuild)\n", name)
			return nil
		}
	}

	type pair struct{ path, file string }
	var pairs []pair
	for _, src := range cfg.Sources {
		for _, file := range src.Files {
			pairs = append(pairs, pair{src.Path, file})
		}
	}

	symbols := make([]string, len(pairs))
	var eg errgroup.Group
	for i, p := range pairs {
		i, p := i, p
		eg.Go(func() error {
			data, err := fetchSVG(client, p.path, p.file)
			if err != nil {
				return fmt.Errorf("sprite %q, file %q: %w", name, p.file, err)
			}
			id := strings.TrimSuffix(p.file, ".svg")
			sym, err := processSVG(data, id)
			if err != nil {
				return fmt.Errorf("sprite %q, file %q: %w", name, p.file, err)
			}
			symbols[i] = sym
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">` + "\n")
	sb.WriteString("  <defs>\n")
	for _, sym := range symbols {
		sb.WriteString(sym)
		sb.WriteString("\n")
	}
	sb.WriteString("  </defs>\n")
	sb.WriteString("</svg>\n")

	output := sb.String()
	if dryRun {
		fmt.Fprintf(out, "--- [dry-run] %s -> %s (%d icons) ---\n%s", name, cfg.Target, len(pairs), output)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Target), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(cfg.Target), err)
	}
	if err := os.WriteFile(cfg.Target, []byte(output), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", cfg.Target, err)
	}
	fmt.Fprintf(out, "  ✓ %s -> %s (%d icons)\n", name, cfg.Target, len(pairs))
	return nil
}

func fetchSVG(client *http.Client, base, file string) (data []byte, err error) {
	switch {
	case strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://"):
		url := base + file
		resp, err := client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("GET %s: %w", url, err)
		}
		defer closeOnReturn(&err, resp.Body, "response body for %s", url)
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	case strings.HasPrefix(base, "file://"):
		path := filepath.Join(strings.TrimPrefix(base, "file://"), file)
		return os.ReadFile(path)
	default:
		return os.ReadFile(filepath.Join(base, file))
	}
}

func processSVG(data []byte, id string) (string, error) {
	s := string(data)

	viewBox := "0 0 24 24"
	if m := reViewBox.FindStringSubmatch(s); m != nil {
		viewBox = m[1]
	}

	var presAttrsBuf strings.Builder
	if m := reOuterSVGTag.FindStringSubmatch(s); m != nil {
		for _, match := range rePresAttr.FindAllStringSubmatch(m[1], -1) {
			fmt.Fprintf(&presAttrsBuf, " %s=%q", strings.ToLower(match[1]), match[2])
		}
	}
	presAttrs := presAttrsBuf.String()

	inner := s
	if m := reOuterSVG.FindStringSubmatch(s); m != nil {
		inner = m[1]
	}
	inner = reScript.ReplaceAllString(inner, "")
	inner = reEventAttr.ReplaceAllString(inner, "")

	var lines []string
	for _, line := range strings.Split(inner, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, "      "+line)
		}
	}

	if len(lines) == 0 {
		return fmt.Sprintf(`    <symbol id=%q viewBox=%q%s/>`, id, viewBox, presAttrs), nil
	}
	return fmt.Sprintf("    <symbol id=%q viewBox=%q%s>\n%s\n    </symbol>", id, viewBox, presAttrs, strings.Join(lines, "\n")), nil
}

func allRemoteSources(sources []config.SourcesConfig) bool {
	for _, src := range sources {
		if !strings.HasPrefix(src.Path, "http://") && !strings.HasPrefix(src.Path, "https://") {
			return false
		}
	}
	return len(sources) > 0
}
