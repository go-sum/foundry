package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-sum/foundry/pkg/web/serve"
)

const (
	testCSRFHexKey      = "0000000000000000000000000000000000000000000000000000000000000001" // for-tests-only
	testAuthTokenHexKey = "0000000000000000000000000000000000000000000000000000000000000002" // for-tests-only
)

func setupTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "testing")
	t.Setenv("SECURITY_CSRF_KEY", testCSRFHexKey)
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", testAuthTokenHexKey)
	t.Setenv("SITE_BASE_URL", "http://test.local")

	dir, err := os.MkdirTemp("", "static-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) }) //nolint:errcheck
	t.Setenv("TEST_STATIC_DIR", dir)
}

func mustNew(t *testing.T) *App {
	t.Helper()
	a, err := New(context.Background())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return a
}

func TestApp_Healthz_Returns200(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).router.Serve)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
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
	t.Setenv("SESSION_STORE", "cookie")
	// 32-byte AES key expressed as 64 hex chars.
	t.Setenv("SECURITY_SESSION_KEY", "0000000000000000000000000000000000000000000000000000000000000002")

	h := serve.ToHTTPHandler(mustNew(t).router.Serve)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
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
	t.Setenv("SESSION_STORE", "cookie")
	t.Setenv("SECURITY_SESSION_KEY", "")

	_, err := New(context.Background())
	if err == nil {
		t.Fatal("expected error for missing SECURITY_SESSION_KEY, got nil")
	}
}

// TestApp_GET_SetsSessionCookie verifies that the session middleware issues a
// session cookie on the first request. CSRF uses session-backed tokens when a
// session is present (no separate csrf double-submit cookie).
func TestApp_GET_SetsSessionCookie(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).router.Serve)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

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
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}
