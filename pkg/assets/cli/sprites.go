package main

import (
	"os"

	"github.com/go-sum/assets/build"
	"github.com/go-sum/assets/config"
	"github.com/spf13/cobra"
)

func newSpritesCmd() *cobra.Command {
	var configPath, spriteName string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sprites",
		Short: "Build SVG sprite sheets",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			return build.BuildSprites(cfg, build.SpriteOptions{Name: spriteName, DryRun: dryRun}, build.DefaultClient, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", config.DefaultConfigPath, "path to .assets.yaml")
	cmd.Flags().StringVar(&spriteName, "sprite", "", "build only this named sprite (default: all enabled)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print output without writing files")
	return cmd
}
