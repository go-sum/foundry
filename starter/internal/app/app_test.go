package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/serve"
	"github.com/go-sum/foundry/pkg/web/session"
)

type stubSessionStore struct {
	stopCount int
}

type stubKVStore struct {
	pingErr    error
	closeErr   error
	closeCount int
}

const (
	testCSRFHexKey      = "0000000000000000000000000000000000000000000000000000000000000001" // for-tests-only
	testAuthTokenHexKey = "0000000000000000000000000000000000000000000000000000000000000002" // for-tests-only
	testSessionHexKey   = "0000000000000000000000000000000000000000000000000000000000000003" // for-tests-only
)

func setupTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "testing")
	t.Setenv("SECURITY_CSRF_KEY", testCSRFHexKey)
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", testAuthTokenHexKey)
	t.Setenv("SECURITY_SESSION_KEY", testSessionHexKey)
	t.Setenv("SITE_BASE_URL", "http://test.local")

	dir, err := os.MkdirTemp("", "static-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) }) //nolint:errcheck // cleanup: best-effort removal, failure is non-fatal in tests
	t.Setenv("TEST_STATIC_DIR", dir)
}

func mustNew(t *testing.T) *App {
	t.Helper()
	a, err := New(context.Background())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Errorf("App.Close() error = %v", err)
		}
	})
	return a
}

func newSecurityHarness(t *testing.T) (http.Handler, *App) {
	t.Helper()
	a := mustNew(t)
	handler := web.Chain(
		func(c *web.Context) (web.Response, error) {
			if c.Method() == http.MethodGet {
				return web.Text(http.StatusOK, c.URL().Scheme+"\n"+secure.CSRFToken(c)), nil
			}
			return web.Text(http.StatusOK, "ok"), nil
		},
		session.Middleware(a.Session),
		secure.CSRF(a.CSRF),
		secure.OriginGuard(secure.OriginGuardConfig{TrustedOrigins: a.Origins, ServerOrigin: a.ServerOrigin}),
	)
	srv, err := serve.NewServer(handler, a.Config.Server)
	if err != nil {
		t.Fatalf("serve.NewServer: %v", err)
	}
	return srv.Handler, a
}

func extractSchemeAndToken(t *testing.T, body string) (string, string) {
	t.Helper()
	parts := strings.SplitN(strings.TrimSpace(body), "\n", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		t.Fatalf("unexpected harness response body %q", body)
	}
	return parts[0], parts[1]
}

func sessionCookieValue(t *testing.T, resp *http.Response) string {
	t.Helper()
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session" {
			return cookie.Name + "=" + cookie.Value
		}
	}
	t.Fatalf("session cookie not found in response")
	return ""
}

func (s *stubSessionStore) Read(context.Context, string) ([]byte, int64, error) {
	return nil, 0, session.ErrSessionNotFound
}

func (s *stubSessionStore) Save(context.Context, string, []byte, time.Time, time.Duration, int64) (string, error) {
	return "", nil
}

func (s *stubSessionStore) Delete(context.Context, string) error {
	return nil
}

func (s *stubSessionStore) Stop() {
	s.stopCount++
}

func (s *stubKVStore) Ping(context.Context) error { return s.pingErr }

func (s *stubKVStore) Get(context.Context, string) ([]byte, error) {
	return nil, kv.ErrNotFound
}

func (s *stubKVStore) Set(context.Context, string, []byte, kv.SetOptions) error { return nil }

func (s *stubKVStore) Delete(context.Context, ...string) error { return nil }

func (s *stubKVStore) Exists(context.Context, ...string) (int64, error) { return 0, nil }

func (s *stubKVStore) Close() error {
	s.closeCount++
	return s.closeErr
}

func (s *stubKVStore) SessionRead(context.Context, string, time.Time) ([]byte, int64, bool, error) {
	return nil, 0, false, nil
}

func (s *stubKVStore) SessionSave(context.Context, string, []byte, time.Time, time.Duration, int64, time.Time) (int64, bool, bool, error) {
	return 1, false, false, nil
}

func TestApp_Healthz_Returns200(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).router.Serve)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Host = "test.local"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", string(body), "ok")
	}
}

func TestApp_UnknownPath_Returns404(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).router.Serve)
	req := httptest.NewRequest(http.MethodGet, "/definitely-does-not-exist", nil)
	req.Host = "test.local"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestApp_GET_SetsRequestIDHeader(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).router.Serve)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Host = "test.local"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header not set")
	}
}

// TestApp_CookieSessionStore_BootsAndServesRequests verifies that the app starts
// successfully when SESSION_STORE=cookie with a valid encryption key.
func TestApp_CookieSessionStore_BootsAndServesRequests(t *testing.T) {
	setupTestEnv(t)

	h := serve.ToHTTPHandler(mustNew(t).router.Serve)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Host = "test.local"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestApp_CookieSessionStore_MissingKey_ReturnsError verifies that New() returns
// a descriptive error when SESSION_STORE=cookie but SECURITY_SESSION_KEY is absent.
func TestApp_CookieSessionStore_MissingKey_ReturnsError(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("SECURITY_SESSION_KEY", "")

	_, err := New(context.Background())
	if err == nil {
		t.Fatal("expected error for missing SECURITY_SESSION_KEY, got nil")
	}
}

func TestApp_MemorySessionStore_AllowedInTesting(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("SESSION_STORE", "memory")

	h := serve.ToHTTPHandler(mustNew(t).router.Serve)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Host = "test.local"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestApp_KVSessionStore_AllowedInTesting(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("SESSION_STORE", "kv")

	store := &stubKVStore{}
	a, err := New(context.Background(), WithKVStoreFactory(func(context.Context, Runtime) (kv.Store, error) {
		return store, nil
	}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := a.Close(); err != nil {
			t.Errorf("App.Close() error = %v", err)
		}
	})

	h := serve.ToHTTPHandler(a.router.Serve)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Host = "test.local"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestApp_KVSessionStore_UnreachableKV_ReturnsError(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("SESSION_STORE", "kv")

	_, err := New(context.Background(), WithKVStoreFactory(func(context.Context, Runtime) (kv.Store, error) {
		return &stubKVStore{pingErr: errors.New("dial tcp: connection refused")}, nil
	}))
	if err == nil {
		t.Fatal("expected error for unreachable KV, got nil")
	}
}

// TestApp_GET_SetsSessionCookie verifies that the session middleware issues a
// session cookie on the first request. CSRF uses session-backed tokens when a
// session is present (no separate csrf double-submit cookie).
func TestApp_GET_SetsSessionCookie(t *testing.T) {
	setupTestEnv(t)
	h, _ := newSecurityHarness(t)
	req := httptest.NewRequest(http.MethodGet, "/form", nil)
	req.Host = "test.local"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	resp := rec.Result()
	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == "session" {
			found = true
			break
		}
	}
	if !found {
		t.Error("session cookie not set in response")
	}
}

func TestApp_GET_SetsSecurityHeaders(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).router.Serve)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Host = "test.local"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}

func TestApp_SecurityHarness_TrustedProxyAcceptsHTTPSOrigin(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("SERVER_TRUSTED_PROXIES", "192.0.2.0/24")
	a := mustNew(t)
	if got := len(a.Config.Server.TrustedProxies); got != 1 {
		t.Fatalf("TrustedProxies length = %d, want 1", got)
	}

	// Override ServerOrigin to HTTPS to simulate a production deployment
	// behind a TLS-terminating proxy where SITE_BASE_URL is the external URL.
	a.ServerOrigin = "https://test.local"
	a.CSRF.ServerOrigin = "https://test.local"

	handler := web.Chain(
		func(c *web.Context) (web.Response, error) {
			if c.Method() == http.MethodGet {
				return web.Text(http.StatusOK, c.URL().Scheme+"\n"+secure.CSRFToken(c)), nil
			}
			return web.Text(http.StatusOK, "ok"), nil
		},
		session.Middleware(a.Session),
		secure.CSRF(a.CSRF),
		secure.OriginGuard(secure.OriginGuardConfig{TrustedOrigins: a.Origins, ServerOrigin: a.ServerOrigin}),
	)
	srv, err := serve.NewServer(handler, a.Config.Server)
	if err != nil {
		t.Fatalf("serve.NewServer: %v", err)
	}
	h := srv.Handler

	getReq := httptest.NewRequest(http.MethodGet, "/form", nil)
	getReq.Host = "test.local"
	getReq.RemoteAddr = "192.0.2.1:1234"
	getReq.Header.Set("X-Forwarded-Proto", "https")
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /form status = %d, want %d", getRec.Code, http.StatusOK)
	}

	scheme, token := extractSchemeAndToken(t, getRec.Body.String())
	if scheme != "https" {
		t.Fatalf("GET /form scheme = %q, want %q", scheme, "https")
	}
	cookie := sessionCookieValue(t, getRec.Result())

	postReq := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader("name=Proxy+User"))
	postReq.Host = "test.local"
	postReq.RemoteAddr = "192.0.2.1:1234"
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("X-Forwarded-Proto", "https")
	postReq.Header.Set("Origin", "https://test.local")
	postReq.Header.Set("Cookie", cookie)
	postReq.Header.Set("X-CSRF-Token", token)
	postRec := httptest.NewRecorder()
	h.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("POST /form status = %d, want %d", postRec.Code, http.StatusOK)
	}
}

// TestApp_SecurityHarness_UntrustedProxy_SchemeMismatch verifies that
// X-Forwarded-Proto from an untrusted peer is ignored, keeping the server
// scheme at "http". A POST that sends Origin: https://test.local is then
// rejected by the CSRF middleware because the server-perceived origin
// (http://test.local) does not match the claimed HTTPS origin — demonstrating
// that an attacker cannot upgrade the perceived scheme via a rogue X-Forwarded-Proto.
func TestApp_SecurityHarness_UntrustedProxy_SchemeMismatch(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("SERVER_TRUSTED_PROXIES", "192.0.2.0/24")
	h, a := newSecurityHarness(t)
	if got := len(a.Config.Server.TrustedProxies); got != 1 {
		t.Fatalf("TrustedProxies length = %d, want 1", got)
	}

	// GET from untrusted peer with X-Forwarded-Proto: https — scheme must stay "http".
	getReq := httptest.NewRequest(http.MethodGet, "/form", nil)
	getReq.Host = "test.local"
	getReq.RemoteAddr = "203.0.113.9:4321" // outside 192.0.2.0/24
	getReq.Header.Set("X-Forwarded-Proto", "https")
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /form status = %d, want %d", getRec.Code, http.StatusOK)
	}

	scheme, token := extractSchemeAndToken(t, getRec.Body.String())
	// Untrusted proxy cannot elevate scheme — must remain "http".
	if scheme != "http" {
		t.Fatalf("GET /form scheme = %q, want %q (untrusted X-Forwarded-Proto must be ignored)", scheme, "http")
	}
	cookie := sessionCookieValue(t, getRec.Result())

	// POST with Origin: https://test.local — rejected because server sees http://test.local.
	// This tests that an attacker spoofing X-Forwarded-Proto: https cannot craft an
	// Origin that matches the elevated scheme, since the elevation is not trusted.
	postReq := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader("name=Proxy+User"))
	postReq.Host = "test.local"
	postReq.RemoteAddr = "203.0.113.9:4321"
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("X-Forwarded-Proto", "https")
	postReq.Header.Set("Origin", "https://test.local")
	postReq.Header.Set("Cookie", cookie)
	postReq.Header.Set("X-CSRF-Token", token)
	postRec := httptest.NewRecorder()
	h.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Fatalf("POST /form status = %d, want %d", postRec.Code, http.StatusForbidden)
	}
}

func TestAppClose_StopsSessionStore(t *testing.T) {
	store := &stubSessionStore{}
	a := &App{sessionStore: store}

	if err := a.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if store.stopCount != 1 {
		t.Fatalf("stopCount = %d, want 1", store.stopCount)
	}
}

func TestAppClose_ClosesSharedKVOnce(t *testing.T) {
	store := &stubKVStore{}
	a := &App{Services: Services{KVStore: store}}

	if err := a.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if store.closeCount != 1 {
		t.Fatalf("closeCount = %d, want 1", store.closeCount)
	}
}

// TestNew_CleansUpSessionStoreOnRouteRegistrationError must not run in parallel:
// it swaps the package-level newMemorySessionStore var, which is not goroutine-safe.
func TestApp_AllowedHosts_RejectsRequestWithBadHost(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).router.Serve)

	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	req.Host = "evil.attacker.com"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMisdirectedRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusMisdirectedRequest)
	}
}

func TestApp_Healthz_SkipsAllowedHosts(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).router.Serve)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Host = "10.0.0.5" // pod IP, not in AllowedHosts
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func newOAuthRouteSeparationHarness(t *testing.T) http.Handler {
	setupTestEnv(t)
	a := mustNew(t)
	routes := provider.DefaultRouteConfig()
	rt := router.New()
	coreMw, err := coreMiddleware(rt, a.Runtime, a.Security)
	if err != nil {
		t.Fatalf("coreMiddleware() error = %v", err)
	}
	rt.Use(coreMw...)
	router.Register(rt,
		router.Layout(router.Nodes(
			[]router.Node{router.Use(apiMiddleware(a.Security, routes.Token.Pattern)...)},
			[]router.Node{router.POST(routes.Token.Pattern, routes.Token.Name, func(_ *web.Context) (web.Response, error) {
				return web.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"}), nil
			})},
			[]router.Node{router.Layout(router.Nodes(
				[]router.Node{router.Use(secure.CSRF(a.CSRF))},
				[]router.Node{router.POST(routes.AuthorizePost.Pattern, routes.AuthorizePost.Name, func(_ *web.Context) (web.Response, error) {
					return web.Text(http.StatusOK, "ok"), nil
				})},
			)...)},
		)...),
	)
	rt.Freeze()
	return serve.ToHTTPHandler(rt.Serve)
}

func TestApp_OAuthTokenEndpoint_NotBlockedByCSRFMiddleware(t *testing.T) {
	h := newOAuthRouteSeparationHarness(t)

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader("client_id=test-client"))
	req.Host = "test.local"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), `"error":"invalid_request"`) {
		t.Fatalf("body = %q, want oauth invalid_request response", rec.Body.String())
	}
}

func TestApp_OAuthAuthorizePost_RemainsCSRFProtected(t *testing.T) {
	h := newOAuthRouteSeparationHarness(t)

	req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader("action=approve"))
	req.Host = "test.local"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("Status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestApp_APIMiddleware_OriginGuardSkipsTokenRoute(t *testing.T) {
	setupTestEnv(t)
	a := mustNew(t)
	routes := provider.DefaultRouteConfig()
	rt := router.New()
	coreMw2, err := coreMiddleware(rt, a.Runtime, a.Security)
	if err != nil {
		t.Fatalf("coreMiddleware() error = %v", err)
	}
	rt.Use(coreMw2...)
	router.Register(rt,
		router.Layout(router.Nodes(
			[]router.Node{router.Use(apiMiddleware(a.Security, routes.Token.Pattern)...)},
			[]router.Node{router.POST(routes.Token.Pattern, routes.Token.Name, func(_ *web.Context) (web.Response, error) {
				return web.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"}), nil
			})},
		)...),
	)
	rt.Freeze()
	h := serve.ToHTTPHandler(rt.Serve)

	req := httptest.NewRequest(http.MethodPost, routes.Token.Pattern, strings.NewReader("client_id=test-client"))
	req.Host = "test.local"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestNew_CleansUpSessionStoreOnRouteRegistrationError(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("SESSION_STORE", "memory")
	t.Setenv("TEST_STATIC_DIR", filepath.Join(t.TempDir(), "missing"))

	store := &stubSessionStore{}
	_, err := New(context.Background(), WithSessionStoreFactory(func() session.Store { return store }))
	if err == nil {
		t.Fatal("New() error = nil, want non-nil")
	}
	if store.stopCount != 1 {
		t.Fatalf("stopCount = %d, want 1", store.stopCount)
	}
}
