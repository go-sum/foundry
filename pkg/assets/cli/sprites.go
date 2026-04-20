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
	var incremental bool
	var stateFilePath string

	cmd := &cobra.Command{
		Use:   "sprites",
		Short: "Build SVG sprite sheets",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			opts := build.SpriteOptions{Name: spriteName, DryRun: dryRun}
			if incremental {
				state := build.LoadState(stateFilePath)
				return build.BuildSpritesIfChanged(cfg, opts, build.DefaultClient, state, os.Stdout)
			}
			return build.BuildSprites(cfg, opts, build.DefaultClient, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", config.DefaultConfigPath, "path to .assets.yaml")
	cmd.Flags().StringVar(&spriteName, "sprite", "", "build only this named sprite (default: all enabled)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print output without writing files")
	cmd.Flags().BoolVar(&incremental, "incremental", false, "skip build if inputs have not changed since last run")
	cmd.Flags().StringVar(&stateFilePath, "state-file", "tmp/.assets-state.json", "build state file for change detection")
	return cmd
}
