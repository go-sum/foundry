package ddl

import (
	"regexp"
	"strings"
)

var (
	reAlterAddCol  = regexp.MustCompile(`(?i)^\s*ALTER\s+TABLE\s+(\S+)\s+ADD\s+COLUMN\s+(.+);$`)
	reAlterDropCol = regexp.MustCompile(`(?i)^\s*ALTER\s+TABLE\s+(\S+)\s+DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?(\S+);`)
	reDropTable    = regexp.MustCompile(`(?i)^\s*DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?(\S+)`)
	reDropIndex    = regexp.MustCompile(`(?i)^\s*DROP\s+INDEX\s+(?:IF\s+EXISTS\s+)?(\S+);`)
	reDropExt      = regexp.MustCompile(`(?i)^\s*DROP\s+EXTENSION\s+(?:IF\s+EXISTS\s+)?(\S+);`)
	reDropFunc     = regexp.MustCompile(`(?i)^\s*DROP\s+FUNCTION\s+(?:IF\s+EXISTS\s+)?(\S+);`)
	reDropTrigger  = regexp.MustCompile(`(?i)^\s*DROP\s+TRIGGER\s+(?:IF\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+);`)
)

// Apply takes a base schema and migration SQL containing DDL operations,
// returns a new schema with those operations applied.
// Handles: CREATE TABLE, ALTER TABLE ADD/DROP COLUMN, DROP TABLE,
// CREATE/DROP INDEX, CREATE/DROP EXTENSION, CREATE/DROP FUNCTION,
// CREATE/DROP TRIGGER, and DO $$ trigger blocks.
// Unrecognized statements are silently skipped (best-effort).
func Apply(base *Schema, sql string) *Schema {
	result := deepCopySchema(base)

	// Merge CREATE statements from Parse into result.
	parsed := Parse(sql)
	mergeInto(result, parsed)

	// Scan line-by-line for ALTER/DROP mutations.
	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		if m := reAlterAddCol.FindStringSubmatch(trimmed); m != nil {
			tableName := stripSchemaPrefix(m[1])
			rawCol := strings.TrimSpace(m[2])
			col := parseColumnFromRaw(rawCol)
			applyAddColumn(result, tableName, col)
			continue
		}

		if m := reAlterDropCol.FindStringSubmatch(trimmed); m != nil {
			tableName := stripSchemaPrefix(m[1])
			colName := strings.TrimRight(m[2], ";")
			applyDropColumn(result, tableName, colName)
			continue
		}

		if m := reDropTable.FindStringSubmatch(trimmed); m != nil {
			tableName := stripSchemaPrefix(strings.TrimRight(m[1], ";"))
			// Strip CASCADE and trailing punctuation.
			tableName = strings.Fields(tableName)[0]
			tableName = strings.TrimRight(tableName, ";")
			applyDropTable(result, tableName)
			continue
		}

		if m := reDropIndex.FindStringSubmatch(trimmed); m != nil {
			idxName := strings.TrimRight(m[1], ";")
			applyDropIndex(result, idxName)
			continue
		}

		if m := reDropExt.FindStringSubmatch(trimmed); m != nil {
			extName := strings.TrimRight(m[1], ";")
			applyDropExtension(result, extName)
			continue
		}

		if m := reDropFunc.FindStringSubmatch(trimmed); m != nil {
			funcName := strings.TrimRight(m[1], ";")
			applyDropFunction(result, funcName)
			continue
		}

		if m := reDropTrigger.FindStringSubmatch(trimmed); m != nil {
			trigName := strings.TrimRight(m[1], ";")
			applyDropTrigger(result, trigName)
			continue
		}
	}

	return result
}

// deepCopySchema returns a deep copy of a Schema.
func deepCopySchema(s *Schema) *Schema {
	if s == nil {
		return &Schema{}
	}
	result := &Schema{}

	result.Extensions = make([]string, len(s.Extensions))
	copy(result.Extensions, s.Extensions)

	result.Tables = make([]Table, len(s.Tables))
	for i, t := range s.Tables {
		cols := make([]Column, len(t.Columns))
		copy(cols, t.Columns)
		result.Tables[i] = Table{Name: t.Name, Columns: cols}
	}

	result.Indexes = make([]Index, len(s.Indexes))
	copy(result.Indexes, s.Indexes)

	result.Functions = make([]Function, len(s.Functions))
	copy(result.Functions, s.Functions)

	result.Triggers = make([]Trigger, len(s.Triggers))
	copy(result.Triggers, s.Triggers)

	return result
}

// mergeInto adds objects from src into dst without duplicating entries.
// For tables already present, columns are not re-added (ADD COLUMN handles that).
// Functions and triggers are added or replaced.
func mergeInto(dst, src *Schema) {
	// Extensions: add if not already present.
	extSet := toSet(dst.Extensions)
	for _, e := range src.Extensions {
		if !extSet[e] {
			dst.Extensions = append(dst.Extensions, e)
			extSet[e] = true
		}
	}

	// Tables: add tables not already present.
	existingTables := make(map[string]bool, len(dst.Tables))
	for _, t := range dst.Tables {
		existingTables[t.Name] = true
	}
	for _, t := range src.Tables {
		if !existingTables[t.Name] {
			cols := make([]Column, len(t.Columns))
			copy(cols, t.Columns)
			dst.Tables = append(dst.Tables, Table{Name: t.Name, Columns: cols})
			existingTables[t.Name] = true
		}
	}

	// Indexes: add if not already present.
	existingIdxs := make(map[string]bool, len(dst.Indexes))
	for _, idx := range dst.Indexes {
		existingIdxs[idx.Name] = true
	}
	for _, idx := range src.Indexes {
		if !existingIdxs[idx.Name] {
			dst.Indexes = append(dst.Indexes, idx)
			existingIdxs[idx.Name] = true
		}
	}

	// Functions: add or replace.
	existingFuncs := make(map[string]int, len(dst.Functions))
	for i, fn := range dst.Functions {
		existingFuncs[fn.Name] = i
	}
	for _, fn := range src.Functions {
		if i, exists := existingFuncs[fn.Name]; exists {
			dst.Functions[i] = fn
		} else {
			dst.Functions = append(dst.Functions, fn)
			existingFuncs[fn.Name] = len(dst.Functions) - 1
		}
	}

	// Triggers: add or replace.
	existingTrigs := make(map[string]int, len(dst.Triggers))
	for i, trig := range dst.Triggers {
		existingTrigs[trig.Name] = i
	}
	for _, trig := range src.Triggers {
		if i, exists := existingTrigs[trig.Name]; exists {
			dst.Triggers[i] = trig
		} else {
			dst.Triggers = append(dst.Triggers, trig)
			existingTrigs[trig.Name] = len(dst.Triggers) - 1
		}
	}
}

// parseColumnFromRaw parses a column definition from the text after ADD COLUMN.
func parseColumnFromRaw(raw string) Column {
	// raw looks like: "name TEXT NOT NULL" or "id UUID PRIMARY KEY DEFAULT gen_random_uuid()"
	m := reColumnDef.FindStringSubmatch(raw)
	if m == nil {
		return Column{Raw: raw}
	}
	return parseColumn(m[1], m[2], raw)
}

// stripSchemaPrefix removes a "schema." prefix from a table name.
func stripSchemaPrefix(name string) string {
	if idx := strings.LastIndex(name, "."); idx != -1 {
		return name[idx+1:]
	}
	return name
}

func applyAddColumn(s *Schema, tableName string, col Column) {
	for i, t := range s.Tables {
		if t.Name == tableName {
			// Only add if not already present.
			for _, c := range t.Columns {
				if c.Name == col.Name {
					return
				}
			}
			s.Tables[i].Columns = append(s.Tables[i].Columns, col)
			return
		}
	}
}

func applyDropColumn(s *Schema, tableName, colName string) {
	for i, t := range s.Tables {
		if t.Name == tableName {
			cols := make([]Column, 0, len(t.Columns))
			for _, c := range t.Columns {
				if c.Name != colName {
					cols = append(cols, c)
				}
			}
			s.Tables[i].Columns = cols
			return
		}
	}
}

func applyDropTable(s *Schema, tableName string) {
	tables := make([]Table, 0, len(s.Tables))
	for _, t := range s.Tables {
		if t.Name != tableName {
			tables = append(tables, t)
		}
	}
	s.Tables = tables
}

func applyDropIndex(s *Schema, idxName string) {
	idxs := make([]Index, 0, len(s.Indexes))
	for _, idx := range s.Indexes {
		if idx.Name != idxName {
			idxs = append(idxs, idx)
		}
	}
	s.Indexes = idxs
}

func applyDropExtension(s *Schema, extName string) {
	exts := make([]string, 0, len(s.Extensions))
	for _, e := range s.Extensions {
		if e != extName {
			exts = append(exts, e)
		}
	}
	s.Extensions = exts
}

func applyDropFunction(s *Schema, funcName string) {
	fns := make([]Function, 0, len(s.Functions))
	for _, fn := range s.Functions {
		if fn.Name != funcName {
			fns = append(fns, fn)
		}
	}
	s.Functions = fns
}

func applyDropTrigger(s *Schema, trigName string) {
	trigs := make([]Trigger, 0, len(s.Triggers))
	for _, trig := range s.Triggers {
		if trig.Name != trigName {
			trigs = append(trigs, trig)
		}
	}
	s.Triggers = trigs
}
