package auth

// ProviderConfig holds the OAuth 2.0 / OIDC configuration for a single external provider.
// Populate it directly or use Discover + ApplyDiscovery for OIDC providers.
type ProviderConfig struct {
	// Issuer is the OIDC issuer URL (e.g., "https://accounts.google.com").
	// Used for issuer validation. Optional for plain OAuth 2.0.
	Issuer string

	// ClientID is the OAuth 2.0 client identifier registered with the provider.
	ClientID string

	// ClientSecret is the OAuth 2.0 client secret. Empty for public clients (PKCE-only).
	ClientSecret string

	// AuthorizationEndpoint is the authorization server's authorization endpoint URL.
	AuthorizationEndpoint string

	// TokenEndpoint is the authorization server's token endpoint URL.
	TokenEndpoint string

	// UserinfoEndpoint is the URL of the userinfo endpoint. Optional.
	UserinfoEndpoint string

	// JWKSURI is the URL of the JSON Web Key Set. Optional; used for ID token verification.
	JWKSURI string

	// Scopes are the OAuth scopes to request. Defaults to ["openid", "email", "profile"].
	Scopes []string

	// RedirectURL is the client's registered redirect URI (callback URL).
	RedirectURL string
}

// EffectiveScopes returns the configured scopes, or the default OIDC scopes if none are set.
func (p ProviderConfig) EffectiveScopes() []string {
	if len(p.Scopes) > 0 {
		return p.Scopes
	}
	return []string{"openid", "email", "profile"}
}
