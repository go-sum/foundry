package view

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
)

func TestNewHTMXRequest(t *testing.T) {
	h := web.NewHeaders()
	h.Set("HX-Request", "true")
	h.Set("HX-Boosted", "true")

	got := NewHTMXRequest(h)
	if !got.Enabled || !got.Boosted {
		t.Fatalf("got %+v", got)
	}
}

func TestRequestIsPartial(t *testing.T) {
	req := Request{HTMX: HTMXRequest{Enabled: true}}
	if !req.IsPartial() {
		t.Fatal("expected partial")
	}
	req = Request{HTMX: HTMXRequest{Enabled: true, Boosted: true}}
	if req.IsPartial() {
		t.Fatal("boosted requests should not be partial")
	}
}

func TestNewRequest(t *testing.T) {
	u, _ := url.Parse("/hello/World")
	req := web.NewRequest(http.MethodGet, u)
	req.Headers.Set("HX-Request", "true")
	c := web.NewContext(context.Background(), req)

	routes := []router.Route{{Method: "GET", Pattern: "/", Name: "home.show"}}
	vr := NewRequest(c, routes)

	if vr.CurrentPath != "/hello/World" {
		t.Fatalf("CurrentPath = %q", vr.CurrentPath)
	}
	if !vr.HTMX.Enabled {
		t.Fatal("HTMX.Enabled should be true")
	}
	if len(vr.Routes) != 1 {
		t.Fatalf("Routes len = %d, want 1", len(vr.Routes))
	}
}
