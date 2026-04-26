package provider

import (
	"context"

	"github.com/google/uuid"
)

// ClientStore provides read access to registered OAuth 2.0 clients.
type ClientStore interface {
	GetClientByClientID(ctx context.Context, clientID string) (OAuthClient, error)
}

// CodeStore manages single-use authorization codes.
type CodeStore interface {
	CreateCode(ctx context.Context, code AuthorizationCode) error
	GetCode(ctx context.Context, code string) (AuthorizationCode, error)
	MarkCodeUsed(ctx context.Context, code string) error
	DeleteExpiredCodes(ctx context.Context) error
}

// TokenStore manages access and refresh tokens.
type TokenStore interface {
	CreateToken(ctx context.Context, token OAuthToken) error
	GetTokenByHash(ctx context.Context, hash string) (OAuthToken, error)
	RevokeToken(ctx context.Context, id uuid.UUID) error
	RevokeTokensByUserAndClient(ctx context.Context, userID uuid.UUID, clientID string) error
	DeleteExpiredTokens(ctx context.Context) error
}

// ConsentStore persists user consent decisions.
type ConsentStore interface {
	GetConsent(ctx context.Context, userID uuid.UUID, clientID string) (Consent, error)
	SaveConsent(ctx context.Context, consent Consent) error
	RevokeConsent(ctx context.Context, userID uuid.UUID, clientID string) error
}
