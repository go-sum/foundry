package cmd

import (
	"github.com/go-sum/db"
	"github.com/go-sum/db/compose"
	"github.com/spf13/cobra"
)

// Config holds all dependencies for the db command tree.
type Config struct {
	Registry      *db.Registry
	SeedRegistry  *db.SeedRegistry
	DSNFunc       func() (string, error)
	MigrationsDir string
	PlanDB        compose.PlanDBConfig
}

func (c *Config) dsnFunc() func() (string, error) {
	if c.DSNFunc != nil {
		return c.DSNFunc
	}
	return db.DSN
}

func (c *Config) migrationsDir() string {
	if c.MigrationsDir != "" {
		return c.MigrationsDir
	}
	return "db/migrations"
}

// NewDBCommand returns a cobra command subtree for database management.
// Mount it onto any application's root command.
func NewDBCommand(cfg Config) *cobra.Command {
	root := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
	}

	root.AddCommand(newMigrateCmd(cfg))
	root.AddCommand(newRollbackCmd(cfg))
	root.AddCommand(newStatusCmd(cfg))
	root.AddCommand(newCreateCmd(cfg))
	root.AddCommand(newComposeCmd(cfg))
	root.AddCommand(newSeedCmd(cfg))
	root.AddCommand(newLintCmd(cfg))
	root.AddCommand(newHealthCmd(cfg))

	return root
}
