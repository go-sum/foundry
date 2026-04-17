package render

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/go-sum/web"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func TestComponent(t *testing.T) {
	node := h.Div(g.Text("hello"))
	resp, err := Component(node)
	if err != nil {
		t.Fatalf("Component error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
	}
	if ct := resp.Headers.Get("Content-Type"); ct != "text/html; charset=UTF-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=UTF-8", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	resp.Body.Close()

	want := "<div>hello</div>"
	if string(body) != want {
		t.Errorf("body = %q, want %q", string(body), want)
	}
}

func TestComponentWithStatus(t *testing.T) {
	node := h.P(g.Text("error"))
	resp, err := ComponentWithStatus(http.StatusBadRequest, node)
	if err != nil {
		t.Fatalf("ComponentWithStatus error: %v", err)
	}

	if resp.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusBadRequest)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	resp.Body.Close()

	want := "<p>error</p>"
	if string(body) != want {
		t.Errorf("body = %q, want %q", string(body), want)
	}
}

func TestFragment(t *testing.T) {
	node := h.Span(g.Text("frag"))
	resp, err := Fragment(node)
	if err != nil {
		t.Fatalf("Fragment error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	resp.Body.Close()

	want := "<span>frag</span>"
	if string(body) != want {
		t.Errorf("body = %q, want %q", string(body), want)
	}
}

func TestComponentWithStatus_RenderError(t *testing.T) {
	// errNode is a gomponents node whose Render always returns an error.
	errNode := g.NodeFunc(func(w io.Writer) error {
		return errors.New("intentional render failure")
	})

	resp, err := ComponentWithStatus(http.StatusOK, errNode)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusInternalServerError {
		t.Errorf("Error.Status = %d, want %d", webErr.Status, http.StatusInternalServerError)
	}
	if resp.Status != 0 {
		t.Errorf("Response.Status = %d, want 0 (zero value)", resp.Status)
	}
	// No cause text in the error's public message.
	if webErr.Message != "" {
		t.Errorf("Error.Message = %q, want empty (no cause leak)", webErr.Message)
	}
}

func TestRenderNode(t *testing.T) {
	node := h.Div(h.Class("test"), g.Text("content"))
	got := RenderNode(t, node)
	want := `<div class="test">content</div>`
	if got != want {
		t.Errorf("RenderNode = %q, want %q", got, want)
	}
}
