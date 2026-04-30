package authn

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/session"
)

func testSessionConfig() session.Config {
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

func runAuthRequest(t *testing.T, handler web.Handler, requestURL *url.URL, extraHeaders map[string]string) (web.Response, error) {
	t.Helper()
	req := web.NewRequest(http.MethodGet, requestURL)
	for k, v := range extraHeaders {
		req.Headers.Set(k, v)
	}
	return handler(web.NewContext(context.Background(), req))
}

func TestRequireAuth_PanicsWithoutSessionMiddleware(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic, got none")
		}
	}()
	handler := RequireAuth(func() string { return "/signin" })(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/protected"})
	_, _ = handler(web.NewContext(context.Background(), req))
}

func TestRequireAuth_LoadsIdentityFromSessionWithoutLoadSession(t *testing.T) {
	sessionMW := session.Middleware(testSessionConfig())
	setAuthMW := func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			sess, ok := session.FromContext(c)
			if !ok {
				t.Fatal("session not in context")
			}
			if err := SetAuth(sess, "user-123", "Casey", true); err != nil {
				t.Fatalf("SetAuth: %v", err)
			}
			return next(c)
		}
	}

	var gotID string
	var gotIdentity Identity
	handler := sessionMW(setAuthMW(RequireAuth(func() string { return "/signin" })(func(c *web.Context) (web.Response, error) {
		gotID = UserID(c)
		gotIdentity = GetIdentity(c)
		return web.Respond(http.StatusNoContent), nil
	})))

	resp, err := runAuthRequest(t, handler, &url.URL{Path: "/account"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusNoContent {
		t.Fatalf("Status = %d, want %d", resp.Status, http.StatusNoContent)
	}
	if gotID != "user-123" {
		t.Fatalf("UserID = %q, want %q", gotID, "user-123")
	}
	if !gotIdentity.IsAuthenticated {
		t.Fatal("Identity.IsAuthenticated = false, want true")
	}
	if !gotIdentity.IsVerified {
		t.Fatal("Identity.IsVerified = false, want true")
	}
	if gotIdentity.DisplayName != "Casey" {
		t.Fatalf("Identity.DisplayName = %q, want %q", gotIdentity.DisplayName, "Casey")
	}
}

func TestRequireAuth_UnauthenticatedFullPageRedirectsWithReturnTo(t *testing.T) {
	sessionMW := session.Middleware(testSessionConfig())
	handler := sessionMW(RequireAuth(func() string { return "/signin" })(func(_ *web.Context) (web.Response, error) {
		t.Fatal("protected handler should not be called")
		return web.Respond(http.StatusOK), nil
	}))

	resp, err := runAuthRequest(t, handler, &url.URL{Path: "/protected", RawQuery: "tab=profile"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusSeeOther {
		t.Fatalf("Status = %d, want %d", resp.Status, http.StatusSeeOther)
	}
	if got := resp.Headers.Get("Location"); got != "/signin?return_to=%2Fprotected%3Ftab%3Dprofile" {
		t.Fatalf("Location = %q, want %q", got, "/signin?return_to=%2Fprotected%3Ftab%3Dprofile")
	}
}

func TestRequireAuth_UnauthenticatedHTMXSetsRedirectHeader(t *testing.T) {
	sessionMW := session.Middleware(testSessionConfig())
	handler := sessionMW(RequireAuth(func() string { return "/signin?source=auth" })(func(_ *web.Context) (web.Response, error) {
		t.Fatal("protected handler should not be called")
		return web.Respond(http.StatusOK), nil
	}))

	resp, err := runAuthRequest(t, handler, &url.URL{Path: "/protected"}, map[string]string{"HX-Request": "true"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusUnauthorized {
		t.Fatalf("Status = %d, want %d", resp.Status, http.StatusUnauthorized)
	}
	if got := resp.Headers.Get("HX-Redirect"); got != "/signin?source=auth&return_to=%2Fprotected" {
		t.Fatalf("HX-Redirect = %q, want %q", got, "/signin?source=auth&return_to=%2Fprotected")
	}
}

func TestEnsureIdentityFromSession_SkipsWhenContextMatchesSession(t *testing.T) {
	sessionMW := session.Middleware(testSessionConfig())
	handler := sessionMW(func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok {
			t.Fatal("session not in context")
		}
		if err := SetAuth(sess, "user-123", "Casey", true); err != nil {
			t.Fatalf("SetAuth: %v", err)
		}

		SetUserID(c, "user-123")
		SetIdentity(c, Identity{
			IsAuthenticated: true,
			IsVerified:      true,
			DisplayName:     "Casey",
		})

		if changed := ensureIdentityFromSession(c, sess); changed {
			t.Fatal("ensureIdentityFromSession() = true, want false")
		}
		return web.Respond(http.StatusNoContent), nil
	})

	resp, err := runAuthRequest(t, handler, &url.URL{Path: "/account"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusNoContent {
		t.Fatalf("Status = %d, want %d", resp.Status, http.StatusNoContent)
	}
}

func TestRequireAuth_RefreshesIdentityWhenSessionChangesAfterLoadSession(t *testing.T) {
	sessionMW := session.Middleware(testSessionConfig())
	setAuthMW := func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			sess, ok := session.FromContext(c)
			if !ok {
				t.Fatal("session not in context")
			}
			if err := SetAuth(sess, "user-456", "Jordan", false); err != nil {
				t.Fatalf("SetAuth: %v", err)
			}
			return next(c)
		}
	}

	var gotID string
	var gotIdentity Identity
	handler := sessionMW(LoadSession()(setAuthMW(RequireAuth(func() string { return "/signin" })(func(c *web.Context) (web.Response, error) {
		gotID = UserID(c)
		gotIdentity = GetIdentity(c)
		return web.Respond(http.StatusNoContent), nil
	}))))

	resp, err := runAuthRequest(t, handler, &url.URL{Path: "/account"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusNoContent {
		t.Fatalf("Status = %d, want %d", resp.Status, http.StatusNoContent)
	}
	if gotID != "user-456" {
		t.Fatalf("UserID = %q, want %q", gotID, "user-456")
	}
	if !gotIdentity.IsAuthenticated {
		t.Fatal("Identity.IsAuthenticated = false, want true")
	}
	if gotIdentity.IsVerified {
		t.Fatal("Identity.IsVerified = true, want false")
	}
	if gotIdentity.DisplayName != "Jordan" {
		t.Fatalf("Identity.DisplayName = %q, want %q", gotIdentity.DisplayName, "Jordan")
	}
}
