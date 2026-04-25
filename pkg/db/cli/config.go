package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-sum/db"
	"github.com/go-sum/db/codegen"
	"github.com/go-sum/db/compose"
)

type dbConfig struct {
	Schema     []schemaEntry        `yaml:"schema"`
	Migrations string               `yaml:"migrations"`
	Codegen    codegen.Config       `yaml:"codegen"`
	PlanDB     compose.PlanDBConfig `yaml:"plan_db"`
	baseDir    string
}

type schemaEntry struct {
	Name         string   `yaml:"name"`
	Path         string   `yaml:"path"`
	Priority     int      `yaml:"priority"`
	HealthTables []string `yaml:"health_tables"`
	External     bool     `yaml:"external"`
}

type fileSchema struct {
	name     string
	sql      string
	priority int
	tables   []string
}

func (f fileSchema) Name() string           { return f.name }
func (f fileSchema) SQL() string            { return f.sql }
func (f fileSchema) Priority() int          { return f.priority }
func (f fileSchema) HealthTables() []string { return f.tables }

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
	if cfg.Migrations != "" && !filepath.IsAbs(cfg.Migrations) {
		cfg.Migrations = filepath.Join(baseDir, cfg.Migrations)
	}
	for i := range cfg.Schema {
		if !filepath.IsAbs(cfg.Schema[i].Path) {
			cfg.Schema[i].Path = filepath.Join(baseDir, cfg.Schema[i].Path)
		}
	}
	cfg.baseDir = baseDir
	return &cfg, nil
}

func (c *dbConfig) buildRegistry() (*db.Registry, error) {
	reg := db.NewRegistry()
	for _, entry := range c.Schema {
		sql, err := os.ReadFile(entry.Path)
		if err != nil {
			return nil, fmt.Errorf("schema %s: %w", entry.Path, err)
		}
		name := strings.TrimSuffix(filepath.Base(entry.Path), ".sql")
		tables := entry.HealthTables
		if len(tables) == 0 {
			tables = nil
		}
		reg.Register(fileSchema{
			name:     name,
			sql:      string(sql),
			priority: entry.Priority,
			tables:   tables,
		})
	}
	return reg, nil
}

func (c *dbConfig) migrationsDir() string {
	if c.Migrations != "" {
		return c.Migrations
	}
	return "db/migrations"
}

func (c *dbConfig) queriesDir() string {
	if c.Codegen.Queries == "" {
		return filepath.Join(c.baseDir, "sql/queries")
	}
	if filepath.IsAbs(c.Codegen.Queries) {
		return c.Codegen.Queries
	}
	return filepath.Join(c.baseDir, c.Codegen.Queries)
}

func (c *dbConfig) schemaFiles() []string {
	paths := make([]string, len(c.Schema))
	for i, s := range c.Schema {
		paths[i] = s.Path
	}
	return paths
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
