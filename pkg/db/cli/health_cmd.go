package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-sum/db"
	"github.com/spf13/cobra"
)

func newHealthCmd(configPath *string) *cobra.Command {
	var tablesFlag string

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Verify database connectivity and required tables",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
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
				return err
			}
			defer pool.Close()

			var tables []string
			if tablesFlag != "" {
				for _, t := range strings.Split(tablesFlag, ",") {
					if t = strings.TrimSpace(t); t != "" {
						tables = append(tables, t)
					}
				}
			} else {
				reg, err := cfg.buildRegistry()
				if err != nil {
					return err
				}
				tables = reg.HealthTables()
			}

			if err := db.Health(ctx, pool, tables...); err != nil {
				return err
			}

			fmt.Println("Database is healthy.")
			return nil
		},
	}

	cmd.Flags().StringVar(&tablesFlag, "tables", "", "comma-separated list of tables to verify")

	return cmd
}
