package auth

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PasskeyCredential represents a stored WebAuthn credential for a user.
type PasskeyCredential struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	CredentialID    []byte
	Name            string
	PublicKey       []byte
	PublicKeyAlg    int64
	AttestationType string
	AAGUID          []byte
	SignCount       int64
	CloneWarning    bool
	BackupEligible  bool
	BackupState     bool
	Transports      []string
	Attachment      string
	LastUsedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PasskeyCredentialParameter is a neutral representation of a WebAuthn credential
// parameter (algorithm + type) for round-tripping through ceremony state.
type PasskeyCredentialParameter struct {
	Type      string
	Algorithm int64
}

// PasskeyCeremony is a neutral carrier for in-progress WebAuthn ceremony state
// held by the caller between Begin* and Finish* calls. Service implementations
// translate to/from their library's native session type at the boundary.
type PasskeyCeremony struct {
	Challenge            []byte
	RelyingPartyID       string
	UserID               []byte
	AllowedCredentialIDs [][]byte
	UserVerification     string
	Mediation            string
	Extensions           map[string]any
	Expires              time.Time
	CredentialParameters []PasskeyCredentialParameter
}

// PasskeyCreationOptions is the JSON payload returned to the browser for
// navigator.credentials.create(). PublicKey is an opaque raw message produced
// by the service implementation — the handler forwards it unchanged.
type PasskeyCreationOptions struct {
	PublicKey json.RawMessage `json:"publicKey"`
}

// PasskeyRequestOptions mirrors the payload for navigator.credentials.get().
type PasskeyRequestOptions struct {
	PublicKey json.RawMessage `json:"publicKey"`
}
