package schema

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContactSubmission is the domain model for the contact_submissions table.
type ContactSubmission struct {
	Id string
	Name string
	Email string
	Message string
	IpAddress string
	CreatedAt time.Time
}

func scanContactSubmission(row pgx.Row) (ContactSubmission, error) {
	var m ContactSubmission
	err := row.Scan(
		&m.Id,
		&m.Name,
		&m.Email,
		&m.Message,
		&m.IpAddress,
		&m.CreatedAt,
	)
	return m, err
}

// Store provides CRUD operations for contact_submissions.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a Store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const insertContactSubmission = `
INSERT INTO contact_submissions (name, email, message, ip_address)
VALUES ($1, $2, $3, $4)
RETURNING id, name, email, message, ip_address, created_at`

// InsertContactSubmission inserts a new record and returns the created row.
func (s *Store) InsertContactSubmission(ctx context.Context, m ContactSubmission) (ContactSubmission, error) {
	return scanContactSubmission(s.pool.QueryRow(ctx, insertContactSubmission,
		m.Name,
		m.Email,
		m.Message,
		m.IpAddress,
	))
}

const getContactSubmission = `
SELECT id, name, email, message, ip_address, created_at FROM contact_submissions
WHERE id = $1`

// GetContactSubmission returns a record by id.
func (s *Store) GetContactSubmission(ctx context.Context, id string) (ContactSubmission, error) {
	return scanContactSubmission(s.pool.QueryRow(ctx, getContactSubmission, id))
}

const listContactSubmissions = `
SELECT id, name, email, message, ip_address, created_at FROM contact_submissions
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`

// ListContactSubmissions returns a paginated list ordered by created_at descending.
func (s *Store) ListContactSubmissions(ctx context.Context, limit, offset int32) ([]ContactSubmission, error) {
	rows, err := s.pool.Query(ctx, listContactSubmissions, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ContactSubmission
	for rows.Next() {
		m, err := scanContactSubmission(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

const deleteContactSubmission = `
DELETE FROM contact_submissions
WHERE id = $1`

// DeleteContactSubmission removes a record by id.
func (s *Store) DeleteContactSubmission(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, deleteContactSubmission, id)
	return err
}
