package cmd

import (
	"fmt"

	"github.com/go-sum/db/migrate"
	"github.com/spf13/cobra"
)

func newCreateCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new empty migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			name := args[0]

			path, err := migrate.Create(dir, name)
			if err != nil {
				return err
			}

			fmt.Printf("Created: %s\n", path)
			return nil
		},
	}

	cmd.Flags().String("dir", cfg.migrationsDir(), "migrations directory")

	return cmd
}
