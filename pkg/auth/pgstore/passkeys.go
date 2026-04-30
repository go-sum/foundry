package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func scanCredential(s scanner) (auth.PasskeyCredential, error) {
	var c auth.PasskeyCredential
	err := s.Scan(
		&c.ID,
		&c.UserID,
		&c.CredentialID,
		&c.Name,
		&c.PublicKey,
		&c.PublicKeyAlg,
		&c.AttestationType,
		&c.AAGUID,
		&c.SignCount,
		&c.CloneWarning,
		&c.BackupEligible,
		&c.BackupState,
		&c.Transports,
		&c.Attachment,
		&c.LastUsedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	return c, err
}

const createPasskeyCredential = `
INSERT INTO webauthn_credentials (
    user_id, credential_id, name, public_key, public_key_alg,
    attestation_type, aaguid, sign_count, clone_warning,
    backup_eligible, backup_state, transports, attachment, last_used_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13, $14
)
RETURNING id, user_id, credential_id, name, public_key, public_key_alg, attestation_type, aaguid, sign_count, clone_warning, backup_eligible, backup_state, transports, attachment, last_used_at, created_at, updated_at`

// CreateCredential persists a new WebAuthn credential.
func (s *Store) CreateCredential(ctx context.Context, cred auth.PasskeyCredential) (auth.PasskeyCredential, error) {
	transports := cred.Transports
	if transports == nil {
		transports = []string{}
	}
	c, err := scanCredential(s.pool.QueryRow(ctx, createPasskeyCredential,
		cred.UserID,
		cred.CredentialID,
		cred.Name,
		cred.PublicKey,
		cred.PublicKeyAlg,
		cred.AttestationType,
		cred.AAGUID,
		cred.SignCount,
		cred.CloneWarning,
		cred.BackupEligible,
		cred.BackupState,
		transports,
		cred.Attachment,
		cred.LastUsedAt,
	))
	if err != nil {
		return auth.PasskeyCredential{}, mapError(err, "auth: create credential",
			onUniqueViolation(auth.ErrPasskeyAlreadyRegistered),
		)
	}
	return c, nil
}

const getPasskeyCredentialByCredentialID = `
SELECT id, user_id, credential_id, name, public_key, public_key_alg, attestation_type, aaguid, sign_count, clone_warning, backup_eligible, backup_state, transports, attachment, last_used_at, created_at, updated_at FROM webauthn_credentials
WHERE credential_id = $1`

// GetByCredentialID returns a credential by its WebAuthn credential ID.
func (s *Store) GetByCredentialID(ctx context.Context, credentialID []byte) (auth.PasskeyCredential, error) {
	c, err := scanCredential(s.pool.QueryRow(ctx, getPasskeyCredentialByCredentialID, credentialID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.PasskeyCredential{}, auth.ErrPasskeyNotFound
		}
		return auth.PasskeyCredential{}, err
	}
	return c, nil
}

const getPasskeyCredentialByIDForUser = `
SELECT id, user_id, credential_id, name, public_key, public_key_alg, attestation_type, aaguid, sign_count, clone_warning, backup_eligible, backup_state, transports, attachment, last_used_at, created_at, updated_at FROM webauthn_credentials
WHERE id = $1 AND user_id = $2`

// GetByIDForUser returns a credential by its UUID, scoped to the given user.
func (s *Store) GetByIDForUser(ctx context.Context, userID, id uuid.UUID) (auth.PasskeyCredential, error) {
	c, err := scanCredential(s.pool.QueryRow(ctx, getPasskeyCredentialByIDForUser, id, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.PasskeyCredential{}, auth.ErrPasskeyNotFound
		}
		return auth.PasskeyCredential{}, err
	}
	return c, nil
}

const listPasskeyCredentialsByUserID = `
SELECT id, user_id, credential_id, name, public_key, public_key_alg, attestation_type, aaguid, sign_count, clone_warning, backup_eligible, backup_state, transports, attachment, last_used_at, created_at, updated_at FROM webauthn_credentials
WHERE user_id = $1
ORDER BY created_at DESC, id DESC`

// ListByUserID returns all credentials belonging to the given user.
func (s *Store) ListByUserID(ctx context.Context, userID uuid.UUID) ([]auth.PasskeyCredential, error) {
	rows, err := s.pool.Query(ctx, listPasskeyCredentialsByUserID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var creds []auth.PasskeyCredential
	for rows.Next() {
		c, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return creds, nil
}

const touchPasskeyCredential = `
UPDATE webauthn_credentials
SET sign_count    = GREATEST(sign_count, $2),
    clone_warning = clone_warning OR $3,
    last_used_at  = $4,
    updated_at    = NOW()
WHERE id = $1`

// TouchPasskeyCredential updates the sign count, clone warning, and last-used timestamp.
func (s *Store) TouchPasskeyCredential(ctx context.Context, id uuid.UUID, signCount int64, cloneWarning bool, lastUsed time.Time) error {
	_, err := s.pool.Exec(ctx, touchPasskeyCredential, id, signCount, cloneWarning, &lastUsed)
	return err
}

const renamePasskeyCredential = `
UPDATE webauthn_credentials
SET name = $3
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, credential_id, name, public_key, public_key_alg, attestation_type, aaguid, sign_count, clone_warning, backup_eligible, backup_state, transports, attachment, last_used_at, created_at, updated_at`

// RenameCredential changes the display name of a credential, scoped to the given user.
func (s *Store) RenameCredential(ctx context.Context, id, userID uuid.UUID, name string) (auth.PasskeyCredential, error) {
	c, err := scanCredential(s.pool.QueryRow(ctx, renamePasskeyCredential, id, userID, name))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.PasskeyCredential{}, auth.ErrPasskeyNotFound
		}
		return auth.PasskeyCredential{}, err
	}
	return c, nil
}

const deletePasskeyCredential = `
DELETE FROM webauthn_credentials
WHERE id = $1 AND user_id = $2
RETURNING id`

// DeleteCredential removes a credential, scoped to the given user.
// Returns ErrPasskeyNotFound if no matching row exists.
func (s *Store) DeleteCredential(ctx context.Context, id, userID uuid.UUID) error {
	var returnedID uuid.UUID
	err := s.pool.QueryRow(ctx, deletePasskeyCredential, id, userID).Scan(&returnedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.ErrPasskeyNotFound
		}
		return err
	}
	return nil
}
