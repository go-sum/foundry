package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-sum/foundry/pkg/db/ddl"
)

// SchemaInput describes a single schema entry for migration creation.
type SchemaInput struct {
	Name     string
	SQL      string
	Priority int
	Group    string // was: Discrete bool
}

// CreateConfig holds parameters for Create.
type CreateConfig struct {
	Schemas       []SchemaInput
	MigrationsDir string
	// BaseSchemas is required. Keyed by schema name. Populate via BuildBaseline
	// or migrate.IntrospectDSN+ddl.Filter. Missing keys are treated as empty schemas.
	BaseSchemas map[string]*ddl.Schema
}

// CreateResult holds the paths of migration files created.
type CreateResult struct {
	Files []string
}

// Create detects schema changes and writes migration files.
// BaseSchemas must be non-nil; use BuildBaseline to derive it from existing migrations.
// Schemas sharing a Group value are merged into a single migration file named after the group.
// Ungrouped schemas (Group == "") are merged into a single combined file named after name.
// Groups are written in ascending order of their maximum member Priority.
// Returns an empty CreateResult if no changes are detected.
func Create(cfg CreateConfig, name string) (*CreateResult, error) {
	schemas := make([]SchemaInput, len(cfg.Schemas))
	copy(schemas, cfg.Schemas)
	sort.SliceStable(schemas, func(i, j int) bool {
		return schemas[i].Priority < schemas[j].Priority
	})

	if cfg.BaseSchemas == nil {
		return nil, fmt.Errorf("migrate create: BaseSchemas is required")
	}

	type diffEntry struct {
		schema SchemaInput
		result *ddl.DiffResult
	}

	type schemaGroup struct {
		name    string
		entries []diffEntry
		maxPri  int
	}

	groupMap := map[string]*schemaGroup{}
	var groupOrder []string // insertion order preserves priority-tie stability

	for _, schema := range schemas {
		var snapshot *ddl.Schema
		if s, ok := cfg.BaseSchemas[schema.Name]; ok {
			snapshot = s
		} else {
			snapshot = &ddl.Schema{}
		}

		current := ddl.Parse(schema.SQL)
		result := ddl.Diff(snapshot, current)

		if result.Empty {
			continue
		}

		g, ok := groupMap[schema.Group]
		if !ok {
			g = &schemaGroup{name: schema.Group}
			groupMap[schema.Group] = g
			groupOrder = append(groupOrder, schema.Group)
		}
		g.entries = append(g.entries, diffEntry{schema: schema, result: result})
		if schema.Priority > g.maxPri {
			g.maxPri = schema.Priority
		}
	}

	if len(groupMap) == 0 {
		return &CreateResult{}, nil
	}

	sortedGroups := make([]*schemaGroup, 0, len(groupOrder))
	for _, gname := range groupOrder {
		sortedGroups = append(sortedGroups, groupMap[gname])
	}
	sort.SliceStable(sortedGroups, func(i, j int) bool {
		return sortedGroups[i].maxPri < sortedGroups[j].maxPri
	})

	if err := os.MkdirAll(cfg.MigrationsDir, 0o755); err != nil {
		return nil, fmt.Errorf("migrate create: mkdir migrations: %w", err)
	}

	seq, err := nextSequenceNumber(cfg.MigrationsDir)
	if err != nil {
		return nil, err
	}

	var files []string

	for _, g := range sortedGroups {
		var upParts, downParts []string
		for _, entry := range g.entries {
			upParts = append(upParts, entry.result.UpSQL)
			downParts = append(downParts, entry.result.DownSQL)
		}
		upSQL := strings.Join(upParts, "\n\n")
		downSQL := strings.Join(downParts, "\n\n")

		migName := g.name
		if migName == "" {
			migName = name
			if migName == "" {
				migName = "migrate_schema"
			}
		}

		fileName := fmt.Sprintf("%05d_%s.sql", seq, sanitizeName(migName))
		filePath := filepath.Join(cfg.MigrationsDir, fileName)
		content := formatMigrationFile(upSQL, downSQL)
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("migrate create: write %s: %w", filePath, err)
		}
		files = append(files, filePath)
		seq++
	}

	return &CreateResult{Files: files}, nil
}

// BuildBaseline replays all existing migration Up sections in order and
// returns a per-schema baseline map ready for use as CreateConfig.BaseSchemas.
func BuildBaseline(migrationsDir string, schemas []SchemaInput) (map[string]*ddl.Schema, error) {
	migs, err := ParseDir(migrationsDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("migrate build baseline: %w", err)
	}
	combined := &ddl.Schema{}
	for _, m := range migs {
		combined = ddl.Apply(combined, m.UpSQL)
	}
	result := make(map[string]*ddl.Schema, len(schemas))
	for _, s := range schemas {
		result[s.Name] = ddl.Filter(combined, ddl.Parse(s.SQL))
	}
	return result, nil
}

func formatMigrationFile(upSQL, downSQL string) string {
	return fmt.Sprintf("-- Auto-generated migration - do not edit\n\n-- +migrate Up\n%s\n\n-- +migrate Down\n%s\n", upSQL, downSQL)
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
