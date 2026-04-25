package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/go-sum/db"
	"github.com/go-sum/db/migrate"
	"github.com/spf13/cobra"
)

func newMigrateCmd(configPath *string) *cobra.Command {
	var dryRun bool
	var toVersion int64
	var dir string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending migrations",
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

			migrDir := dir
			if migrDir == "" {
				migrDir = cfg.migrationsDir()
			}

			lintResults, lintErr := migrate.Lint(migrDir)
			if lintErr != nil {
				return fmt.Errorf("migrate: pre-flight lint: %w", lintErr)
			}
			if len(lintResults) > 0 {
				printLintResults(os.Stderr, lintResults)
				if migrate.HasErrors(lintResults) {
					return fmt.Errorf("migrate: lint found errors — fix migration files before applying")
				}
				fmt.Fprintln(os.Stderr, "migrate: lint warnings found — proceeding")
			}

			ctx := context.Background()

			if dryRun {
				fmt.Fprintln(os.Stderr, "Dry run — no changes will be applied")
				statuses, err := migrate.Status(ctx, dsn, migrDir)
				if err != nil {
					return err
				}
				pending := 0
				for _, s := range statuses {
					if !s.Applied {
						fmt.Fprintf(os.Stderr, "  pending: %d  %s\n", s.Version, filepath.Base(s.Source))
						pending++
					}
				}
				if pending == 0 {
					fmt.Fprintln(os.Stderr, "  (no pending migrations)")
				}
				return nil
			}

			if toVersion > 0 {
				if err := migrate.UpTo(ctx, dsn, migrDir, toVersion); err != nil {
					return err
				}
				storeFingerprintAfterMigrate(ctx, cfg, dsn)
				return nil
			}
			if err := migrate.Up(ctx, dsn, migrDir); err != nil {
				return err
			}
			storeFingerprintAfterMigrate(ctx, cfg, dsn)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "migrations directory (default: from config)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print SQL without executing")
	cmd.Flags().Int64Var(&toVersion, "to", 0, "migrate up to this version")

	return cmd
}

func newRollbackCmd(configPath *string) *cobra.Command {
	var toVersion int64
	var dir string

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback the last applied migration",
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

			migrDir := dir
			if migrDir == "" {
				migrDir = cfg.migrationsDir()
			}

			ctx := context.Background()

			if toVersion > 0 {
				return migrate.DownTo(ctx, dsn, migrDir, toVersion)
			}
			return migrate.Down(ctx, dsn, migrDir)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "migrations directory (default: from config)")
	cmd.Flags().Int64Var(&toVersion, "to", 0, "roll back to this version")

	return cmd
}

func newStatusCmd(configPath *string) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
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

			migrDir := dir
			if migrDir == "" {
				migrDir = cfg.migrationsDir()
			}

			ctx := context.Background()
			statuses, err := migrate.Status(ctx, dsn, migrDir)
			if err != nil {
				return err
			}

			if len(statuses) == 0 {
				fmt.Println("No migrations found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "VERSION\tAPPLIED\tSOURCE")
			for _, s := range statuses {
				applied := "no"
				if s.Applied {
					applied = "yes"
				}
				fmt.Fprintf(w, "%d\t%s\t%s\n", s.Version, applied, s.Source)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "migrations directory (default: from config)")

	return cmd
}

// storeFingerprintAfterMigrate computes the schema fingerprint and stores it in
// the database after a successful migration. Errors are logged as warnings and
// do not cause the migrate command to fail.
func storeFingerprintAfterMigrate(ctx context.Context, cfg *dbConfig, dsn string) {
	reg, err := cfg.buildRegistry()
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: fingerprint: build registry:", err)
		return
	}
	pool, err := db.ConnectDSN(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: fingerprint: connect:", err)
		return
	}
	defer pool.Close()
	if err := db.StoreFingerprint(ctx, pool, reg.Fingerprint()); err != nil {
		fmt.Fprintln(os.Stderr, "warning: fingerprint: store:", err)
	}
}
