package migrate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// LintResult describes a single lint finding in a migration file.
type LintResult struct {
	File     string
	Line     int
	Severity string
	Rule     string
	Message  string
}

var (
	reDropColumn       = regexp.MustCompile(`(?i)\bDROP\s+COLUMN\b`)
	reAlterColumnType  = regexp.MustCompile(`(?i)\bALTER\s+COLUMN\b.+\bTYPE\b`)
	reNotNullNoDefault = regexp.MustCompile(`(?i)\bNOT\s+NULL\b`)
	reHasDefault       = regexp.MustCompile(`(?i)\bDEFAULT\b`)
	reDropTableNoIF    = regexp.MustCompile(`(?i)\bDROP\s+TABLE\b`)
	reDropTableIF      = regexp.MustCompile(`(?i)\bDROP\s+TABLE\s+IF\s+EXISTS\b`)
	reUnboundedUpdate  = regexp.MustCompile(`(?i)^\s*UPDATE\s+\S+\s+SET\b`)
	reHasWhere         = regexp.MustCompile(`(?i)\bWHERE\b`)
	reUnboundedDelete  = regexp.MustCompile(`(?i)^\s*DELETE\s+FROM\b`)
)

// Lint reads all .sql files in dir and checks for dangerous DDL patterns.
// Only the Up section (between -- +goose Up and -- +goose Down) is analyzed.
func Lint(dir string) ([]LintResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("lint: read dir: %w", err)
	}

	var results []LintResult

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}

		filePath := filepath.Join(dir, e.Name())
		fileResults, err := lintFile(filePath)
		if err != nil {
			return nil, err
		}
		results = append(results, fileResults...)
	}

	return results, nil
}

func lintFile(path string) (_ []LintResult, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("lint: open %s: %w", path, err)
	}
	defer f.Close() //nolint:errcheck

	var results []LintResult
	inUp := false
	lineNum := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "-- +goose Up") {
			inUp = true
			continue
		}
		if strings.HasPrefix(trimmed, "-- +goose Down") {
			inUp = false
			continue
		}

		if !inUp {
			continue
		}

		if reDropColumn.MatchString(line) {
			results = append(results, LintResult{
				File:     path,
				Line:     lineNum,
				Severity: "warning",
				Rule:     "drop-column",
				Message:  "DROP COLUMN is destructive and may cause data loss",
			})
		}

		if reAlterColumnType.MatchString(line) {
			results = append(results, LintResult{
				File:     path,
				Line:     lineNum,
				Severity: "warning",
				Rule:     "alter-column-type",
				Message:  "ALTER COLUMN ... TYPE may cause a full table rewrite",
			})
		}

		if reNotNullNoDefault.MatchString(line) && !reHasDefault.MatchString(line) {
			results = append(results, LintResult{
				File:     path,
				Line:     lineNum,
				Severity: "warning",
				Rule:     "not-null-no-default",
				Message:  "NOT NULL added without DEFAULT may require a full table scan on existing rows",
			})
		}

		if reDropTableNoIF.MatchString(line) && !reDropTableIF.MatchString(line) {
			results = append(results, LintResult{
				File:     path,
				Line:     lineNum,
				Severity: "error",
				Rule:     "drop-table",
				Message:  "DROP TABLE without IF EXISTS will error if table does not exist",
			})
		}

		if reUnboundedUpdate.MatchString(line) && !reHasWhere.MatchString(line) {
			results = append(results, LintResult{
				File:     path,
				Line:     lineNum,
				Severity: "error",
				Rule:     "unbounded-update",
				Message:  "UPDATE without WHERE clause will affect all rows",
			})
		}

		if reUnboundedDelete.MatchString(line) && !reHasWhere.MatchString(line) {
			results = append(results, LintResult{
				File:     path,
				Line:     lineNum,
				Severity: "error",
				Rule:     "unbounded-delete",
				Message:  "DELETE without WHERE clause will delete all rows",
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("lint: scan %s: %w", path, err)
	}

	return results, nil
}
