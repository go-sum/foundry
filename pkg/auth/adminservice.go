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

// UpdateUser updates user fields by ID.
func (s *AdminService) UpdateUser(ctx context.Context, id uuid.UUID, input UpdateUserInput) (User, error) {
	return s.users.UpdateUser(ctx, id, input.Email, input.DisplayName, input.Role)
}

// DeleteUser removes a user by ID.
func (s *AdminService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.users.DeleteUser(ctx, id)
}

// HasAdmin reports whether at least one admin user exists.
func (s *AdminService) HasAdmin(ctx context.Context) (bool, error) {
	return s.users.HasAdmin(ctx)
}

// ElevateToAdmin promotes a user to admin, provided no admin exists yet.
func (s *AdminService) ElevateToAdmin(ctx context.Context, userID uuid.UUID) (User, error) {
	hasAdmin, err := s.users.HasAdmin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("AdminService.ElevateToAdmin: %w", err)
	}
	if hasAdmin {
		return User{}, ErrAdminExists
	}
	user, err := s.users.UpdateUser(ctx, userID, "", "", string(RoleAdmin))
	if err != nil {
		return User{}, fmt.Errorf("AdminService.ElevateToAdmin: %w", err)
	}
	return user, nil
}
