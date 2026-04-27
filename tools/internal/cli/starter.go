package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-sum/foundry/tools/internal/starter"
	"github.com/spf13/cobra"
)

// Config holds the global CLI configuration shared by all subcommands.
type Config struct {
	DryRun   bool
	RepoRoot string
}

// NewStarterCmd returns the "starter" subcommand group.
func NewStarterCmd(cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "starter",
		Short: "Template cloning and verification",
	}

	cmd.AddCommand(
		newStarterCloneCmd(cfg),
		newStarterVerifyCmd(cfg),
		newStarterListCmd(cfg),
	)

	return cmd
}

func newStarterCloneCmd(cfg *Config) *cobra.Command {
	var (
		target string
		module string
		source string
	)

	cmd := &cobra.Command{
		Use:   "clone",
		Short: "Clone the foundry starter template into a new application directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			src := source
			if src == "" {
				src = resolveSourceRoot(cfg.RepoRoot)
			}

			opts := starter.CloneOptions{
				Source: src,
				Target: target,
				Module: module,
			}
			return starter.RunClone(opts, os.Stdout)
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "destination directory path (required)")
	cmd.Flags().StringVar(&module, "module", "", "new Go module path, e.g. github.com/myorg/myapp (required)")
	cmd.Flags().StringVar(&source, "source", "", "foundry repository root (default: auto-detect)")

	_ = cmd.MarkFlagRequired("target")
	_ = cmd.MarkFlagRequired("module")

	return cmd
}

func newStarterVerifyCmd(cfg *Config) *cobra.Command {
	var source string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Clone to a temp dir, build, and vet to verify the starter template is healthy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			src := source
			if src == "" {
				src = resolveSourceRoot(cfg.RepoRoot)
			}
			return starter.RunVerify(src, os.Stdout)
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "foundry repository root (default: auto-detect)")

	return cmd
}

func newStarterListCmd(cfg *Config) *cobra.Command {
	var source string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List files that would be copied during clone",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			src := source
			if src == "" {
				src = resolveSourceRoot(cfg.RepoRoot)
			}

			manifestPath := filepath.Join(src, "tools", "starter", "manifest.yaml")
			manifest, err := starter.LoadManifest(manifestPath)
			if err != nil {
				return err
			}

			files, err := gitLsFiles(src)
			if err != nil {
				return err
			}

			count := 0
			for _, f := range files {
				if starter.IsExcluded(manifest, f) {
					continue
				}
				fmt.Fprintln(cmd.OutOrStdout(), f)
				count++
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d files\n", count)
			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "foundry repository root (default: auto-detect)")

	return cmd
}

// resolveSourceRoot returns the source root: if repoRoot is already set by PersistentPreRunE,
// return it; otherwise fall back to FOUNDRY_ROOT or walk up from cwd.
func resolveSourceRoot(repoRoot string) string {
	if repoRoot != "" {
		return repoRoot
	}
	if v := os.Getenv("FOUNDRY_ROOT"); v != "" {
		return v
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

// gitLsFiles returns the list of file paths tracked by git in the given directory.
func gitLsFiles(dir string) ([]string, error) {
	cmd := exec.Command("git", "-C", dir, "ls-files",
		"--cached",
		"--others",
		"--exclude-standard",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files in %s: %w", dir, err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		rel := filepath.ToSlash(line)
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			continue
		}
		files = append(files, rel)
	}
	return files, nil
}
