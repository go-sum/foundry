package provider

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	auth "github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/go-sum/foundry/pkg/web/validate"
	"github.com/google/uuid"
)

type fakeConsentStore struct {
	consent   Consent
	getErr    error
	saved     []Consent
	saveErr   error
	revokeErr error
}

func (f *fakeConsentStore) GetConsent(_ context.Context, _ uuid.UUID, _ string) (Consent, error) {
	if f.getErr != nil {
		return Consent{}, f.getErr
	}
	return f.consent, nil
}

func (f *fakeConsentStore) SaveConsent(_ context.Context, consent Consent) error {
	f.saved = append(f.saved, consent)
	return f.saveErr
}

func (f *fakeConsentStore) RevokeConsent(_ context.Context, _ uuid.UUID, _ string) error {
	return f.revokeErr
}

func testProviderSessionConfig() session.Config {
	return session.Config{
		Store: session.NewMemoryStore(),
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}
}

func runAuthorizeSubmit(t *testing.T, h *AuthorizeHandler, contentType, body string, userID uuid.UUID, seed func(*session.Session) error) (web.Response, *session.Session, error) {
	t.Helper()

	sessionMW := session.Middleware(testProviderSessionConfig())
	reqURL, err := url.Parse("/oauth/authorize")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	req := web.NewRequest(http.MethodPost, reqURL)
	req.Headers.Set("Content-Type", contentType)
	req.SetBody(io.NopCloser(strings.NewReader(body)))

	var captured *session.Session
	resp, callErr := sessionMW(func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok {
			t.Fatal("session not found in context")
		}
		captured = sess
		if seed != nil {
			if err := seed(sess); err != nil {
				t.Fatalf("seed session: %v", err)
			}
		}
		auth.SetUserID(c, userID.String())
		return h.Submit(c)
	})(web.NewContext(context.Background(), req))
	if callErr != nil {
		return web.Response{}, captured, callErr
	}
	return resp, captured, nil
}

func pendingAuthorizeParams() AuthorizeParams {
	return AuthorizeParams{
		ClientID:      "client-1",
		RedirectURI:   "https://app.example.com/callback",
		Scopes:        []string{"openid", "email"},
		State:         "state-123",
		Nonce:         "nonce-123",
		CodeChallenge: "challenge-123",
	}
}

func newAuthorizeHandlerForSubmit(codes CodeStore, consents ConsentStore) *AuthorizeHandler {
	return &AuthorizeHandler{
		codes:     codes,
		consents:  consents,
		config:    ApplyDefaults(Config{}),
		validator: validate.New(),
		logger:    slog.Default(),
	}
}

func assertSessionHasPendingParams(t *testing.T, sess *session.Session, want bool) {
	t.Helper()
	_, ok, err := session.Get[AuthorizeParams](sess, authzParamsSessionKey)
	if err != nil {
		t.Fatalf("session.Get(%q): %v", authzParamsSessionKey, err)
	}
	if ok != want {
		t.Fatalf("session has pending params = %v, want %v", ok, want)
	}
}

func TestIsAllowedRedirectURI_ExactMatch(t *testing.T) {
	registered := []string{
		"https://app.example.com/callback",
		"https://app.example.com/other",
	}
	if !isAllowedRedirectURI(registered, "https://app.example.com/callback") {
		t.Error("isAllowedRedirectURI: exact match returned false, want true")
	}
}

func TestIsAllowedRedirectURI_NoMatch(t *testing.T) {
	registered := []string{
		"https://app.example.com/callback",
	}
	cases := []string{
		"https://evil.example.com/callback",
		"https://app.example.com/callback/extra",
		"https://app.example.com/callbackx",
		"",
		"https://app.example.com/callback?foo=bar",
	}
	for _, requested := range cases {
		if isAllowedRedirectURI(registered, requested) {
			t.Errorf("isAllowedRedirectURI: %q should not match registered URIs", requested)
		}
	}
}

func TestIsAllowedRedirectURI_EmptyRegistered(t *testing.T) {
	if isAllowedRedirectURI(nil, "https://app.example.com/callback") {
		t.Error("isAllowedRedirectURI: nil registered list returned true, want false")
	}
	if isAllowedRedirectURI([]string{}, "https://app.example.com/callback") {
		t.Error("isAllowedRedirectURI: empty registered list returned true, want false")
	}
}

func TestIsAllowedRedirectURI_MultipleRegistered(t *testing.T) {
	registered := []string{
		"https://app.example.com/cb1",
		"https://app.example.com/cb2",
		"https://other.example.com/cb",
	}
	for _, r := range registered {
		if !isAllowedRedirectURI(registered, r) {
			t.Errorf("isAllowedRedirectURI: %q should match but did not", r)
		}
	}
}

func TestParseScopes_MultipleScopes(t *testing.T) {
	got := parseScopes("openid email profile")
	want := []string{"openid", "email", "profile"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseScopes = %v, want %v", got, want)
	}
}

func TestParseScopes_SingleScope(t *testing.T) {
	got := parseScopes("openid")
	want := []string{"openid"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseScopes = %v, want %v", got, want)
	}
}

func TestParseScopes_EmptyString(t *testing.T) {
	got := parseScopes("")
	if got != nil {
		t.Errorf("parseScopes(\"\") = %v, want nil", got)
	}
}

func TestParseScopes_ExtraWhitespace(t *testing.T) {
	// strings.Fields handles multiple spaces.
	got := parseScopes("openid  email   profile")
	want := []string{"openid", "email", "profile"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseScopes (extra whitespace) = %v, want %v", got, want)
	}
}

func TestScopesGranted_AllGranted(t *testing.T) {
	granted := []string{"openid", "email", "profile"}
	requested := []string{"openid", "email"}
	if !scopesGranted(granted, requested) {
		t.Error("scopesGranted: all requested scopes are granted, but returned false")
	}
}

func TestScopesGranted_ExactMatch(t *testing.T) {
	granted := []string{"openid", "email"}
	requested := []string{"openid", "email"}
	if !scopesGranted(granted, requested) {
		t.Error("scopesGranted: exact match should return true")
	}
}

func TestScopesGranted_PartialGranted(t *testing.T) {
	granted := []string{"openid"}
	requested := []string{"openid", "email"}
	if scopesGranted(granted, requested) {
		t.Error("scopesGranted: partial grant should return false")
	}
}

func TestScopesGranted_NoneGranted(t *testing.T) {
	granted := []string{"openid"}
	requested := []string{"email", "profile"}
	if scopesGranted(granted, requested) {
		t.Error("scopesGranted: none of the requested scopes are granted, but returned true")
	}
}

func TestScopesGranted_EmptyRequested(t *testing.T) {
	granted := []string{"openid", "email"}
	// Empty requested set is vacuously true.
	if !scopesGranted(granted, nil) {
		t.Error("scopesGranted: empty requested should return true (vacuously)")
	}
}

func TestScopesGranted_EmptyBoth(t *testing.T) {
	if !scopesGranted(nil, nil) {
		t.Error("scopesGranted: both empty should return true")
	}
}

func TestScopesGranted_EmptyGranted(t *testing.T) {
	requested := []string{"openid"}
	if scopesGranted(nil, requested) {
		t.Error("scopesGranted: empty granted with non-empty requested should return false")
	}
}

func TestAuthorizeSubmit_RequiresExplicitAction(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	params := pendingAuthorizeParams()

	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantCode    web.Code
	}{
		{
			name:        "missing action",
			contentType: "application/x-www-form-urlencoded",
			body:        "",
			wantStatus:  http.StatusUnprocessableEntity,
			wantCode:    web.CodeValidation,
		},
		{
			name:        "invalid action",
			contentType: "application/x-www-form-urlencoded",
			body:        "action=maybe",
			wantStatus:  http.StatusUnprocessableEntity,
			wantCode:    web.CodeValidation,
		},
		{
			name:        "unsupported media type",
			contentType: "text/plain",
			body:        "action=approve",
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    web.CodeUnsupportedMedia,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes := &fakeCodeStore{}
			consents := &fakeConsentStore{}
			h := newAuthorizeHandlerForSubmit(codes, consents)

			_, sess, err := runAuthorizeSubmit(t, h, tt.contentType, tt.body, userID, func(sess *session.Session) error {
				return sess.Set(authzParamsSessionKey, params)
			})
			if err == nil {
				t.Fatal("Submit error = nil, want error")
			}

			var webErr *web.Error
			if !errors.As(err, &webErr) {
				t.Fatalf("Submit error = %T, want *web.Error", err)
			}
			if webErr.Status != tt.wantStatus {
				t.Fatalf("error status = %d, want %d", webErr.Status, tt.wantStatus)
			}
			if webErr.Code != tt.wantCode {
				t.Fatalf("error code = %q, want %q", webErr.Code, tt.wantCode)
			}

			assertSessionHasPendingParams(t, sess, true)
			if len(codes.codes) != 0 {
				t.Fatalf("codes created = %d, want 0", len(codes.codes))
			}
			if len(consents.saved) != 0 {
				t.Fatalf("consents saved = %d, want 0", len(consents.saved))
			}
		})
	}
}

func TestAuthorizeSubmit_DenyRedirectsAndClearsSession(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	params := pendingAuthorizeParams()
	h := newAuthorizeHandlerForSubmit(&fakeCodeStore{}, &fakeConsentStore{})

	resp, sess, err := runAuthorizeSubmit(t, h, "application/x-www-form-urlencoded", "action=deny", userID, func(sess *session.Session) error {
		return sess.Set(authzParamsSessionKey, params)
	})
	if err != nil {
		t.Fatalf("Submit error: %v", err)
	}
	if resp.Status != http.StatusFound {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusFound)
	}

	redirectURL, err := url.Parse(resp.Headers.Get("Location"))
	if err != nil {
		t.Fatalf("url.Parse(Location): %v", err)
	}
	if got := redirectURL.Query().Get("error"); got != "access_denied" {
		t.Fatalf("error query = %q, want %q", got, "access_denied")
	}
	if got := redirectURL.Query().Get("state"); got != params.State {
		t.Fatalf("state query = %q, want %q", got, params.State)
	}

	assertSessionHasPendingParams(t, sess, false)
}

func TestAuthorizeSubmit_ApproveSavesConsentIssuesCodeAndClearsSession(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	params := pendingAuthorizeParams()
	codes := &fakeCodeStore{}
	consents := &fakeConsentStore{}
	h := newAuthorizeHandlerForSubmit(codes, consents)

	resp, sess, err := runAuthorizeSubmit(t, h, "application/x-www-form-urlencoded", "action=approve", userID, func(sess *session.Session) error {
		return sess.Set(authzParamsSessionKey, params)
	})
	if err != nil {
		t.Fatalf("Submit error: %v", err)
	}
	if resp.Status != http.StatusFound {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusFound)
	}
	if len(consents.saved) != 1 {
		t.Fatalf("consents saved = %d, want 1", len(consents.saved))
	}
	savedConsent := consents.saved[0]
	if savedConsent.UserID != userID {
		t.Fatalf("saved consent user_id = %s, want %s", savedConsent.UserID, userID)
	}
	if savedConsent.ClientID != params.ClientID {
		t.Fatalf("saved consent client_id = %q, want %q", savedConsent.ClientID, params.ClientID)
	}
	if !reflect.DeepEqual(savedConsent.Scopes, params.Scopes) {
		t.Fatalf("saved consent scopes = %v, want %v", savedConsent.Scopes, params.Scopes)
	}
	if len(codes.codes) != 1 {
		t.Fatalf("codes created = %d, want 1", len(codes.codes))
	}

	var savedCode AuthorizationCode
	for _, code := range codes.codes {
		savedCode = code
	}
	if savedCode.UserID != userID {
		t.Fatalf("saved code user_id = %s, want %s", savedCode.UserID, userID)
	}
	if savedCode.ClientID != params.ClientID {
		t.Fatalf("saved code client_id = %q, want %q", savedCode.ClientID, params.ClientID)
	}
	if savedCode.RedirectURI != params.RedirectURI {
		t.Fatalf("saved code redirect_uri = %q, want %q", savedCode.RedirectURI, params.RedirectURI)
	}
	if !reflect.DeepEqual(savedCode.Scopes, params.Scopes) {
		t.Fatalf("saved code scopes = %v, want %v", savedCode.Scopes, params.Scopes)
	}

	redirectURL, err := url.Parse(resp.Headers.Get("Location"))
	if err != nil {
		t.Fatalf("url.Parse(Location): %v", err)
	}
	if got := redirectURL.Query().Get("state"); got != params.State {
		t.Fatalf("state query = %q, want %q", got, params.State)
	}
	if got := redirectURL.Query().Get("code"); got == "" {
		t.Fatal("code query is empty, want authorization code")
	}

	assertSessionHasPendingParams(t, sess, false)
}
