package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var configPath string

	root := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
	}
	root.PersistentFlags().StringVar(&configPath, "config", "db/schema.yaml", "path to schema.yaml config file")

	root.AddCommand(
		newMigrateCmd(&configPath),
		newSeedCmd(&configPath),
		newScaffoldCmd(&configPath),
		newLintCmd(&configPath),
		newHealthCmd(&configPath),
		newWriteSchemaCmd(&configPath),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
