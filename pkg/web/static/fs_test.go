package static

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-sum/web"
)

// newFSContext creates a web.Context for testing with an optional URL path.
func newFSContext(method, urlPath string) *web.Context {
	u, _ := url.Parse(urlPath)
	req := web.NewRequest(method, u)
	return web.NewContext(context.Background(), req)
}

// testFS returns an in-memory fstest.MapFS with a set of test files.
func testFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data:    []byte("<h1>Home</h1>"),
			Mode:    0o644,
			ModTime: time.Now(),
		},
		"hello.txt": &fstest.MapFile{
			Data:    []byte("hello world"),
			Mode:    0o644,
			ModTime: time.Now(),
		},
		"app.js": &fstest.MapFile{
			Data:    []byte("console.log('hi');"),
			Mode:    0o644,
			ModTime: time.Now(),
		},
		"sub/index.html": &fstest.MapFile{
			Data:    []byte("<h1>Sub</h1>"),
			Mode:    0o644,
			ModTime: time.Now(),
		},
	}
}

func TestFSHandler_GET_ExistingFile(t *testing.T) {
	fsys := testFS()
	h := FSHandler(fsys, Options{Index: true})

	c := newFSContext(http.MethodGet, "/hello.txt")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if resp.Body == nil {
		t.Fatal("expected body")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "hello world" {
		t.Errorf("body = %q, want %q", string(body), "hello world")
	}
}

func TestFSHandler_GET_ContentType(t *testing.T) {
	fsys := testFS()
	h := FSHandler(fsys, Options{Index: true})

	c := newFSContext(http.MethodGet, "/app.js")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	ct := resp.Headers.Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Errorf("Content-Type = %q, want javascript content type", ct)
	}
}

func TestFSHandler_GET_IndexFile(t *testing.T) {
	fsys := testFS()
	h := FSHandler(fsys, Options{Index: true})

	c := newFSContext(http.MethodGet, "/")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "<h1>Home</h1>" {
		t.Errorf("body = %q, want %q", string(body), "<h1>Home</h1>")
	}
}

func TestFSHandler_GET_MissingFile_Returns404(t *testing.T) {
	fsys := testFS()
	h := FSHandler(fsys, Options{Index: true})

	c := newFSContext(http.MethodGet, "/nonexistent.txt")
	_, err := h(c)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T", err)
	}
	if webErr.Status != http.StatusNotFound {
		t.Errorf("status = %d, want %d", webErr.Status, http.StatusNotFound)
	}
}

func TestFSHandler_POST_MethodNotAllowed(t *testing.T) {
	fsys := testFS()
	h := FSHandler(fsys, Options{Index: true})

	c := newFSContext(http.MethodPost, "/hello.txt")
	_, err := h(c)
	if err == nil {
		t.Fatal("expected error for disallowed method, got nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T", err)
	}
	if webErr.Status != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", webErr.Status, http.StatusMethodNotAllowed)
	}
}

func TestFSHandler_HEAD_ExistingFile(t *testing.T) {
	fsys := testFS()
	h := FSHandler(fsys, Options{Index: true})

	c := newFSContext(http.MethodHead, "/hello.txt")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if resp.Body != nil {
		t.Error("expected nil body for HEAD request")
	}
	if got := resp.Headers.Get("Content-Length"); got == "" {
		t.Error("expected Content-Length header")
	}
}

func TestFSHandler_MissingFile_NotFoundHandler_Called(t *testing.T) {
	fsys := testFS()
	called := false
	notFound := func(c *web.Context) (web.Response, error) {
		called = true
		return web.Response{Status: http.StatusTeapot}, nil
	}
	h := FSHandler(fsys, Options{Index: true, NotFound: notFound})

	c := newFSContext(http.MethodGet, "/nonexistent.txt")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !called {
		t.Error("NotFound handler was not called on file miss")
	}
	if resp.Status != http.StatusTeapot {
		t.Errorf("status = %d, want %d", resp.Status, http.StatusTeapot)
	}
}

func TestFSHandler_FilterReject_NotFoundHandler_Called(t *testing.T) {
	fsys := testFS()
	called := false
	notFound := func(c *web.Context) (web.Response, error) {
		called = true
		return web.Response{Status: http.StatusTeapot}, nil
	}
	filter := func(path string) bool { return false }
	h := FSHandler(fsys, Options{Index: true, Filter: filter, NotFound: notFound})

	c := newFSContext(http.MethodGet, "/hello.txt")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !called {
		t.Error("NotFound handler was not called when Filter rejected file")
	}
	if resp.Status != http.StatusTeapot {
		t.Errorf("status = %d, want %d", resp.Status, http.StatusTeapot)
	}
}
