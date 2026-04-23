package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	docs "github.com/go-sum/docs"
	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	var source, destination string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build Hugo documentation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return build(source, destination)
		},
	}

	cmd.Flags().StringVar(&source, "source", ".docs", "Hugo source directory")
	cmd.Flags().StringVar(&destination, "destination", filepath.Join("public", docs.DefaultDocsDir), "output directory for built documentation")

	return cmd
}

// build invokes Hugo to compile source into destination.
// Any stale output under destination is removed before building.
// Relative paths are resolved against the current working directory.
func build(source, destination string) error {
	if _, err := os.Stat(source); os.IsNotExist(err) {
		fmt.Printf("%s not found — scaffolding from template\n", source)
		if err := scaffold(source); err != nil {
			return err
		}
	}
	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve destination %s: %w", destination, err)
	}
	if err := os.RemoveAll(absDestination); err != nil {
		return fmt.Errorf("remove %s: %w", absDestination, err)
	}
	if err := os.MkdirAll(filepath.Dir(absDestination), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(absDestination), err)
	}

	hugo := exec.Command("hugo",
		"--source", source,
		"--destination", absDestination,
		"--quiet",
	)
	hugo.Stdout = os.Stdout
	hugo.Stderr = os.Stderr
	if err := hugo.Run(); err != nil {
		return fmt.Errorf("hugo: %w", err)
	}
	return nil
}
