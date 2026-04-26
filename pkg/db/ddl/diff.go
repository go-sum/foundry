package ddl

import (
	"fmt"
	"strings"
)

// DiffResult holds the Up and Down SQL for a schema migration.
type DiffResult struct {
	UpSQL   string
	DownSQL string
	Empty   bool
}

// Diff computes the DDL changes to go from baseline to desired.
// Returns Empty=true if no changes detected.
func Diff(baseline, desired *Schema) *DiffResult {
	var up, down []string

	// --- Extensions ---
	baseExts := toSet(baseline.Extensions)
	for _, ext := range desired.Extensions {
		if !baseExts[ext] {
			up = append(up, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", ext))
			down = append(down, fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", ext))
		}
	}

	desiredExts := toSet(desired.Extensions)
	for _, ext := range baseline.Extensions {
		if !desiredExts[ext] {
			up = append(up, fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", ext))
			down = append(down, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", ext))
		}
	}

	// --- Tables ---
	baseTables := tablesByName(baseline.Tables)
	for _, t := range desired.Tables {
		bt, exists := baseTables[t.Name]
		if !exists {
			up = append(up, reconstructTable(t))
			down = append(down, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", t.Name))
			continue
		}
		// Table exists — diff columns.
		baseCols := colsByName(bt.Columns)
		for _, col := range t.Columns {
			if !baseCols[col.Name] {
				up = append(up, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", t.Name, col.Raw))
				down = append(down, fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s;", t.Name, col.Name))
			}
		}
		// Removed columns.
		desiredCols := colsByName(t.Columns)
		for _, col := range bt.Columns {
			if !desiredCols[col.Name] {
				up = append(up, fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s;", t.Name, col.Name))
				down = append(down, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", t.Name, col.Raw))
			}
		}
	}

	desiredTables := tablesByName(desired.Tables)
	for _, t := range baseline.Tables {
		if _, exists := desiredTables[t.Name]; !exists {
			up = append(up, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", t.Name))
			down = append(down, reconstructTable(t))
		}
	}

	// --- Indexes ---
	baseIdxs := indexesByName(baseline.Indexes)
	for _, idx := range desired.Indexes {
		if _, exists := baseIdxs[idx.Name]; !exists {
			up = append(up, idx.Raw+";")
			down = append(down, fmt.Sprintf("DROP INDEX IF EXISTS %s;", idx.Name))
		}
	}

	desiredIdxs := indexesByName(desired.Indexes)
	for _, idx := range baseline.Indexes {
		if _, exists := desiredIdxs[idx.Name]; !exists {
			up = append(up, fmt.Sprintf("DROP INDEX IF EXISTS %s;", idx.Name))
			down = append(down, idx.Raw+";")
		}
	}

	// --- Functions ---
	baseFuncs := funcsByName(baseline.Functions)
	for _, fn := range desired.Functions {
		bf, exists := baseFuncs[fn.Name]
		if !exists || bf.Body != fn.Body {
			body := fn.Body
			if !strings.HasSuffix(strings.TrimSpace(body), ";") {
				body += ";"
			}
			up = append(up, body)
			down = append(down, fmt.Sprintf("DROP FUNCTION IF EXISTS %s;", fn.Name))
		}
	}

	desiredFuncs := funcsByName(desired.Functions)
	for _, fn := range baseline.Functions {
		if _, exists := desiredFuncs[fn.Name]; !exists {
			up = append(up, fmt.Sprintf("DROP FUNCTION IF EXISTS %s;", fn.Name))
			body := fn.Body
			if !strings.HasSuffix(strings.TrimSpace(body), ";") {
				body += ";"
			}
			down = append(down, body)
		}
	}

	// --- Triggers ---
	baseTrigs := triggersByName(baseline.Triggers)
	for _, trig := range desired.Triggers {
		bt, exists := baseTrigs[trig.Name]
		if !exists || bt.Body != trig.Body {
			up = append(up, trig.Body)
			down = append(down, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", trig.Name, trig.Table))
		}
	}

	desiredTrigs := triggersByName(desired.Triggers)
	for _, trig := range baseline.Triggers {
		if _, exists := desiredTrigs[trig.Name]; !exists {
			up = append(up, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", trig.Name, trig.Table))
			down = append(down, trig.Body)
		}
	}

	upSQL := strings.Join(up, "\n\n")
	downSQL := strings.Join(down, "\n\n")

	return &DiffResult{
		UpSQL:   upSQL,
		DownSQL: downSQL,
		Empty:   strings.TrimSpace(upSQL) == "",
	}
}

// reconstructTable builds a CREATE TABLE IF NOT EXISTS statement from a Table.
func reconstructTable(t Table) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s (\n", t.Name)
	for i, col := range t.Columns {
		if i < len(t.Columns)-1 {
			fmt.Fprintf(&b, "    %s,\n", col.Raw)
		} else {
			fmt.Fprintf(&b, "    %s\n", col.Raw)
		}
	}
	b.WriteString(");")
	return b.String()
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func tablesByName(tables []Table) map[string]Table {
	m := make(map[string]Table, len(tables))
	for _, t := range tables {
		m[t.Name] = t
	}
	return m
}

func colsByName(cols []Column) map[string]bool {
	m := make(map[string]bool, len(cols))
	for _, c := range cols {
		m[c.Name] = true
	}
	return m
}

func indexesByName(idxs []Index) map[string]Index {
	m := make(map[string]Index, len(idxs))
	for _, idx := range idxs {
		m[idx.Name] = idx
	}
	return m
}

func funcsByName(fns []Function) map[string]Function {
	m := make(map[string]Function, len(fns))
	for _, fn := range fns {
		m[fn.Name] = fn
	}
	return m
}

func triggersByName(trigs []Trigger) map[string]Trigger {
	m := make(map[string]Trigger, len(trigs))
	for _, t := range trigs {
		m[t.Name] = t
	}
	return m
}
