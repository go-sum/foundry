package dbcli

import (
	"github.com/go-sum/foundry/pkg/db"
	"github.com/spf13/cobra"
)

// Option configures NewRootCommand.
type Option func(*options)

type options struct {
	resolver db.SchemaResolver
}

// WithResolver provides a SchemaResolver for schema entries whose source files
// are not available on the filesystem (standalone apps consuming packages as Go
// modules).
func WithResolver(r db.SchemaResolver) Option {
	return func(o *options) { o.resolver = r }
}

// NewRootCommand returns the db CLI root cobra command. Call Execute() to run.
func NewRootCommand(opts ...Option) *cobra.Command {
	var o options
	for _, fn := range opts {
		fn(&o)
	}

	var configPath string

	root := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
	}
	root.PersistentFlags().StringVar(&configPath, "config", "db/schema.yaml",
		"path to schema.yaml config file")

	root.AddCommand(
		newMigrateCmd(&configPath, o.resolver),
		newSeedCmd(&configPath, o.resolver),
		newScaffoldCmd(&configPath, o.resolver),
		newLintCmd(&configPath, o.resolver),
		newHealthCmd(&configPath, o.resolver),
		newWriteSchemaCmd(&configPath, o.resolver),
	)

	return root
}
