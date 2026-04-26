package provider

import "time"

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

// ApplyDefaults fills in zero values with sensible defaults.
func ApplyDefaults(cfg Config) Config {
	if cfg.CodeTTL <= 0 {
		cfg.CodeTTL = 5 * time.Minute
	}
	if cfg.AccessTokenTTL <= 0 {
		cfg.AccessTokenTTL = time.Hour
	}
	if cfg.RefreshTokenTTL <= 0 {
		cfg.RefreshTokenTTL = 30 * 24 * time.Hour
	}
	// RequirePKCE defaults to true (zero value is false, so we always set it).
	cfg.RequirePKCE = true
	return cfg
}
