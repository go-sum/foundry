package auth

import (
	"time"

	"github.com/google/uuid"
)

// Role is a typed string for user access levels.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User is the auth module's view of an application user.
// It contains only the fields required for authentication and session management.
type User struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	Role        Role
	Verified    bool
	WebAuthnID  []byte // opaque random ID for WebAuthn user handle; nil until first passkey
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// BeginSignupInput carries validated data for starting a signup verification flow.
type BeginSignupInput struct {
	Email       string `form:"email"        validate:"required,email,max=255"`
	DisplayName string `form:"display_name" validate:"required,min=1,max=255"`
	ReturnTo    string `form:"return_to"`
}

// BeginSigninInput carries the email address for starting a signin verification flow.
type BeginSigninInput struct {
	Email    string `form:"email"     validate:"required,email,max=255"`
	ReturnTo string `form:"return_to"`
}

// BeginEmailChangeInput carries the target email address for a signed-in user.
type BeginEmailChangeInput struct {
	Email string `form:"email" validate:"required,email,max=255"`
}

// VerifyInput carries data from the verification form.
type VerifyInput struct {
	Code  string `form:"code"  validate:"required,len=6,numeric"`
	Token string `form:"token" validate:"omitempty"`
}

// UpdateUserInput carries validated data for updating an existing user.
// Empty strings are treated as "no change" by the COALESCE logic in the SQL query.
type UpdateUserInput struct {
	Email       string `form:"email"        validate:"omitempty,email,max=255"`
	DisplayName string `form:"display_name" validate:"omitempty,max=255"`
	Role        string `form:"role"         validate:"omitempty,oneof=user admin"`
}
