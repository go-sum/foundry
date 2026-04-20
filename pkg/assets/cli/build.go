package main

import (
	"os"

	"github.com/go-sum/assets/build"
	"github.com/go-sum/assets/config"
	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	var configPath string
	var minify bool
	var incremental bool
	var stateFilePath string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build asset types: all, css, js, fonts",
	}
	cmd.PersistentFlags().StringVar(&configPath, "config", config.DefaultConfigPath, "path to .assets.yaml")
	cmd.PersistentFlags().BoolVar(&minify, "minify", false, "minify compiled CSS and JS output")
	cmd.PersistentFlags().BoolVar(&incremental, "incremental", false, "skip build if inputs have not changed since last run")
	cmd.PersistentFlags().StringVar(&stateFilePath, "state-file", "tmp/.assets-state.json", "build state file for change detection")

	loadCfg := func() (*config.Config, error) {
		return config.Load(configPath)
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: "Build CSS, JS, and fonts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			if incremental {
				state := build.LoadState(stateFilePath)
				if err := build.BuildCSSIfChanged(cfg, minify, state, os.Stdout); err != nil {
					return err
				}
				// JS and fonts do not participate in change detection.
				if err := build.RemoveStaleJS(cfg, os.Stdout); err != nil {
					return err
				}
				if err := build.DownloadJS(cfg, build.DefaultClient, os.Stdout); err != nil {
					return err
				}
				if err := build.BundleJS(cfg, minify, os.Stdout); err != nil {
					return err
				}
				return build.DownloadFonts(cfg, build.DefaultClient, os.Stdout)
			}
			return build.Build(cfg, build.Options{Minify: minify, JS: true, CSS: true, Fonts: true}, build.DefaultClient, os.Stdout)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "css",
		Short: "Build CSS only",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			if incremental {
				state := build.LoadState(stateFilePath)
				return build.BuildCSSIfChanged(cfg, minify, state, os.Stdout)
			}
			return build.BuildCSS(cfg, minify, os.Stdout)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "js",
		Short: "Build JS only (download + bundle)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			if err := build.RemoveStaleJS(cfg, os.Stdout); err != nil {
				return err
			}
			if err := build.DownloadJS(cfg, build.DefaultClient, os.Stdout); err != nil {
				return err
			}
			return build.BundleJS(cfg, minify, os.Stdout)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "fonts",
		Short: "Download font files only",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			return build.DownloadFonts(cfg, build.DefaultClient, os.Stdout)
		},
	})

	return cmd
}
