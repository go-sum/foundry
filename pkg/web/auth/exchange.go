package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// TokenResponse holds the parsed response from an OAuth 2.0 token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// TokenError represents an OAuth 2.0 error response from the token endpoint.
type TokenError struct {
	ErrorCode   string `json:"error"`
	Description string `json:"error_description,omitempty"`
	URI         string `json:"error_uri,omitempty"`
}

func (e *TokenError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("oauth token error: %s: %s", e.ErrorCode, e.Description)
	}
	return "oauth token error: " + e.ErrorCode
}

// ExchangeParams holds the parameters for the authorization code exchange.
type ExchangeParams struct {
	// TokenEndpoint is the provider's token endpoint URL.
	TokenEndpoint string
	// ClientID is the OAuth 2.0 client identifier.
	ClientID string
	// ClientSecret is the client secret. Leave empty for public clients (PKCE-only).
	ClientSecret string
	// Code is the authorization code received from the authorization endpoint.
	Code string
	// RedirectURI must exactly match the redirect_uri used in the authorization request.
	RedirectURI string
	// CodeVerifier is the PKCE code verifier from the OAuthTransaction.
	CodeVerifier string
}

// ExchangeCode exchanges an authorization code for tokens at the token endpoint.
// Uses application/x-www-form-urlencoded POST per RFC 6749 Section 4.1.3.
// If the token endpoint returns an OAuth error response, the error will be *TokenError.
func ExchangeCode(ctx context.Context, client *http.Client, params ExchangeParams) (TokenResponse, error) {
	if client == nil {
		client = http.DefaultClient
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", params.Code)
	form.Set("redirect_uri", params.RedirectURI)
	form.Set("client_id", params.ClientID)
	form.Set("code_verifier", params.CodeVerifier)
	if params.ClientSecret != "" {
		form.Set("client_secret", params.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("oauth: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("oauth: token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("oauth: read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var tokenErr TokenError
		if jsonErr := json.Unmarshal(body, &tokenErr); jsonErr == nil && tokenErr.ErrorCode != "" {
			return TokenResponse{}, &tokenErr
		}
		return TokenResponse{}, fmt.Errorf("oauth: token endpoint returned status %d", resp.StatusCode)
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return TokenResponse{}, fmt.Errorf("oauth: decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return TokenResponse{}, errors.New("oauth: token response missing access_token")
	}
	return tr, nil
}
