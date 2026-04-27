package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateToken implements provider.TokenStore.
func (s *Store) CreateToken(ctx context.Context, token provider.OAuthToken) error {
	const q = `
		INSERT INTO oauth_tokens
			(id, token_hash, token_type, client_id, user_id, scopes, revoked, parent_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	scopes := token.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	var parentID pgtype.UUID
	if token.ParentID != nil {
		parentID = pgtype.UUID{Bytes: ([16]byte)(*token.ParentID), Valid: true}
	}

	_, err := s.pool.Exec(ctx, q,
		token.ID, token.TokenHash, token.TokenType,
		token.ClientID, token.UserID, scopes,
		token.Revoked, parentID,
		token.ExpiresAt, token.CreatedAt,
	)
	return err
}

// GetTokenByHash implements provider.TokenStore.
func (s *Store) GetTokenByHash(ctx context.Context, hash string) (provider.OAuthToken, error) {
	const q = `
		SELECT id, token_hash, token_type, client_id, user_id, scopes, revoked, parent_id, expires_at, created_at
		FROM oauth_tokens
		WHERE token_hash = $1`

	var t provider.OAuthToken
	var parentID pgtype.UUID
	err := s.pool.QueryRow(ctx, q, hash).Scan(
		&t.ID, &t.TokenHash, &t.TokenType,
		&t.ClientID, &t.UserID, &t.Scopes,
		&t.Revoked, &parentID,
		&t.ExpiresAt, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return provider.OAuthToken{}, provider.ErrTokenNotFound
		}
		return provider.OAuthToken{}, err
	}
	if parentID.Valid {
		id := uuid.UUID(parentID.Bytes)
		t.ParentID = &id
	}
	return t, nil
}

// RevokeToken implements provider.TokenStore.
func (s *Store) RevokeToken(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE oauth_tokens SET revoked = true WHERE id = $1`
	_, err := s.pool.Exec(ctx, q, id)
	return err
}

// RevokeTokensByUserAndClient implements provider.TokenStore.
func (s *Store) RevokeTokensByUserAndClient(ctx context.Context, userID uuid.UUID, clientID string) error {
	const q = `UPDATE oauth_tokens SET revoked = true WHERE user_id = $1 AND client_id = $2`
	_, err := s.pool.Exec(ctx, q, userID, clientID)
	return err
}

// DeleteExpiredTokens implements provider.TokenStore.
func (s *Store) DeleteExpiredTokens(ctx context.Context) error {
	const q = `DELETE FROM oauth_tokens WHERE expires_at < $1 AND revoked = true`
	_, err := s.pool.Exec(ctx, q, time.Now().UTC())
	return err
}
