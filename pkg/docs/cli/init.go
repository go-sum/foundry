package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed template
var templateFS embed.FS

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Scaffold a .docs/ Hugo source directory",
		Long: `Scaffold a barebones .docs/ directory in the current working directory.

The scaffolded directory contains layouts, CSS, JS, and a starter content page
ready to build with: go run ./pkg/docs/cli build`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return initDocs()
		},
	}
}

func initDocs() error {
	const target = ".docs"
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s already exists", target)
	}
	if err := scaffold(target); err != nil {
		return err
	}
	fmt.Printf("created %s/\n", target)
	fmt.Println("next steps:")
	fmt.Printf("  edit %s/hugo.toml to set the title\n", target)
	fmt.Printf("  add markdown files under %s/content/\n", target)
	fmt.Println("  go run ./pkg/docs/cli build")
	return nil
}

// scaffold copies the embedded Hugo template into target.
// Files with a .tmpl extension are written without that suffix (e.g. go.mod.tmpl → go.mod).
func scaffold(target string) error {
	err := fs.WalkDir(templateFS, "template", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("template", path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, strings.TrimSuffix(rel, ".tmpl"))
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("scaffold %s: %w", target, err)
	}
	return nil
}
