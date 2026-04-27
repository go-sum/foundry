package contact

import (
	"context"

	coredb "github.com/go-sum/foundry/pkg/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persists contact submissions.
type Repository interface {
	Create(ctx context.Context, s *Submission) error
}

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a Repository backed by pool.
func NewRepository(pool *pgxpool.Pool) *pgRepository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) Create(ctx context.Context, s *Submission) error {
	const q = `
		INSERT INTO contact_submissions (name, email, message, ip_address)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, q, s.Name, s.Email, s.Message, s.IPAddress).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return coredb.MapError(err, "contact: insert submission")
	}
	return nil
}
