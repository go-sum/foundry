package web

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

// fakeRenderer is a test double for Renderer.
type fakeRenderer struct {
	called bool
	name   string
	status int
	data   any
	err    error
}

func (f *fakeRenderer) Render(c *Context, status int, name string, data any) (Response, error) {
	f.called = true
	f.status = status
	f.name = name
	f.data = data
	if f.err != nil {
		return Response{}, f.err
	}
	return HTML(status, "<html>"+name+"</html>"), nil
}

func TestWithRenderer_StoresInContext(t *testing.T) {
	r := &fakeRenderer{}
	mw := WithRenderer(r)

	var capturedCtx *Context
	next := Handler(func(c *Context) (Response, error) {
		capturedCtx = c
		return Response{}, nil
	})

	c := NewContext(context.Background(), Request{})
	_, err := mw(next)(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := Get[Renderer](capturedCtx, rendererKey{})
	if !ok {
		t.Fatal("Renderer not found in context after WithRenderer middleware")
	}
	if got != r {
		t.Fatal("Renderer in context is not the one passed to WithRenderer")
	}
}

func TestRenderTemplate_CallsRenderer(t *testing.T) {
	r := &fakeRenderer{}
	c := NewContext(context.Background(), Request{})
	c.Set(rendererKey{}, r)

	resp, err := RenderTemplate(c, http.StatusOK, "my-template", map[string]string{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.called {
		t.Fatal("Renderer.Render was not called")
	}
	if r.name != "my-template" {
		t.Fatalf("template name = %q, want %q", r.name, "my-template")
	}
	if r.status != http.StatusOK {
		t.Fatalf("status = %d, want %d", r.status, http.StatusOK)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("response status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestRenderTemplate_NoRenderer_ReturnsErrInternal(t *testing.T) {
	c := NewContext(context.Background(), Request{})

	_, err := RenderTemplate(c, http.StatusOK, "my-template", nil)
	if err == nil {
		t.Fatal("expected error when no Renderer installed, got nil")
	}
	var webErr *Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", webErr.Status, http.StatusInternalServerError)
	}
}
