package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	docs "github.com/go-sum/docs"
	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	var source, destination string
	var overwrite bool

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build Hugo documentation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return build(source, destination, overwrite)
		},
	}

	cmd.Flags().StringVar(&source, "source", ".docs", "Hugo source directory")
	cmd.Flags().StringVar(&destination, "destination", filepath.Join("public", docs.DefaultDocsDir), "destination for documentation")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "remove destination directory before building to clear stale output")

	return cmd
}

// validateDestination rejects empty, filesystem-root, and single-component destinations
// to prevent accidental RemoveAll on critical paths.
func validateDestination(dest string) error {
	if dest == "" {
		return fmt.Errorf("destination must not be empty")
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	volRoot := filepath.VolumeName(abs) + string(filepath.Separator)
	if abs == volRoot || abs == "/" {
		return fmt.Errorf("destination must not be a filesystem root: %s", abs)
	}
	parts := strings.Split(filepath.ToSlash(strings.TrimPrefix(filepath.ToSlash(abs), "/")), "/")
	nonEmpty := 0
	for _, p := range parts {
		if p != "" && p != "." {
			nonEmpty++
		}
	}
	if nonEmpty < 2 {
		return fmt.Errorf("destination path must have at least 2 components, got: %s", abs)
	}
	return nil
}

// build invokes Hugo to compile source into destination.
// When overwrite is true, destination is removed first to clear stale output.
// Relative paths are resolved against the current working directory.
func build(source, destination string, overwrite bool) error {
	if err := validateDestination(destination); err != nil {
		return err
	}
	if _, err := exec.LookPath("hugo"); err != nil {
		return fmt.Errorf("hugo not found on PATH — install from https://gohugo.io/installation/")
	}
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
	if overwrite {
		if err := os.RemoveAll(absDestination); err != nil {
			return fmt.Errorf("remove %s: %w", absDestination, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(absDestination), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(absDestination), err)
	}

	hugo := exec.Command("hugo", "--source", source, "--destination", absDestination, "--quiet")
	hugo.Stdout = os.Stdout
	hugo.Stderr = os.Stderr
	if err := hugo.Run(); err != nil {
		return fmt.Errorf("hugo: %w", err)
	}
	return nil
}
