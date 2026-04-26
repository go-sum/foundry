package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration represents a parsed migration file.
type Migration struct {
	Version int64
	Name    string
	Source  string
	UpSQL   string
	DownSQL string
}

// MigrationStatus describes the state of a single migration.
type MigrationStatus struct {
	Version   int64
	Name      string
	Applied   bool
	AppliedAt time.Time
	Source    string
}

// ParseFile reads a .sql migration file and extracts Up/Down sections.
// Sections are delimited by "-- +migrate Up" and "-- +migrate Down".
func ParseFile(path string) (*Migration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("migrate: parse file %s: %w", path, err)
	}

	base := filepath.Base(path)
	var version int64
	if _, err := fmt.Sscanf(base, "%d_", &version); err != nil {
		return nil, fmt.Errorf("migrate: parse version from %s: %w", base, err)
	}

	name := ""
	if idx := strings.Index(base, "_"); idx != -1 {
		rest := base[idx+1:]
		name = strings.TrimSuffix(rest, ".sql")
	}

	const (
		sectionNone = iota
		sectionUp
		sectionDown
	)

	section := sectionNone
	var upLines, downLines []string

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- +migrate Up") {
			section = sectionUp
			continue
		}
		if strings.HasPrefix(trimmed, "-- +migrate Down") {
			section = sectionDown
			continue
		}
		switch section {
		case sectionUp:
			upLines = append(upLines, line)
		case sectionDown:
			downLines = append(downLines, line)
		}
	}

	return &Migration{
		Version: version,
		Name:    name,
		Source:  path,
		UpSQL:   strings.TrimSpace(strings.Join(upLines, "\n")),
		DownSQL: strings.TrimSpace(strings.Join(downLines, "\n")),
	}, nil
}

// ParseDir scans a directory for NNNNN_*.sql files (top-level only, no recursion).
// Skips files/dirs starting with "." to exclude .schema/ snapshot directory.
// Returns migrations sorted by version ascending.
func ParseDir(dir string) ([]*Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("migrate: read dir %s: %w", dir, err)
	}

	var migrations []*Migration
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		m, err := ParseFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// SplitStatements splits SQL on semicolons, respecting $$ dollar-quoted blocks.
// Blank statements (whitespace only) are omitted from results.
func SplitStatements(sql string) []string {
	var stmts []string
	var current strings.Builder
	inDollar := false

	i := 0
	for i < len(sql) {
		if i+1 < len(sql) && sql[i] == '$' && sql[i+1] == '$' {
			inDollar = !inDollar
			current.WriteString("$$")
			i += 2
			continue
		}
		if !inDollar && sql[i] == ';' {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				stmts = append(stmts, stmt+";")
			}
			current.Reset()
			i++
			continue
		}
		current.WriteByte(sql[i])
		i++
	}

	if stmt := strings.TrimSpace(current.String()); stmt != "" {
		stmts = append(stmts, stmt)
	}

	return stmts
}
