package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
)

//go:embed all:template
var templateFS embed.FS

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Scaffold a .assets.yaml configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
	}
}

func initConfig() error {
	const target = ".assets.yaml"
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s already exists", target)
	}
	err := fs.WalkDir(templateFS, "template", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("scaffold %s: %w", target, err)
	}
	fmt.Printf("created %s\n", target)
	fmt.Println("next steps:")
	fmt.Println("  edit .assets.yaml to configure your assets")
	fmt.Println("  go run github.com/go-sum/foundry/pkg/assets/cli build all")
	fmt.Println("  go run github.com/go-sum/foundry/pkg/assets/cli sprites")
	return nil
}
