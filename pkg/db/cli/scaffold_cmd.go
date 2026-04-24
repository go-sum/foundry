package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-sum/db/codegen"
	"github.com/spf13/cobra"
)

func newScaffoldCmd(configPath *string) *cobra.Command {
	var force, skipExisting bool

	cmd := &cobra.Command{
		Use:   "scaffold [table]",
		Short: "Generate CRUD query files from schema definitions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}

			// Collect SQL from non-external local schema files only.
			var schemaSQLParts []string
			for _, entry := range cfg.Schema {
				if entry.External {
					continue
				}
				data, err := os.ReadFile(entry.Path)
				if err != nil {
					return fmt.Errorf("scaffold: read schema %s: %w", entry.Path, err)
				}
				schemaSQLParts = append(schemaSQLParts, string(data))
			}
			schemaSQL := strings.Join(schemaSQLParts, "\n")

			tableName := ""
			if len(args) > 0 {
				tableName = args[0]
			}

			paths, err := codegen.ScaffoldTable(schemaSQL, tableName, cfg.queriesDir(), force, skipExisting)
			if err != nil {
				return err
			}

			for _, p := range paths {
				fmt.Printf("Created: %s\n", p)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing query files")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "skip tables that already have a query file")
	return cmd
}
