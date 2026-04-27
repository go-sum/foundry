package dbcli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/db/migrate"
	"github.com/go-sum/foundry/pkg/db/seed"
)

type dbConfig struct {
	Schema     []schemaEntry    `yaml:"schema"`
	Migrations migrationsConfig `yaml:"migrations"`
	baseDir    string
	resolver   db.SchemaResolver
}

type migrationsConfig struct {
	Destination string `yaml:"destination"`
}

type schemaEntry struct {
	Name     string `yaml:"name"`
	Source   string `yaml:"source"`
	Priority int    `yaml:"priority"`
	Scaffold *bool  `yaml:"scaffold"`
	Group    string `yaml:"group"`
	Seed     string `yaml:"seed"`
}

// shouldScaffold reports whether scaffold code should be generated for this
// entry. Defaults to true when the scaffold field is omitted (nil).
func (e schemaEntry) shouldScaffold() bool {
	return e.Scaffold == nil || *e.Scaffold
}

type fileSchema struct {
	name     string
	sql      string
	priority int
}

func (f fileSchema) Name() string  { return f.name }
func (f fileSchema) SQL() string   { return f.sql }
func (f fileSchema) Priority() int { return f.priority }

func loadConfig(path string) (*dbConfig, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("config path: %w", err)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	expanded := expandEnvWithDefaults(string(data))
	var cfg dbConfig
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	baseDir := filepath.Dir(absPath)
	if cfg.Migrations.Destination != "" && !filepath.IsAbs(cfg.Migrations.Destination) {
		cfg.Migrations.Destination = filepath.Join(baseDir, cfg.Migrations.Destination)
	}
	for i := range cfg.Schema {
		if cfg.Schema[i].Source != "" && !filepath.IsAbs(cfg.Schema[i].Source) {
			cfg.Schema[i].Source = filepath.Join(baseDir, cfg.Schema[i].Source)
		}
		if cfg.Schema[i].Seed != "" && !filepath.IsAbs(cfg.Schema[i].Seed) {
			cfg.Schema[i].Seed = filepath.Join(baseDir, cfg.Schema[i].Seed)
		}
	}
	cfg.baseDir = baseDir
	return &cfg, nil
}

// resolveSQL returns the SQL for an entry using filesystem-first, resolver-fallback
// resolution. The CLI uses filesystem-first so that developer edits to schema
// files are immediately visible during compose. The resolver is used as a
// fallback for standalone apps that do not have the source files on disk.
func (c *dbConfig) resolveSQL(entry schemaEntry) (string, error) {
	name := entry.Name
	if name == "" && entry.Source != "" {
		name = strings.TrimSuffix(filepath.Base(entry.Source), ".sql")
	}

	// Filesystem first: picks up developer edits during compose.
	if entry.Source != "" {
		data, err := os.ReadFile(entry.Source)
		if err == nil {
			return string(data), nil
		}
		if c.resolver == nil {
			return "", fmt.Errorf("schema %s: %w", entry.Source, err)
		}
		// Fall through to resolver.
	}

	// Resolver fallback: for standalone apps without source files on disk.
	if c.resolver != nil {
		if sql, ok := c.resolver[name]; ok {
			return sql, nil
		}
	}

	return "", fmt.Errorf("schema %q: source file not found and no resolver entry", name)
}

func (c *dbConfig) buildRegistry() (*db.Registry, error) {
	reg := db.NewRegistry()
	for _, entry := range c.Schema {
		name := entry.Name
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(entry.Source), ".sql")
		}
		sql, err := c.resolveSQL(entry)
		if err != nil {
			return nil, err
		}
		reg.Register(fileSchema{name: name, sql: sql, priority: entry.Priority})
	}
	return reg, nil
}

func (c *dbConfig) buildSchemaInputs() ([]migrate.SchemaInput, error) {
	inputs := make([]migrate.SchemaInput, 0, len(c.Schema))
	for _, entry := range c.Schema {
		name := entry.Name
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(entry.Source), ".sql")
		}
		sql, err := c.resolveSQL(entry)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, migrate.SchemaInput{
			Name:     name,
			SQL:      sql,
			Priority: entry.Priority,
			Group: entry.Group,
		})
	}
	return inputs, nil
}

func (c *dbConfig) buildSeedEntries() ([]seed.Entry, error) {
	var entries []seed.Entry
	for _, e := range c.Schema {
		if e.Seed == "" {
			continue
		}
		sql, err := os.ReadFile(e.Seed)
		if err != nil {
			return nil, fmt.Errorf("seed %s: %w", e.Seed, err)
		}
		expanded := expandEnvWithDefaults(string(sql))
		name := e.Name
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(e.Source), ".sql")
		}
		entries = append(entries, seed.Entry{Name: name, Priority: e.Priority, SQL: expanded})
	}
	return entries, nil
}

func (c *dbConfig) migrationsDir() string {
	if c.Migrations.Destination != "" {
		return c.Migrations.Destination
	}
	return "db/migrations"
}

func (c *dbConfig) dsnFunc() func() (string, error) {
	return db.DSN
}

func expandEnvWithDefaults(s string) string {
	return os.Expand(s, func(key string) string {
		if idx := strings.Index(key, ":-"); idx != -1 {
			envKey := key[:idx]
			defVal := key[idx+2:]
			if val, ok := os.LookupEnv(envKey); ok && val != "" {
				return val
			}
			return defVal
		}
		return os.Getenv(key)
	})
}
