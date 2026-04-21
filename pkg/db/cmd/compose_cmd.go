package cmd

import (
	"context"
	"fmt"

	"github.com/go-sum/db/compose"
	"github.com/spf13/cobra"
)

func newComposeCmd(cfg Config) *cobra.Command {
	var diffOnly bool

	cmd := &cobra.Command{
		Use:   "compose <name>",
		Short: "Generate a migration from schema registry diff",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			name := args[0]

			composeCfg := compose.Config{
				Registry:      cfg.Registry,
				MigrationsDir: dir,
				PlanDB:        cfg.PlanDB,
				DiffOnly:      diffOnly,
			}

			ctx := context.Background()
			path, err := compose.Generate(ctx, composeCfg, name)
			if err != nil {
				return err
			}

			if path != "" {
				fmt.Printf("Created: %s\n", path)
			}
			return nil
		},
	}

	cmd.Flags().String("dir", cfg.migrationsDir(), "migrations directory")
	cmd.Flags().BoolVar(&diffOnly, "diff-only", false, "print diff without writing a file")

	return cmd
}
