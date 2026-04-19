package build

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-sum/assets/config"
)

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
