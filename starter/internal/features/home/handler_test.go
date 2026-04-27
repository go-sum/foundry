package home

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/page"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/render"
)

func TestHandlerShow_NoCheckers(t *testing.T) {
	h := NewHandler(nil)

	u, _ := url.Parse("/")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	vr := view.NewRequest(c)
	want := render.RenderNode(t, page.HomePage(vr, nil))
	if string(body) != want {
		t.Fatalf("body mismatch\ngot:  %s\nwant: %s", string(body), want)
	}

	// HTMX partial mode
	req.Headers.Set("HX-Request", "true")
	c = web.NewContext(context.Background(), req)
	resp, err = h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	want = render.RenderNode(t, page.HomeContent(vr, nil))
	if string(body) != want {
		t.Fatalf("partial body mismatch\ngot:  %s\nwant: %s", string(body), want)
	}
}

func TestHandlerShow_HealthyChecker(t *testing.T) {
	called := false
	checkers := []Checker{{
		Name: "Database",
		Fn: func(_ context.Context) error {
			called = true
			return nil
		},
	}}
	h := NewHandler(checkers)

	u, _ := url.Parse("/")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("health checker was not called")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	vr := view.NewRequest(c)
	want := render.RenderNode(t, page.HomePage(vr, []page.ServiceStatus{{Name: "Database", Healthy: true}}))
	if string(body) != want {
		t.Fatalf("body mismatch\ngot:  %s\nwant: %s", string(body), want)
	}
}

func TestHandlerShow_UnhealthyChecker(t *testing.T) {
	checkers := []Checker{{
		Name: "Database",
		Fn:   func(_ context.Context) error { return context.DeadlineExceeded },
	}}
	h := NewHandler(checkers)

	u, _ := url.Parse("/")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	vr := view.NewRequest(c)
	want := render.RenderNode(t, page.HomePage(vr, []page.ServiceStatus{{Name: "Database", Healthy: false}}))
	if string(body) != want {
		t.Fatalf("body mismatch\ngot:  %s\nwant: %s", string(body), want)
	}
}
