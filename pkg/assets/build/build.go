package build

import (
	"io"
	"net/http"

	"github.com/go-sum/assets/config"
)

// Options controls which asset types to build.
type Options struct {
	Minify bool
	JS     bool
	CSS    bool
	Fonts  bool
}

// DefaultOptions returns options that build all asset types.
func DefaultOptions() Options {
	return Options{JS: true, CSS: true, Fonts: true}
}

// Build runs the full asset build pipeline in sequence:
// stale JS removal → JS downloads → JS bundling → font downloads → CSS.
func Build(cfg *config.Config, opts Options, client *http.Client, out io.Writer) error {
	if opts.JS {
		if err := RemoveStaleJS(cfg, out); err != nil {
			return err
		}
		if err := DownloadJS(cfg, client, out); err != nil {
			return err
		}
		if err := BundleJS(cfg, opts.Minify, out); err != nil {
			return err
		}
	}
	if opts.Fonts {
		if err := DownloadFonts(cfg, client, out); err != nil {
			return err
		}
	}
	if opts.CSS {
		if err := BuildCSS(cfg, opts.Minify, out); err != nil {
			return err
		}
	}
	return nil
}
