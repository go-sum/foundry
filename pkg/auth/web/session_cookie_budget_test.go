package authweb

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/cookiecodec"
	"github.com/go-sum/foundry/pkg/web/session"
	authn "github.com/go-sum/foundry/pkg/web/authn"
	"github.com/google/uuid"
)

const (
	testSessionCookieValueBudget = 3800
	testSetCookieHeaderBudget    = 4096
)

func cookieSessionConfig(t *testing.T) session.Config {
	t.Helper()
	codec, err := cookiecodec.New(cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("32-byte-key-for-aead-encryption!")},
		Mode:    cookiecodec.AEAD,
	})
	if err != nil {
		t.Fatalf("cookiecodec.New: %v", err)
	}
	return session.Config{
		Store: session.NewCookieStore(codec),
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}
}

func writeCookieSession(t *testing.T, seed func(*session.Session) error) string {
	t.Helper()
	cfg := cookieSessionConfig(t)
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, err := session.Middleware(cfg)(func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok {
			t.Fatal("session missing from context")
		}
		if err := seed(sess); err != nil {
			t.Fatalf("seed session: %v", err)
		}
		return web.Respond(http.StatusOK), nil
	})(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("Middleware error = %v", err)
	}
	setCookie := resp.Headers.Get("Set-Cookie")
	if setCookie == "" {
		t.Fatal("Set-Cookie header missing")
	}
	return setCookie
}

func assertCookieBudget(t *testing.T, setCookie string) {
	t.Helper()
	if got := len(setCookie); got >= testSetCookieHeaderBudget {
		t.Fatalf("Set-Cookie header length = %d, want < %d", got, testSetCookieHeaderBudget)
	}
	value := strings.SplitN(setCookie, ";", 2)[0]
	if got := len(value) - len("sess="); got >= testSessionCookieValueBudget {
		t.Fatalf("cookie value length = %d, want < %d", got, testSessionCookieValueBudget)
	}
}

func TestPendingFlow_FitsCookieSessionBudget(t *testing.T) {
	setCookie := writeCookieSession(t, func(sess *session.Session) error {
		return setPendingFlow(sess, auth.PendingFlow{
			Purpose:     auth.FlowSignup,
			Email:       "casey.longname@example.com",
			DisplayName: "Casey Longname For Cookie Budget Coverage",
			Role:        auth.RoleUser,
			UserID:      uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Secret:      strings.Repeat("a", 64),
			IssuedAt:    time.Now().UTC(),
			ExpiresAt:   time.Now().UTC().Add(10 * time.Minute),
			Attempts:    4,
			ReturnTo:    "/settings/security?tab=signin&source=cookie-budget",
		})
	})
	assertCookieBudget(t, setCookie)
}

func TestAuthenticatedSession_FitsCookieSessionBudget(t *testing.T) {
	setCookie := writeCookieSession(t, func(sess *session.Session) error {
		return authn.SetAuth(sess,
			"22222222-2222-2222-2222-222222222222",
			"Jordan Example Authenticated User",
			true,
		)
	})
	assertCookieBudget(t, setCookie)
}

func TestPasskeyCeremony_FitsCookieSessionBudget(t *testing.T) {
	setCookie := writeCookieSession(t, func(sess *session.Session) error {
		allowed := make([][]byte, 0, 6)
		for i := range 6 {
			allowed = append(allowed, []byte(strings.Repeat(string(rune('a'+i)), 64)))
		}
		return setPasskeyCeremony(sess, passkeyCeremonyState{
			Operation: "register",
			Ceremony: auth.PasskeyCeremony{
				Challenge:            []byte(strings.Repeat("c", 32)),
				RelyingPartyID:       "example.com",
				UserID:               []byte(strings.Repeat("u", 64)),
				AllowedCredentialIDs: allowed,
				UserVerification:     "preferred",
				Mediation:            "conditional",
				Extensions:           map[string]any{"credProps": true},
				Expires:              time.Now().UTC().Add(5 * time.Minute),
				CredentialParameters: []auth.PasskeyCredentialParameter{
					{Type: "public-key", Algorithm: -7},
					{Type: "public-key", Algorithm: -257},
				},
			},
			UserID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		})
	})
	assertCookieBudget(t, setCookie)
}
