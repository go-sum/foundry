package view

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/componentry/icons"
	"github.com/go-sum/foundry/internal/view/layout"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
	"github.com/go-sum/web/session"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func TestNewHTMXRequest(t *testing.T) {
	h := web.NewHeaders()
	h.Set("HX-Request", "true")
	h.Set("HX-Boosted", "true")
	h.Set("HX-Trigger", "save")
	h.Set("HX-Target", "#content")
	h.Set("HX-Trigger-Name", "submit")
	h.Set("HX-Current-URL", "http://test.local/form")

	got := NewHTMXRequest(h)
	want := HTMXRequest{
		Enabled:     true,
		Boosted:     true,
		Trigger:     "save",
		Target:      "#content",
		TriggerName: "submit",
		CurrentURL:  "http://test.local/form",
	}
	if got != want {
		t.Fatalf("NewHTMXRequest() = %+v, want %+v", got, want)
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

func TestRequestOptions(t *testing.T) {
	got := Request{}
	for _, opt := range []RequestOption{
		WithCSRFToken("csrf-token"),
		WithRequestID("req-123"),
		WithNonce("nonce-123"),
		WithFlash("Saved", "Sent"),
	} {
		opt(&got)
	}

	want := Request{
		CSRFToken: "csrf-token",
		RequestID: "req-123",
		Nonce:     "nonce-123",
		Flash:     []string{"Saved", "Sent"},
	}
	if got.CSRFToken != want.CSRFToken || got.RequestID != want.RequestID || got.Nonce != want.Nonce {
		t.Fatalf("Request options result = %+v, want %+v", got, want)
	}
	if len(got.Flash) != 2 || got.Flash[0] != "Saved" || got.Flash[1] != "Sent" {
		t.Fatalf("Flash = %#v, want %#v", got.Flash, want.Flash)
	}
}

func TestWithIconRegistry(t *testing.T) {
	r := icons.NewRegistry()
	req := Request{}
	WithIconRegistry(r)(&req)
	if req.Icons != r {
		t.Fatalf("WithIconRegistry: got %v, want %v", req.Icons, r)
	}
}

func TestWithIconRegistry_nil(t *testing.T) {
	req := Request{}
	WithIconRegistry(nil)(&req)
	if req.Icons != nil {
		t.Fatalf("WithIconRegistry nil: got non-nil Icons")
	}
}

func TestNewRequest(t *testing.T) {
	u, _ := url.Parse("/hello/World")
	req := web.NewRequest(http.MethodGet, u)
	req.Headers.Set("HX-Request", "true")
	req.Headers.Set("HX-Trigger", "load")
	c := web.NewContext(context.Background(), req)
	c.Set(web.RequestIDKey{}, "req-123")

	vr := NewRequest(
		c,
		WithRequestID("override-req"),
		WithCSRFToken("csrf-123"),
		WithNonce("nonce-123"),
		WithFlash("Saved"),
	)

	if vr.CurrentPath != "/hello/World" {
		t.Fatalf("CurrentPath = %q", vr.CurrentPath)
	}
	if !vr.HTMX.Enabled {
		t.Fatal("HTMX.Enabled should be true")
	}
	if vr.HTMX.Trigger != "load" {
		t.Fatalf("HTMX.Trigger = %q, want %q", vr.HTMX.Trigger, "load")
	}
	if vr.RequestID != "override-req" {
		t.Fatalf("RequestID = %q, want %q", vr.RequestID, "override-req")
	}
	if vr.CSRFToken != "csrf-123" {
		t.Fatalf("CSRFToken = %q, want %q", vr.CSRFToken, "csrf-123")
	}
	if vr.Nonce != "nonce-123" {
		t.Fatalf("Nonce = %q, want %q", vr.Nonce, "nonce-123")
	}
	if len(vr.Flash) != 1 || vr.Flash[0] != "Saved" {
		t.Fatalf("Flash = %#v, want %#v", vr.Flash, []string{"Saved"})
	}
}

func TestNewRequest_PopsFlashFromSession(t *testing.T) {
	store := session.NewMemoryStore()
	t.Cleanup(store.Stop)

	cfg := session.NewConfig(session.Settings{
		CookieName:   "session",
		IdleTTL:      time.Minute,
		AbsoluteTTL:  time.Hour,
		CookieSecure: false,
	}, store)

	u, _ := url.Parse("/hello")
	writeFlash := session.Middleware(cfg)(func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok {
			t.Fatal("session missing from context")
		}
		if err := sess.Flash("flash", []string{"Saved", "Sent"}); err != nil {
			t.Fatalf("Flash() error = %v", err)
		}
		return web.Text(http.StatusOK, "ok"), nil
	})

	req1 := web.NewRequest(http.MethodGet, u)
	resp1, err := writeFlash(web.NewContext(context.Background(), req1))
	if err != nil {
		t.Fatalf("writeFlash() error = %v", err)
	}
	if err := resp1.Body.Close(); err != nil {
		t.Fatalf("resp1 Close() error = %v", err)
	}

	cookieHeader := resp1.Headers.Get("Set-Cookie")
	if cookieHeader == "" {
		t.Fatal("Set-Cookie header = empty, want session cookie")
	}

	var captured Request
	readFlash := session.Middleware(cfg)(func(c *web.Context) (web.Response, error) {
		captured = NewRequest(c)
		return web.Text(http.StatusOK, "ok"), nil
	})

	req2 := web.NewRequest(http.MethodGet, u)
	req2.Headers.Set("Cookie", strings.Split(cookieHeader, ";")[0])
	resp2, err := readFlash(web.NewContext(context.Background(), req2))
	if err != nil {
		t.Fatalf("readFlash() error = %v", err)
	}
	if _, err := io.ReadAll(resp2.Body); err != nil {
		t.Fatalf("ReadAll(resp2) error = %v", err)
	}
	if err := resp2.Body.Close(); err != nil {
		t.Fatalf("resp2 Close() error = %v", err)
	}

	if len(captured.Flash) != 2 || captured.Flash[0] != "Saved" || captured.Flash[1] != "Sent" {
		t.Fatalf("Flash = %#v, want %#v", captured.Flash, []string{"Saved", "Sent"})
	}
}

func TestRequestPage(t *testing.T) {
	req := Request{
		CSRFToken: "csrf-123",
		Nonce:     "nonce-123",
		Flash:     []string{"Saved"},
	}

	got := render.RenderNode(t, req.Page("Greeting", h.Div(g.Text("hello"))))
	want := render.RenderNode(t, layout.Page(layout.Props{
		Title:     "Greeting",
		Nonce:     "nonce-123",
		CSRFToken: "csrf-123",
		Flash:     []string{"Saved"},
		Children:  []g.Node{h.Div(g.Text("hello"))},
	}))
	if got != want {
		t.Fatalf("Page() = %q, want %q", got, want)
	}
}
