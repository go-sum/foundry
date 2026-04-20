package build

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-sum/assets/config"
)

// BuildCSSIfChanged builds each CSS entry only when its tracked input files
// have changed since the last successful build. State is read from and written
// back to state. Unchanged entries are skipped with a log message.
func BuildCSSIfChanged(cfg *config.Config, minify bool, state *StateFile, out io.Writer) error {
	for _, entry := range cfg.CSS {
		patterns, err := CSSWatchPaths(entry.Input)
		if err != nil {
			// If we cannot determine watch paths, build unconditionally.
			if buildErr := buildCSSEntry(entry, minify, out); buildErr != nil {
				return buildErr
			}
			continue
		}

		files, err := ExpandGlobs("/", patterns)
		if err != nil {
			// Fall back to unconditional build on glob errors.
			if buildErr := buildCSSEntry(entry, minify, out); buildErr != nil {
				return buildErr
			}
			continue
		}

		key := "css:" + entry.Output
		changed, err := state.HasChanged(key, files)
		if err != nil || !changed {
			if err == nil {
				fmt.Fprintf(out, "  ↷ css %s: no changes, skipping\n", entry.Input)
				continue
			}
			// HasChanged error → rebuild (fail open)
		}

		if err := buildCSSEntry(entry, minify, out); err != nil {
			return err
		}
		_ = state.MarkBuilt(key, files)
	}
	return nil
}

func BuildCSS(cfg *config.Config, minify bool, out io.Writer) error {
	for _, entry := range cfg.CSS {
		if err := buildCSSEntry(entry, minify, out); err != nil {
			return err
		}
	}
	return nil
}

func buildCSSEntry(entry config.CSSConfig, minify bool, out io.Writer) error {
	if err := os.MkdirAll(filepath.Dir(entry.Output), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(entry.Output), err)
	}
	args := []string{"-i", entry.Input, "-o", entry.Output}
	if minify {
		args = append(args, "--minify")
	}
	cmd := exec.Command(entry.Tool, args...)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", entry.Tool, err)
	}
	fmt.Fprintf(out, "  ✓ css %s -> %s\n", entry.Input, entry.Output)
	return nil
}
