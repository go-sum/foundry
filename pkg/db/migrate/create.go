package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-sum/foundry/pkg/db/ddl"
)

// snapshotDir resolves the effective snapshot directory for a CreateConfig.
func (cfg *CreateConfig) snapshotDir() string {
	if cfg.SnapshotDir != "" {
		return cfg.SnapshotDir
	}
	return filepath.Join(cfg.MigrationsDir, ".schema")
}

// SchemaInput describes a single schema entry for migration creation.
type SchemaInput struct {
	Name     string
	SQL      string
	Priority int
	Discrete bool
}

// CreateConfig holds parameters for Create.
type CreateConfig struct {
	Schemas       []SchemaInput
	MigrationsDir string
	// SnapshotDir overrides the default <MigrationsDir>/.schema/ snapshot location.
	// When set (e.g. a /tmp path), snapshot reads and writes use this directory.
	SnapshotDir string
	// BaseSchemas, when non-nil, is used as the diff baseline instead of reading
	// snapshot files. Keyed by schema name. Missing keys are treated as empty schemas.
	BaseSchemas map[string]*ddl.Schema
}

// CreateResult holds the paths of migration files created.
type CreateResult struct {
	Files []string
}

// Create detects schema changes via snapshot comparison and writes migration files.
// For discrete schemas with changes: own migration file per schema.
// For non-discrete schemas with changes: single combined migration file.
// Snapshots are stored in <MigrationsDir>/.schema/<name>.sql.
// Returns an empty CreateResult if no changes detected.
func Create(cfg CreateConfig, name string) (*CreateResult, error) {
	schemas := make([]SchemaInput, len(cfg.Schemas))
	copy(schemas, cfg.Schemas)
	sort.SliceStable(schemas, func(i, j int) bool {
		return schemas[i].Priority < schemas[j].Priority
	})

	snapshotDir := cfg.snapshotDir()

	type diffEntry struct {
		schema SchemaInput
		result *ddl.DiffResult
	}

	var discreteDiffs []diffEntry
	var nonDiscreteDiffs []diffEntry

	for _, schema := range schemas {
		var snapshot *ddl.Schema
		if cfg.BaseSchemas != nil {
			if s, ok := cfg.BaseSchemas[schema.Name]; ok {
				snapshot = s
			} else {
				snapshot = &ddl.Schema{}
			}
		} else {
			snapshotPath := filepath.Join(snapshotDir, schema.Name+".sql")
			snapshotSQL := ""
			data, err := os.ReadFile(snapshotPath)
			if err == nil {
				snapshotSQL = string(data)
			}
			snapshot = ddl.Parse(snapshotSQL)
		}

		current := ddl.Parse(schema.SQL)
		result := ddl.Diff(snapshot, current)

		if result.Empty {
			continue
		}

		if schema.Discrete {
			discreteDiffs = append(discreteDiffs, diffEntry{schema: schema, result: result})
		} else {
			nonDiscreteDiffs = append(nonDiscreteDiffs, diffEntry{schema: schema, result: result})
		}
	}

	if len(discreteDiffs) == 0 && len(nonDiscreteDiffs) == 0 {
		return &CreateResult{}, nil
	}

	if err := os.MkdirAll(cfg.MigrationsDir, 0o755); err != nil {
		return nil, fmt.Errorf("migrate create: mkdir migrations: %w", err)
	}

	seq, err := nextSequenceNumber(cfg.MigrationsDir)
	if err != nil {
		return nil, err
	}

	var files []string

	for _, entry := range discreteDiffs {
		fileName := fmt.Sprintf("%05d_%s.sql", seq, sanitizeName(entry.schema.Name))
		filePath := filepath.Join(cfg.MigrationsDir, fileName)
		content := formatMigrationFile(entry.result.UpSQL, entry.result.DownSQL)
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("migrate create: write %s: %w", filePath, err)
		}
		files = append(files, filePath)
		seq++
	}

	if len(nonDiscreteDiffs) > 0 {
		migName := name
		if migName == "" {
			migName = "migrate_schema"
		}
		var upParts, downParts []string
		for _, entry := range nonDiscreteDiffs {
			upParts = append(upParts, entry.result.UpSQL)
			downParts = append(downParts, entry.result.DownSQL)
		}
		upSQL := strings.Join(upParts, "\n\n")
		downSQL := strings.Join(downParts, "\n\n")

		fileName := fmt.Sprintf("%05d_%s.sql", seq, sanitizeName(migName))
		filePath := filepath.Join(cfg.MigrationsDir, fileName)
		content := formatMigrationFile(upSQL, downSQL)
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("migrate create: write %s: %w", filePath, err)
		}
		files = append(files, filePath)
	}

	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return nil, fmt.Errorf("migrate create: mkdir snapshot dir: %w", err)
	}
	for _, schema := range schemas {
		snapshotPath := filepath.Join(snapshotDir, schema.Name+".sql")
		if err := os.WriteFile(snapshotPath, []byte(schema.SQL), 0o644); err != nil {
			return nil, fmt.Errorf("migrate create: write snapshot %s: %w", snapshotPath, err)
		}
	}

	return &CreateResult{Files: files}, nil
}

func formatMigrationFile(upSQL, downSQL string) string {
	return fmt.Sprintf("-- +migrate Up\n%s\n\n-- +migrate Down\n%s\n", upSQL, downSQL)
}

func nextSequenceNumber(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("migrate: read migrations dir: %w", err)
	}

	max := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(e.Name(), "%d", &n); err == nil {
			if n > max {
				max = n
			}
		}
	}
	return max + 1, nil
}

func sanitizeName(name string) string {
	if i := strings.IndexByte(name, '_'); i > 0 && strings.TrimLeft(name[:i], "0123456789") == "" {
		name = name[i+1:]
	}
	replacer := strings.NewReplacer(" ", "_", "-", "_", "/", "_")
	return strings.ToLower(replacer.Replace(name))
}
