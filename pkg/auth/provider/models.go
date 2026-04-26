package provider

import (
	"time"

	"github.com/google/uuid"
)

// OAuthClient represents a registered OAuth 2.0 client application.
type OAuthClient struct {
	ID           uuid.UUID
	ClientID     string
	ClientSecret string    // bcrypt hash; empty for public clients
	Name         string
	RedirectURIs []string
	Scopes       []string
	Public       bool      // true = PKCE-only, no client_secret
	FirstParty   bool      // true = auto-approve consent; full security flow still runs
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AuthorizationCode represents a single-use authorization code.
type AuthorizationCode struct {
	Code          string
	ClientID      string
	UserID        uuid.UUID
	RedirectURI   string
	Scopes        []string
	CodeChallenge string    // PKCE S256 challenge
	Nonce         string    // OIDC nonce, may be empty
	ExpiresAt     time.Time
	CreatedAt     time.Time
	Used          bool
}

// OAuthToken represents an issued access or refresh token.
type OAuthToken struct {
	ID        uuid.UUID
	TokenHash string     // SHA-256 hex digest of the opaque token value
	TokenType string     // "access" or "refresh"
	ClientID  string
	UserID    uuid.UUID
	Scopes    []string
	Revoked   bool
	ParentID  *uuid.UUID // for refresh rotation: ID of the replaced token
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Consent records a user's approval of specific scopes for a client.
type Consent struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ClientID  string
	Scopes    []string
	CreatedAt time.Time
	UpdatedAt time.Time
}
