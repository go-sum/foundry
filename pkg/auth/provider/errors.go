package provider

import "errors"

var (
	ErrClientNotFound       = errors.New("provider: client not found")
	ErrInvalidRedirectURI   = errors.New("provider: invalid redirect_uri")
	ErrInvalidScope         = errors.New("provider: invalid scope")
	ErrCodeExpired          = errors.New("provider: authorization code expired")
	ErrCodeUsed             = errors.New("provider: authorization code already used")
	ErrCodeNotFound         = errors.New("provider: authorization code not found")
	ErrPKCERequired         = errors.New("provider: code_challenge is required")
	ErrPKCEFailed           = errors.New("provider: PKCE verification failed")
	ErrTokenRevoked         = errors.New("provider: token revoked")
	ErrTokenExpired         = errors.New("provider: token expired")
	ErrTokenNotFound        = errors.New("provider: token not found")
	ErrInvalidGrant         = errors.New("provider: invalid grant")
	ErrUnsupportedGrantType = errors.New("provider: unsupported grant type")
	ErrMissingChallenge     = errors.New("provider: code_challenge_method must be S256")
	ErrConsentNotFound      = errors.New("provider: consent not found")
)
