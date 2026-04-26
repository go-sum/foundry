package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// UserinfoClaims holds the standard OIDC userinfo claims.
type UserinfoClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	Picture       string `json:"picture,omitempty"`
}

// FetchUserinfo calls the userinfo endpoint with the bearer access token
// and returns the decoded standard claims.
func FetchUserinfo(ctx context.Context, client *http.Client, endpoint, accessToken string) (UserinfoClaims, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return UserinfoClaims{}, fmt.Errorf("oauth: build userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return UserinfoClaims{}, fmt.Errorf("oauth: userinfo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UserinfoClaims{}, fmt.Errorf("oauth: userinfo endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserinfoClaims{}, fmt.Errorf("oauth: read userinfo response: %w", err)
	}

	var claims UserinfoClaims
	if err := json.Unmarshal(body, &claims); err != nil {
		return UserinfoClaims{}, fmt.Errorf("oauth: decode userinfo response: %w", err)
	}
	return claims, nil
}
