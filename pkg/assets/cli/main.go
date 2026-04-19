package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "assets",
		Short: "Asset pipeline: build, sprite, and init commands",
	}
	root.AddCommand(
		newBuildCmd(),
		newSpritesCmd(),
		newInitCmd(),
	)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
