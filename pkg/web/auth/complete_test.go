package auth

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/go-sum/web"
	websession "github.com/go-sum/web/session"
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
	mw(func(c *web.Context) (web.Response, error) {
		sess, ok := websession.FromContext(c)
		if !ok {
			t.Fatal("session not found in context")
		}
		captured = sess
		return web.Respond(http.StatusOK), nil
	})(web.NewContext(context.Background(), req))

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
