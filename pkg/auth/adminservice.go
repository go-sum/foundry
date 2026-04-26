package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// AdminStore provides user management operations for the admin service.
type AdminStore interface {
	UserReader
	ListUsers(ctx context.Context, limit, offset int32) ([]User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, email, displayName, role string) (User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	CountUsers(ctx context.Context) (int64, error)
	HasAdmin(ctx context.Context) (bool, error)
	// IsLastAdmin reports whether id is an admin AND is the only admin in the
	// system. Returns false when id is not an admin or more than one admin exists.
	IsLastAdmin(ctx context.Context, id uuid.UUID) (bool, error)
	// ElevateToAdmin atomically promotes id to admin, returning ErrAdminExists
	// if any admin already exists. The check-and-update is a single SQL statement
	// to prevent a TOCTOU race during initial bootstrap.
	ElevateToAdmin(ctx context.Context, id uuid.UUID) (User, error)
}

// AdminService provides admin-level user management operations.
type AdminService struct {
	users AdminStore
}

// NewAdminService returns a new AdminService.
func NewAdminService(users AdminStore) *AdminService {
	return &AdminService{users: users}
}

// CountUsers returns the total number of users.
func (s *AdminService) CountUsers(ctx context.Context) (int64, error) {
	return s.users.CountUsers(ctx)
}

// ListUsers returns a paginated list of users.
func (s *AdminService) ListUsers(ctx context.Context, page, perPage int) ([]User, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	return s.users.ListUsers(ctx, int32(perPage), int32(offset))
}

// GetUser returns a single user by ID.
func (s *AdminService) GetUser(ctx context.Context, id uuid.UUID) (User, error) {
	return s.users.GetUserByID(ctx, id)
}

// UpdateUser updates user fields by ID. Returns ErrLastAdmin when the update
// would demote the only remaining admin account.
func (s *AdminService) UpdateUser(ctx context.Context, id uuid.UUID, input UpdateUserInput) (User, error) {
	if input.Role == string(RoleUser) {
		if last, err := s.users.IsLastAdmin(ctx, id); err != nil {
			return User{}, fmt.Errorf("AdminService.UpdateUser: check last admin: %w", err)
		} else if last {
			return User{}, ErrLastAdmin
		}
	}
	return s.users.UpdateUser(ctx, id, input.Email, input.DisplayName, input.Role)
}

// DeleteUser removes a user by ID. Returns ErrLastAdmin when the target is
// the only remaining admin account.
func (s *AdminService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if last, err := s.users.IsLastAdmin(ctx, id); err != nil {
		return fmt.Errorf("AdminService.DeleteUser: check last admin: %w", err)
	} else if last {
		return ErrLastAdmin
	}
	return s.users.DeleteUser(ctx, id)
}

// HasAdmin reports whether at least one admin user exists.
func (s *AdminService) HasAdmin(ctx context.Context) (bool, error) {
	return s.users.HasAdmin(ctx)
}

// ElevateToAdmin atomically promotes a user to admin, provided no admin exists yet.
func (s *AdminService) ElevateToAdmin(ctx context.Context, userID uuid.UUID) (User, error) {
	user, err := s.users.ElevateToAdmin(ctx, userID)
	if err != nil {
		return User{}, fmt.Errorf("AdminService.ElevateToAdmin: %w", err)
	}
	return user, nil
}
