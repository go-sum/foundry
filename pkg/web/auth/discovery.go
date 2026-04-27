package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DiscoveryDocument represents the OpenID Connect discovery metadata
// returned by the /.well-known/openid-configuration endpoint.
type DiscoveryDocument struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	UserinfoEndpoint              string   `json:"userinfo_endpoint,omitempty"`
	JWKSURI                       string   `json:"jwks_uri,omitempty"`
	ScopesSupported               []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported        []string `json:"response_types_supported,omitempty"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported,omitempty"`
}

// Discover fetches the OpenID Connect discovery document from the given issuer.
// It requests issuer + "/.well-known/openid-configuration".
func Discover(ctx context.Context, client *http.Client, issuer string) (DiscoveryDocument, error) {
	if client == nil {
		client = http.DefaultClient
	}
	discoveryURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return DiscoveryDocument{}, fmt.Errorf("oauth: build discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return DiscoveryDocument{}, fmt.Errorf("oauth: discovery request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return DiscoveryDocument{}, fmt.Errorf("oauth: discovery endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return DiscoveryDocument{}, fmt.Errorf("oauth: read discovery response: %w", err)
	}

	var doc DiscoveryDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return DiscoveryDocument{}, fmt.Errorf("oauth: decode discovery response: %w", err)
	}
	return doc, nil
}

// ApplyDiscovery merges a DiscoveryDocument into a ProviderConfig,
// filling in endpoint fields that are currently empty.
// Existing non-empty values in cfg are preserved.
func ApplyDiscovery(cfg *ProviderConfig, doc DiscoveryDocument) {
	if cfg.Issuer == "" {
		cfg.Issuer = doc.Issuer
	}
	if cfg.AuthorizationEndpoint == "" {
		cfg.AuthorizationEndpoint = doc.AuthorizationEndpoint
	}
	if cfg.TokenEndpoint == "" {
		cfg.TokenEndpoint = doc.TokenEndpoint
	}
	if cfg.UserinfoEndpoint == "" {
		cfg.UserinfoEndpoint = doc.UserinfoEndpoint
	}
	if cfg.JWKSURI == "" {
		cfg.JWKSURI = doc.JWKSURI
	}
}
