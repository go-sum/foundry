package cmd

import (
	"context"
	"fmt"

	"github.com/go-sum/db"
	"github.com/spf13/cobra"
)

func newSeedCmd(cfg Config) *cobra.Command {
	var envFlag string

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Run seed data for the given environment",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if cfg.SeedRegistry == nil {
				return fmt.Errorf("seed: no SeedRegistry configured")
			}

			dsn, err := cfg.dsnFunc()()
			if err != nil {
				return err
			}

			ctx := context.Background()
			pool, err := db.ConnectDSN(ctx, dsn)
			if err != nil {
				return err
			}
			defer pool.Close()

			env := db.Environment(envFlag)
			return cfg.SeedRegistry.Run(ctx, pool, env)
		},
	}

	cmd.Flags().StringVar(&envFlag, "env", "dev", "target environment (dev|test|prod)")

	return cmd
}
