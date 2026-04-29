package provider

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/cookiecodec"
	"github.com/go-sum/foundry/pkg/web/session"
)

func oauthCookieSessionConfig(t *testing.T) session.Config {
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

func TestAuthorizeParams_FitCookieSessionBudget(t *testing.T) {
	cfg := oauthCookieSessionConfig(t)
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, err := session.Middleware(cfg)(func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok {
			t.Fatal("session missing from context")
		}
		if err := sess.Set(authzParamsSessionKey, AuthorizeParams{
			ClientID:      "starter-first-party-client",
			RedirectURI:   "https://app.example.com/callback/oauth?source=cookie-budget&view=consent",
			Scopes:        []string{"openid", "email", "profile", "offline_access"},
			State:         strings.Repeat("s", 48),
			Nonce:         strings.Repeat("n", 48),
			CodeChallenge: strings.Repeat("c", 64),
		}); err != nil {
			t.Fatalf("sess.Set(%q): %v", authzParamsSessionKey, err)
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
	if got := len(setCookie); got >= 4096 {
		t.Fatalf("Set-Cookie header length = %d, want < 4096", got)
	}
	value := strings.SplitN(setCookie, ";", 2)[0]
	if got := len(value) - len("sess="); got >= 3800 {
		t.Fatalf("cookie value length = %d, want < 3800", got)
	}
}
