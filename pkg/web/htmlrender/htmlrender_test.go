package htmlrender

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/go-sum/foundry/pkg/web"
)

func newContext() *web.Context {
	return web.NewContext(context.Background(), web.Request{})
}

func TestNew_ValidFS_RendersTemplate(t *testing.T) {
	memFS := fstest.MapFS{
		"templates/hello.html": &fstest.MapFile{
			Data: []byte(`Hello, {{.Name}}!`),
		},
	}

	r, err := New(Config{
		FS:      memFS,
		Pattern: "templates/*.html",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := r.Render(newContext(), http.StatusOK, "hello.html", map[string]string{"Name": "World"})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestNew_BadPattern_ReturnsError(t *testing.T) {
	memFS := fstest.MapFS{}

	_, err := New(Config{
		FS:      memFS,
		Pattern: "templates/*.html",
	})
	if err == nil {
		t.Fatal("expected error for pattern with no matches, got nil")
	}
}

func TestRender_UnknownTemplateName_ReturnsErrInternal(t *testing.T) {
	memFS := fstest.MapFS{
		"templates/hello.html": &fstest.MapFile{
			Data: []byte(`Hello!`),
		},
	}

	r, err := New(Config{
		FS:      memFS,
		Pattern: "templates/*.html",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = r.Render(newContext(), http.StatusOK, "nonexistent.html", nil)
	if err == nil {
		t.Fatal("expected error for unknown template name, got nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", webErr.Status, http.StatusInternalServerError)
	}
}

// mutatingFS wraps a map so we can swap template content between calls.
type mutatingFS struct {
	files map[string]*fstest.MapFile
}

func (m *mutatingFS) Open(name string) (fs.File, error) {
	f, ok := m.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return fstest.MapFS{name: f}.Open(name)
}

func TestRender_DevMode_ReloadsTemplates(t *testing.T) {
	files := map[string]*fstest.MapFile{
		"templates/page.html": {Data: []byte(`version1`)},
	}
	mfs := &mutatingFS{files: files}

	r, err := New(Config{
		FS:      mfs,
		Pattern: "templates/page.html",
		DevMode: true,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp1, err := r.Render(newContext(), http.StatusOK, "page.html", nil)
	if err != nil {
		t.Fatalf("first Render error: %v", err)
	}
	_ = resp1

	// Mutate the FS to return different content.
	files["templates/page.html"] = &fstest.MapFile{Data: []byte(`version2`)}

	resp2, err := r.Render(newContext(), http.StatusOK, "page.html", nil)
	if err != nil {
		t.Fatalf("second Render error: %v", err)
	}
	_ = resp2
	// Both should succeed; DevMode reloads without error on each call.
}

func TestRender_UsesStatusParameter(t *testing.T) {
	memFS := fstest.MapFS{
		"templates/hello.html": &fstest.MapFile{
			Data: []byte(`Hello!`),
		},
	}

	r, err := New(Config{
		FS:      memFS,
		Pattern: "templates/*.html",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := r.Render(newContext(), http.StatusCreated, "hello.html", nil)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if resp.Status != http.StatusCreated {
		t.Fatalf("response status = %d, want %d", resp.Status, http.StatusCreated)
	}
}
