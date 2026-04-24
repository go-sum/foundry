package db

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// SchemaProvider declares a package's desired-state DDL.
type SchemaProvider interface {
	Name() string
	SQL() string
	Priority() int
}

// Registry collects SchemaProviders and composes them into a single SQL string.
// Register must complete before any concurrent call to Providers, Compose, or HealthTables.
type Registry struct {
	providers []SchemaProvider
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds one or more providers to the registry.
func (r *Registry) Register(providers ...SchemaProvider) {
	r.providers = append(r.providers, providers...)
}

// Providers returns a priority-sorted copy of all registered providers.
func (r *Registry) Providers() []SchemaProvider {
	sorted := make([]SchemaProvider, len(r.providers))
	copy(sorted, r.providers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority() < sorted[j].Priority()
	})
	return sorted
}

// Compose returns all provider SQL concatenated in priority order, separated by double newlines.
func (r *Registry) Compose() string {
	providers := r.Providers()
	if len(providers) == 0 {
		return ""
	}

	var result strings.Builder
	result.WriteString(providers[0].SQL())
	for _, p := range providers[1:] {
		result.WriteString("\n\n" + p.SQL())
	}
	return result.String()
}

// HealthTables returns the combined table names from all registered providers
// that implement HealthTables() []string.
func (r *Registry) HealthTables() []string {
	type healthTabler interface {
		HealthTables() []string
	}
	var tables []string
	for _, p := range r.Providers() {
		if ht, ok := p.(healthTabler); ok {
			tables = append(tables, ht.HealthTables()...)
		}
	}
	return tables
}

type simpleSchema struct {
	name     string
	sql      string
	priority int
}

func (s simpleSchema) Name() string           { return s.name }
func (s simpleSchema) SQL() string            { return s.sql }
func (s simpleSchema) Priority() int          { return s.priority }
func (s simpleSchema) HealthTables() []string { return []string{s.name} }

// NewSchema returns a SchemaProvider for the given name, SQL, and priority.
// The provider's Name is also used as its health-check table name.
func NewSchema(name, sql string, priority int) SchemaProvider {
	return simpleSchema{name: name, sql: sql, priority: priority}
}

// yamlSchemaCfg holds the schema section of a schema.yaml config file.
type yamlSchemaCfg struct {
	Schema []yamlSchemaEntry `yaml:"schema"`
}

type yamlSchemaEntry struct {
	Path         string   `yaml:"path"`
	Priority     int      `yaml:"priority"`
	HealthTables []string `yaml:"health_tables"`
	External     bool     `yaml:"external"`
}

// yamlSchema is a SchemaProvider whose SQL was loaded from an embedded filesystem.
type yamlSchema struct {
	name     string
	sql      string
	priority int
	tables   []string
}

func (y yamlSchema) Name() string           { return y.name }
func (y yamlSchema) SQL() string            { return y.sql }
func (y yamlSchema) Priority() int          { return y.priority }
func (y yamlSchema) HealthTables() []string { return y.tables }

// LoadRegistryFromYAML parses a schema.yaml config, loads non-external schema
// files from schemaFiles, and registers them together with the explicitly
// provided extras (which supply SQL for external: true entries).
func LoadRegistryFromYAML(configYAML []byte, schemaFiles fs.FS, extras ...SchemaProvider) (*Registry, error) {
	var cfg yamlSchemaCfg
	if err := yaml.Unmarshal(configYAML, &cfg); err != nil {
		return nil, fmt.Errorf("db: parse schema yaml: %w", err)
	}

	reg := NewRegistry()
	reg.Register(extras...)

	for _, entry := range cfg.Schema {
		if entry.External {
			continue
		}
		sql, err := fs.ReadFile(schemaFiles, entry.Path)
		if err != nil {
			return nil, fmt.Errorf("db: read schema %s: %w", entry.Path, err)
		}
		name := strings.TrimSuffix(path.Base(entry.Path), ".sql")
		reg.Register(yamlSchema{
			name:     name,
			sql:      string(sql),
			priority: entry.Priority,
			tables:   entry.HealthTables,
		})
	}

	return reg, nil
}
