package db

import (
	"crypto/sha256"
	"encoding/hex"
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
// Register must complete before any concurrent call to Providers or Compose.
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

// Fingerprint returns the hex-encoded SHA-256 hash of the composed schema SQL.
func (r *Registry) Fingerprint() string {
	hash := sha256.Sum256([]byte(r.Compose()))
	return hex.EncodeToString(hash[:])
}

// SchemaResolver maps schema names (from schema.yaml `name` field) to their
// SQL content. Used to provide SQL for package-owned schemas at both runtime
// (via WithResolver) and in the CLI (via dbcli.WithResolver).
type SchemaResolver map[string]string

// LoadOption configures LoadRegistryFromYAML.
type LoadOption func(*loadConfig)

type loadConfig struct {
	resolver SchemaResolver
}

// WithResolver returns a LoadOption that supplies a SchemaResolver for schema
// entries whose source files are not available on the filesystem (standalone
// apps consuming packages as Go modules). The resolver takes precedence over
// embed.FS when both are available for the same entry name.
func WithResolver(r SchemaResolver) LoadOption {
	return func(cfg *loadConfig) { cfg.resolver = r }
}

type simpleSchema struct {
	name     string
	sql      string
	priority int
}

func (s simpleSchema) Name() string  { return s.name }
func (s simpleSchema) SQL() string   { return s.sql }
func (s simpleSchema) Priority() int { return s.priority }

// NewSchema returns a SchemaProvider for the given name, SQL, and priority.
func NewSchema(name, sql string, priority int) SchemaProvider {
	return simpleSchema{name: name, sql: sql, priority: priority}
}

// yamlSchemaCfg holds the schema section of a schema.yaml config file.
type yamlSchemaCfg struct {
	Schema []yamlSchemaEntry `yaml:"schema"`
}

type yamlSchemaEntry struct {
	Name     string `yaml:"name"`
	Source   string `yaml:"source"`
	Priority int    `yaml:"priority"`
	Scaffold *bool  `yaml:"scaffold"`
}

// yamlSchema is a SchemaProvider whose SQL was loaded from an embedded filesystem.
type yamlSchema struct {
	name     string
	sql      string
	priority int
}

func (y yamlSchema) Name() string  { return y.name }
func (y yamlSchema) SQL() string   { return y.sql }
func (y yamlSchema) Priority() int { return y.priority }

// LoadRegistryFromYAML parses a schema.yaml config and registers each schema
// entry using resolver-first, embed.FS-fallback resolution:
//
//  1. If a WithResolver option was supplied and the resolver contains the entry
//     name, the resolver SQL is used (standalone apps without source files).
//  2. Otherwise, the entry's source path is read from schemaFiles (monorepo
//     embed.FS usage).
//  3. If neither path resolves, an error is returned.
func LoadRegistryFromYAML(configYAML []byte, schemaFiles fs.FS, opts ...LoadOption) (*Registry, error) {
	var cfg yamlSchemaCfg
	if err := yaml.Unmarshal(configYAML, &cfg); err != nil {
		return nil, fmt.Errorf("db: parse schema yaml: %w", err)
	}

	var lc loadConfig
	for _, o := range opts {
		o(&lc)
	}

	reg := NewRegistry()

	for _, entry := range cfg.Schema {
		name := entry.Name
		if name == "" {
			name = strings.TrimSuffix(path.Base(entry.Source), ".sql")
		}

		// Resolver-first: resolver takes precedence when available.
		if lc.resolver != nil {
			if sql, ok := lc.resolver[name]; ok {
				reg.Register(yamlSchema{
					name:     name,
					sql:      sql,
					priority: entry.Priority,
				})
				continue
			}
		}

		// Fallback to embed.FS.
		if entry.Source != "" {
			sql, err := fs.ReadFile(schemaFiles, entry.Source)
			if err != nil {
				return nil, fmt.Errorf("db: read schema %s: %w", entry.Source, err)
			}
			reg.Register(yamlSchema{
				name:     name,
				sql:      string(sql),
				priority: entry.Priority,
			})
			continue
		}

		return nil, fmt.Errorf("db: schema %q: no resolver entry and no source path", name)
	}

	return reg, nil
}
