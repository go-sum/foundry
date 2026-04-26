// Package pgstore implements the auth module's persistence interfaces
// (UserWriter, AdminStore, CredentialStore) using PostgreSQL with pgx/v5.
package pgstore

import "github.com/jackc/pgx/v5/pgxpool"

// Store implements auth.UserWriter, auth.AdminStore, and auth.CredentialStore
// backed by PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a Store. The pool is externally managed and not closed by the Store.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
