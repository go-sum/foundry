package dbcli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/db/migrate"
	"github.com/spf13/cobra"
)

func newMigrateCmd(configPath *string, resolver db.SchemaResolver) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migration commands",
	}
	cmd.AddCommand(
		newMigrateComposeCmd(configPath, resolver),
		newMigrateApplyCmd(configPath, resolver),
		newMigrateStatusCmd(configPath, resolver),
		newMigrateRollbackCmd(configPath, resolver),
	)
	return cmd
}

func newMigrateComposeCmd(configPath *string, resolver db.SchemaResolver) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "compose [name]",
		Short: "Detect schema changes and generate a migration file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}
			cfg.resolver = resolver
			schemas, err := cfg.buildSchemaInputs()
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

			createCfg := migrate.CreateConfig{
				Schemas:       schemas,
				MigrationsDir: migrDir,
			}

			baseSchemas, err := migrate.BuildBaseline(migrDir, schemas)
			if err != nil {
				return fmt.Errorf("migrate compose: build baseline: %w", err)
			}
			createCfg.BaseSchemas = baseSchemas

			result, err := migrate.Create(createCfg, name)
			if err != nil {
				return err
			}
			if len(result.Files) == 0 {
				fmt.Println("migrate compose: no schema changes detected")
				return nil
			}
			for _, f := range result.Files {
				fmt.Printf("Created: %s\n", f)
			}
			lintResults, lintErr := migrate.Lint(migrDir)
			if lintErr != nil {
				return fmt.Errorf("migrate compose: post-create lint: %w", lintErr)
			}
			if len(lintResults) > 0 {
				printLintResults(os.Stderr, lintResults)
				if migrate.HasErrors(lintResults) {
					return fmt.Errorf("migrate compose: lint errors in generated migration")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "migrations directory (default: from config)")
	return cmd
}

func newMigrateApplyCmd(configPath *string, resolver db.SchemaResolver) *cobra.Command {
	var dir string
	var dryRun bool
	var toVersion int64
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply pending migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}
			cfg.resolver = resolver
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
				return fmt.Errorf("migrate apply: pre-flight lint: %w", lintErr)
			}
			if len(lintResults) > 0 {
				printLintResults(os.Stderr, lintResults)
				if migrate.HasErrors(lintResults) {
					return fmt.Errorf("migrate apply: lint errors — fix before applying")
				}
				fmt.Fprintln(os.Stderr, "migrate apply: lint warnings — proceeding")
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
			} else {
				if err := migrate.Up(ctx, dsn, migrDir); err != nil {
					return err
				}
			}
			storeFingerprintAfterApply(ctx, cfg, dsn, migrDir)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "migrations directory (default: from config)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print pending migrations without applying")
	cmd.Flags().Int64Var(&toVersion, "to", 0, "apply up to this version")
	return cmd
}

func newMigrateStatusCmd(configPath *string, resolver db.SchemaResolver) *cobra.Command {
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
			cfg.resolver = resolver
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
			fmt.Fprintln(w, "VERSION\tAPPLIED\tAPPLIED AT\tSOURCE")
			for _, s := range statuses {
				applied := "no"
				appliedAt := ""
				if s.Applied {
					applied = "yes"
					appliedAt = s.AppliedAt.Format(time.RFC3339)
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Version, applied, appliedAt, filepath.Base(s.Source))
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "migrations directory (default: from config)")
	return cmd
}

func newMigrateRollbackCmd(configPath *string, resolver db.SchemaResolver) *cobra.Command {
	var dir string
	var toVersion int64
	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Roll back the most recently applied migration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}
			cfg.resolver = resolver
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

func storeFingerprintAfterApply(ctx context.Context, cfg *dbConfig, dsn, migrDir string) {
	reg, err := cfg.buildRegistry()
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: fingerprint: build registry:", err)
		return
	}
	if err := migrate.StoreFingerprintDSN(ctx, dsn, migrDir, reg.Fingerprint()); err != nil {
		fmt.Fprintln(os.Stderr, "warning: fingerprint:", err)
	}
}
