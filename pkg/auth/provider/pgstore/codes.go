package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/jackc/pgx/v5"
)

// CreateCode implements provider.CodeStore.
func (s *Store) CreateCode(ctx context.Context, code provider.AuthorizationCode) error {
	const q = `
		INSERT INTO oauth_authorization_codes
			(code, client_id, user_id, redirect_uri, scopes, code_challenge, nonce, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	scopes := code.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	_, err := s.pool.Exec(ctx, q,
		code.Code, code.ClientID, code.UserID, code.RedirectURI,
		scopes, code.CodeChallenge, code.Nonce,
		code.ExpiresAt, code.CreatedAt,
	)
	return err
}

// GetCode implements provider.CodeStore.
func (s *Store) GetCode(ctx context.Context, code string) (provider.AuthorizationCode, error) {
	const q = `
		SELECT code, client_id, user_id, redirect_uri, scopes, code_challenge, nonce, used, expires_at, created_at
		FROM oauth_authorization_codes
		WHERE code = $1`

	var c provider.AuthorizationCode
	err := s.pool.QueryRow(ctx, q, code).Scan(
		&c.Code, &c.ClientID, &c.UserID, &c.RedirectURI,
		&c.Scopes, &c.CodeChallenge, &c.Nonce, &c.Used,
		&c.ExpiresAt, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return provider.AuthorizationCode{}, provider.ErrCodeNotFound
		}
		return provider.AuthorizationCode{}, err
	}
	return c, nil
}

// MarkCodeUsed implements provider.CodeStore.
func (s *Store) MarkCodeUsed(ctx context.Context, code string) error {
	const q = `UPDATE oauth_authorization_codes SET used = true WHERE code = $1`
	_, err := s.pool.Exec(ctx, q, code)
	return err
}

// DeleteExpiredCodes implements provider.CodeStore.
func (s *Store) DeleteExpiredCodes(ctx context.Context) error {
	const q = `DELETE FROM oauth_authorization_codes WHERE expires_at < $1`
	_, err := s.pool.Exec(ctx, q, time.Now().UTC())
	return err
}
