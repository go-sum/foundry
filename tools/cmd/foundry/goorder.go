package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-sum/foundry/tools/internal/godeclsort"
	"github.com/spf13/cobra"
)

func newGoOrderCommand() *cobra.Command {
	var write bool
	var list bool

	cmd := &cobra.Command{
		Use:   "go-order <file.go> [file.go...]",
		Short: "Reorder top-level Go declarations into type, const, func groups",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGoOrder(args, write, list, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVarP(&write, "write", "w", false, "write result back to the file")
	cmd.Flags().BoolVarP(&list, "list", "l", false, "print files that would change without writing them")

	return cmd
}

func runGoOrder(paths []string, write, list bool, out io.Writer) error {
	if write && list {
		return fmt.Errorf("go-order: --write and --list cannot be used together")
	}

	for _, arg := range paths {
		path := filepath.Clean(arg)
		if filepath.Ext(path) != ".go" {
			return fmt.Errorf("go-order: %s is not a .go file", arg)
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("go-order: read %s: %w", path, err)
		}

		formatted, err := godeclsort.ReorderSource(src)
		if err != nil {
			return fmt.Errorf("go-order: format %s: %w", path, err)
		}

		if list {
			if !bytes.Equal(src, formatted) {
				if _, err := fmt.Fprintln(out, path); err != nil {
					return fmt.Errorf("go-order: write stdout: %w", err)
				}
			}
			continue
		}

		if write {
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("go-order: stat %s: %w", path, err)
			}
			if err := os.WriteFile(path, formatted, info.Mode()); err != nil {
				return fmt.Errorf("go-order: write %s: %w", path, err)
			}
			continue
		}

		if _, err := out.Write(formatted); err != nil {
			return fmt.Errorf("go-order: write stdout: %w", err)
		}
	}

	return nil
}
