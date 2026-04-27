package seed

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-sum/foundry/pkg/db/migrate"
)

const historyDDL = `CREATE TABLE IF NOT EXISTS _seed_history (
    name       TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

// Entry describes a seed to apply.
type Entry struct {
	Name     string
	Priority int
	SQL      string
}

// Status describes whether a seed has been applied.
type Status struct {
	Name      string
	Applied   bool
	AppliedAt time.Time
}

// Apply runs seed entries in priority order, skipping already-applied seeds.
// Each seed executes in its own transaction using migrate.SplitStatements for $$ safety.
// Returns count of applied seeds.
func Apply(ctx context.Context, dsn string, entries []Entry) (int, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return 0, fmt.Errorf("seed: open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close() //nolint:errcheck

	if _, err := db.ExecContext(ctx, historyDDL); err != nil {
		return 0, fmt.Errorf("seed: ensure history table: %w", err)
	}

	rows, err := db.QueryContext(ctx, `SELECT name FROM _seed_history`)
	if err != nil {
		return 0, fmt.Errorf("seed: query applied: %w", err)
	}
	applied := make(map[string]bool)
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close() //nolint:errcheck
			return 0, fmt.Errorf("seed: scan applied: %w", err)
		}
		applied[n] = true
	}
	rows.Close() //nolint:errcheck
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("seed: iterate applied: %w", err)
	}

	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	count := 0
	for _, entry := range sorted {
		if applied[entry.Name] {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return count, fmt.Errorf("seed: begin tx for %s: %w", entry.Name, err)
		}

		for _, stmt := range migrate.SplitStatements(entry.SQL) {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				tx.Rollback() //nolint:errcheck
				return count, fmt.Errorf("seed: apply %s: %w", entry.Name, err)
			}
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO _seed_history (name) VALUES ($1)`,
			entry.Name,
		); err != nil {
			tx.Rollback() //nolint:errcheck
			return count, fmt.Errorf("seed: record %s: %w", entry.Name, err)
		}

		if err := tx.Commit(); err != nil {
			return count, fmt.Errorf("seed: commit %s: %w", entry.Name, err)
		}
		count++
	}
	return count, nil
}

// GetStatus returns which seeds have been applied.
func GetStatus(ctx context.Context, dsn string, known []string) ([]Status, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("seed: open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close() //nolint:errcheck

	rows, err := db.QueryContext(ctx, `SELECT name, applied_at FROM _seed_history`)
	if err != nil {
		return nil, fmt.Errorf("seed: query history: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	appliedAt := make(map[string]time.Time)
	for rows.Next() {
		var n string
		var t time.Time
		if err := rows.Scan(&n, &t); err != nil {
			return nil, fmt.Errorf("seed: scan history: %w", err)
		}
		appliedAt[n] = t
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("seed: iterate history: %w", err)
	}

	statuses := make([]Status, len(known))
	for i, name := range known {
		t, ok := appliedAt[name]
		statuses[i] = Status{
			Name:      name,
			Applied:   ok,
			AppliedAt: t,
		}
	}
	return statuses, nil
}

// Reset truncates _seed_history so seeds can be re-applied.
func Reset(ctx context.Context, dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("seed: open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close() //nolint:errcheck

	if _, err := db.ExecContext(ctx, `TRUNCATE _seed_history`); err != nil {
		return fmt.Errorf("seed: reset: %w", err)
	}
	return nil
}
