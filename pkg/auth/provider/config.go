package provider

import (
	"time"

	webauth "github.com/go-sum/foundry/pkg/web/auth"
)

// Config holds configuration for the OAuth 2.0 Authorization Server.
type Config struct {
	// Issuer is the issuer identifier (e.g., "https://auth.example.com").
	Issuer string

	// CodeTTL is the lifetime of authorization codes. Default: 5 minutes.
	CodeTTL time.Duration

	// AccessTokenTTL is the lifetime of access tokens. Default: 1 hour.
	AccessTokenTTL time.Duration

	// RefreshTokenTTL is the lifetime of refresh tokens. Default: 30 days.
	RefreshTokenTTL time.Duration

	// RequirePKCE forces all clients to use PKCE. Default: true.
	RequirePKCE bool
}

// InitialProviderConfig returns the package's sane defaults for Config.
// Use this as the starting point and override individual fields as needed.
func InitialProviderConfig() Config {
	return Config{
		CodeTTL:         5 * time.Minute,
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 30 * 24 * time.Hour,
		RequirePKCE:     true,
	}
}

// BuildOAuthClient constructs the webauth.ProviderConfig for a client connecting
// to this authorization server. Endpoint URLs are derived from issuer and the
// default route patterns. Call this whenever the issuer changes.
func BuildOAuthClient(issuer, clientID, redirectPath string) webauth.ProviderConfig {
	routes := DefaultRouteConfig()
	return webauth.ProviderConfig{
		Issuer:                issuer,
		ClientID:              clientID,
		AuthorizationEndpoint: issuer + routes.Authorize.Pattern,
		TokenEndpoint:         issuer + routes.Token.Pattern,
		UserinfoEndpoint:      issuer + routes.Userinfo.Pattern,
		RedirectURL:           issuer + redirectPath,
	}
}

// ApplyDefaults fills in zero values with sensible defaults.
// Deprecated: use InitialProviderConfig and override fields directly.
func ApplyDefaults(cfg Config) Config {
	defaults := InitialProviderConfig()
	if cfg.CodeTTL <= 0 {
		cfg.CodeTTL = defaults.CodeTTL
	}
	if cfg.AccessTokenTTL <= 0 {
		cfg.AccessTokenTTL = defaults.AccessTokenTTL
	}
	if cfg.RefreshTokenTTL <= 0 {
		cfg.RefreshTokenTTL = defaults.RefreshTokenTTL
	}
	cfg.RequirePKCE = true
	return cfg
}
