package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-sum/db/codegen"
	"github.com/spf13/cobra"
)

func newCodegenCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "codegen",
		Short: "Generate Go code from SQL queries",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}

			if cfg.Codegen.Queries == "" {
				return fmt.Errorf("codegen: no queries directory configured in schema.yaml")
			}

			if !hasQueryFiles(cfg.queriesDir()) {
				fmt.Println("codegen: no query files found, skipping.")
				return nil
			}

			if err := codegen.Generate(cfg.baseDir, cfg.schemaFiles(), cfg.Codegen); err != nil {
				return err
			}

			fmt.Println("Code generation complete.")
			return nil
		},
	}
}

func hasQueryFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			return true
		}
	}
	return false
}
