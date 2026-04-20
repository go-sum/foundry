package demos

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/componentry/showcase"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
)

func TestHandlerShow(t *testing.T) {
	t.Parallel()

	h := NewHandler(nil)

	u, _ := url.Parse("/demos/")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	vr := view.NewRequest(c)
	want := render.RenderNode(t, vr.Page("Component Showcase", showcase.Showcase()))
	if string(body) != want {
		t.Fatalf("full page body mismatch\ngot:  %s\nwant: %s", string(body), want)
	}
}

func TestHandlerShow_HTMXReturnsFullPage(t *testing.T) {
	// nil partial means HTMX requests receive the full page instead of a fragment.
	t.Parallel()

	h := NewHandler(nil)

	u, _ := url.Parse("/demos/")
	req := web.NewRequest(http.MethodGet, u)
	req.Headers.Set("HX-Request", "true")
	c := web.NewContext(context.Background(), req)
	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	vr := view.NewRequest(c)
	want := render.RenderNode(t, vr.Page("Component Showcase", showcase.Showcase()))
	if string(body) != want {
		t.Fatalf("HTMX body mismatch\ngot:  %s\nwant: %s", string(body), want)
	}
}
