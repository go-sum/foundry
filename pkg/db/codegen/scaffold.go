package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// IndexDef holds the parsed definition of a CREATE INDEX statement.
type IndexDef struct {
	Name     string
	Columns  []string
	IsUnique bool
	Where    string // partial index condition; if non-empty, skip from query generation
}

// TableDef holds the parsed definition of a single CREATE TABLE statement.
type TableDef struct {
	Name    string
	Columns []ColumnDef
	Indexes []IndexDef
}

// ColumnDef holds information about a single column within a table.
type ColumnDef struct {
	Name       string
	HasDefault bool
	IsPK       bool
	IsUnique   bool   // inline UNIQUE constraint
	FKRef      string // table name from REFERENCES, empty if none
}

var (
	reScaffoldCreateTable  = regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s*\(`)
	reScaffoldColumnDef    = regexp.MustCompile(`(?i)^\s*([a-z_][a-z0-9_]*)\s+\S+`)
	reScaffoldCreateIndex  = regexp.MustCompile(`(?i)^\s*CREATE\s+INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)\s*\(([^)]+)\)`)
	reScaffoldCreateUIndex = regexp.MustCompile(`(?i)^\s*CREATE\s+UNIQUE\s+INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)\s*\(([^)]+)\)`)
	reScaffoldWhereClause  = regexp.MustCompile(`(?i)\)\s*WHERE\s+(.+)$`)
	reScaffoldReferences   = regexp.MustCompile(`(?i)\bREFERENCES\s+(\S+)\s*\(`)
)

// ParseTables parses CREATE TABLE statements from schemaSQL and returns a
// TableDef for each table found.
func ParseTables(schemaSQL string) []TableDef {
	lines := strings.Split(schemaSQL, "\n")

	var tables []TableDef
	var current *TableDef
	capturing := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !capturing {
			m := reScaffoldCreateTable.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			rawName := m[1]
			// Strip schema prefix (e.g. public.foo → foo)
			if idx := strings.LastIndex(rawName, "."); idx != -1 {
				rawName = rawName[idx+1:]
			}
			// Strip surrounding quotes
			rawName = strings.Trim(rawName, `"`)
			current = &TableDef{Name: rawName}
			capturing = true
			continue
		}

		// End of CREATE TABLE block
		if trimmed == ");" || trimmed == ")" {
			tables = append(tables, *current)
			current = nil
			capturing = false
			continue
		}

		// Skip table-level constraints — not column definitions
		upper := strings.ToUpper(trimmed)
		if strings.HasPrefix(upper, "CONSTRAINT") ||
			strings.HasPrefix(upper, "UNIQUE") ||
			strings.HasPrefix(upper, "FOREIGN") ||
			strings.HasPrefix(upper, "CHECK") ||
			strings.HasPrefix(upper, "PRIMARY KEY (") {
			continue
		}

		m := reScaffoldColumnDef.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		upperLine := strings.ToUpper(line)
		col := ColumnDef{
			Name:       m[1],
			HasDefault: strings.Contains(upperLine, "DEFAULT"),
			IsPK:       strings.Contains(upperLine, "PRIMARY KEY"),
			IsUnique:   strings.Contains(upperLine, "UNIQUE"),
		}
		if rm := reScaffoldReferences.FindStringSubmatch(line); rm != nil {
			col.FKRef = strings.Trim(rm[1], `"`)
		}
		current.Columns = append(current.Columns, col)
	}

	// Second pass: find CREATE [UNIQUE] INDEX statements and attach to tables.
	// First, join multi-line index statements into single lines.
	var indexLines []string
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		upper := strings.ToUpper(trimmed)
		if !strings.HasPrefix(upper, "CREATE INDEX") && !strings.HasPrefix(upper, "CREATE UNIQUE INDEX") {
			continue
		}
		// Accumulate continuation lines until we hit a semicolon.
		joined := trimmed
		for !strings.HasSuffix(strings.TrimSpace(joined), ";") && i+1 < len(lines) {
			i++
			joined += " " + strings.TrimSpace(lines[i])
		}
		indexLines = append(indexLines, joined)
	}

	tablesByName := make(map[string]*TableDef, len(tables))
	for i := range tables {
		tablesByName[tables[i].Name] = &tables[i]
	}

	for _, line := range indexLines {
		var idxName, tableName, colsRaw string
		isUnique := false

		if m := reScaffoldCreateUIndex.FindStringSubmatch(line); m != nil {
			idxName = m[1]
			tableName = m[2]
			colsRaw = m[3]
			isUnique = true
		} else if m := reScaffoldCreateIndex.FindStringSubmatch(line); m != nil {
			idxName = m[1]
			tableName = m[2]
			colsRaw = m[3]
		} else {
			continue
		}

		tableName = strings.Trim(tableName, `"`)

		td, ok := tablesByName[tableName]
		if !ok {
			continue
		}

		// Parse columns from the column list.
		rawCols := strings.Split(colsRaw, ",")
		var cols []string
		for _, rc := range rawCols {
			rc = strings.TrimSpace(rc)
			// Strip ASC/DESC suffixes.
			rc = strings.TrimSuffix(rc, " ASC")
			rc = strings.TrimSuffix(rc, " asc")
			rc = strings.TrimSuffix(rc, " DESC")
			rc = strings.TrimSuffix(rc, " desc")
			rc = strings.TrimSpace(rc)
			if rc != "" {
				cols = append(cols, rc)
			}
		}

		idx := IndexDef{
			Name:     idxName,
			Columns:  cols,
			IsUnique: isUnique,
		}

		// Check for WHERE clause on the same line.
		if wm := reScaffoldWhereClause.FindStringSubmatch(line); wm != nil {
			idx.Where = strings.TrimRight(strings.TrimSpace(wm[1]), ";")
		}

		td.Indexes = append(td.Indexes, idx)
	}

	return tables
}

// toPascalCase converts a snake_case identifier to PascalCase.
// Example: contact_submissions → ContactSubmissions
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

// singularize removes a trailing "s" when the preceding character is a
// consonant (not "s" and not a vowel). This handles common plural forms
// like "submissions" → "submission", "jobs" → "job", "users" → "user"
// while leaving words like "address" (ends "ss") and "status" (ends "us") intact.
func singularize(s string) string {
	if len(s) < 2 {
		return s
	}
	if !strings.HasSuffix(s, "s") {
		return s
	}
	prev := s[len(s)-2]
	const vowels = "aeiou"
	// Keep if preceded by 's' (double-s) or a vowel.
	if prev == 's' || strings.ContainsRune(vowels, rune(prev)) {
		return s
	}
	return s[:len(s)-1]
}

// pluralPascal returns the PascalCase form of the table name (already plural).
func pluralPascal(tableName string) string {
	return toPascalCase(tableName)
}

// singularPascal returns the PascalCase form of the singularized table name.
func singularPascal(tableName string) string {
	return toPascalCase(singularize(tableName))
}

// colsKey returns a deduplication key for a set of columns (sorted, comma-joined).
func colsKey(cols []string) string {
	sorted := make([]string, len(cols))
	copy(sorted, cols)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

// colsPascal joins PascalCase column names with "And".
func colsPascal(cols []string) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = toPascalCase(c)
	}
	return strings.Join(parts, "And")
}

// GenerateQueries produces CRUD SQL query annotations for the given TableDef.
func GenerateQueries(t TableDef) string {
	// Determine insert columns: exclude PK+DEFAULT columns and DEFAULT timestamp columns.
	var insertCols []string
	for _, col := range t.Columns {
		if col.IsPK && col.HasDefault {
			continue
		}
		if col.HasDefault && (col.Name == "created_at" || col.Name == "updated_at") {
			continue
		}
		insertCols = append(insertCols, col.Name)
	}

	// Determine PK column.
	pkCol := ""
	for _, col := range t.Columns {
		if col.IsPK {
			pkCol = col.Name
			break
		}
	}
	if pkCol == "" && len(t.Columns) > 0 {
		pkCol = t.Columns[0].Name
	}

	// Determine order-by column.
	orderCol := pkCol
	for _, col := range t.Columns {
		if col.Name == "created_at" {
			orderCol = "created_at"
			break
		}
	}

	// Check for updated_at column.
	hasUpdatedAt := false
	for _, col := range t.Columns {
		if col.Name == "updated_at" {
			hasUpdatedAt = true
			break
		}
	}

	sp := singularPascal(t.Name)
	pp := pluralPascal(t.Name)

	var b strings.Builder

	fmt.Fprintf(&b, "-- Auto-generated CRUD queries for %s\n", t.Name)
	fmt.Fprintf(&b, "-- Edit as needed, then run: db codegen\n")

	// INSERT block
	fmt.Fprintln(&b)
	if len(insertCols) == 0 {
		fmt.Fprintf(&b, "-- No insertable columns (all have defaults)\n")
	} else {
		placeholders := make([]string, len(insertCols))
		for i := range insertCols {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		fmt.Fprintf(&b, "-- name: Insert%s :one\n", sp)
		fmt.Fprintf(&b, "INSERT INTO %s (%s)\n", t.Name, strings.Join(insertCols, ", "))
		fmt.Fprintf(&b, "VALUES (%s)\n", strings.Join(placeholders, ", "))
		fmt.Fprintf(&b, "RETURNING *;\n")
	}

	// GET by PK block
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "-- name: Get%s :one\n", sp)
	fmt.Fprintf(&b, "SELECT * FROM %s\n", t.Name)
	fmt.Fprintf(&b, "WHERE %s = $1;\n", pkCol)

	// GetBy* queries — from inline UNIQUE columns and unique indexes (non-partial).
	generated := make(map[string]bool)

	// Inline UNIQUE columns (single-column).
	for _, col := range t.Columns {
		if !col.IsUnique {
			continue
		}
		key := colsKey([]string{col.Name})
		if generated[key] {
			continue
		}
		generated[key] = true
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "-- name: Get%sBy%s :one\n", sp, colsPascal([]string{col.Name}))
		fmt.Fprintf(&b, "SELECT * FROM %s\n", t.Name)
		fmt.Fprintf(&b, "WHERE %s = $1;\n", col.Name)
	}

	// Unique indexes (non-partial).
	for _, idx := range t.Indexes {
		if !idx.IsUnique || idx.Where != "" {
			continue
		}
		key := colsKey(idx.Columns)
		if generated[key] {
			continue
		}
		generated[key] = true
		whereParts := make([]string, len(idx.Columns))
		for i, c := range idx.Columns {
			whereParts[i] = fmt.Sprintf("%s = $%d", c, i+1)
		}
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "-- name: Get%sBy%s :one\n", sp, colsPascal(idx.Columns))
		fmt.Fprintf(&b, "SELECT * FROM %s\n", t.Name)
		fmt.Fprintf(&b, "WHERE %s;\n", strings.Join(whereParts, " AND "))
	}

	// LIST default block
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "-- name: List%s :many\n", pp)
	fmt.Fprintf(&b, "SELECT * FROM %s\n", t.Name)
	fmt.Fprintf(&b, "ORDER BY %s DESC\n", orderCol)
	fmt.Fprintf(&b, "LIMIT $1 OFFSET $2;\n")

	// ListBy* queries — from non-unique non-partial indexes.
	listGenerated := make(map[string]bool)

	for _, idx := range t.Indexes {
		if idx.IsUnique || idx.Where != "" {
			continue
		}
		key := colsKey(idx.Columns)
		if listGenerated[key] {
			continue
		}
		listGenerated[key] = true
		whereParts := make([]string, len(idx.Columns))
		for i, c := range idx.Columns {
			whereParts[i] = fmt.Sprintf("%s = $%d", c, i+1)
		}
		nextParam := len(idx.Columns) + 1
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "-- name: List%sBy%s :many\n", pp, colsPascal(idx.Columns))
		fmt.Fprintf(&b, "SELECT * FROM %s\n", t.Name)
		fmt.Fprintf(&b, "WHERE %s\n", strings.Join(whereParts, " AND "))
		fmt.Fprintf(&b, "ORDER BY %s DESC\n", orderCol)
		fmt.Fprintf(&b, "LIMIT $%d OFFSET $%d;\n", nextParam, nextParam+1)
	}

	// FK-derived ListBy queries (if not already covered by an index).
	for _, col := range t.Columns {
		if col.FKRef == "" {
			continue
		}
		key := colsKey([]string{col.Name})
		if listGenerated[key] {
			continue
		}
		listGenerated[key] = true
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "-- name: List%sBy%s :many\n", pp, colsPascal([]string{col.Name}))
		fmt.Fprintf(&b, "SELECT * FROM %s\n", t.Name)
		fmt.Fprintf(&b, "WHERE %s = $1\n", col.Name)
		fmt.Fprintf(&b, "ORDER BY %s DESC\n", orderCol)
		fmt.Fprintf(&b, "LIMIT $2 OFFSET $3;\n")
	}

	// UPDATE block — only if table has updated_at column.
	if hasUpdatedAt {
		var mutableCols []string
		for _, col := range t.Columns {
			if col.IsPK || col.Name == "created_at" || col.Name == "updated_at" {
				continue
			}
			mutableCols = append(mutableCols, col.Name)
		}
		if len(mutableCols) > 0 {
			setParts := make([]string, len(mutableCols))
			for i, c := range mutableCols {
				setParts[i] = fmt.Sprintf("%s = $%d", c, i+2) // $1 is PK
			}
			fmt.Fprintln(&b)
			fmt.Fprintf(&b, "-- name: Update%s :one\n", sp)
			fmt.Fprintf(&b, "UPDATE %s\n", t.Name)
			fmt.Fprintf(&b, "SET %s\n", strings.Join(setParts, ", "))
			fmt.Fprintf(&b, "WHERE %s = $1\n", pkCol)
			fmt.Fprintf(&b, "RETURNING *;\n")
		}
	}

	// DELETE block
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "-- name: Delete%s :exec\n", sp)
	fmt.Fprintf(&b, "DELETE FROM %s\n", t.Name)
	fmt.Fprintf(&b, "WHERE %s = $1;\n", pkCol)

	return b.String()
}

// ScaffoldTable parses schemaSQL and writes CRUD query files to outDir.
// If tableName is non-empty, only that table is scaffolded.
// Behaviour when the output file already exists:
//   - force=false, skipExisting=false → error (default: protect manual edits)
//   - force=true                      → overwrite
//   - skipExisting=true               → silently skip (used by db:compose)
//
// Returns the list of file paths created.
func ScaffoldTable(schemaSQL, tableName, outDir string, force, skipExisting bool) ([]string, error) {
	tables := ParseTables(schemaSQL)

	if tableName != "" {
		var filtered []TableDef
		for _, t := range tables {
			if t.Name == tableName {
				filtered = append(filtered, t)
				break
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("scaffold: table %q not found in schema", tableName)
		}
		tables = filtered
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("scaffold: create output dir: %w", err)
	}

	var created []string
	for _, t := range tables {
		outPath := filepath.Join(outDir, t.Name+".sql")
		if _, err := os.Stat(outPath); err == nil {
			switch {
			case force:
				// overwrite — fall through
			case skipExisting:
				continue
			default:
				return nil, fmt.Errorf("scaffold: %s already exists (use --force to overwrite)", outPath)
			}
		}
		content := GenerateQueries(t)
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("scaffold: write %s: %w", outPath, err)
		}
		created = append(created, outPath)
	}

	return created, nil
}
