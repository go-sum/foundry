package auth

import (
	"errors"
	"net/url"
	"testing"
)

func TestAuthorizationURL_RequiredParams(t *testing.T) {
	endpoint := "https://accounts.example.com/oauth/authorize"
	params := AuthURLParams{
		ClientID:      "client123",
		RedirectURI:   "https://app.example.com/callback",
		Scopes:        []string{"openid", "email"},
		State:         "randomstate",
		CodeChallenge: "somechallenge",
	}

	rawURL, err := AuthorizationURL(endpoint, params)
	if err != nil {
		t.Fatalf("AuthorizationURL error: %v", err)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse result URL error: %v", err)
	}
	q := u.Query()

	assertParam := func(key, want string) {
		t.Helper()
		if got := q.Get(key); got != want {
			t.Errorf("param %q = %q, want %q", key, got, want)
		}
	}

	assertParam("response_type", "code")
	assertParam("client_id", "client123")
	assertParam("redirect_uri", "https://app.example.com/callback")
	assertParam("scope", "openid email")
	assertParam("state", "randomstate")
	assertParam("code_challenge", "somechallenge")
	assertParam("code_challenge_method", "S256")
}

func TestAuthorizationURL_IncludesNonceWhenNonEmpty(t *testing.T) {
	rawURL, err := AuthorizationURL("https://idp.example.com/auth", AuthURLParams{
		ClientID:      "cid",
		RedirectURI:   "https://app/cb",
		Scopes:        []string{"openid"},
		State:         "s",
		Nonce:         "mynonce",
		CodeChallenge: "ch",
	})
	if err != nil {
		t.Fatalf("AuthorizationURL error: %v", err)
	}
	u, _ := url.Parse(rawURL)
	if got := u.Query().Get("nonce"); got != "mynonce" {
		t.Errorf("nonce = %q, want %q", got, "mynonce")
	}
}

func TestAuthorizationURL_OmitsNonceWhenEmpty(t *testing.T) {
	rawURL, err := AuthorizationURL("https://idp.example.com/auth", AuthURLParams{
		ClientID:      "cid",
		RedirectURI:   "https://app/cb",
		Scopes:        []string{"openid"},
		State:         "s",
		Nonce:         "",
		CodeChallenge: "ch",
	})
	if err != nil {
		t.Fatalf("AuthorizationURL error: %v", err)
	}
	u, _ := url.Parse(rawURL)
	if u.Query().Has("nonce") {
		t.Error("nonce param present, want absent when Nonce is empty")
	}
}

func TestAuthorizationURL_EmptyEndpoint(t *testing.T) {
	_, err := AuthorizationURL("", AuthURLParams{})
	if !errors.Is(err, ErrMissingEndpoint) {
		t.Fatalf("empty endpoint: got %v, want ErrMissingEndpoint", err)
	}
}

func TestBeginOAuth_ReturnsValidTransactionAndURL(t *testing.T) {
	provider := ProviderConfig{
		ClientID:              "myapp",
		RedirectURL:           "https://app.example.com/callback",
		AuthorizationEndpoint: "https://idp.example.com/auth",
	}

	tx, rawURL, err := BeginOAuth(provider, "/after-login")
	if err != nil {
		t.Fatalf("BeginOAuth error: %v", err)
	}

	if tx.State == "" {
		t.Error("tx.State is empty")
	}
	if tx.Verifier == "" {
		t.Error("tx.Verifier is empty")
	}
	if tx.ReturnTo != "/after-login" {
		t.Errorf("tx.ReturnTo = %q, want %q", tx.ReturnTo, "/after-login")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL error: %v", err)
	}
	q := u.Query()
	if got := q.Get("client_id"); got != "myapp" {
		t.Errorf("client_id = %q, want %q", got, "myapp")
	}
	if got := q.Get("redirect_uri"); got != "https://app.example.com/callback" {
		t.Errorf("redirect_uri = %q, want %q", got, "https://app.example.com/callback")
	}
}
