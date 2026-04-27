package view

import (
	"io"
	"net/http"
	"testing"

	"github.com/go-sum/foundry/pkg/web/render"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func TestRender(t *testing.T) {
	full := h.HTML(h.Body(g.Text("full")))
	partial := h.Div(g.Text("partial"))

	t.Run("full page when not partial", func(t *testing.T) {
		req := Request{HTMX: HTMXRequest{Enabled: false}}
		resp, err := Render(req, full, partial)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck

		if resp.Status != http.StatusOK {
			t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
		}
		want := render.RenderNode(t, full)
		if string(body) != want {
			t.Errorf("full page body = %q, want %q", string(body), want)
		}
	})

	t.Run("partial when htmx request", func(t *testing.T) {
		req := Request{HTMX: HTMXRequest{Enabled: true}}
		resp, err := Render(req, full, partial)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck

		want := render.RenderNode(t, partial)
		if string(body) != want {
			t.Errorf("partial body = %q, want %q", string(body), want)
		}
	})

	t.Run("full when partial is nil", func(t *testing.T) {
		req := Request{HTMX: HTMXRequest{Enabled: true}}
		resp, err := Render(req, full, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck

		want := render.RenderNode(t, full)
		if string(body) != want {
			t.Errorf("full page (nil partial) body = %q, want %q", string(body), want)
		}
	})
}

func TestRenderWithStatus(t *testing.T) {
	full := h.Div(g.Text("full"))
	partial := h.Span(g.Text("partial"))

	req := Request{HTMX: HTMXRequest{Enabled: false}}
	resp := RenderWithStatus(req, http.StatusCreated, full, partial)

	if resp.Status != http.StatusCreated {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusCreated)
	}
}

func TestRenderWithStatus_Partial(t *testing.T) {
	full := h.Div(g.Text("full"))
	partial := h.Span(g.Text("partial"))

	req := Request{HTMX: HTMXRequest{Enabled: true}}
	resp := RenderWithStatus(req, http.StatusCreated, full, partial)

	if resp.Status != http.StatusCreated {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusCreated)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close() //nolint:errcheck

	want := render.RenderNode(t, partial)
	if string(body) != want {
		t.Errorf("partial body = %q, want %q", string(body), want)
	}
}
