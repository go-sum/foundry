package main

import (
	"cmp"
	"os"

	"github.com/go-sum/db"
	dbcmd "github.com/go-sum/db/cmd"
	"github.com/go-sum/db/compose"
	"github.com/go-sum/queue/pgstore"

	"github.com/go-sum/foundry/db/dbschema"
)

type queueSchema struct{}

func (queueSchema) Name() string  { return "queue_jobs" }
func (queueSchema) SQL() string   { return pgstore.SchemaSQL }
func (queueSchema) Priority() int { return 50 }

func main() {
	schemaReg := db.NewRegistry()
	schemaReg.Register(dbschema.Schema, dbschema.ContactSchema, queueSchema{})

	seedReg := db.NewSeedRegistry()
	seedReg.Register(dbschema.ContactSeeder)

	cmd := dbcmd.NewDBCommand(dbcmd.Config{
		Registry:      schemaReg,
		SeedRegistry:  seedReg,
		MigrationsDir: "db/migrations",
		PlanDB: compose.PlanDBConfig{
			Host:     cmp.Or(os.Getenv("PLAN_DB_HOST"), "db"),
			Port:     cmp.Or(os.Getenv("PLAN_DB_PORT"), "5432"),
			User:     cmp.Or(os.Getenv("POSTGRES_USER"), "postgres"),
			Password: cmp.Or(os.Getenv("POSTGRES_PASSWORD"), "postgres"),
			Database: cmp.Or(os.Getenv("PLAN_DB_NAME"), "foundry_plan"),
		},
	})

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
