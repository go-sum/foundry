package ddl

import (
	"regexp"
	"strings"
)

// Schema holds all DDL objects parsed from a SQL string.
type Schema struct {
	Extensions []string
	Tables     []Table
	Indexes    []Index
	Functions  []Function
	Triggers   []Trigger
}

// Table represents a CREATE TABLE definition.
type Table struct {
	Name    string
	Columns []Column
}

// Column represents a single column within a table.
type Column struct {
	Name     string
	Type     string // normalized: lowercase first word, parens stripped
	Nullable bool
	Default  string // actual default expression (empty = no default)
	IsPK     bool
	IsUnique bool
	FKRef    string // referenced table name, empty if none
	Raw      string // full column definition line (trimmed, no trailing comma)
}

// Index represents a CREATE INDEX definition.
type Index struct {
	Name     string
	Table    string
	Columns  []string
	IsUnique bool
	Where    string
	Raw      string // full CREATE [UNIQUE] INDEX ... statement (no semicolon)
}

// Function represents a CREATE OR REPLACE FUNCTION definition.
type Function struct {
	Name string
	Body string // full CREATE OR REPLACE FUNCTION ... statement, trimmed
}

// Trigger represents a DO $$ ... $$; trigger block.
type Trigger struct {
	Name  string
	Table string
	Body  string // full DO $$ ... $$; block, trimmed
}

var (
	reCreateTable  = regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s*\(`)
	reColumnDef    = regexp.MustCompile(`(?i)^\s*([a-z_][a-z0-9_]*)\s+(\S+)`)
	reCreateIndex  = regexp.MustCompile(`(?i)^\s*CREATE\s+INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)\s*\(([^)]+)\)`)
	reCreateUIndex = regexp.MustCompile(`(?i)^\s*CREATE\s+UNIQUE\s+INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)\s*\(([^)]+)\)`)
	reWhereClause  = regexp.MustCompile(`(?i)\)\s*WHERE\s+(.+)$`)
	reReferences   = regexp.MustCompile(`(?i)\bREFERENCES\s+(\S+)\s*\(`)
	reExtension    = regexp.MustCompile(`(?i)CREATE\s+EXTENSION\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)`)
	reFuncName     = regexp.MustCompile(`(?i)CREATE\s+OR\s+REPLACE\s+FUNCTION\s+(\S+)\s*\(`)
	reTriggerName  = regexp.MustCompile(`(?i)CREATE\s+TRIGGER\s+(\S+)`)
	reTriggerOn    = regexp.MustCompile(`(?i)\bON\s+(\S+)`)
)

// Parse extracts all DDL objects from a SQL string.
// Errors are silently skipped (best-effort parser).
func Parse(sql string) *Schema {
	s := &Schema{}
	lines := strings.Split(sql, "\n")

	var (
		// Table parsing state
		capturingTable bool
		currentTable   *Table

		// Function parsing state
		capturingFunc bool
		funcLines     []string
		dollarDepth   int

		// DO block (trigger) parsing state
		capturingDo bool
		doLines     []string
	)

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// --- Function capture ---
		if capturingFunc {
			funcLines = append(funcLines, line)
			// Toggle dollar-quote depth on $$
			depth := strings.Count(line, "$$")
			dollarDepth += depth
			// dollarDepth started at 1 (opening $$). Even count means we've
			// closed all dollar-quote blocks.
			if dollarDepth%2 == 0 && strings.Contains(trimmed, ";") {
				fn := parseFunction(funcLines)
				if fn != nil {
					s.Functions = append(s.Functions, *fn)
				}
				capturingFunc = false
				funcLines = nil
				dollarDepth = 0
			}
			continue
		}

		// --- DO block capture ---
		if capturingDo {
			doLines = append(doLines, line)
			if strings.HasSuffix(trimmed, "$$;") || trimmed == "END $$;" {
				trig := parseTrigger(doLines)
				if trig != nil {
					s.Triggers = append(s.Triggers, *trig)
				}
				capturingDo = false
				doLines = nil
			}
			continue
		}

		// --- Table capture ---
		if capturingTable {
			if trimmed == ");" || trimmed == ")" {
				s.Tables = append(s.Tables, *currentTable)
				currentTable = nil
				capturingTable = false
				continue
			}

			upper := strings.ToUpper(trimmed)
			if strings.HasPrefix(upper, "CONSTRAINT") ||
				strings.HasPrefix(upper, "UNIQUE") ||
				strings.HasPrefix(upper, "FOREIGN") ||
				strings.HasPrefix(upper, "CHECK") ||
				strings.HasPrefix(upper, "PRIMARY KEY (") {
				continue
			}

			m := reColumnDef.FindStringSubmatch(line)
			if m == nil {
				continue
			}

			col := parseColumn(m[1], m[2], line)
			currentTable.Columns = append(currentTable.Columns, col)
			continue
		}

		// --- Extension ---
		if m := reExtension.FindStringSubmatch(line); m != nil {
			name := strings.Trim(strings.TrimRight(m[1], ";"), `"'`)
			s.Extensions = append(s.Extensions, name)
			continue
		}

		// --- CREATE TABLE ---
		if m := reCreateTable.FindStringSubmatch(line); m != nil {
			rawName := m[1]
			if idx := strings.LastIndex(rawName, "."); idx != -1 {
				rawName = rawName[idx+1:]
			}
			rawName = strings.Trim(rawName, `"`)
			currentTable = &Table{Name: rawName}
			capturingTable = true
			continue
		}

		// --- CREATE FUNCTION ---
		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, "CREATE OR REPLACE FUNCTION") || strings.Contains(upper, "CREATE FUNCTION") {
			funcLines = []string{line}
			dollarDepth = strings.Count(line, "$$")
			capturingFunc = true
			continue
		}

		// --- DO block ---
		if strings.HasPrefix(upper, "DO $$") || strings.HasPrefix(upper, "DO $$ ") {
			doLines = []string{line}
			// Check if the DO block closes on the same line (shouldn't happen in practice)
			if strings.HasSuffix(trimmed, "$$;") && len(trimmed) > 5 {
				trig := parseTrigger(doLines)
				if trig != nil {
					s.Triggers = append(s.Triggers, *trig)
				}
				doLines = nil
			} else {
				capturingDo = true
			}
			continue
		}

		// --- CREATE INDEX (potentially multi-line) ---
		upperTrimmed := strings.ToUpper(trimmed)
		if strings.HasPrefix(upperTrimmed, "CREATE INDEX") || strings.HasPrefix(upperTrimmed, "CREATE UNIQUE INDEX") {
			joined := trimmed
			for !strings.HasSuffix(strings.TrimSpace(joined), ";") && i+1 < len(lines) {
				i++
				joined += " " + strings.TrimSpace(lines[i])
			}
			idx := parseIndex(joined)
			if idx != nil {
				s.Indexes = append(s.Indexes, *idx)
			}
			continue
		}
	}

	return s
}

// parseColumn builds a Column from the matched name, raw type, and full line.
func parseColumn(name, rawType, line string) Column {
	upperLine := strings.ToUpper(line)
	trimmed := strings.TrimSpace(line)
	raw := strings.TrimRight(trimmed, ",")

	// Normalize type: lowercase, strip parens.
	t := strings.ToLower(rawType)
	t = strings.TrimSuffix(t, ",")
	if idx := strings.Index(t, "("); idx != -1 {
		t = t[:idx]
	}

	col := Column{
		Name:     name,
		Type:     t,
		Nullable: !strings.Contains(upperLine, "NOT NULL"),
		IsPK:     strings.Contains(upperLine, "PRIMARY KEY"),
		IsUnique: strings.Contains(upperLine, "UNIQUE"),
		Raw:      raw,
	}

	if col.IsPK {
		col.Nullable = false
	}

	if strings.Contains(upperLine, "DEFAULT") {
		col.Default = extractDefault(line)
	}

	if rm := reReferences.FindStringSubmatch(line); rm != nil {
		col.FKRef = strings.Trim(rm[1], `"`)
		// Strip trailing punctuation like (id) from the match
		col.FKRef = strings.Split(col.FKRef, "(")[0]
		col.FKRef = strings.TrimRight(col.FKRef, `"`)
	}

	return col
}

// extractDefault scans the line for the DEFAULT keyword and returns the
// expression that follows it. Stops at a comma only when not inside
// parens or quotes, then strips any trailing `,` or `)`.
func extractDefault(line string) string {
	upper := strings.ToUpper(line)
	idx := strings.Index(upper, "DEFAULT ")
	if idx == -1 {
		return ""
	}
	expr := strings.TrimSpace(line[idx+8:])

	var result strings.Builder
	depth := 0
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(expr); i++ {
		ch := expr[i]

		if inQuote {
			result.WriteByte(ch)
			if ch == quoteChar {
				inQuote = false
			}
			continue
		}

		if ch == '\'' || ch == '"' {
			inQuote = true
			quoteChar = ch
			result.WriteByte(ch)
			continue
		}

		if ch == '(' {
			depth++
			result.WriteByte(ch)
			continue
		}

		if ch == ')' {
			if depth == 0 {
				// Outside all parens — stop here (table column list closing paren).
				break
			}
			depth--
			result.WriteByte(ch)
			continue
		}

		if ch == ',' && depth == 0 {
			break
		}

		result.WriteByte(ch)
	}

	return strings.TrimRight(strings.TrimSpace(result.String()), ",")
}

// parseIndex parses a single joined CREATE [UNIQUE] INDEX line.
func parseIndex(line string) *Index {
	// Strip trailing semicolon for Raw storage.
	raw := strings.TrimRight(strings.TrimSpace(line), ";")

	var (
		idxName   string
		tableName string
		colsRaw   string
		isUnique  bool
	)

	if m := reCreateUIndex.FindStringSubmatch(line); m != nil {
		idxName = m[1]
		tableName = m[2]
		colsRaw = m[3]
		isUnique = true
	} else if m := reCreateIndex.FindStringSubmatch(line); m != nil {
		idxName = m[1]
		tableName = m[2]
		colsRaw = m[3]
	} else {
		return nil
	}

	tableName = strings.Trim(tableName, `"`)

	rawCols := strings.Split(colsRaw, ",")
	var cols []string
	for _, rc := range rawCols {
		rc = strings.TrimSpace(rc)
		rc = strings.TrimSuffix(rc, " ASC")
		rc = strings.TrimSuffix(rc, " asc")
		rc = strings.TrimSuffix(rc, " DESC")
		rc = strings.TrimSuffix(rc, " desc")
		rc = strings.TrimSpace(rc)
		if rc != "" {
			cols = append(cols, rc)
		}
	}

	idx := &Index{
		Name:     idxName,
		Table:    tableName,
		Columns:  cols,
		IsUnique: isUnique,
		Raw:      raw,
	}

	if wm := reWhereClause.FindStringSubmatch(line); wm != nil {
		idx.Where = strings.TrimRight(strings.TrimSpace(wm[1]), ";")
	}

	return idx
}

// parseFunction extracts a Function from captured lines.
func parseFunction(lines []string) *Function {
	if len(lines) == 0 {
		return nil
	}
	body := strings.TrimSpace(strings.Join(lines, "\n"))
	m := reFuncName.FindStringSubmatch(lines[0])
	if m == nil {
		return nil
	}
	name := strings.TrimRight(m[1], "(")
	return &Function{Name: name, Body: body}
}

// parseTrigger extracts a Trigger from captured DO block lines.
func parseTrigger(lines []string) *Trigger {
	if len(lines) == 0 {
		return nil
	}
	body := strings.TrimSpace(strings.Join(lines, "\n"))

	var name, table string
	for _, l := range lines {
		if name == "" {
			if m := reTriggerName.FindStringSubmatch(l); m != nil {
				name = m[1]
			}
		}
		if table == "" && name != "" {
			// Look for ON <table> after the trigger name line.
			if m := reTriggerOn.FindStringSubmatch(l); m != nil {
				candidate := strings.Trim(m[1], `"`)
				// Skip "pg_trigger" from the IF NOT EXISTS guard.
				if !strings.EqualFold(candidate, "pg_trigger") {
					table = candidate
				}
			}
		}
	}

	if name == "" {
		return nil
	}

	return &Trigger{Name: name, Table: table, Body: body}
}
