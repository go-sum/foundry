package contact

import (
	"context"

	coredb "github.com/go-sum/foundry/pkg/db"
)

// Repository persists contact submissions.
type Repository interface {
	Create(ctx context.Context, s *Submission) error
}

type pgRepository struct {
	db coredb.DBTX
}

// NewRepository creates a Repository backed by db.
func NewRepository(db coredb.DBTX) *pgRepository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, s *Submission) error {
	const q = `
		INSERT INTO contact_submissions (name, email, message, ip_address)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	err := r.db.QueryRow(ctx, q, s.Name, s.Email, s.Message, s.IPAddress).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return coredb.MapError(err, "contact: insert submission")
	}
	return nil
}
