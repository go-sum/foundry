package db

import "fmt"

// MapError translates a database error to a domain error using the provided
// mapping functions. The first mapping that returns a non-nil error wins.
// If no mapping matches, the original error is wrapped with the operation context.
func MapError(err error, op string, mappings ...func(error) error) error {
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

// OnUniqueViolation returns a mapping function that returns domainErr when
// the database error is a unique constraint violation (SQLSTATE 23505).
func OnUniqueViolation(domainErr error) func(error) error {
	return func(err error) error {
		if IsUniqueViolation(err) {
			return domainErr
		}
		return nil
	}
}

// OnForeignKeyViolation returns a mapping function that returns domainErr when
// the database error is a foreign key constraint violation (SQLSTATE 23503).
func OnForeignKeyViolation(domainErr error) func(error) error {
	return func(err error) error {
		if IsForeignKeyViolation(err) {
			return domainErr
		}
		return nil
	}
}
