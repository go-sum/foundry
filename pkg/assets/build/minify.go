package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-sum/foundry/pkg/assets/config"
)

// MinifyJS minifies each source JS file into its target path using esbuild's
// Transform API. Unlike BundleJS, no import resolution or bundling is performed —
// each source file is transformed in isolation. Intended for self-contained
// inline scripts that are embedded via go:embed.
func MinifyJS(cfg *config.Config, out io.Writer) error {
	for _, entry := range cfg.JS.Minify {
		if err := minifyJSOne(entry, out); err != nil {
			return err
		}
	}
	return nil
}

func minifyJSOne(entry config.JSMinify, out io.Writer) error {
	if entry.Source == "" || entry.Target == "" {
		return nil
	}
	src, err := os.ReadFile(entry.Source)
	if err != nil {
		return fmt.Errorf("minify js: read %s: %w", entry.Source, err)
	}
	result := api.Transform(string(src), api.TransformOptions{
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		LegalComments:     api.LegalCommentsNone,
		Target:            api.ES2020,
	})
	if len(result.Errors) > 0 {
		return fmt.Errorf("minify js: transform %s: %s", entry.Source, result.Errors[0].Text)
	}
	if err := os.MkdirAll(filepath.Dir(entry.Target), 0o755); err != nil {
		return fmt.Errorf("minify js: mkdir %s: %w", filepath.Dir(entry.Target), err)
	}
	if err := os.WriteFile(entry.Target, result.Code, 0o644); err != nil {
		return fmt.Errorf("minify js: write %s: %w", entry.Target, err)
	}
	fmt.Fprintf(out, "  ✓ minified %s -> %s\n", entry.Source, entry.Target)
	return nil
}
