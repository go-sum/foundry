package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/health"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/site"
	"github.com/go-sum/foundry/pkg/web/static"
)

func testRouteContext(method, rawURL string) *web.Context {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	req := web.NewRequest(method, u)
	return web.NewContext(context.Background(), req)
}

func TestRegisterStaticRoutes_ServesFilesFromConfiguredPrefix(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "css"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "css", "app.css"), []byte("body{color:red}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	rt := router.New()
	err := registerStaticRoutes(rt, static.AssetsConfig{
		PublicDir: dir,
		URLPrefix: "/assets",
	})
	if err != nil {
		t.Fatalf("registerStaticRoutes() error = %v", err)
	}

	resp, err := rt.Serve(testRouteContext(http.MethodGet, "/assets/css/app.css"))
	if err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got, want := resp.Status, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got, want := string(body), "body{color:red}"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := resp.Headers.Get("Cache-Control"), "no-cache"; got != want {
		t.Fatalf("Cache-Control = %q, want %q", got, want)
	}
}

func TestRegisterRoutes_ReturnsErrorWhenStaticRootCannotBeOpened(t *testing.T) {
	rt := router.New()
	s := site.New(site.Config{BaseURL: "http://test.local"})
	err := RegisterRoutes(rt, DefaultRouteConfig(), Security{}, Services{}, static.AssetsConfig{
		PublicDir: filepath.Join(t.TempDir(), "missing"),
		URLPrefix: "/assets",
	}, t.TempDir(), s, Presentation{})
	if err == nil {
		t.Fatal("RegisterRoutes() error = nil, want non-nil")
	}
}

func TestRegisterRoutes_RegistersPublicAndStaticNamedRoutes(t *testing.T) {
	dir := t.TempDir()
	rt := router.New()
	s := site.New(site.Config{BaseURL: "http://test.local"})
	csrf, err := secure.NewCSRFConfigFromHex(testCSRFHexKey)
	if err != nil {
		t.Fatalf("secure.NewCSRFConfigFromHex() error = %v", err)
	}

	err = RegisterRoutes(rt, DefaultRouteConfig(), Security{CSRF: csrf, Origins: []string{"http://test.local"}}, Services{}, static.AssetsConfig{
		PublicDir: dir,
		URLPrefix: "/assets",
	}, dir, s, Presentation{})
	if err != nil {
		t.Fatalf("RegisterRoutes() error = %v", err)
	}

	cases := []struct {
		name   string
		params map[string]string
		want   string
	}{
		{name: "static.assets", params: map[string]string{"rest": "css/app.css"}, want: "/assets/css/app.css"},
		{name: "meta.robots", want: "/robots.txt"},
		{name: "meta.sitemap", want: "/sitemap.xml"},
		{name: "health.check", want: "/healthz"},
		{name: "home.show", want: "/"},
		{name: "docs.index", want: "/docs"},
		{name: "docs.show", params: map[string]string{"path": "guide/intro"}, want: "/docs/guide/intro"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reverseErr := rt.Reverse(tc.name, tc.params)
			if reverseErr != nil {
				t.Fatalf("Reverse(%q) error = %v", tc.name, reverseErr)
			}
			if got != tc.want {
				t.Fatalf("Reverse(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestHealthHandler_ReturnsOKWithNoCheckers(t *testing.T) {
	h := health.Handler()
	c := testRouteContext(http.MethodGet, "/healthz")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("health.Handler() error = %v", err)
	}
	if got, want := resp.Status, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
}

func TestHealthHandler_ReturnsUnavailableWhenCheckerFails(t *testing.T) {
	failing := health.CheckerFunc(func(_ context.Context) error { return errors.New("connection refused") })
	h := health.Handler(failing)
	c := testRouteContext(http.MethodGet, "/healthz")
	_, err := h(c)
	if err == nil {
		t.Fatal("health.Handler() error = nil, want non-nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("err type = %T, want *web.Error", err)
	}
	if got, want := webErr.Status, http.StatusServiceUnavailable; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
}

func TestUnavailableHandler_ReturnsUnavailableError(t *testing.T) {
	h := unavailableHandler("contact")
	c := testRouteContext(http.MethodGet, "/contact")
	_, err := h(c)
	if err == nil {
		t.Fatal("unavailableHandler() error = nil, want non-nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("err type = %T, want *web.Error", err)
	}
	if got, want := webErr.Status, http.StatusServiceUnavailable; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
}
