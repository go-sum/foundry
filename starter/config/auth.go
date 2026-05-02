package config

import (
	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/auth/provider"
	cfgpkg "github.com/go-sum/foundry/pkg/config"
	webauth "github.com/go-sum/foundry/pkg/web/auth"
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
	Provider provider.Config

	// OAuthClient is the pre-built OAuth 2.0 client configuration for this application.
	// All endpoint URLs are derived from Provider.Issuer; rebuild via
	// provider.BuildOAuthClient whenever Provider.Issuer changes.
	OAuthClient webauth.ProviderConfig
}

// productionAuth builds the AuthConfig from environment variables and secrets.
// On missing or invalid SECURITY_AUTH_TOKEN_KEY the returned AuthConfig has nil
// token keys; validation catches this via the required,min=1 tag on Token.Secrets.
func productionAuth(siteBaseURL string) AuthConfig {
	masterKeys, _ := auth.ParseTokenKeys(cfgpkg.ExpandSecret("SECURITY_AUTH_TOKEN_KEY"))
	verifyKeys, identityKeys, _ := auth.DeriveTokenSubkeys(masterKeys)
	issuer := cfgpkg.ExpandEnv("AUTH_ISSUER", siteBaseURL)
	return AuthConfig{
		TokenKeys: verifyKeys,
		Identity: auth.Config{
			EmailTOTP: auth.EmailTOTPConfig{
				Enabled:       true,
				PeriodSeconds: 300,
			},
			Token: auth.TokenConfig{
				Secrets: identityKeys,
			},
		},
		Provider:    provider.Config{Issuer: issuer},
		OAuthClient: provider.BuildOAuthClient(issuer, cfgpkg.ExpandEnv("AUTH_FIRST_PARTY_CLIENT_ID", "starter-app"), "/auth/callback"),
	}
}
