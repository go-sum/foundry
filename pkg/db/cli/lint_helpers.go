package main

import (
	"io"
	"text/tabwriter"
	"fmt"

	"github.com/go-sum/foundry/pkg/db/migrate"
)

func printLintResults(w io.Writer, results []migrate.LintResult) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FILE\tLINE\tSEVERITY\tRULE\tMESSAGE")
	for _, r := range results {
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\n", r.File, r.Line, r.Severity, r.Rule, r.Message)
	}
	tw.Flush() //nolint:errcheck
}
