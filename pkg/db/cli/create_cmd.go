package main

import (
	"fmt"

	"github.com/go-sum/db/migrate"
	"github.com/spf13/cobra"
)

func newCreateCmd(configPath *string) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new empty migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}

			migrDir := dir
			if migrDir == "" {
				migrDir = cfg.migrationsDir()
			}

			path, err := migrate.Create(migrDir, args[0])
			if err != nil {
				return err
			}

			fmt.Printf("Created: %s\n", path)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "migrations directory (default: from config)")

	return cmd
}
