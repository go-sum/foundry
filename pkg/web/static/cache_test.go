package static

import (
	"net/http"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
)

func TestVersionedCacheControl_versioned(t *testing.T) {
	inner := func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}
	h := VersionedCacheControl("v")(inner)

	c := newContext(http.MethodGet, "/static/css/app.css?v=abc123")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if got := resp.Headers.Get("Cache-Control"); got != immutableCacheControl {
		t.Errorf("Cache-Control = %q, want %q", got, immutableCacheControl)
	}
}

func TestVersionedCacheControl_unversioned(t *testing.T) {
	inner := func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}
	h := VersionedCacheControl("v")(inner)

	c := newContext(http.MethodGet, "/static/css/app.css")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if got := resp.Headers.Get("Cache-Control"); got != noCacheControl {
		t.Errorf("Cache-Control = %q, want %q", got, noCacheControl)
	}
}

func TestVersionedCacheControl_handlerValuePreserved(t *testing.T) {
	const handlerCC = "no-store"
	inner := func(c *web.Context) (web.Response, error) {
		resp := web.Respond(http.StatusOK)
		resp.Headers.Set("Cache-Control", handlerCC)
		return resp, nil
	}
	h := VersionedCacheControl("v")(inner)

	// Even with a version param, the handler's value must not be overwritten.
	c := newContext(http.MethodGet, "/static/css/app.css?v=abc123")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if got := resp.Headers.Get("Cache-Control"); got != handlerCC {
		t.Errorf("Cache-Control = %q, want %q", got, handlerCC)
	}
}

func TestVersionedCacheControl_emptyParam(t *testing.T) {
	inner := func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}
	h := VersionedCacheControl("v")(inner)

	c := newContext(http.MethodGet, "/static/css/app.css?v=")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if got := resp.Headers.Get("Cache-Control"); got != noCacheControl {
		t.Errorf("Cache-Control = %q, want %q", got, noCacheControl)
	}
}
