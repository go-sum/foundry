package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/go-sum/auth"
	authdb "github.com/go-sum/auth/pgstore/db"
	coredb "github.com/go-sum/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func toCredential(c authdb.WebauthnCredential) auth.PasskeyCredential {
	var lastUsed *time.Time
	if c.LastUsedAt.Valid {
		lastUsed = &c.LastUsedAt.Time
	}
	return auth.PasskeyCredential{
		ID:              c.ID,
		UserID:          c.UserID,
		CredentialID:    c.CredentialID,
		Name:            c.Name,
		PublicKey:       c.PublicKey,
		PublicKeyAlg:    c.PublicKeyAlg,
		AttestationType: c.AttestationType,
		AAGUID:          c.Aaguid,
		SignCount:       c.SignCount,
		CloneWarning:    c.CloneWarning,
		BackupEligible:  c.BackupEligible,
		BackupState:     c.BackupState,
		Transports:      c.Transports,
		Attachment:      c.Attachment,
		LastUsedAt:      lastUsed,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

// CreateCredential persists a new WebAuthn credential.
func (s *Store) CreateCredential(ctx context.Context, cred auth.PasskeyCredential) (auth.PasskeyCredential, error) {
	var lastUsed pgtype.Timestamptz
	if cred.LastUsedAt != nil {
		lastUsed = pgtype.Timestamptz{Time: *cred.LastUsedAt, Valid: true}
	}
	transports := cred.Transports
	if transports == nil {
		transports = []string{}
	}
	row, err := s.q.CreatePasskeyCredential(ctx, authdb.CreatePasskeyCredentialParams{
		UserID:          cred.UserID,
		CredentialID:    cred.CredentialID,
		Name:            cred.Name,
		PublicKey:       cred.PublicKey,
		PublicKeyAlg:    cred.PublicKeyAlg,
		AttestationType: cred.AttestationType,
		Aaguid:          cred.AAGUID,
		SignCount:       cred.SignCount,
		CloneWarning:    cred.CloneWarning,
		BackupEligible:  cred.BackupEligible,
		BackupState:     cred.BackupState,
		Transports:      transports,
		Attachment:      cred.Attachment,
		LastUsedAt:      lastUsed,
	})
	if err != nil {
		return auth.PasskeyCredential{}, coredb.MapError(err, "auth: create credential",
			coredb.OnUniqueViolation(auth.ErrPasskeyAlreadyRegistered),
		)
	}
	return toCredential(row), nil
}

// GetByCredentialID returns a credential by its WebAuthn credential ID.
func (s *Store) GetByCredentialID(ctx context.Context, credentialID []byte) (auth.PasskeyCredential, error) {
	row, err := s.q.GetPasskeyCredentialByCredentialID(ctx, credentialID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.PasskeyCredential{}, auth.ErrPasskeyNotFound
		}
		return auth.PasskeyCredential{}, err
	}
	return toCredential(row), nil
}

// GetByIDForUser returns a credential by its UUID, scoped to the given user.
func (s *Store) GetByIDForUser(ctx context.Context, userID, id uuid.UUID) (auth.PasskeyCredential, error) {
	row, err := s.q.GetPasskeyCredentialByIDForUser(ctx, authdb.GetPasskeyCredentialByIDForUserParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.PasskeyCredential{}, auth.ErrPasskeyNotFound
		}
		return auth.PasskeyCredential{}, err
	}
	return toCredential(row), nil
}

// ListByUserID returns all credentials belonging to the given user.
func (s *Store) ListByUserID(ctx context.Context, userID uuid.UUID) ([]auth.PasskeyCredential, error) {
	rows, err := s.q.ListPasskeyCredentialsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	creds := make([]auth.PasskeyCredential, len(rows))
	for i, r := range rows {
		creds[i] = toCredential(r)
	}
	return creds, nil
}

// TouchPasskeyCredential updates the sign count, clone warning, and last-used timestamp.
func (s *Store) TouchPasskeyCredential(ctx context.Context, id uuid.UUID, signCount int64, cloneWarning bool, lastUsed time.Time) error {
	return s.q.TouchPasskeyCredential(ctx, authdb.TouchPasskeyCredentialParams{
		ID:           id,
		SignCount:    signCount,
		CloneWarning: cloneWarning,
		LastUsedAt:   pgtype.Timestamptz{Time: lastUsed, Valid: true},
	})
}

// RenameCredential changes the display name of a credential, scoped to the given user.
func (s *Store) RenameCredential(ctx context.Context, id, userID uuid.UUID, name string) (auth.PasskeyCredential, error) {
	row, err := s.q.RenamePasskeyCredential(ctx, authdb.RenamePasskeyCredentialParams{
		ID:     id,
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.PasskeyCredential{}, auth.ErrPasskeyNotFound
		}
		return auth.PasskeyCredential{}, err
	}
	return toCredential(row), nil
}

// DeleteCredential removes a credential, scoped to the given user.
// Returns ErrPasskeyNotFound if no matching row exists.
func (s *Store) DeleteCredential(ctx context.Context, id, userID uuid.UUID) error {
	_, err := s.q.DeletePasskeyCredential(ctx, authdb.DeletePasskeyCredentialParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.ErrPasskeyNotFound
		}
		return err
	}
	return nil
}
