package auth

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web"
	websession "github.com/go-sum/foundry/pkg/web/session"
)

// sessionFromMiddleware creates a real *session.Session by running a request
// through session middleware and capturing the session from the context.
func sessionFromMiddleware(t *testing.T) *websession.Session {
	t.Helper()

	store := websession.NewMemoryStore()
	t.Cleanup(store.Stop)

	cfg := websession.Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}
	mw := websession.Middleware(cfg)

	var captured *websession.Session
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	if _, err := mw(func(c *web.Context) (web.Response, error) {
		sess, ok := websession.FromContext(c)
		if !ok {
			t.Fatal("session not found in context")
		}
		captured = sess
		return web.Respond(http.StatusOK), nil
	})(web.NewContext(context.Background(), req)); err != nil {
		t.Fatalf("middleware error: %v", err)
	}

	return captured
}

func TestCompleteAuth_ReturnsReturnTo(t *testing.T) {
	sess := sessionFromMiddleware(t)
	tx := OAuthTransaction{
		State:    "some-state",
		Nonce:    "some-nonce",
		Verifier: "some-verifier",
		ReturnTo: "/dashboard",
	}

	returnTo, err := CompleteAuth(context.Background(), sess, tx)
	if err != nil {
		t.Fatalf("CompleteAuth error: %v", err)
	}
	if returnTo != "/dashboard" {
		t.Errorf("returnTo = %q, want %q", returnTo, "/dashboard")
	}
}

func TestCompleteAuth_RegeneratesSession(t *testing.T) {
	sess := sessionFromMiddleware(t)
	tx := OAuthTransaction{
		State:    "some-state",
		Nonce:    "some-nonce",
		Verifier: "some-verifier",
		ReturnTo: "/",
	}

	_, err := CompleteAuth(context.Background(), sess, tx)
	if err != nil {
		t.Fatalf("CompleteAuth error: %v", err)
	}
	// After Regenerate(), the token is cleared (set to "").
	// ID() returns the current token, which is now empty.
	if sess.ID() != "" {
		t.Errorf("sess.ID() = %q after CompleteAuth, want empty (Regenerate clears token)", sess.ID())
	}
}

func TestCompleteAuth_UnsetsTransaction(t *testing.T) {
	sess := sessionFromMiddleware(t)
	if err := sess.Set(SessionKey, "test-value"); err != nil {
		t.Fatalf("sess.Set error: %v", err)
	}
	if !sess.Has(SessionKey) {
		t.Fatal("expected session to have SessionKey before CompleteAuth")
	}

	tx := OAuthTransaction{
		State:    "some-state",
		Nonce:    "some-nonce",
		Verifier: "some-verifier",
		ReturnTo: "/dashboard",
	}
	if _, err := CompleteAuth(context.Background(), sess, tx); err != nil {
		t.Fatalf("CompleteAuth error: %v", err)
	}

	if sess.Has(SessionKey) {
		t.Error("expected SessionKey to be unset after CompleteAuth")
	}
}

func TestCompleteAuth_ExpiredTransaction(t *testing.T) {
	sess := sessionFromMiddleware(t)
	originalID := sess.ID()

	tx := OAuthTransaction{
		State:     "some-state",
		Nonce:     "some-nonce",
		Verifier:  "some-verifier",
		ReturnTo:  "/dashboard",
		CreatedAt: time.Now().UTC().Add(-11 * time.Minute),
	}

	_, err := CompleteAuth(context.Background(), sess, tx)
	if !errors.Is(err, ErrTransactionExpired) {
		t.Fatalf("CompleteAuth expired: got %v, want ErrTransactionExpired", err)
	}
	// Session must NOT have been regenerated.
	if sess.ID() != originalID {
		t.Errorf("session ID changed on expired transaction; Regenerate must not be called")
	}
}

func TestCompleteAuth_ZeroCreatedAt(t *testing.T) {
	sess := sessionFromMiddleware(t)
	tx := OAuthTransaction{
		State:    "some-state",
		Nonce:    "some-nonce",
		Verifier: "some-verifier",
		ReturnTo: "/home",
		// CreatedAt is zero — TTL check must be skipped.
	}

	returnTo, err := CompleteAuth(context.Background(), sess, tx)
	if err != nil {
		t.Fatalf("CompleteAuth zero CreatedAt error: %v", err)
	}
	if returnTo != "/home" {
		t.Errorf("returnTo = %q, want %q", returnTo, "/home")
	}
}
