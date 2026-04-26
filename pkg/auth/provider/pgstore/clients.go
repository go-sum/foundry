package pgstore

import (
	"context"
	"errors"

	"github.com/go-sum/auth/provider"
	"github.com/jackc/pgx/v5"
)

// GetClientByClientID implements provider.ClientStore.
func (s *Store) GetClientByClientID(ctx context.Context, clientID string) (provider.OAuthClient, error) {
	const q = `
		SELECT id, client_id, client_secret, name, redirect_uris, scopes, public, first_party, created_at, updated_at
		FROM oauth_clients
		WHERE client_id = $1`

	var c provider.OAuthClient
	err := s.pool.QueryRow(ctx, q, clientID).Scan(
		&c.ID, &c.ClientID, &c.ClientSecret, &c.Name,
		&c.RedirectURIs, &c.Scopes, &c.Public, &c.FirstParty,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return provider.OAuthClient{}, provider.ErrClientNotFound
		}
		return provider.OAuthClient{}, err
	}
	return c, nil
}
