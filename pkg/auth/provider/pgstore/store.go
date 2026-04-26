// Package pgstore implements the OAuth 2.0 provider module's persistence interfaces
// (ClientStore, CodeStore, TokenStore, ConsentStore) using PostgreSQL with pgx/v5.
package pgstore

import "github.com/jackc/pgx/v5/pgxpool"

// Store implements provider.ClientStore, provider.CodeStore, provider.TokenStore,
// and provider.ConsentStore backed by PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a Store. The pool is externally managed and not closed by the Store.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
