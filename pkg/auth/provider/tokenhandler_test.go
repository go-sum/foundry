package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/go-sum/foundry/pkg/web"
	webauth "github.com/go-sum/foundry/pkg/web/auth"
	"github.com/go-sum/foundry/pkg/web/validate"
)

// ---------------------------------------------------------------------------
// Fake stores
// ---------------------------------------------------------------------------

type fakeCodeStore struct {
	codes  map[string]AuthorizationCode
	marked []string
	err    error
}

func (f *fakeCodeStore) CreateCode(_ context.Context, code AuthorizationCode) error {
	if f.codes == nil {
		f.codes = make(map[string]AuthorizationCode)
	}
	f.codes[code.Code] = code
	return f.err
}

func (f *fakeCodeStore) GetCode(_ context.Context, code string) (AuthorizationCode, error) {
	if f.err != nil {
		return AuthorizationCode{}, f.err
	}
	c, ok := f.codes[code]
	if !ok {
		return AuthorizationCode{}, ErrCodeNotFound
	}
	return c, nil
}

func (f *fakeCodeStore) MarkCodeUsed(_ context.Context, code string) error {
	f.marked = append(f.marked, code)
	if f.codes != nil {
		c := f.codes[code]
		c.Used = true
		f.codes[code] = c
	}
	return f.err
}

func (f *fakeCodeStore) DeleteExpiredCodes(_ context.Context) error { return f.err }

// ---------------------------------------------------------------------------

type fakeTokenStore struct {
	tokens map[string]OAuthToken // keyed by TokenHash
	byID   map[uuid.UUID]OAuthToken
	err    error
}

func (f *fakeTokenStore) CreateToken(_ context.Context, token OAuthToken) error {
	if f.err != nil {
		return f.err
	}
	if f.tokens == nil {
		f.tokens = make(map[string]OAuthToken)
		f.byID = make(map[uuid.UUID]OAuthToken)
	}
	f.tokens[token.TokenHash] = token
	f.byID[token.ID] = token
	return nil
}

func (f *fakeTokenStore) GetTokenByHash(_ context.Context, hash string) (OAuthToken, error) {
	if f.err != nil {
		return OAuthToken{}, f.err
	}
	t, ok := f.tokens[hash]
	if !ok {
		return OAuthToken{}, ErrTokenNotFound
	}
	return t, nil
}

func (f *fakeTokenStore) RevokeToken(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if t, ok := f.byID[id]; ok {
		t.Revoked = true
		f.byID[id] = t
		f.tokens[t.TokenHash] = t
	}
	return nil
}

func (f *fakeTokenStore) RevokeTokensByUserAndClient(_ context.Context, _ uuid.UUID, _ string) error {
	return f.err
}

func (f *fakeTokenStore) DeleteExpiredTokens(_ context.Context) error { return f.err }

// ---------------------------------------------------------------------------

type fakeClientStore struct {
	clients map[string]OAuthClient
	err     error
}

func (f *fakeClientStore) GetClientByClientID(_ context.Context, clientID string) (OAuthClient, error) {
	if f.err != nil {
		return OAuthClient{}, f.err
	}
	c, ok := f.clients[clientID]
	if !ok {
		return OAuthClient{}, ErrClientNotFound
	}
	return c, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newTokenHandler builds a TokenHandler with the given stores and a real validator.
func newTokenHandler(codes CodeStore, tokens TokenStore) *TokenHandler {
	return &TokenHandler{
		clients:   &fakeClientStore{},
		codes:     codes,
		tokens:    tokens,
		config:    ApplyDefaults(Config{}),
		validator: validate.New(),
		logger:    slog.Default(),
	}
}

// formContext builds a *web.Context with an application/x-www-form-urlencoded body.
func formContext(values url.Values) *web.Context {
	u, _ := url.Parse("/oauth/token")
	req := web.NewRequest("POST", u)
	body := values.Encode()
	req.SetBody(io.NopCloser(strings.NewReader(body)))
	req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	return web.NewContext(context.Background(), req)
}

// readJSON decodes the JSON body of a web.Response into a map[string]string.
func readJSON(t *testing.T, resp web.Response) map[string]string {
	t.Helper()
	if resp.Body == nil {
		t.Fatal("response body is nil")
	}
	var m map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decoding JSON response: %v", err)
	}
	return m
}

// readJSONAny decodes the JSON body into a map[string]interface{}.
func readJSONAny(t *testing.T, resp web.Response) map[string]interface{} {
	t.Helper()
	if resp.Body == nil {
		t.Fatal("response body is nil")
	}
	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decoding JSON response: %v", err)
	}
	return m
}

// validCode builds a non-expired, non-used AuthorizationCode.
func validCode(clientID, redirectURI string, scopes []string) AuthorizationCode {
	return AuthorizationCode{
		Code:        "valid-code",
		ClientID:    clientID,
		UserID:      uuid.New(),
		RedirectURI: redirectURI,
		Scopes:      scopes,
		ExpiresAt:   time.Now().UTC().Add(5 * time.Minute),
		CreatedAt:   time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// Exchange — routing-only tests (no second body read required)
// ---------------------------------------------------------------------------

func TestTokenHandler_Exchange_MissingGrantType(t *testing.T) {
	h := newTokenHandler(&fakeCodeStore{}, &fakeTokenStore{})

	// Omit grant_type entirely.
	c := formContext(url.Values{
		"code": {"some-code"},
	})

	resp, err := h.Exchange(c)
	if err != nil {
		t.Fatalf("Exchange error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_request" {
		t.Errorf("error = %q, want invalid_request", body["error"])
	}
}

func TestTokenHandler_Exchange_UnsupportedGrantType(t *testing.T) {
	h := newTokenHandler(&fakeCodeStore{}, &fakeTokenStore{})

	c := formContext(url.Values{
		"grant_type": {"client_credentials"},
	})

	resp, err := h.Exchange(c)
	if err != nil {
		t.Fatalf("Exchange error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "unsupported_grant_type" {
		t.Errorf("error = %q, want unsupported_grant_type", body["error"])
	}
}

// ---------------------------------------------------------------------------
// handleAuthorizationCode — direct invocation (same package, avoids double-read)
// ---------------------------------------------------------------------------

func TestHandleAuthorizationCode_ValidExchange(t *testing.T) {
	codes := &fakeCodeStore{}
	tokens := &fakeTokenStore{}
	h := newTokenHandler(codes, tokens)

	code := validCode("client1", "https://app.example.com/cb", []string{"openid"})
	codes.codes = map[string]AuthorizationCode{"valid-code": code}

	c := formContext(url.Values{})

	resp, err := h.handleAuthorizationCode(c, "valid-code", "https://app.example.com/cb", "client1", "")
	if err != nil {
		t.Fatalf("handleAuthorizationCode error: %v", err)
	}
	if resp.Status != 200 {
		errBody := readJSON(t, resp)
		t.Fatalf("status = %d, want 200; body=%v", resp.Status, errBody)
	}

	body := readJSONAny(t, resp)
	if v, _ := body["access_token"].(string); v == "" {
		t.Error("access_token is empty")
	}
	if v, _ := body["refresh_token"].(string); v == "" {
		t.Error("refresh_token is empty")
	}
	if v, _ := body["token_type"].(string); v != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", v)
	}
	// Code should be marked used.
	if len(codes.marked) != 1 || codes.marked[0] != "valid-code" {
		t.Errorf("MarkCodeUsed not called with correct code; marked=%v", codes.marked)
	}
}

func TestHandleAuthorizationCode_CodeNotFound(t *testing.T) {
	codes := &fakeCodeStore{codes: map[string]AuthorizationCode{}}
	h := newTokenHandler(codes, &fakeTokenStore{})

	c := formContext(url.Values{})

	resp, err := h.handleAuthorizationCode(c, "nonexistent", "https://app.example.com/cb", "client1", "")
	if err != nil {
		t.Fatalf("handleAuthorizationCode error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleAuthorizationCode_CodeAlreadyUsed(t *testing.T) {
	code := validCode("client1", "https://app.example.com/cb", []string{"openid"})
	code.Used = true
	codes := &fakeCodeStore{
		codes: map[string]AuthorizationCode{"used-code": code},
	}
	h := newTokenHandler(codes, &fakeTokenStore{})

	c := formContext(url.Values{})

	resp, err := h.handleAuthorizationCode(c, "used-code", "https://app.example.com/cb", "client1", "")
	if err != nil {
		t.Fatalf("handleAuthorizationCode error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleAuthorizationCode_CodeExpired(t *testing.T) {
	code := validCode("client1", "https://app.example.com/cb", []string{"openid"})
	code.ExpiresAt = time.Now().UTC().Add(-time.Minute) // already expired
	codes := &fakeCodeStore{
		codes: map[string]AuthorizationCode{"expired-code": code},
	}
	h := newTokenHandler(codes, &fakeTokenStore{})

	c := formContext(url.Values{})

	resp, err := h.handleAuthorizationCode(c, "expired-code", "https://app.example.com/cb", "client1", "")
	if err != nil {
		t.Fatalf("handleAuthorizationCode error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleAuthorizationCode_ClientIDMismatch(t *testing.T) {
	code := validCode("correct-client", "https://app.example.com/cb", []string{"openid"})
	codes := &fakeCodeStore{
		codes: map[string]AuthorizationCode{"code1": code},
	}
	h := newTokenHandler(codes, &fakeTokenStore{})

	c := formContext(url.Values{})

	resp, err := h.handleAuthorizationCode(c, "code1", "https://app.example.com/cb", "wrong-client", "")
	if err != nil {
		t.Fatalf("handleAuthorizationCode error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleAuthorizationCode_RedirectURIMismatch(t *testing.T) {
	code := validCode("client1", "https://app.example.com/cb", []string{"openid"})
	codes := &fakeCodeStore{
		codes: map[string]AuthorizationCode{"code1": code},
	}
	h := newTokenHandler(codes, &fakeTokenStore{})

	c := formContext(url.Values{})

	resp, err := h.handleAuthorizationCode(c, "code1", "https://evil.example.com/cb", "client1", "")
	if err != nil {
		t.Fatalf("handleAuthorizationCode error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleAuthorizationCode_PKCERequiredButMissing(t *testing.T) {
	verifier, err := webauth.NewVerifier()
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	challenge, err := webauth.Challenge(verifier)
	if err != nil {
		t.Fatalf("Challenge: %v", err)
	}

	code := validCode("client1", "https://app.example.com/cb", []string{"openid"})
	code.CodeChallenge = challenge
	codes := &fakeCodeStore{
		codes: map[string]AuthorizationCode{"code1": code},
	}
	h := newTokenHandler(codes, &fakeTokenStore{})

	// No code_verifier provided.
	c := formContext(url.Values{})

	resp, err := h.handleAuthorizationCode(c, "code1", "https://app.example.com/cb", "client1", "")
	if err != nil {
		t.Fatalf("handleAuthorizationCode error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleAuthorizationCode_PKCEWrongVerifier(t *testing.T) {
	verifier, err := webauth.NewVerifier()
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	challenge, err := webauth.Challenge(verifier)
	if err != nil {
		t.Fatalf("Challenge: %v", err)
	}

	code := validCode("client1", "https://app.example.com/cb", []string{"openid"})
	code.CodeChallenge = challenge
	codes := &fakeCodeStore{
		codes: map[string]AuthorizationCode{"code1": code},
	}
	h := newTokenHandler(codes, &fakeTokenStore{})

	wrongVerifier, _ := webauth.NewVerifier()
	c := formContext(url.Values{})

	resp, err := h.handleAuthorizationCode(c, "code1", "https://app.example.com/cb", "client1", wrongVerifier)
	if err != nil {
		t.Fatalf("handleAuthorizationCode error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleAuthorizationCode_PKCEValidVerifier(t *testing.T) {
	verifier, err := webauth.NewVerifier()
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	challenge, err := webauth.Challenge(verifier)
	if err != nil {
		t.Fatalf("Challenge: %v", err)
	}

	code := validCode("client1", "https://app.example.com/cb", []string{"openid"})
	code.CodeChallenge = challenge
	codes := &fakeCodeStore{
		codes: map[string]AuthorizationCode{"code1": code},
	}
	h := newTokenHandler(codes, &fakeTokenStore{})

	c := formContext(url.Values{})

	resp, err := h.handleAuthorizationCode(c, "code1", "https://app.example.com/cb", "client1", verifier)
	if err != nil {
		t.Fatalf("handleAuthorizationCode error: %v", err)
	}
	if resp.Status != 200 {
		body := readJSON(t, resp)
		t.Fatalf("status = %d, want 200; body=%v", resp.Status, body)
	}
}

func TestHandleAuthorizationCode_ScopesPropagatedToTokens(t *testing.T) {
	scopes := []string{"openid", "email", "profile"}
	code := validCode("client1", "https://app.example.com/cb", scopes)
	codes := &fakeCodeStore{
		codes: map[string]AuthorizationCode{"code1": code},
	}
	tokens := &fakeTokenStore{}
	h := newTokenHandler(codes, tokens)

	c := formContext(url.Values{})

	resp, err := h.handleAuthorizationCode(c, "code1", "https://app.example.com/cb", "client1", "")
	if err != nil {
		t.Fatalf("handleAuthorizationCode error: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("status = %d, want 200", resp.Status)
	}

	body := readJSONAny(t, resp)
	wantScope := fmt.Sprintf("%s %s %s", scopes[0], scopes[1], scopes[2])
	gotScope, ok := body["scope"].(string)
	if !ok {
		t.Fatalf("scope field missing or wrong type: %v", body["scope"])
	}
	if gotScope != wantScope {
		t.Errorf("scope = %q, want %q", gotScope, wantScope)
	}
}

// ---------------------------------------------------------------------------
// handleRefreshToken — direct invocation
// ---------------------------------------------------------------------------

// buildRefreshToken stores a refresh token in the fake store and returns the raw token string.
func buildRefreshToken(t *testing.T, tokens *fakeTokenStore, clientID string, userID uuid.UUID, scopes []string) string {
	t.Helper()
	raw, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	tok := OAuthToken{
		ID:        uuid.New(),
		TokenHash: HashToken(raw),
		TokenType: "refresh",
		ClientID:  clientID,
		UserID:    userID,
		Scopes:    scopes,
		Revoked:   false,
		ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	if err := tokens.CreateToken(context.Background(), tok); err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	return raw
}

func TestHandleRefreshToken_Valid(t *testing.T) {
	tokens := &fakeTokenStore{}
	userID := uuid.New()
	rawRefresh := buildRefreshToken(t, tokens, "client1", userID, []string{"openid"})

	h := newTokenHandler(&fakeCodeStore{}, tokens)

	c := formContext(url.Values{})

	resp, err := h.handleRefreshToken(c, rawRefresh, "client1")
	if err != nil {
		t.Fatalf("handleRefreshToken error: %v", err)
	}
	if resp.Status != 200 {
		errBody := readJSON(t, resp)
		t.Fatalf("status = %d, want 200; body=%v", resp.Status, errBody)
	}
	body := readJSONAny(t, resp)
	accessToken, _ := body["access_token"].(string)
	refreshToken, _ := body["refresh_token"].(string)
	if accessToken == "" {
		t.Error("access_token is empty")
	}
	if refreshToken == "" {
		t.Error("refresh_token is empty")
	}
	// Token rotation: new refresh token must differ from the old one.
	if refreshToken == rawRefresh {
		t.Error("refresh_token was not rotated; old and new values are identical")
	}
}

func TestHandleRefreshToken_NotFound(t *testing.T) {
	tokens := &fakeTokenStore{}
	h := newTokenHandler(&fakeCodeStore{}, tokens)

	c := formContext(url.Values{})

	resp, err := h.handleRefreshToken(c, "nonexistent-token", "client1")
	if err != nil {
		t.Fatalf("handleRefreshToken error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleRefreshToken_Revoked(t *testing.T) {
	tokens := &fakeTokenStore{}
	userID := uuid.New()
	rawRefresh := buildRefreshToken(t, tokens, "client1", userID, []string{"openid"})

	// Mark the token revoked directly.
	hash := HashToken(rawRefresh)
	tok := tokens.tokens[hash]
	tok.Revoked = true
	tokens.tokens[hash] = tok
	tokens.byID[tok.ID] = tok

	h := newTokenHandler(&fakeCodeStore{}, tokens)

	c := formContext(url.Values{})

	resp, err := h.handleRefreshToken(c, rawRefresh, "client1")
	if err != nil {
		t.Fatalf("handleRefreshToken error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleRefreshToken_Expired(t *testing.T) {
	tokens := &fakeTokenStore{}
	userID := uuid.New()
	rawRefresh := buildRefreshToken(t, tokens, "client1", userID, []string{"openid"})

	// Expire the token.
	hash := HashToken(rawRefresh)
	tok := tokens.tokens[hash]
	tok.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	tokens.tokens[hash] = tok
	tokens.byID[tok.ID] = tok

	h := newTokenHandler(&fakeCodeStore{}, tokens)

	c := formContext(url.Values{})

	resp, err := h.handleRefreshToken(c, rawRefresh, "client1")
	if err != nil {
		t.Fatalf("handleRefreshToken error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleRefreshToken_ClientIDMismatch(t *testing.T) {
	tokens := &fakeTokenStore{}
	userID := uuid.New()
	rawRefresh := buildRefreshToken(t, tokens, "correct-client", userID, []string{"openid"})

	h := newTokenHandler(&fakeCodeStore{}, tokens)

	c := formContext(url.Values{})

	resp, err := h.handleRefreshToken(c, rawRefresh, "wrong-client")
	if err != nil {
		t.Fatalf("handleRefreshToken error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

func TestHandleRefreshToken_WrongTokenType(t *testing.T) {
	// Insert a token with type "access" and try to redeem it as a refresh token.
	tokens := &fakeTokenStore{}
	rawToken, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	tok := OAuthToken{
		ID:        uuid.New(),
		TokenHash: HashToken(rawToken),
		TokenType: "access", // wrong type
		ClientID:  "client1",
		UserID:    uuid.New(),
		Scopes:    []string{"openid"},
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	_ = tokens.CreateToken(context.Background(), tok)

	h := newTokenHandler(&fakeCodeStore{}, tokens)

	c := formContext(url.Values{})

	resp, err := h.handleRefreshToken(c, rawToken, "client1")
	if err != nil {
		t.Fatalf("handleRefreshToken error: %v", err)
	}
	if resp.Status != 400 {
		t.Fatalf("status = %d, want 400", resp.Status)
	}
	body := readJSON(t, resp)
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", body["error"])
	}
}

// ---------------------------------------------------------------------------
// oauthErrorJSON helper tests
// ---------------------------------------------------------------------------

func TestOAuthErrorJSON_WithDescription(t *testing.T) {
	m := oauthErrorJSON("invalid_grant", "code expired")
	if m["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", m["error"])
	}
	if m["error_description"] != "code expired" {
		t.Errorf("error_description = %q, want %q", m["error_description"], "code expired")
	}
}

func TestOAuthErrorJSON_NoDescription(t *testing.T) {
	m := oauthErrorJSON("invalid_request", "")
	if m["error"] != "invalid_request" {
		t.Errorf("error = %q, want invalid_request", m["error"])
	}
	if _, ok := m["error_description"]; ok {
		t.Error("error_description should be absent when description is empty")
	}
}

// ---------------------------------------------------------------------------
// joinScopes helper tests
// ---------------------------------------------------------------------------

func TestJoinScopes(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"openid", "email", "profile"}, "openid email profile"},
		{[]string{"openid"}, "openid"},
		{[]string{}, ""},
		{nil, ""},
	}
	for _, tc := range cases {
		got := joinScopes(tc.in)
		if got != tc.want {
			t.Errorf("joinScopes(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
