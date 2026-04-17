package htmx

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/web"
)

func TestVaryMiddleware_AppendsHXRequest(t *testing.T) {
	mw := VaryMiddleware()

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest("GET", &url.URL{Path: "/"})
	c := web.NewContext(context.Background(), req)
	resp, _ := handler(c)

	got := resp.Headers.Get("Vary")
	if got != "HX-Request" {
		t.Errorf("Vary = %q, want %q", got, "HX-Request")
	}
}

func TestVaryMiddleware_PreservesExistingVary(t *testing.T) {
	mw := VaryMiddleware()

	handler := mw(func(c *web.Context) (web.Response, error) {
		resp := web.Respond(http.StatusOK)
		resp.Headers.Set("Vary", "Accept")
		return resp, nil
	})

	req := web.NewRequest("GET", &url.URL{Path: "/"})
	c := web.NewContext(context.Background(), req)
	resp, _ := handler(c)

	values := resp.Headers.Values("Vary")
	foundAccept := false
	foundHXRequest := false
	for _, v := range values {
		if v == "Accept" {
			foundAccept = true
		}
		if v == "HX-Request" {
			foundHXRequest = true
		}
	}
	if !foundAccept {
		t.Errorf("Vary does not contain Accept; got %v", values)
	}
	if !foundHXRequest {
		t.Errorf("Vary does not contain HX-Request; got %v", values)
	}
}

func TestVaryMiddleware_CallsNext(t *testing.T) {
	mw := VaryMiddleware()

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest("GET", &url.URL{Path: "/"})
	c := web.NewContext(context.Background(), req)
	_, _ = handler(c)

	if !called {
		t.Error("VaryMiddleware did not call next handler")
	}
}
