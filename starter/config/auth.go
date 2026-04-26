package config

import (
	"errors"
	"fmt"

	"github.com/go-sum/auth"
	cfgpkg "github.com/go-sum/config"
	webauth "github.com/go-sum/web/auth"
)

// AuthConfig consolidates all authentication and authorization settings.
// This is the single place where pkg/auth (identity), pkg/auth/provider
// (OAuth Authorization Server), and pkg/web/auth (OAuth client toolkit) are
// connected for the application.
type AuthConfig struct {
	// TokenKeys holds AEAD key material for encrypting auth verification tokens.
	// Loaded from SECURITY_AUTH_TOKEN_KEY.
	TokenKeys [][]byte

	// Identity configures the core identity provider (pkg/auth).
	Identity auth.Config

	// Provider configures the built-in OAuth 2.0 Authorization Server (pkg/auth/provider).
	Provider ProviderAuthConfig

	// FirstParty configures the app's own built-in OAuth 2.1 client.
	FirstParty FirstPartyConfig
}

// ProviderAuthConfig configures the built-in OAuth 2.0 Authorization Server.
type ProviderAuthConfig struct {
	// Issuer is the publicly reachable base URL of this authorization server.
	// Defaults to Site.BaseURL when empty. Override with AUTH_ISSUER env var.
	Issuer string
}

// FirstPartyConfig configures the app's own first-party OAuth 2.1 client.
// The client is a public client (PKCE-only, no secret) because it runs on
// the same origin as the Authorization Server.
type FirstPartyConfig struct {
	// ClientID is the OAuth 2.0 client_id registered for this app.
	// Override with AUTH_FIRST_PARTY_CLIENT_ID env var. Default: "starter-app".
	ClientID string

	// RedirectPath is the path-only callback URL (e.g. "/auth/callback").
	RedirectPath string
}

// DefaultAuth builds the default AuthConfig for production, reading secrets
// from environment variables. The siteBaseURL is used as the default OAuth
// issuer when AUTH_ISSUER is not explicitly set.
func DefaultAuth(siteBaseURL string) (AuthConfig, error) {
	tokenKeys, err := auth.ParseTokenKeys(cfgpkg.ExpandSecret("SECURITY_AUTH_TOKEN_KEY"))
	if err != nil {
		if errors.Is(err, auth.ErrTokenKeyMissing) {
			return AuthConfig{}, fmt.Errorf("%w: set SECURITY_AUTH_TOKEN_KEY environment variable", ErrAuthTokenKeyMissing)
		}
		return AuthConfig{}, fmt.Errorf("%w", ErrAuthTokenKeyInvalid)
	}
	return AuthConfig{
		TokenKeys: tokenKeys,
		Identity: auth.Config{
			EmailTOTP: auth.EmailTOTPConfig{
				Enabled:       true,
				PeriodSeconds: 300,
			},
			Token: auth.TokenConfig{
				Secrets: tokenKeys,
			},
		},
		Provider: ProviderAuthConfig{
			Issuer: cfgpkg.ExpandEnv("AUTH_ISSUER", siteBaseURL),
		},
		FirstParty: FirstPartyConfig{
			ClientID:     cfgpkg.ExpandEnv("AUTH_FIRST_PARTY_CLIENT_ID", "starter-app"),
			RedirectPath: "/auth/callback",
		},
	}, nil
}

// ClientConfig returns a pkg/web/auth ProviderConfig pre-filled for the local
// OAuth 2.0 provider. Endpoints are derived from the Issuer and the well-known
// OAuth route paths registered by pkg/auth/provider.
func (c AuthConfig) ClientConfig(clientID, clientSecret, redirectURL string) webauth.ProviderConfig {
	issuer := c.Provider.Issuer
	return webauth.ProviderConfig{
		Issuer:                issuer,
		ClientID:              clientID,
		ClientSecret:          clientSecret,
		AuthorizationEndpoint: issuer + "/oauth/authorize",
		TokenEndpoint:         issuer + "/oauth/token",
		UserinfoEndpoint:      issuer + "/oauth/userinfo",
		RedirectURL:           redirectURL,
		Scopes:                []string{"openid", "email", "profile"},
	}
}

// FirstPartyClientConfig returns the ProviderConfig for the built-in first-party
// OAuth client. The redirect URL is derived from the Issuer and the configured
// RedirectPath. The first-party client is always a public client (no secret).
func (c AuthConfig) FirstPartyClientConfig() webauth.ProviderConfig {
	redirectURL := c.Provider.Issuer + c.FirstParty.RedirectPath
	return c.ClientConfig(c.FirstParty.ClientID, "", redirectURL)
}
