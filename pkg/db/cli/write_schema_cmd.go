package main

import (
	"os"

	"github.com/spf13/cobra"
)

func newWriteSchemaCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "write-schema <path>",
		Short: "Write composed schema SQL to a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}

			reg, err := cfg.buildRegistry()
			if err != nil {
				return err
			}

			sql := reg.Compose()
			return os.WriteFile(args[0], []byte(sql), 0o644)
		},
	}
}
