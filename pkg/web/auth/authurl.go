package auth

import (
	"errors"
	"net/url"
	"strings"
)

var ErrMissingEndpoint = errors.New("oauth: authorization endpoint is required")

// AuthURLParams holds the parameters for constructing an OAuth 2.0 authorization URL.
type AuthURLParams struct {
	ClientID      string
	RedirectURI   string
	Scopes        []string
	State         string
	Nonce         string // included only when non-empty
	CodeChallenge string
}

// AuthorizationURL constructs the authorization redirect URL from the endpoint and params.
// Always uses response_type=code and code_challenge_method=S256.
func AuthorizationURL(endpoint string, params AuthURLParams) (string, error) {
	if endpoint == "" {
		return "", ErrMissingEndpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", params.ClientID)
	q.Set("redirect_uri", params.RedirectURI)
	q.Set("scope", strings.Join(params.Scopes, " "))
	q.Set("state", params.State)
	if params.Nonce != "" {
		q.Set("nonce", params.Nonce)
	}
	q.Set("code_challenge", params.CodeChallenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// BeginOAuth creates a new OAuthTransaction, derives the PKCE challenge, and builds
// the authorization URL for the given provider. Store the returned transaction in the
// session; redirect the user to the returned URL.
func BeginOAuth(provider ProviderConfig, returnTo string) (OAuthTransaction, string, error) {
	tx, err := NewTransaction(returnTo)
	if err != nil {
		return OAuthTransaction{}, "", err
	}
	challenge, err := Challenge(tx.Verifier)
	if err != nil {
		return OAuthTransaction{}, "", err
	}
	authURL, err := AuthorizationURL(provider.AuthorizationEndpoint, AuthURLParams{
		ClientID:      provider.ClientID,
		RedirectURI:   provider.RedirectURL,
		Scopes:        provider.EffectiveScopes(),
		State:         tx.State,
		Nonce:         tx.Nonce,
		CodeChallenge: challenge,
	})
	if err != nil {
		return OAuthTransaction{}, "", err
	}
	return tx, authURL, nil
}
