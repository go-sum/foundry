package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscover_Success(t *testing.T) {
	want := DiscoveryDocument{
		Issuer:                "https://idp.example.com",
		AuthorizationEndpoint: "https://idp.example.com/auth",
		TokenEndpoint:         "https://idp.example.com/token",
		UserinfoEndpoint:      "https://idp.example.com/userinfo",
		JWKSURI:               "https://idp.example.com/.well-known/jwks.json",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	// Discover appends /.well-known/openid-configuration, so use the server URL as issuer.
	got, err := Discover(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}
	if got.Issuer != want.Issuer {
		t.Errorf("Issuer = %q, want %q", got.Issuer, want.Issuer)
	}
	if got.AuthorizationEndpoint != want.AuthorizationEndpoint {
		t.Errorf("AuthorizationEndpoint = %q, want %q", got.AuthorizationEndpoint, want.AuthorizationEndpoint)
	}
	if got.TokenEndpoint != want.TokenEndpoint {
		t.Errorf("TokenEndpoint = %q, want %q", got.TokenEndpoint, want.TokenEndpoint)
	}
	if got.UserinfoEndpoint != want.UserinfoEndpoint {
		t.Errorf("UserinfoEndpoint = %q, want %q", got.UserinfoEndpoint, want.UserinfoEndpoint)
	}
	if got.JWKSURI != want.JWKSURI {
		t.Errorf("JWKSURI = %q, want %q", got.JWKSURI, want.JWKSURI)
	}
}

func TestApplyDiscovery_FillsEmptyFields(t *testing.T) {
	cfg := ProviderConfig{} // all fields empty
	doc := DiscoveryDocument{
		Issuer:                "https://idp.example.com",
		AuthorizationEndpoint: "https://idp.example.com/auth",
		TokenEndpoint:         "https://idp.example.com/token",
		UserinfoEndpoint:      "https://idp.example.com/userinfo",
		JWKSURI:               "https://idp.example.com/.well-known/jwks.json",
	}

	ApplyDiscovery(&cfg, doc)

	if cfg.Issuer != doc.Issuer {
		t.Errorf("Issuer = %q, want %q", cfg.Issuer, doc.Issuer)
	}
	if cfg.AuthorizationEndpoint != doc.AuthorizationEndpoint {
		t.Errorf("AuthorizationEndpoint = %q, want %q", cfg.AuthorizationEndpoint, doc.AuthorizationEndpoint)
	}
	if cfg.TokenEndpoint != doc.TokenEndpoint {
		t.Errorf("TokenEndpoint = %q, want %q", cfg.TokenEndpoint, doc.TokenEndpoint)
	}
	if cfg.UserinfoEndpoint != doc.UserinfoEndpoint {
		t.Errorf("UserinfoEndpoint = %q, want %q", cfg.UserinfoEndpoint, doc.UserinfoEndpoint)
	}
	if cfg.JWKSURI != doc.JWKSURI {
		t.Errorf("JWKSURI = %q, want %q", cfg.JWKSURI, doc.JWKSURI)
	}
}

func TestApplyDiscovery_PreservesNonEmptyFields(t *testing.T) {
	cfg := ProviderConfig{
		Issuer:                "https://my-issuer.com",
		AuthorizationEndpoint: "https://my-auth.com/authorize",
		TokenEndpoint:         "https://my-token.com/token",
	}
	doc := DiscoveryDocument{
		Issuer:                "https://idp.example.com",
		AuthorizationEndpoint: "https://idp.example.com/auth",
		TokenEndpoint:         "https://idp.example.com/token",
		UserinfoEndpoint:      "https://idp.example.com/userinfo",
	}

	ApplyDiscovery(&cfg, doc)

	// Pre-existing values must not be overwritten.
	if cfg.Issuer != "https://my-issuer.com" {
		t.Errorf("Issuer overwritten; got %q, want %q", cfg.Issuer, "https://my-issuer.com")
	}
	if cfg.AuthorizationEndpoint != "https://my-auth.com/authorize" {
		t.Errorf("AuthorizationEndpoint overwritten; got %q", cfg.AuthorizationEndpoint)
	}
	if cfg.TokenEndpoint != "https://my-token.com/token" {
		t.Errorf("TokenEndpoint overwritten; got %q", cfg.TokenEndpoint)
	}
	// Empty field should be filled.
	if cfg.UserinfoEndpoint != "https://idp.example.com/userinfo" {
		t.Errorf("UserinfoEndpoint = %q, want %q", cfg.UserinfoEndpoint, "https://idp.example.com/userinfo")
	}
}

func TestDiscover_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := Discover(context.Background(), srv.Client(), srv.URL)
	if err == nil {
		t.Fatal("expected error for non-200 discovery response, got nil")
	}
}
