package pgstore

import (
	"context"
	"errors"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// scanner is satisfied by pgx.Row (from QueryRow) and pgx.Rows (during Next() iteration).
type scanner interface {
	Scan(dest ...any) error
}

func scanUser(s scanner) (auth.User, error) {
	var u auth.User
	err := s.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.Verified, &u.WebAuthnID, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

const getUserByID = `
SELECT id, email, display_name, role, verified, webauthn_id, created_at, updated_at
FROM users
WHERE id = $1`

// GetUserByID returns a user by ID.
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (auth.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, getUserByID, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, err
	}
	return u, nil
}

const getUserByEmail = `
SELECT id, email, display_name, role, verified, webauthn_id, created_at, updated_at
FROM users
WHERE email = $1`

// GetUserByEmail returns a user by email address.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (auth.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, getUserByEmail, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, err
	}
	return u, nil
}

const createUser = `
INSERT INTO users (email, display_name, role, verified)
VALUES ($1, $2, $3, $4)
RETURNING id, email, display_name, role, verified, webauthn_id, created_at, updated_at`

// CreateUser inserts a new user and returns the created record.
func (s *Store) CreateUser(ctx context.Context, email, displayName string, role auth.Role, verified bool) (auth.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, createUser, email, displayName, string(role), verified))
	if err != nil {
		return auth.User{}, mapError(err, "auth: create user",
			onUniqueViolation(auth.ErrEmailTaken),
		)
	}
	return u, nil
}

const updateUserEmail = `
UPDATE users
SET email = $2
WHERE id = $1
RETURNING id, email, display_name, role, verified, webauthn_id, created_at, updated_at`

// UpdateUserEmail changes the email address for a user.
func (s *Store) UpdateUserEmail(ctx context.Context, id uuid.UUID, email string) (auth.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, updateUserEmail, id, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, mapError(err, "auth: update email",
			onUniqueViolation(auth.ErrEmailTaken),
		)
	}
	return u, nil
}

const setWebAuthnID = `
UPDATE users SET webauthn_id = $2 WHERE id = $1
RETURNING id, email, display_name, role, verified, webauthn_id, created_at, updated_at`

// SetWebAuthnID sets the WebAuthn user handle unconditionally.
func (s *Store) SetWebAuthnID(ctx context.Context, id uuid.UUID, webauthnID []byte) (auth.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, setWebAuthnID, id, webauthnID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, mapError(err, "auth: set webauthn id",
			onUniqueViolation(auth.ErrWebAuthnIDAlreadySet),
		)
	}
	return u, nil
}

const setWebAuthnIDIfNull = `
UPDATE users SET webauthn_id = $2 WHERE id = $1 AND webauthn_id IS NULL
RETURNING id, email, display_name, role, verified, webauthn_id, created_at, updated_at`

// SetWebAuthnIDIfNull sets the WebAuthn user handle only when it is currently NULL.
// Returns ErrWebAuthnIDAlreadySet if the handle was already assigned.
func (s *Store) SetWebAuthnIDIfNull(ctx context.Context, id uuid.UUID, webauthnID []byte) (auth.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, setWebAuthnIDIfNull, id, webauthnID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrWebAuthnIDAlreadySet
		}
		return auth.User{}, mapError(err, "auth: set webauthn id if null",
			onUniqueViolation(auth.ErrWebAuthnIDAlreadySet),
		)
	}
	return u, nil
}

const getUserByWebAuthnID = `
SELECT id, email, display_name, role, verified, webauthn_id, created_at, updated_at
FROM users WHERE webauthn_id = $1`

// GetUserByWebAuthnID returns a user by their WebAuthn user handle.
func (s *Store) GetUserByWebAuthnID(ctx context.Context, webauthnID []byte) (auth.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, getUserByWebAuthnID, webauthnID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, err
	}
	return u, nil
}

const listUsers = `
SELECT id, email, display_name, role, verified, webauthn_id, created_at, updated_at FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`

// ListUsers returns a paginated list of users ordered by creation date descending.
func (s *Store) ListUsers(ctx context.Context, limit, offset int32) ([]auth.User, error) {
	rows, err := s.pool.Query(ctx, listUsers, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []auth.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

const updateUser = `
UPDATE users
SET
    email        = COALESCE(NULLIF($1::text, ''), email),
    display_name = COALESCE(NULLIF($2::text, ''), display_name),
    role         = COALESCE(NULLIF($3::text, ''), role)
WHERE id = $4
RETURNING id, email, display_name, role, verified, webauthn_id, created_at, updated_at`

// UpdateUser updates user fields by ID. Empty strings are treated as "no change"
// by the COALESCE logic in the SQL query.
func (s *Store) UpdateUser(ctx context.Context, id uuid.UUID, email, displayName, role string) (auth.User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, updateUser, email, displayName, role, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, mapError(err, "auth: update user",
			onUniqueViolation(auth.ErrEmailTaken),
		)
	}
	return u, nil
}

const deleteUser = `
DELETE FROM users
WHERE id = $1`

// DeleteUser removes a user by ID.
func (s *Store) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, deleteUser, id)
	return err
}

const countUsers = `SELECT COUNT(*) FROM users`

// CountUsers returns the total number of users.
func (s *Store) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := s.pool.QueryRow(ctx, countUsers).Scan(&count)
	return count, err
}

const hasAdminUser = `SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin')`

// HasAdmin reports whether at least one admin user exists.
func (s *Store) HasAdmin(ctx context.Context) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, hasAdminUser).Scan(&exists)
	return exists, err
}

// IsLastAdmin reports whether id is an admin and is the only admin. A single
// query checks both conditions atomically, avoiding a TOCTOU between role
// lookup and count.
func (s *Store) IsLastAdmin(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = `
		SELECT COUNT(*) = 1
		FROM users
		WHERE role = 'admin'
		  AND EXISTS (SELECT 1 FROM users WHERE id = $1 AND role = 'admin')`
	var isLast bool
	if err := s.pool.QueryRow(ctx, q, id).Scan(&isLast); err != nil {
		return false, err
	}
	return isLast, nil
}

// ElevateToAdmin atomically promotes id to admin only when no admin exists.
// A single UPDATE ... WHERE NOT EXISTS statement eliminates the TOCTOU race
// that a separate HasAdmin check would introduce.
func (s *Store) ElevateToAdmin(ctx context.Context, id uuid.UUID) (auth.User, error) {
	const q = `
		UPDATE users SET role = 'admin'
		WHERE id = $1
		  AND NOT EXISTS (SELECT 1 FROM users WHERE role = 'admin')
		RETURNING id, email, display_name, role, verified, webauthn_id, created_at, updated_at`
	u, err := scanUser(s.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrAdminExists
		}
		return auth.User{}, err
	}
	return u, nil
}
