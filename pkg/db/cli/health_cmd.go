package main

import (
	"context"
	"fmt"

	"github.com/go-sum/foundry/pkg/db"
	"github.com/spf13/cobra"
)

func newHealthCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Verify database connectivity and schema fingerprint",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}
			dsn, err := cfg.dsnFunc()()
			if err != nil {
				return err
			}
			ctx := context.Background()
			pool, err := db.ConnectDSN(ctx, dsn)
			if err != nil {
				return fmt.Errorf("health: connect: %w", err)
			}
			defer pool.Close()
			if err := db.Health(ctx, pool); err != nil {
				return fmt.Errorf("health: ping: %w", err)
			}
			reg, err := cfg.buildRegistry()
			if err != nil {
				return fmt.Errorf("health: build registry: %w", err)
			}
			if err := db.VerifyFingerprint(ctx, pool, reg.Fingerprint()); err != nil {
				return fmt.Errorf("health: %w", err)
			}
			fmt.Println("OK")
			return nil
		},
	}
	return cmd
}
