package codegen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sqlc-dev/sqlc/pkg/cli"
	"gopkg.in/yaml.v3"
)

// Config holds code generation settings from schema.yaml.
type Config struct {
	Queries         string `yaml:"queries"`
	Out             string `yaml:"out"`
	Package         string `yaml:"package"`
	EmitJSONTags    bool   `yaml:"emit_json_tags"`
	EmitEmptySlices bool   `yaml:"emit_empty_slices"`
}

// Generate produces Go code from SQL schema files and query annotations.
// baseDir is the directory containing schema.yaml; relative paths in cfg
// are resolved against it. schemaFiles must be absolute paths.
func Generate(baseDir string, schemaFiles []string, cfg Config) error {
	// sqlc resolves schema paths relative to the config file's location.
	// Convert absolute schema paths to relative-to-baseDir so they resolve
	// correctly when the temp config file is placed in baseDir.
	relSchema := make([]string, len(schemaFiles))
	for i, f := range schemaFiles {
		rel, err := filepath.Rel(baseDir, f)
		if err != nil {
			return fmt.Errorf("codegen: schema rel path: %w", err)
		}
		relSchema[i] = rel
	}

	sqlcCfg := sqlcConfig{
		Version: "2",
		SQL: []sqlcSQL{{
			Engine:  "postgresql",
			Queries: cfg.Queries,
			Schema:  relSchema,
			Gen: sqlcGen{Go: sqlcGenGo{
				Package:         cfg.Package,
				Out:             cfg.Out,
				SQLPackage:      "pgx/v5",
				EmitJSONTags:    cfg.EmitJSONTags,
				EmitEmptySlices: cfg.EmitEmptySlices,
			}},
		}},
	}

	data, err := yaml.Marshal(sqlcCfg)
	if err != nil {
		return fmt.Errorf("codegen: marshal config: %w", err)
	}

	// Place the temp config in baseDir so sqlc resolves relative paths from
	// the same directory as schema.yaml, regardless of the process working dir.
	f, err := os.CreateTemp(baseDir, ".sqlc-gen-*.yaml")
	if err != nil {
		return fmt.Errorf("codegen: create temp config: %w", err)
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath) //nolint:errcheck

	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("codegen: write temp config: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("codegen: close temp config: %w", err)
	}

	if code := cli.Run([]string{"generate", "--file", tmpPath}); code != 0 {
		return fmt.Errorf("codegen: sqlc exited with code %d", code)
	}
	return nil
}

type sqlcConfig struct {
	Version string    `yaml:"version"`
	SQL     []sqlcSQL `yaml:"sql"`
}

type sqlcSQL struct {
	Engine  string   `yaml:"engine"`
	Queries string   `yaml:"queries"`
	Schema  []string `yaml:"schema"`
	Gen     sqlcGen  `yaml:"gen"`
}

type sqlcGen struct {
	Go sqlcGenGo `yaml:"go"`
}

type sqlcGenGo struct {
	Package         string `yaml:"package"`
	Out             string `yaml:"out"`
	SQLPackage      string `yaml:"sql_package"`
	EmitJSONTags    bool   `yaml:"emit_json_tags"`
	EmitEmptySlices bool   `yaml:"emit_empty_slices"`
}
