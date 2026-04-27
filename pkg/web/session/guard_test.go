package session

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
)

func runGuardRequest(t *testing.T, sessionMW web.Middleware, handler web.Handler, extraHeaders map[string]string) (web.Response, error) {
	t.Helper()
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/protected"})
	for k, v := range extraHeaders {
		req.Headers.Set(k, v)
	}
	return sessionMW(handler)(web.NewContext(context.Background(), req))
}

func TestGuard_PanicsWithoutSessionMiddleware(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic, got none")
		}
	}()
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/protected"})
	c := web.NewContext(context.Background(), req)
	handler := Guard(DefaultGuardConfig())(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})
	_, _ = handler(c)
}

func TestGuard_UnauthenticatedFullPage_RedirectsToSignin(t *testing.T) {
	sessionMW := Middleware(testMemoryConfig(t))
	guard := Guard(DefaultGuardConfig())

	called := false
	resp, err := runGuardRequest(t, sessionMW, guard(func(_ *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	}), nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("inner handler should not be called for unauthenticated request")
	}
	if resp.Status != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusSeeOther)
	}
	if loc := resp.Headers.Get("Location"); loc != "/signin" {
		t.Errorf("Location = %q, want /signin", loc)
	}
}

func TestGuard_UnauthenticatedHTMX_Returns401(t *testing.T) {
	sessionMW := Middleware(testMemoryConfig(t))
	guard := Guard(DefaultGuardConfig())

	called := false
	_, err := runGuardRequest(t, sessionMW, guard(func(_ *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	}), map[string]string{"HX-Request": "true"})

	if called {
		t.Error("inner handler should not be called for unauthenticated HTMX request")
	}
	if err == nil {
		t.Fatal("expected error for unauthenticated HTMX request, got nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", webErr.Status, http.StatusUnauthorized)
	}
}

func TestGuard_Authenticated_PassesThrough(t *testing.T) {
	sessionMW := Middleware(testMemoryConfig(t))
	guard := Guard(DefaultGuardConfig())

	called := false
	inner := guard(func(_ *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})
	setAuthedThenGuard := func(c *web.Context) (web.Response, error) {
		sess, ok := FromContext(c)
		if !ok {
			t.Fatal("session not in context")
		}
		_ = sess.Set("authed", true)
		return inner(c)
	}

	resp, err := runGuardRequest(t, sessionMW, setAuthedThenGuard, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("inner handler should be called for authenticated request")
	}
	if resp.Status != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestGuard_CustomRedirectPath(t *testing.T) {
	sessionMW := Middleware(testMemoryConfig(t))
	guard := Guard(GuardConfig{RedirectPath: "/login"})

	resp, err := runGuardRequest(t, sessionMW, guard(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}), nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusSeeOther)
	}
	if loc := resp.Headers.Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}
}

func TestGuard_CustomCheck(t *testing.T) {
	sessionMW := Middleware(testMemoryConfig(t))
	guard := Guard(GuardConfig{
		RedirectPath: "/signin",
		Check:        func(s *Session) bool { return s.Has("user_id") },
	})

	called := false
	inner := guard(func(_ *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	// "authed" key does not satisfy custom check — should redirect.
	setWrongKey := func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("authed", true)
		return inner(c)
	}
	resp, err := runGuardRequest(t, sessionMW, setWrongKey, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("inner handler should not be called when custom check fails")
	}
	if resp.Status != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusSeeOther)
	}

	// "user_id" key satisfies custom check — should pass through.
	called = false
	setRightKey := func(c *web.Context) (web.Response, error) {
		sess, _ := FromContext(c)
		_ = sess.Set("user_id", "abc")
		return inner(c)
	}
	resp, err = runGuardRequest(t, sessionMW, setRightKey, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("inner handler should be called when custom check passes")
	}
	if resp.Status != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestGuard_CustomOnUnauthenticated(t *testing.T) {
	sessionMW := Middleware(testMemoryConfig(t))
	customCalled := false
	guard := Guard(GuardConfig{
		OnUnauthenticated: func(_ *web.Context) (web.Response, error) {
			customCalled = true
			return web.Respond(http.StatusForbidden), nil
		},
	})

	resp, err := runGuardRequest(t, sessionMW, guard(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}), nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !customCalled {
		t.Error("custom OnUnauthenticated should be called")
	}
	if resp.Status != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusForbidden)
	}
}
