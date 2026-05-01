package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	configpkg "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/render"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/serve"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/go-sum/foundry/pkg/web/site"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
	"github.com/go-sum/foundry/pkg/web/viewstate/errorpage"
)

func testAppContext(method, rawURL string) *web.Context {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	req := web.NewRequest(method, u)
	return web.NewContext(context.Background(), req)
}

func TestProvideSecurity_BuildsOriginsAndSessionConfig(t *testing.T) {
	cfg := &configpkg.Config{
		Env:     configpkg.Testing,
		CSRF:    secure.DefaultCSRFConfig(),
		CSP:     secure.CSPNonceConfig{CSPTemplate: "default-src 'self'"},
		Headers: secure.DefaultHeadersConfig(),
		Session: session.Settings{
			CookieName:   "app-session",
			IdleTTL:      30 * time.Minute,
			AbsoluteTTL:  12 * time.Hour,
			CookieSecure: true,
		},
		SessionStore: "memory",
		Site: site.Config{
			BaseURL:         "https://app.example.com",
			OriginAllowlist: []string{"https://admin.example.com", "https://cdn.example.com"},
		},
	}

	sec, store, err := provideSecurity(context.Background(), Runtime{Config: cfg}, nil, nil)
	if err != nil {
		t.Fatalf("provideSecurity() error = %v", err)
	}
	if store == nil {
		t.Fatal("store = nil, want memory store")
	}
	if stoppable, ok := store.(interface{ Stop() }); ok {
		t.Cleanup(stoppable.Stop)
	}
	if got, want := sec.CSP.CSPTemplate, "default-src 'self'"; got != want {
		t.Fatalf("CSPTemplate = %q, want %q", got, want)
	}
	if got, want := sec.Origins, []string{
		"https://app.example.com",
		"https://admin.example.com",
		"https://cdn.example.com",
	}; !slices.Equal(got, want) {
		t.Fatalf("Origins = %#v, want %#v", got, want)
	}
	if got, want := sec.Session.CookieTemplate.Name, "app-session"; got != want {
		t.Fatalf("CookieTemplate.Name = %q, want %q", got, want)
	}
	if got, want := sec.Session.CookieTemplate.Path, "/"; got != want {
		t.Fatalf("CookieTemplate.Path = %q, want %q", got, want)
	}
	if !sec.Session.CookieTemplate.HTTPOnly {
		t.Fatal("CookieTemplate.HTTPOnly = false, want true")
	}
	if got, want := sec.Session.CookieTemplate.SameSite, "Lax"; got != want {
		t.Fatalf("CookieTemplate.SameSite = %q, want %q", got, want)
	}
	if !sec.Session.CookieTemplate.Secure {
		t.Fatal("CookieTemplate.Secure = false, want true")
	}
	if got, want := sec.Session.TTL, 12*time.Hour; got != want {
		t.Fatalf("TTL = %v, want %v", got, want)
	}
	if got, want := sec.Session.IdleTTL, 30*time.Minute; got != want {
		t.Fatalf("IdleTTL = %v, want %v", got, want)
	}
}

func TestProvideSecurity_EmptySessionStoreFailsFast(t *testing.T) {
	cfg := &configpkg.Config{
		Env:     configpkg.Testing,
		CSRF:    secure.DefaultCSRFConfig(),
		CSP:     secure.CSPNonceConfig{CSPTemplate: "default-src 'self'"},
		Headers: secure.DefaultHeadersConfig(),
		Session: session.Settings{
			CookieName:   "app-session",
			IdleTTL:      30 * time.Minute,
			AbsoluteTTL:  12 * time.Hour,
			CookieSecure: true,
		},
		SessionStore: "",
		Site: site.Config{
			BaseURL: "https://app.example.com",
		},
	}

	_, _, err := provideSecurity(context.Background(), Runtime{Config: cfg}, nil, nil)
	if err == nil {
		t.Fatal("provideSecurity() error = nil, want unsupported session store")
	}
	if !strings.Contains(err.Error(), `unsupported store type ""`) {
		t.Fatalf("provideSecurity() error = %v, want unsupported empty store", err)
	}
}

func TestProvideSecurity_InvalidTrustedProxyCIDR_ReturnsError(t *testing.T) {
	cfg := &configpkg.Config{
		Env:     configpkg.Testing,
		CSRF:    secure.DefaultCSRFConfig(),
		CSP:     secure.CSPNonceConfig{CSPTemplate: "default-src 'self'"},
		Headers: secure.DefaultHeadersConfig(),
		Session: session.Settings{
			CookieName:   "app-session",
			IdleTTL:      30 * time.Minute,
			AbsoluteTTL:  12 * time.Hour,
			CookieSecure: true,
		},
		SessionStore: "memory",
		Server: serve.ServerConfig{
			TrustedProxies: []string{"not-a-cidr"},
		},
		Site: site.Config{
			BaseURL: "https://app.example.com",
		},
	}

	_, _, err := provideSecurity(context.Background(), Runtime{Config: cfg}, nil, nil)
	if err == nil {
		t.Fatal("provideSecurity() error = nil, want invalid trusted proxy CIDR")
	}
	if !errors.Is(err, ErrTrustedProxyCIDRInvalid) {
		t.Fatalf("provideSecurity() error = %v, want ErrTrustedProxyCIDRInvalid", err)
	}
}

func TestProvideErrorBoundary_RendersFullAndPartialResponses(t *testing.T) {
	routing := router.New()
	router.Register(routing, router.GET("/", "home.show", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}))

	runtime := Runtime{
		Config: &configpkg.Config{Env: configpkg.Testing},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	handler := provideErrorBoundary(runtime, routing)(func(_ *web.Context) (web.Response, error) {
		return web.Response{}, web.ErrNotFound("missing page")
	})

	fullCtx := testAppContext(http.MethodGet, "/missing")
	fullCtx.Request.Headers.Set("Accept", "text/html")
	fullResp, fullErr := handler(fullCtx)
	if fullErr != nil {
		t.Fatalf("full handler error = %v", fullErr)
	}
	fullBody, err := io.ReadAll(fullResp.Body)
	if err != nil {
		t.Fatalf("ReadAll(full) error = %v", err)
	}
	if err := fullResp.Body.Close(); err != nil {
		t.Fatalf("full body Close() error = %v", err)
	}
	fullWant := render.RenderNode(t, errorpage.ErrorPage(viewstate.NewRequest(fullCtx), web.ErrNotFound("missing page")))
	if got := string(fullBody); got != fullWant {
		t.Fatalf("full body = %q, want %q", got, fullWant)
	}
	if got, want := fullResp.Status, http.StatusNotFound; got != want {
		t.Fatalf("full status = %d, want %d", got, want)
	}

	partialCtx := testAppContext(http.MethodGet, "/missing")
	partialCtx.Request.Headers.Set("HX-Request", "true")
	partialResp, partialErr := handler(partialCtx)
	if partialErr != nil {
		t.Fatalf("partial handler error = %v", partialErr)
	}
	partialBody, err := io.ReadAll(partialResp.Body)
	if err != nil {
		t.Fatalf("ReadAll(partial) error = %v", err)
	}
	if err := partialResp.Body.Close(); err != nil {
		t.Fatalf("partial body Close() error = %v", err)
	}
	partialWant := render.RenderNode(t, errorpage.ErrorContent(web.ErrNotFound("missing page")))
	if got := string(partialBody); got != partialWant {
		t.Fatalf("partial body = %q, want %q", got, partialWant)
	}
	if got, want := partialResp.Status, http.StatusNotFound; got != want {
		t.Fatalf("partial status = %d, want %d", got, want)
	}
}
