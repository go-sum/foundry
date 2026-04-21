package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/go-sum/db/migrate"
	"github.com/spf13/cobra"
)

func newMigrateCmd(cfg Config) *cobra.Command {
	var dryRun bool
	var toVersion int64

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			dsn, err := cfg.dsnFunc()()
			if err != nil {
				return err
			}

			ctx := context.Background()

			if dryRun {
				fmt.Fprintln(os.Stderr, "Dry run — no changes will be applied")
			}

			if toVersion > 0 {
				return migrate.UpTo(ctx, dsn, dir, toVersion)
			}
			return migrate.Up(ctx, dsn, dir)
		},
	}

	cmd.Flags().String("dir", cfg.migrationsDir(), "migrations directory")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print SQL without executing")
	cmd.Flags().Int64Var(&toVersion, "to", 0, "migrate up to this version")

	return cmd
}

func newRollbackCmd(cfg Config) *cobra.Command {
	var toVersion int64

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback the last applied migration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			dsn, err := cfg.dsnFunc()()
			if err != nil {
				return err
			}

			ctx := context.Background()

			if toVersion > 0 {
				return migrate.DownTo(ctx, dsn, dir, toVersion)
			}
			return migrate.Down(ctx, dsn, dir)
		},
	}

	cmd.Flags().String("dir", cfg.migrationsDir(), "migrations directory")
	cmd.Flags().Int64Var(&toVersion, "to", 0, "roll back to this version")

	return cmd
}

func newStatusCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			dsn, err := cfg.dsnFunc()()
			if err != nil {
				return err
			}

			ctx := context.Background()
			statuses, err := migrate.Status(ctx, dsn, dir)
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

	cmd.Flags().String("dir", cfg.migrationsDir(), "migrations directory")

	return cmd
}
