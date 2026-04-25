package queue

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QueueSummary holds aggregate counts for a single named queue.
type QueueSummary struct {
	Name      string
	Pending   int
	Running   int
	Completed int
	Failed    int
	Dead      int
	Total     int
}

// StatusCounts holds per-status counts for a single queue.
type StatusCounts struct {
	Pending   int
	Running   int
	Completed int
	Failed    int
	Dead      int
	Total     int
}

// JobRow holds the data for a single job row.
type JobRow struct {
	ID          string
	Status      string
	Priority    int
	Attempts    int
	MaxAttempts int
	LastError   string
	RunAt       time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// listQueues returns a sorted slice of QueueSummary, one entry per distinct queue name.
func listQueues(ctx context.Context, pool *pgxpool.Pool) ([]QueueSummary, error) {
	rows, err := pool.Query(ctx, `
		SELECT queue, status, COUNT(*) AS cnt
		FROM queue_jobs
		GROUP BY queue, status
		ORDER BY queue, status
	`)
	if err != nil {
		return nil, fmt.Errorf("list queues: %w", err)
	}
	defer rows.Close()

	m := make(map[string]*QueueSummary)
	for rows.Next() {
		var name, status string
		var cnt int
		if err := rows.Scan(&name, &status, &cnt); err != nil {
			return nil, fmt.Errorf("scan queue row: %w", err)
		}
		q, ok := m[name]
		if !ok {
			q = &QueueSummary{Name: name}
			m[name] = q
		}
		switch status {
		case "pending":
			q.Pending += cnt
		case "running":
			q.Running += cnt
		case "completed":
			q.Completed += cnt
		case "failed":
			q.Failed += cnt
		case "dead":
			q.Dead += cnt
		}
		q.Total += cnt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate queue rows: %w", err)
	}

	result := make([]QueueSummary, 0, len(m))
	for _, q := range m {
		result = append(result, *q)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// listJobs returns a paginated list of jobs for the given queue and optional status filter.
// It also returns the total count of matching jobs.
func listJobs(ctx context.Context, pool *pgxpool.Pool, queueName, status string, limit, offset int) ([]JobRow, int, error) {
	var total int
	if validStatus(status) {
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM queue_jobs WHERE queue = $1 AND status = $2`, queueName, status).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count jobs: %w", err)
		}
	} else {
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM queue_jobs WHERE queue = $1`, queueName).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count jobs: %w", err)
		}
	}

	var rows pgx.Rows
	if validStatus(status) {
		r, err := pool.Query(ctx, `
			SELECT id, status, priority, attempts, max_attempts, last_error, run_at, created_at, updated_at
			FROM queue_jobs
			WHERE queue = $1 AND status = $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`, queueName, status, limit, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("list jobs: %w", err)
		}
		rows = r
	} else {
		r, err := pool.Query(ctx, `
			SELECT id, status, priority, attempts, max_attempts, last_error, run_at, created_at, updated_at
			FROM queue_jobs
			WHERE queue = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`, queueName, limit, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("list jobs: %w", err)
		}
		rows = r
	}
	defer rows.Close()

	var jobs []JobRow
	for rows.Next() {
		var uid pgtype.UUID
		var job JobRow
		if err := rows.Scan(
			&uid,
			&job.Status,
			&job.Priority,
			&job.Attempts,
			&job.MaxAttempts,
			&job.LastError,
			&job.RunAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan job row: %w", err)
		}
		job.ID = uid.String()
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate job rows: %w", err)
	}

	return jobs, total, nil
}

// queueStatusCounts returns per-status counts for a single queue.
func queueStatusCounts(ctx context.Context, pool *pgxpool.Pool, queueName string) (StatusCounts, error) {
	rows, err := pool.Query(ctx, `
		SELECT status, COUNT(*)
		FROM queue_jobs
		WHERE queue = $1
		GROUP BY status
	`, queueName)
	if err != nil {
		return StatusCounts{}, fmt.Errorf("queue status counts: %w", err)
	}
	defer rows.Close()

	var counts StatusCounts
	for rows.Next() {
		var status string
		var cnt int
		if err := rows.Scan(&status, &cnt); err != nil {
			return StatusCounts{}, fmt.Errorf("scan status count: %w", err)
		}
		switch status {
		case "pending":
			counts.Pending += cnt
		case "running":
			counts.Running += cnt
		case "completed":
			counts.Completed += cnt
		case "failed":
			counts.Failed += cnt
		case "dead":
			counts.Dead += cnt
		}
		counts.Total += cnt
	}
	if err := rows.Err(); err != nil {
		return StatusCounts{}, fmt.Errorf("iterate status counts: %w", err)
	}
	return counts, nil
}

// validStatus reports whether s is a recognized job status value.
func validStatus(s string) bool {
	switch s {
	case "pending", "running", "completed", "failed", "dead":
		return true
	}
	return false
}
