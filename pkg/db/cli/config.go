package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-sum/db"
	"github.com/go-sum/db/migrate"
	"github.com/go-sum/db/seed"
)

type dbConfig struct {
	Schema     []schemaEntry    `yaml:"schema"`
	Migrations migrationsConfig `yaml:"migrations"`
	baseDir    string
}

type migrationsConfig struct {
	Destination string `yaml:"destination"`
}

type schemaEntry struct {
	Name     string `yaml:"name"`
	Source   string `yaml:"source"`
	Priority int    `yaml:"priority"`
	External bool   `yaml:"external"`
	Discrete bool   `yaml:"discrete"`
	Seed     string `yaml:"seed"`
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
		if !filepath.IsAbs(cfg.Schema[i].Source) {
			cfg.Schema[i].Source = filepath.Join(baseDir, cfg.Schema[i].Source)
		}
		if cfg.Schema[i].Seed != "" && !filepath.IsAbs(cfg.Schema[i].Seed) {
			cfg.Schema[i].Seed = filepath.Join(baseDir, cfg.Schema[i].Seed)
		}
	}
	cfg.baseDir = baseDir
	return &cfg, nil
}

func (c *dbConfig) buildRegistry() (*db.Registry, error) {
	reg := db.NewRegistry()
	for _, entry := range c.Schema {
		sql, err := os.ReadFile(entry.Source)
		if err != nil {
			return nil, fmt.Errorf("schema %s: %w", entry.Source, err)
		}
		name := entry.Name
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(entry.Source), ".sql")
		}
		reg.Register(fileSchema{name: name, sql: string(sql), priority: entry.Priority})
	}
	return reg, nil
}

func (c *dbConfig) buildSchemaInputs() ([]migrate.SchemaInput, error) {
	inputs := make([]migrate.SchemaInput, 0, len(c.Schema))
	for _, entry := range c.Schema {
		sql, err := os.ReadFile(entry.Source)
		if err != nil {
			return nil, fmt.Errorf("schema %s: %w", entry.Source, err)
		}
		name := entry.Name
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(entry.Source), ".sql")
		}
		inputs = append(inputs, migrate.SchemaInput{
			Name:     name,
			SQL:      string(sql),
			Priority: entry.Priority,
			Discrete: entry.Discrete,
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
		name := e.Name
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(e.Source), ".sql")
		}
		entries = append(entries, seed.Entry{Name: name, Priority: e.Priority, SQL: string(sql)})
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
