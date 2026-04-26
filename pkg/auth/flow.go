package auth

import (
	"time"

	"github.com/google/uuid"
)

// FlowPurpose identifies the verification workflow in progress.
type FlowPurpose string

const (
	FlowSignup            FlowPurpose = "signup"
	FlowSignin            FlowPurpose = "signin"
	FlowEmailChange       FlowPurpose = "email_change"
	// FlowAlreadyRegistered is used only as a Notifier purpose when a signup
	// attempt is made for an already-verified email. No code or verify URL is
	// included. The session flow purpose is still FlowSignup to prevent
	// response-timing differentiation.
	FlowAlreadyRegistered FlowPurpose = "already_registered"
)

// PendingFlow is the browser-bound verification state retained between the begin
// and verify steps.
type PendingFlow struct {
	Purpose     FlowPurpose `json:"purpose"`
	Email       string      `json:"email"`
	DisplayName string      `json:"display_name,omitempty"`
	Role        Role        `json:"role,omitempty"`
	UserID      uuid.UUID   `json:"user_id,omitempty"`
	Secret      string      `json:"secret"`
	IssuedAt    time.Time   `json:"issued_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
	Attempts    int         `json:"attempts,omitempty"`
	ReturnTo    string      `json:"return_to,omitempty"`
}

// VerificationToken is the self-contained payload embedded in emailed verify links.
type VerificationToken struct {
	Purpose     FlowPurpose `json:"purpose"`
	Email       string      `json:"email"`
	DisplayName string      `json:"display_name,omitempty"`
	Role        Role        `json:"role,omitempty"`
	UserID      uuid.UUID   `json:"user_id,omitempty"`
	Secret      string      `json:"secret"`
	IssuedAt    time.Time   `json:"issued_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
}

// DeliveryInput is the email payload required for sending a verification message.
type DeliveryInput struct {
	Purpose   FlowPurpose
	Email     string
	Code      string
	VerifyURL string
	ExpiresAt time.Time
}

// VerifyResult describes a successful verification.
type VerifyResult struct {
	Purpose FlowPurpose
	User    User
	Method  string // auth method that produced this result (e.g. "email_totp")
}

// pendingFlowFromToken converts a VerificationToken to a PendingFlow.
// Attempts starts at zero since token-based flows are always fresh.
func pendingFlowFromToken(t VerificationToken) PendingFlow {
	return PendingFlow{
		Purpose:     t.Purpose,
		Email:       t.Email,
		DisplayName: t.DisplayName,
		Role:        t.Role,
		UserID:      t.UserID,
		Secret:      t.Secret,
		IssuedAt:    t.IssuedAt,
		ExpiresAt:   t.ExpiresAt,
	}
}

// verificationTokenFromFlow converts a PendingFlow to a VerificationToken.
// The Attempts counter is intentionally excluded from the token.
func verificationTokenFromFlow(f PendingFlow) VerificationToken {
	return VerificationToken{
		Purpose:     f.Purpose,
		Email:       f.Email,
		DisplayName: f.DisplayName,
		Role:        f.Role,
		UserID:      f.UserID,
		Secret:      f.Secret,
		IssuedAt:    f.IssuedAt,
		ExpiresAt:   f.ExpiresAt,
	}
}

// VerifyPageState supplies the verification screen with display context.
type VerifyPageState struct {
	Purpose   FlowPurpose
	Token     string
	Email     string
	CanResend bool
}
