package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/go-sum/foundry/pkg/db/seed"
	"github.com/spf13/cobra"
)

func newSeedCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed commands",
	}
	cmd.AddCommand(
		newSeedApplyCmd(configPath),
		newSeedStatusCmd(configPath),
		newSeedResetCmd(configPath),
	)
	return cmd
}

func newSeedApplyCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Run pending seed files",
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
			entries, err := cfg.buildSeedEntries()
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("seed apply: no seed files configured")
				return nil
			}
			ctx := context.Background()
			count, err := seed.Apply(ctx, dsn, entries)
			if err != nil {
				return err
			}
			if count == 0 {
				fmt.Println("seed apply: all seeds already applied")
			} else {
				fmt.Printf("seed apply: applied %d seed(s)\n", count)
			}
			return nil
		},
	}
	return cmd
}

func newSeedStatusCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show seed status",
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
			entries, err := cfg.buildSeedEntries()
			if err != nil {
				return err
			}
			known := make([]string, len(entries))
			for i, e := range entries {
				known[i] = e.Name
			}
			ctx := context.Background()
			statuses, err := seed.GetStatus(ctx, dsn, known)
			if err != nil {
				return err
			}
			if len(statuses) == 0 {
				fmt.Println("No seeds configured.")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tAPPLIED\tAPPLIED AT")
			for _, s := range statuses {
				applied := "no"
				appliedAt := ""
				if s.Applied {
					applied = "yes"
					appliedAt = s.AppliedAt.Format(time.RFC3339)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, applied, appliedAt)
			}
			return w.Flush()
		},
	}
	return cmd
}

func newSeedResetCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Clear seed history so seeds can be re-applied",
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
			if err := seed.Reset(ctx, dsn); err != nil {
				return err
			}
			fmt.Println("seed reset: history cleared")
			return nil
		},
	}
	return cmd
}
