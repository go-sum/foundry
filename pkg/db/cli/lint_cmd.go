package dbcli

import (
	"fmt"
	"os"

	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/db/migrate"
	"github.com/spf13/cobra"
)

func newLintCmd(configPath *string, resolver db.SchemaResolver) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Check migration files for dangerous DDL patterns",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}
			cfg.resolver = resolver

			migrDir := dir
			if migrDir == "" {
				migrDir = cfg.migrationsDir()
			}

			results, err := migrate.Lint(migrDir)
			if err != nil {
				return err
			}

			if len(results) == 0 {
				fmt.Println("No issues found.")
				return nil
			}

			printLintResults(os.Stdout, results)
			if migrate.HasErrors(results) {
				return fmt.Errorf("lint: found errors in migrations")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "migrations directory (default: from config)")

	return cmd
}
