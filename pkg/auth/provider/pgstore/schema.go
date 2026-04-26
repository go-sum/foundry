package pgstore

import _ "embed"

//go:embed sql/schema.sql
var SchemaSQL string
