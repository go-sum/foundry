package db

import _ "embed"

//go:embed sql/base.sql
var baseSQL string

type baseSchema struct{}

func (baseSchema) Name() string  { return "base" }
func (baseSchema) SQL() string   { return baseSQL }
func (baseSchema) Priority() int { return 0 }

// BaseSchema provides common PostgreSQL extensions and utility functions.
// Register at priority 0 before all feature schemas.
var BaseSchema baseSchema
