package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-sum/db/compose"
	"github.com/go-sum/db/migrate"
	"github.com/spf13/cobra"
)

func newComposeCmd(configPath *string) *cobra.Command {
	var diffOnly bool
	var dir string

	cmd := &cobra.Command{
		Use:   "compose [name]",
		Short: "Generate a migration from schema registry diff",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}

			reg, err := cfg.buildRegistry()
			if err != nil {
				return err
			}

			migrDir := dir
			if migrDir == "" {
				migrDir = cfg.migrationsDir()
			}

			name := "migrate_schema"
			if len(args) > 0 {
				name = args[0]
			}

			var baseSQL string
			if providers := reg.Providers(); len(providers) > 0 && providers[0].Priority() == 0 {
				baseSQL = providers[0].SQL()
			}

			composeCfg := compose.Config{
				Registry:      reg,
				MigrationsDir: migrDir,
				PlanDB:        cfg.PlanDB,
				DiffOnly:      diffOnly,
				BaseSQL:       baseSQL,
			}

			ctx := context.Background()
			result, err := compose.Generate(ctx, composeCfg, name)
			if err != nil {
				return err
			}

			switch {
			case diffOnly:
				fmt.Print(result.Migration)
			case result.Migration != "":
				if result.InitialSchema != "" {
					fmt.Printf("Created: %s\n", result.InitialSchema)
				}
				fmt.Printf("Created: %s\n", result.Migration)
				lintResults, lintErr := migrate.Lint(migrDir)
				if lintErr != nil {
					return fmt.Errorf("compose: post-generate lint: %w", lintErr)
				}
				if len(lintResults) > 0 {
					printLintResults(os.Stderr, lintResults)
					if migrate.HasErrors(lintResults) {
						return fmt.Errorf("compose: lint found errors in generated migration — review and fix before applying")
					}
				}
			default:
				fmt.Println("compose: no schema changes detected — database is up to date")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "migrations directory (default: from config)")
	cmd.Flags().BoolVar(&diffOnly, "diff-only", false, "print diff without writing a file")

	return cmd
}
