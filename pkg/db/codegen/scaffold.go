package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	Type       string // raw SQL type string (e.g. "TEXT", "UUID", "TIMESTAMPTZ")
	Nullable   bool   // true when NOT NULL is absent
	HasDefault bool
	IsPK       bool
	IsUnique   bool   // inline UNIQUE constraint
	FKRef      string // table name from REFERENCES, empty if none
}

var (
	reScaffoldCreateTable  = regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s*\(`)
	reScaffoldColumnDef    = regexp.MustCompile(`(?i)^\s*([a-z_][a-z0-9_]*)\s+(\S+)`)
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
			Type:       m[2],
			Nullable:   !strings.Contains(upperLine, "NOT NULL"),
			HasDefault: strings.Contains(upperLine, "DEFAULT"),
			IsPK:       strings.Contains(upperLine, "PRIMARY KEY"),
			IsUnique:   strings.Contains(upperLine, "UNIQUE"),
		}
		// PK columns are implicitly NOT NULL
		if col.IsPK {
			col.Nullable = false
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

// goType maps a SQL type string to its Go equivalent.
// nullable=true returns a pointer type for scalar types.
func goType(sqlType string, nullable bool) string {
	base := map[string]string{
		"uuid":        "uuid.UUID",
		"text":        "string",
		"citext":      "string",
		"varchar":     "string",
		"boolean":     "bool",
		"bool":        "bool",
		"integer":     "int32",
		"int":         "int32",
		"bigint":      "int64",
		"timestamptz": "time.Time",
		"jsonb":       "json.RawMessage",
		"bytea":       "[]byte",
	}
	// normalize: take first word, lowercase, strip trailing comma and parenthesized args (e.g. varchar(20))
	t := strings.ToLower(strings.Fields(sqlType)[0])
	t = strings.TrimSuffix(t, ",")
	if idx := strings.Index(t, "("); idx != -1 {
		t = t[:idx]
	}
	if b, ok := base[t]; ok {
		if nullable && b != "[]byte" && b != "json.RawMessage" {
			return "*" + b
		}
		return b
	}
	return "string" // safe fallback
}

// GenerateGoStore produces a complete Go file with inline SQL constants,
// a scan helper, a Store struct, and CRUD methods for the given TableDef.
func GenerateGoStore(t TableDef, pkgName string) string {
	sp := singularPascal(t.Name)
	pp := pluralPascal(t.Name)

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

	// Determine PK Go type.
	pkGoType := "string"
	for _, col := range t.Columns {
		if col.IsPK {
			pkGoType = goType(col.Type, false)
			break
		}
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

	// Collect all column names for SELECT.
	var allCols []string
	for _, col := range t.Columns {
		allCols = append(allCols, col.Name)
	}
	allColsStr := strings.Join(allCols, ", ")

	// Determine required imports.
	needsUUID := false
	needsTime := false
	needsJSON := false
	for _, col := range t.Columns {
		gt := goType(col.Type, col.Nullable)
		gt = strings.TrimPrefix(gt, "*")
		if gt == "uuid.UUID" {
			needsUUID = true
		}
		if gt == "time.Time" {
			needsTime = true
		}
		if gt == "json.RawMessage" {
			needsJSON = true
		}
	}

	var b strings.Builder

	// Package and imports
	fmt.Fprintf(&b, "package %s\n\n", pkgName)
	fmt.Fprintf(&b, "import (\n")
	fmt.Fprintf(&b, "\t\"context\"\n")
	if needsJSON {
		fmt.Fprintf(&b, "\t\"encoding/json\"\n")
	}
	if needsTime {
		fmt.Fprintf(&b, "\t\"time\"\n")
	}
	fmt.Fprintf(&b, "\n")
	if needsUUID {
		fmt.Fprintf(&b, "\t\"github.com/google/uuid\"\n")
	}
	fmt.Fprintf(&b, "\t\"github.com/jackc/pgx/v5\"\n")
	fmt.Fprintf(&b, "\t\"github.com/jackc/pgx/v5/pgxpool\"\n")
	fmt.Fprintf(&b, ")\n\n")

	// Model struct
	fmt.Fprintf(&b, "// %s is the domain model for the %s table.\n", sp, t.Name)
	fmt.Fprintf(&b, "type %s struct {\n", sp)
	for _, col := range t.Columns {
		gt := goType(col.Type, col.Nullable)
		fmt.Fprintf(&b, "\t%s %s\n", toPascalCase(col.Name), gt)
	}
	fmt.Fprintf(&b, "}\n\n")

	// scan helper
	fmt.Fprintf(&b, "func scan%s(row pgx.Row) (%s, error) {\n", sp, sp)
	fmt.Fprintf(&b, "\tvar m %s\n", sp)
	fmt.Fprintf(&b, "\terr := row.Scan(\n")
	for _, col := range t.Columns {
		fmt.Fprintf(&b, "\t\t&m.%s,\n", toPascalCase(col.Name))
	}
	fmt.Fprintf(&b, "\t)\n")
	fmt.Fprintf(&b, "\treturn m, err\n")
	fmt.Fprintf(&b, "}\n\n")

	// Store struct
	fmt.Fprintf(&b, "// Store provides CRUD operations for %s.\n", t.Name)
	fmt.Fprintf(&b, "type Store struct {\n")
	fmt.Fprintf(&b, "\tpool *pgxpool.Pool\n")
	fmt.Fprintf(&b, "}\n\n")

	fmt.Fprintf(&b, "// NewStore creates a Store backed by pool.\n")
	fmt.Fprintf(&b, "func NewStore(pool *pgxpool.Pool) *Store {\n")
	fmt.Fprintf(&b, "\treturn &Store{pool: pool}\n")
	fmt.Fprintf(&b, "}\n\n")

	// INSERT
	if len(insertCols) > 0 {
		placeholders := make([]string, len(insertCols))
		for i := range insertCols {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		fmt.Fprintf(&b, "const insert%s = `\n", sp)
		fmt.Fprintf(&b, "INSERT INTO %s (%s)\n", t.Name, strings.Join(insertCols, ", "))
		fmt.Fprintf(&b, "VALUES (%s)\n", strings.Join(placeholders, ", "))
		fmt.Fprintf(&b, "RETURNING %s`\n\n", allColsStr)

		fmt.Fprintf(&b, "// Insert%s inserts a new record and returns the created row.\n", sp)
		fmt.Fprintf(&b, "func (s *Store) Insert%s(ctx context.Context, m %s) (%s, error) {\n", sp, sp, sp)
		fmt.Fprintf(&b, "\treturn scan%s(s.pool.QueryRow(ctx, insert%s,\n", sp, sp)
		for _, col := range insertCols {
			fmt.Fprintf(&b, "\t\tm.%s,\n", toPascalCase(col))
		}
		fmt.Fprintf(&b, "\t))\n")
		fmt.Fprintf(&b, "}\n\n")
	}

	// GET by PK
	fmt.Fprintf(&b, "const get%s = `\n", sp)
	fmt.Fprintf(&b, "SELECT %s FROM %s\n", allColsStr, t.Name)
	fmt.Fprintf(&b, "WHERE %s = $1`\n\n", pkCol)

	fmt.Fprintf(&b, "// Get%s returns a record by %s.\n", sp, pkCol)
	fmt.Fprintf(&b, "func (s *Store) Get%s(ctx context.Context, %s %s) (%s, error) {\n", sp, pkCol, pkGoType, sp)
	fmt.Fprintf(&b, "\treturn scan%s(s.pool.QueryRow(ctx, get%s, %s))\n", sp, sp, pkCol)
	fmt.Fprintf(&b, "}\n\n")

	// LIST
	fmt.Fprintf(&b, "const list%s = `\n", pp)
	fmt.Fprintf(&b, "SELECT %s FROM %s\n", allColsStr, t.Name)
	fmt.Fprintf(&b, "ORDER BY %s DESC\n", orderCol)
	fmt.Fprintf(&b, "LIMIT $1 OFFSET $2`\n\n")

	fmt.Fprintf(&b, "// List%s returns a paginated list ordered by %s descending.\n", pp, orderCol)
	fmt.Fprintf(&b, "func (s *Store) List%s(ctx context.Context, limit, offset int32) ([]%s, error) {\n", pp, sp)
	fmt.Fprintf(&b, "\trows, err := s.pool.Query(ctx, list%s, limit, offset)\n", pp)
	fmt.Fprintf(&b, "\tif err != nil {\n")
	fmt.Fprintf(&b, "\t\treturn nil, err\n")
	fmt.Fprintf(&b, "\t}\n")
	fmt.Fprintf(&b, "\tdefer rows.Close()\n")
	fmt.Fprintf(&b, "\tvar items []%s\n", sp)
	fmt.Fprintf(&b, "\tfor rows.Next() {\n")
	fmt.Fprintf(&b, "\t\tm, err := scan%s(rows)\n", sp)
	fmt.Fprintf(&b, "\t\tif err != nil {\n")
	fmt.Fprintf(&b, "\t\t\treturn nil, err\n")
	fmt.Fprintf(&b, "\t\t}\n")
	fmt.Fprintf(&b, "\t\titems = append(items, m)\n")
	fmt.Fprintf(&b, "\t}\n")
	fmt.Fprintf(&b, "\treturn items, rows.Err()\n")
	fmt.Fprintf(&b, "}\n\n")

	// UPDATE — only if table has updated_at column.
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
			fmt.Fprintf(&b, "const update%s = `\n", sp)
			fmt.Fprintf(&b, "UPDATE %s\n", t.Name)
			fmt.Fprintf(&b, "SET %s\n", strings.Join(setParts, ", "))
			fmt.Fprintf(&b, "WHERE %s = $1\n", pkCol)
			fmt.Fprintf(&b, "RETURNING %s`\n\n", allColsStr)

			fmt.Fprintf(&b, "// Update%s updates mutable fields of a record by %s.\n", sp, pkCol)
			fmt.Fprintf(&b, "func (s *Store) Update%s(ctx context.Context, m %s) (%s, error) {\n", sp, sp, sp)
			fmt.Fprintf(&b, "\treturn scan%s(s.pool.QueryRow(ctx, update%s,\n", sp, sp)
			fmt.Fprintf(&b, "\t\tm.%s,\n", toPascalCase(pkCol))
			for _, col := range mutableCols {
				fmt.Fprintf(&b, "\t\tm.%s,\n", toPascalCase(col))
			}
			fmt.Fprintf(&b, "\t))\n")
			fmt.Fprintf(&b, "}\n\n")
		}
	}

	// DELETE
	fmt.Fprintf(&b, "const delete%s = `\n", sp)
	fmt.Fprintf(&b, "DELETE FROM %s\n", t.Name)
	fmt.Fprintf(&b, "WHERE %s = $1`\n\n", pkCol)

	fmt.Fprintf(&b, "// Delete%s removes a record by %s.\n", sp, pkCol)
	fmt.Fprintf(&b, "func (s *Store) Delete%s(ctx context.Context, %s %s) error {\n", sp, pkCol, pkGoType)
	fmt.Fprintf(&b, "\t_, err := s.pool.Exec(ctx, delete%s, %s)\n", sp, pkCol)
	fmt.Fprintf(&b, "\treturn err\n")
	fmt.Fprintf(&b, "}\n")

	return b.String()
}

// ScaffoldTable parses schemaSQL and writes CRUD Go store files to outDir.
// If tableName is non-empty, only that table is scaffolded.
// Behaviour when the output file already exists:
//   - force=false, skipExisting=false → error (default: protect manual edits)
//   - force=true                      → overwrite
//   - skipExisting=true               → silently skip (used by db:compose)
//
// Returns the list of file paths created.
func ScaffoldTable(schemaSQL, tableName, pkgName, outDir string, force, skipExisting bool) ([]string, error) {
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
		outPath := filepath.Join(outDir, t.Name+"_store.go")
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
		content := GenerateGoStore(t, pkgName)
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("scaffold: write %s: %w", outPath, err)
		}
		created = append(created, outPath)
	}

	return created, nil
}
