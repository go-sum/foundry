package pgstore

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// mapError translates a database error to a domain error using the provided
// mapping functions. The first mapping that returns a non-nil error wins.
// If no mapping matches, the original error is wrapped with the operation context.
func mapError(err error, op string, mappings ...func(error) error) error {
	if err == nil {
		return nil
	}
	for _, m := range mappings {
		if domainErr := m(err); domainErr != nil {
			return domainErr
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}

// onUniqueViolation returns a mapping function that returns domainErr when
// the database error is a unique constraint violation (SQLSTATE 23505).
func onUniqueViolation(domainErr error) func(error) error {
	return func(err error) error {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domainErr
		}
		return nil
	}
}
