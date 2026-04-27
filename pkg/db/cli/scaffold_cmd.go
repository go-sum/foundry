package dbcli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/db/codegen"
	"github.com/spf13/cobra"
)

func newScaffoldCmd(configPath *string, resolver db.SchemaResolver) *cobra.Command {
	var force, skipExisting bool
	var pkgName, outDir string

	cmd := &cobra.Command{
		Use:   "scaffold [table]",
		Short: "Generate Go store files from schema definitions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}
			cfg.resolver = resolver

			// Collect SQL from scaffold-enabled entries only.
			var schemaSQLParts []string
			for _, entry := range cfg.Schema {
				if !entry.shouldScaffold() {
					continue
				}
				sql, err := cfg.resolveSQL(entry)
				if err != nil {
					return fmt.Errorf("scaffold: read schema: %w", err)
				}
				schemaSQLParts = append(schemaSQLParts, sql)
			}
			schemaSQL := strings.Join(schemaSQLParts, "\n")

			tableName := ""
			if len(args) > 0 {
				tableName = args[0]
			}

			// Determine output directory: default to same directory as first scaffold entry.
			target := outDir
			if target == "" {
				for _, entry := range cfg.Schema {
					if entry.shouldScaffold() {
						target = filepath.Dir(entry.Source)
						break
					}
				}
			}
			if target == "" {
				target = cfg.baseDir
			}

			// Determine package name: default to directory base name.
			pkg := pkgName
			if pkg == "" {
				pkg = filepath.Base(target)
				// sanitize: replace hyphens with underscores
				pkg = strings.ReplaceAll(pkg, "-", "_")
			}

			paths, err := codegen.ScaffoldTable(schemaSQL, tableName, pkg, target, force, skipExisting)
			if err != nil {
				return err
			}

			for _, p := range paths {
				fmt.Printf("Created: %s\n", p)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing store files")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "skip tables that already have a store file")
	cmd.Flags().StringVar(&pkgName, "package", "", "Go package name (default: inferred from output directory name)")
	cmd.Flags().StringVar(&outDir, "out", "", "output directory (default: same directory as the schema file)")
	return cmd
}
