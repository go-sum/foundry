package db

import (
	"embed"

	authpgstore "github.com/go-sum/auth/pgstore"
	providerpgstore "github.com/go-sum/auth/provider/pgstore"
	coredb "github.com/go-sum/db"
	"github.com/go-sum/queue/pgstore"
)

//go:embed schema.yaml
var ConfigYAML []byte

//go:embed sql/schema/*.sql
var SchemaFiles embed.FS

// ExternalSchemas returns the resolver mapping external schema names (declared
// in schema.yaml) to their embedded SQL. This is the single wiring point for
// external package SQL.
func ExternalSchemas() coredb.ExternalResolver {
	return coredb.ExternalResolver{
		"base":             coredb.BaseSchema.SQL(),
		"queue":            pgstore.SchemaSQL,
		"auth":             authpgstore.SchemaSQL,
		"auth_provider":    providerpgstore.SchemaSQL,
	}
}
