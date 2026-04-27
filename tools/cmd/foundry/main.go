package main

import (
	"fmt"
	"os"

	"github.com/go-sum/foundry/tools/internal/cli"
	"github.com/go-sum/foundry/tools/internal/gitops"
	"github.com/spf13/cobra"
)

func main() {
	cfg := &cli.Config{}

	root := &cobra.Command{
		Use:   "foundry",
		Short: "Foundry monorepo toolset",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			root, err := gitops.RepoRoot()
			if err != nil {
				return fmt.Errorf("not inside a git repository: %w", err)
			}
			cfg.RepoRoot = root
			return nil
		},
		SilenceUsage: true,
	}

	root.PersistentFlags().BoolVar(&cfg.DryRun, "dry-run", false, "print actions without executing")

	root.AddCommand(
		cli.NewStarterCmd(cfg),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
