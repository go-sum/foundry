package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/go-sum/db/migrate"
	"github.com/spf13/cobra"
)

func newLintCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Check migration files for dangerous DDL patterns",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, _ := cmd.Flags().GetString("dir")

			results, err := migrate.Lint(dir)
			if err != nil {
				return err
			}

			if len(results) == 0 {
				fmt.Println("No issues found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "FILE\tLINE\tSEVERITY\tRULE\tMESSAGE")
			for _, r := range results {
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", r.File, r.Line, r.Severity, r.Rule, r.Message)
			}
			return w.Flush()
		},
	}

	cmd.Flags().String("dir", cfg.migrationsDir(), "migrations directory")

	return cmd
}
