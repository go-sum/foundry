package pgstore

import (
	"context"
	"errors"

	"github.com/go-sum/auth"
	authdb "github.com/go-sum/auth/pgstore/db"
	coredb "github.com/go-sum/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func toUser(u authdb.User) auth.User {
	return auth.User{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        auth.Role(u.Role),
		Verified:    u.Verified,
		WebAuthnID:  u.WebauthnID,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// GetUserByID returns a user by ID.
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (auth.User, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, err
	}
	return toUser(u), nil
}

// GetUserByEmail returns a user by email address.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (auth.User, error) {
	u, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, err
	}
	return toUser(u), nil
}

// CreateUser inserts a new user and returns the created record.
func (s *Store) CreateUser(ctx context.Context, email, displayName string, role auth.Role, verified bool) (auth.User, error) {
	u, err := s.q.CreateUser(ctx, authdb.CreateUserParams{
		Email:       email,
		DisplayName: displayName,
		Role:        string(role),
		Verified:    verified,
	})
	if err != nil {
		return auth.User{}, coredb.MapError(err, "auth: create user",
			coredb.OnUniqueViolation(auth.ErrEmailTaken),
		)
	}
	return toUser(u), nil
}

// UpdateUserEmail changes the email address for a user.
func (s *Store) UpdateUserEmail(ctx context.Context, id uuid.UUID, email string) (auth.User, error) {
	u, err := s.q.UpdateUserEmail(ctx, authdb.UpdateUserEmailParams{ID: id, Email: email})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, coredb.MapError(err, "auth: update email",
			coredb.OnUniqueViolation(auth.ErrEmailTaken),
		)
	}
	return toUser(u), nil
}

// SetWebAuthnID sets the WebAuthn user handle unconditionally.
func (s *Store) SetWebAuthnID(ctx context.Context, id uuid.UUID, webauthnID []byte) (auth.User, error) {
	u, err := s.q.SetWebAuthnID(ctx, authdb.SetWebAuthnIDParams{ID: id, WebauthnID: webauthnID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, coredb.MapError(err, "auth: set webauthn id",
			coredb.OnUniqueViolation(auth.ErrWebAuthnIDAlreadySet),
		)
	}
	return toUser(u), nil
}

// SetWebAuthnIDIfNull sets the WebAuthn user handle only when it is currently NULL.
// Returns ErrWebAuthnIDAlreadySet if the handle was already assigned.
func (s *Store) SetWebAuthnIDIfNull(ctx context.Context, id uuid.UUID, webauthnID []byte) (auth.User, error) {
	u, err := s.q.SetWebAuthnIDIfNull(ctx, authdb.SetWebAuthnIDIfNullParams{ID: id, WebauthnID: webauthnID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrWebAuthnIDAlreadySet
		}
		return auth.User{}, coredb.MapError(err, "auth: set webauthn id if null",
			coredb.OnUniqueViolation(auth.ErrWebAuthnIDAlreadySet),
		)
	}
	return toUser(u), nil
}

// GetUserByWebAuthnID returns a user by their WebAuthn user handle.
func (s *Store) GetUserByWebAuthnID(ctx context.Context, webauthnID []byte) (auth.User, error) {
	u, err := s.q.GetUserByWebAuthnID(ctx, webauthnID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, err
	}
	return toUser(u), nil
}

// ListUsers returns a paginated list of users ordered by creation date descending.
func (s *Store) ListUsers(ctx context.Context, limit, offset int32) ([]auth.User, error) {
	rows, err := s.q.ListUsers(ctx, authdb.ListUsersParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	users := make([]auth.User, len(rows))
	for i, r := range rows {
		users[i] = toUser(r)
	}
	return users, nil
}

// UpdateUser updates user fields by ID. Empty strings are treated as "no change"
// by the COALESCE logic in the SQL query.
func (s *Store) UpdateUser(ctx context.Context, id uuid.UUID, email, displayName, role string) (auth.User, error) {
	u, err := s.q.UpdateUser(ctx, authdb.UpdateUserParams{
		ID:          id,
		Email:       email,
		DisplayName: displayName,
		Role:        role,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUserNotFound
		}
		return auth.User{}, coredb.MapError(err, "auth: update user",
			coredb.OnUniqueViolation(auth.ErrEmailTaken),
		)
	}
	return toUser(u), nil
}

// DeleteUser removes a user by ID.
func (s *Store) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteUser(ctx, id)
}

// CountUsers returns the total number of users.
func (s *Store) CountUsers(ctx context.Context) (int64, error) {
	return s.q.CountUsers(ctx)
}

// HasAdmin reports whether at least one admin user exists.
func (s *Store) HasAdmin(ctx context.Context) (bool, error) {
	return s.q.HasAdminUser(ctx)
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
	var u authdb.User
	row := s.pool.QueryRow(ctx, q, id)
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.DisplayName,
		&u.Role,
		&u.Verified,
		&u.WebauthnID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrAdminExists
		}
		return auth.User{}, err
	}
	return toUser(u), nil
}
