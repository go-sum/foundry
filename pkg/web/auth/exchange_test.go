package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestExchangeCode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "tok",
			TokenType:   "bearer",
		})
	}))
	defer srv.Close()

	tr, err := ExchangeCode(context.Background(), srv.Client(), ExchangeParams{
		TokenEndpoint: srv.URL,
		ClientID:      "cid",
		Code:          "mycode",
		RedirectURI:   "https://app/cb",
		CodeVerifier:  "verifier",
	})
	if err != nil {
		t.Fatalf("ExchangeCode error: %v", err)
	}
	if tr.AccessToken != "tok" {
		t.Errorf("AccessToken = %q, want %q", tr.AccessToken, "tok")
	}
	if tr.TokenType != "bearer" {
		t.Errorf("TokenType = %q, want %q", tr.TokenType, "bearer")
	}
}

func TestExchangeCode_OAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(TokenError{
			ErrorCode:   "invalid_grant",
			Description: "code expired",
		})
	}))
	defer srv.Close()

	_, err := ExchangeCode(context.Background(), srv.Client(), ExchangeParams{
		TokenEndpoint: srv.URL,
		ClientID:      "cid",
		Code:          "expired",
		RedirectURI:   "https://app/cb",
		CodeVerifier:  "verifier",
	})
	var tokenErr *TokenError
	if !errors.As(err, &tokenErr) {
		t.Fatalf("expected *TokenError, got %T: %v", err, err)
	}
	if tokenErr.ErrorCode != "invalid_grant" {
		t.Errorf("ErrorCode = %q, want %q", tokenErr.ErrorCode, "invalid_grant")
	}
	if tokenErr.Description != "code expired" {
		t.Errorf("Description = %q, want %q", tokenErr.Description, "code expired")
	}
}

func TestExchangeCode_HTTP500NoJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	_, err := ExchangeCode(context.Background(), srv.Client(), ExchangeParams{
		TokenEndpoint: srv.URL,
		ClientID:      "cid",
		Code:          "code",
		RedirectURI:   "https://app/cb",
		CodeVerifier:  "verifier",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.As(err, new(*TokenError)) {
		t.Errorf("expected generic error, got *TokenError")
	}
}

func TestExchangeCode_RequestBodyParams(t *testing.T) {
	var gotBody string
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		gotBody = r.Form.Encode()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "tok", TokenType: "bearer"})
	}))
	defer srv.Close()

	_, err := ExchangeCode(context.Background(), srv.Client(), ExchangeParams{
		TokenEndpoint: srv.URL,
		ClientID:      "testclient",
		ClientSecret:  "secret",
		Code:          "authcode",
		RedirectURI:   "https://app/cb",
		CodeVerifier:  "verifier123",
	})
	if err != nil {
		t.Fatalf("ExchangeCode error: %v", err)
	}

	if !strings.Contains(gotContentType, "application/x-www-form-urlencoded") {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", gotContentType)
	}

	form, _ := url.ParseQuery(gotBody)
	assertField := func(key, want string) {
		t.Helper()
		if got := form.Get(key); got != want {
			t.Errorf("form[%q] = %q, want %q", key, got, want)
		}
	}
	assertField("grant_type", "authorization_code")
	assertField("code", "authcode")
	assertField("redirect_uri", "https://app/cb")
	assertField("client_id", "testclient")
	assertField("code_verifier", "verifier123")
	assertField("client_secret", "secret")
}

func TestExchangeCode_NilClientUsesDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "tok", TokenType: "bearer"})
	}))
	defer srv.Close()

	// nil client should fall back to http.DefaultClient without panicking.
	tr, err := ExchangeCode(context.Background(), nil, ExchangeParams{
		TokenEndpoint: srv.URL,
		ClientID:      "cid",
		Code:          "code",
		RedirectURI:   "https://app/cb",
		CodeVerifier:  "verifier",
	})
	if err != nil {
		t.Fatalf("ExchangeCode nil client error: %v", err)
	}
	if tr.AccessToken != "tok" {
		t.Errorf("AccessToken = %q, want %q", tr.AccessToken, "tok")
	}
}

func TestExchangeCode_MissingAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return a 200 but with no access_token field.
		_, _ = w.Write([]byte(`{"token_type":"bearer"}`))
	}))
	defer srv.Close()

	_, err := ExchangeCode(context.Background(), srv.Client(), ExchangeParams{
		TokenEndpoint: srv.URL,
		ClientID:      "cid",
		Code:          "code",
		RedirectURI:   "https://app/cb",
		CodeVerifier:  "verifier",
	})
	if err == nil {
		t.Fatal("expected error for missing access_token, got nil")
	}
}
