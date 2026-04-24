package contact

import (
	"context"

	"github.com/go-sum/db"
	"github.com/go-sum/foundry/db/sqlcgen"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persists contact submissions.
type Repository interface {
	Create(ctx context.Context, s *Submission) error
}

type pgRepository struct {
	q *sqlcgen.Queries
}

// NewRepository creates a Repository backed by pool.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{q: sqlcgen.New(pool)}
}

func (r *pgRepository) Create(ctx context.Context, s *Submission) error {
	row, err := r.q.InsertContactSubmission(ctx, sqlcgen.InsertContactSubmissionParams{
		Name:      s.Name,
		Email:     s.Email,
		Message:   s.Message,
		IpAddress: s.IPAddress,
	})
	if err != nil {
		return db.MapError(err, "contact: insert submission")
	}
	s.ID = row.ID
	s.CreatedAt = row.CreatedAt.Time
	return nil
}
