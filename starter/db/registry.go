package db

import "embed"

//go:embed schema.yaml
var ConfigYAML []byte

//go:embed sql/schema/*.sql
var SchemaFiles embed.FS
