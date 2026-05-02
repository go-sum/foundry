package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:          "foundry",
		Short:        "Foundry monorepo toolset",
		SilenceUsage: true,
	}
	root.AddCommand(newGoOrderCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
