package auth

import (
	"time"

	"github.com/google/uuid"
)

// FlowPurpose identifies the verification workflow in progress.
type FlowPurpose string

const (
	FlowSignup      FlowPurpose = "signup"
	FlowSignin      FlowPurpose = "signin"
	FlowEmailChange FlowPurpose = "email_change"
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

// VerifyPageState supplies the verification screen with a purpose and prefilled code.
type VerifyPageState struct {
	Purpose   FlowPurpose
	Code      string
	Token     string
	Email     string
	CanResend bool
}
