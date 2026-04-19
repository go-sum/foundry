package build

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-sum/assets/config"
)

func DownloadJS(cfg *config.Config, client *http.Client, out io.Writer) error {
	for _, dl := range cfg.JS.Downloads {
		version := ResolveVersion(dl.Name, dl.Version)
		url := strings.ReplaceAll(dl.URL, "{version}", version)
		downloaded, err := FetchURL(client, url, dl.Target, out)
		if err != nil {
			return fmt.Errorf("js download %s: %w", dl.Name, err)
		}
		if downloaded {
			fmt.Fprintf(out, "  ✓ downloaded %s@%s -> %s\n", dl.Name, version, dl.Target)
		}
	}
	return nil
}

func BundleJS(cfg *config.Config, minify bool, out io.Writer) error {
	for _, bundle := range cfg.JS.Bundles {
		if err := bundleOne(bundle, minify, out); err != nil {
			return err
		}
	}
	return nil
}

func bundleOne(bundle config.JSBundle, minify bool, out io.Writer) error {
	if bundle.Entry == "" || bundle.Target == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(bundle.Target), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(bundle.Target), err)
	}
	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{bundle.Entry},
		Bundle:            true,
		Write:             true,
		Outfile:           bundle.Target,
		Platform:          api.PlatformBrowser,
		Format:            api.FormatIIFE,
		TreeShaking:       api.TreeShakingTrue,
		Target:            api.ES2017,
		MinifyWhitespace:  minify,
		MinifyIdentifiers: minify,
		MinifySyntax:      minify,
		LegalComments:     api.LegalCommentsNone,
	})
	if len(result.Errors) > 0 {
		return fmt.Errorf("bundle %s: %s", bundle.Entry, result.Errors[0].Text)
	}
	fmt.Fprintf(out, "  ✓ bundled %s -> %s\n", bundle.Entry, bundle.Target)
	return nil
}

func RemoveStaleJS(cfg *config.Config, out io.Writer) error {
	managed := make(map[string]bool, len(cfg.JS.Downloads)+len(cfg.JS.Bundles))
	dirs := make(map[string]bool, len(cfg.JS.Downloads)+len(cfg.JS.Bundles))
	for _, dl := range cfg.JS.Downloads {
		if dl.Target == "" {
			continue
		}
		managed[filepath.Clean(dl.Target)] = true
		dirs[filepath.Dir(dl.Target)] = true
	}
	for _, bundle := range cfg.JS.Bundles {
		if bundle.Target == "" {
			continue
		}
		managed[filepath.Clean(bundle.Target)] = true
		dirs[filepath.Dir(bundle.Target)] = true
	}
	for dir := range dirs {
		outputs, err := filepath.Glob(filepath.Join(dir, "*.js"))
		if err != nil {
			return fmt.Errorf("glob %s: %w", dir, err)
		}
		for _, path := range outputs {
			clean := filepath.Clean(path)
			if managed[clean] {
				continue
			}
			if err := os.Remove(clean); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove stale %s: %w", clean, err)
			}
			fmt.Fprintf(out, "  ✗ removed stale %s\n", clean)
		}
	}
	return nil
}
