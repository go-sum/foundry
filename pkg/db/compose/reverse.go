package compose

import (
	"regexp"
	"strings"
)

var (
	reCreateTable         = regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)`)
	reCreateIndex         = regexp.MustCompile(`(?i)^\s*CREATE\s+INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+ON`)
	reCreateUniqueIndex   = regexp.MustCompile(`(?i)^\s*CREATE\s+UNIQUE\s+INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+ON`)
	reAddColumn           = regexp.MustCompile(`(?i)^\s*ALTER\s+TABLE\s+(\S+)\s+ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)`)
	reCreateTrigger       = regexp.MustCompile(`(?i)^\s*CREATE\s+(?:OR\s+REPLACE\s+)?TRIGGER\s+(\S+)`)
	reTriggerOn           = regexp.MustCompile(`(?i)\s+ON\s+(\S+)`)
	reCreateFunction      = regexp.MustCompile(`(?i)^\s*CREATE\s+OR\s+REPLACE\s+FUNCTION\s+(\S+)`)
)

// GenerateDown produces a best-effort Down SQL block from the given Up SQL.
// Lines are processed in reverse order so dependencies are dropped correctly.
// The output is prefixed with a review warning.
func GenerateDown(upSQL string) string {
	lines := strings.Split(upSQL, "\n")
	var downLines []string

	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]

		if m := reCreateTable.FindStringSubmatch(line); m != nil {
			downLines = append(downLines, "DROP TABLE IF EXISTS "+m[1]+" CASCADE;")
			continue
		}

		if m := reCreateUniqueIndex.FindStringSubmatch(line); m != nil {
			downLines = append(downLines, "DROP INDEX IF EXISTS "+m[1]+";")
			continue
		}

		if m := reCreateIndex.FindStringSubmatch(line); m != nil {
			downLines = append(downLines, "DROP INDEX IF EXISTS "+m[1]+";")
			continue
		}

		if m := reAddColumn.FindStringSubmatch(line); m != nil {
			downLines = append(downLines, "ALTER TABLE "+m[1]+" DROP COLUMN IF EXISTS "+m[2]+";")
			continue
		}

		if m := reCreateTrigger.FindStringSubmatch(line); m != nil {
			triggerName := m[1]
			onMatch := reTriggerOn.FindStringSubmatch(line)
			tableName := ""
			if onMatch != nil {
				tableName = " ON " + onMatch[1]
			}
			downLines = append(downLines, "DROP TRIGGER IF EXISTS "+triggerName+tableName+";")
			continue
		}

		if m := reCreateFunction.FindStringSubmatch(line); m != nil {
			downLines = append(downLines, "DROP FUNCTION IF EXISTS "+m[1]+";")
			continue
		}
	}

	if len(downLines) == 0 {
		return "-- REVIEW: auto-generated Down SQL — verify before committing\n"
	}

	return "-- REVIEW: auto-generated Down SQL — verify before committing\n" +
		strings.Join(downLines, "\n") + "\n"
}
