package pgstore

import (
	"context"
	"errors"

	"github.com/go-sum/auth/provider"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetConsent implements provider.ConsentStore.
func (s *Store) GetConsent(ctx context.Context, userID uuid.UUID, clientID string) (provider.Consent, error) {
	const q = `
		SELECT id, user_id, client_id, scopes, created_at, updated_at
		FROM oauth_consents
		WHERE user_id = $1 AND client_id = $2`

	var c provider.Consent
	err := s.pool.QueryRow(ctx, q, userID, clientID).Scan(
		&c.ID, &c.UserID, &c.ClientID, &c.Scopes,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return provider.Consent{}, provider.ErrConsentNotFound
		}
		return provider.Consent{}, err
	}
	return c, nil
}

// SaveConsent implements provider.ConsentStore — upsert on (user_id, client_id).
func (s *Store) SaveConsent(ctx context.Context, consent provider.Consent) error {
	const q = `
		INSERT INTO oauth_consents (id, user_id, client_id, scopes)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, client_id) DO UPDATE
			SET scopes = EXCLUDED.scopes,
				updated_at = NOW()`

	scopes := consent.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	_, err := s.pool.Exec(ctx, q, consent.ID, consent.UserID, consent.ClientID, scopes)
	return err
}

// RevokeConsent implements provider.ConsentStore.
func (s *Store) RevokeConsent(ctx context.Context, userID uuid.UUID, clientID string) error {
	const q = `DELETE FROM oauth_consents WHERE user_id = $1 AND client_id = $2`
	_, err := s.pool.Exec(ctx, q, userID, clientID)
	return err
}
