package static

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
)

// newContext creates a web.Context for testing.
func newContext(method, path string) *web.Context {
	u, _ := url.Parse(path)
	req := web.NewRequest(method, u)
	return web.NewContext(context.Background(), req)
}

// setupRoot creates a temp directory with a few test files and returns an *os.Root.
func setupRoot(t *testing.T) *os.Root {
	t.Helper()
	tmpDir := t.TempDir()

	// Create index.html at root
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<h1>Index</h1>"), 0o644); err != nil {
		t.Fatalf("WriteFile index.html: %v", err)
	}

	// Create a regular file
	if err := os.WriteFile(filepath.Join(tmpDir, "hello.txt"), []byte("hello world"), 0o644); err != nil {
		t.Fatalf("WriteFile hello.txt: %v", err)
	}

	// Create a subdirectory with an index
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("MkdirAll sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "index.html"), []byte("<h1>Sub</h1>"), 0o644); err != nil {
		t.Fatalf("WriteFile sub/index.html: %v", err)
	}

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	t.Cleanup(func() { root.Close() }) //nolint:errcheck
	return root
}

func TestStatic_GET_ExistingFile(t *testing.T) {
	root := setupRoot(t)
	h := Handler(root, Options{Index: true})

	c := newContext(http.MethodGet, "/hello.txt")
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

func TestStatic_GET_DirectoryWithIndex(t *testing.T) {
	root := setupRoot(t)
	h := Handler(root, Options{Index: true})

	c := newContext(http.MethodGet, "/")
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
	if string(body) != "<h1>Index</h1>" {
		t.Errorf("body = %q, want %q", string(body), "<h1>Index</h1>")
	}
}

func TestStatic_GET_MissingFile(t *testing.T) {
	root := setupRoot(t)
	h := Handler(root, Options{Index: true})

	c := newContext(http.MethodGet, "/nonexistent.txt")
	_, err := h(c)

	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T", err)
	}
	if webErr.Status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", webErr.Status, http.StatusNotFound)
	}
}

func TestStatic_HEAD_ExistingFile(t *testing.T) {
	root := setupRoot(t)
	h := Handler(root, Options{Index: true})

	c := newContext(http.MethodHead, "/hello.txt")
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

func TestStatic_MissingFile_NotFoundHandler_Called(t *testing.T) {
	root := setupRoot(t)
	called := false
	notFound := func(c *web.Context) (web.Response, error) {
		called = true
		return web.Response{Status: http.StatusTeapot}, nil
	}
	h := Handler(root, Options{Index: true, NotFound: notFound})

	c := newContext(http.MethodGet, "/nonexistent.txt")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if !called {
		t.Error("NotFound handler was not called on file miss")
	}
	if resp.Status != http.StatusTeapot {
		t.Errorf("status = %d, want %d (from NotFound handler)", resp.Status, http.StatusTeapot)
	}
}

func TestStatic_FilterReject_NotFoundHandler_Called(t *testing.T) {
	root := setupRoot(t)
	called := false
	notFound := func(c *web.Context) (web.Response, error) {
		called = true
		return web.Response{Status: http.StatusTeapot}, nil
	}
	filter := func(path string) bool { return false } // reject everything
	h := Handler(root, Options{Index: true, Filter: filter, NotFound: notFound})

	c := newContext(http.MethodGet, "/hello.txt")
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

func TestStatic_MissingFile_NoNotFoundHandler_Returns404(t *testing.T) {
	root := setupRoot(t)
	h := Handler(root, Options{Index: true}) // no NotFound set

	c := newContext(http.MethodGet, "/nonexistent.txt")
	_, err := h(c)

	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T", err)
	}
	if webErr.Status != http.StatusNotFound {
		t.Errorf("status = %d, want %d (default 404)", webErr.Status, http.StatusNotFound)
	}
}

func TestStatic_POST_MethodNotAllowed(t *testing.T) {
	root := setupRoot(t)
	h := Handler(root, Options{Index: true})

	c := newContext(http.MethodPost, "/hello.txt")
	_, err := h(c)

	if err == nil {
		t.Fatal("expected error for disallowed method, got nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T", err)
	}
	if webErr.Status != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", webErr.Status, http.StatusMethodNotAllowed)
	}
	if webErr.Code != web.CodeMethodNotAllowed {
		t.Errorf("Code = %q, want %q", webErr.Code, web.CodeMethodNotAllowed)
	}
}

// setupRootWithSidecars creates a temp directory with a JS file and its
// pre-compressed sidecars (.gz and .br), returning an *os.Root.
func setupRootWithSidecars(t *testing.T) *os.Root {
	t.Helper()
	tmpDir := t.TempDir()

	original := []byte("console.log('hello');")
	gzipped := []byte("gzip-compressed-bytes")
	brotli := []byte("br-compressed-bytes")

	if err := os.WriteFile(filepath.Join(tmpDir, "app.js"), original, 0o644); err != nil {
		t.Fatalf("WriteFile app.js: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "app.js.gz"), gzipped, 0o644); err != nil {
		t.Fatalf("WriteFile app.js.gz: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "app.js.br"), brotli, 0o644); err != nil {
		t.Fatalf("WriteFile app.js.br: %v", err)
	}

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	t.Cleanup(func() { root.Close() }) //nolint:errcheck
	return root
}

func TestStatic_Precompressed_GzipSidecar(t *testing.T) {
	root := setupRootWithSidecars(t)
	h := Handler(root, Options{Index: true, Precompressed: true})

	c := newContext(http.MethodGet, "/app.js")
	c.Request.Headers.Set("Accept-Encoding", "gzip")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if got := resp.Headers.Get("Content-Encoding"); got != "gzip" {
		t.Errorf("Content-Encoding = %q, want %q", got, "gzip")
	}
	if got := resp.Headers.Get("Content-Type"); got != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", got, "text/javascript; charset=utf-8")
	}
	if got := resp.Headers.Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary = %q, want %q", got, "Accept-Encoding")
	}
	if resp.Body == nil {
		t.Fatal("expected body")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "gzip-compressed-bytes" {
		t.Errorf("body = %q, want %q", string(body), "gzip-compressed-bytes")
	}
}

func TestStatic_Precompressed_BrotliSidecar(t *testing.T) {
	root := setupRootWithSidecars(t)
	h := Handler(root, Options{Index: true, Precompressed: true})

	c := newContext(http.MethodGet, "/app.js")
	c.Request.Headers.Set("Accept-Encoding", "br")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if got := resp.Headers.Get("Content-Encoding"); got != "br" {
		t.Errorf("Content-Encoding = %q, want %q", got, "br")
	}
	if got := resp.Headers.Get("Content-Type"); got != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", got, "text/javascript; charset=utf-8")
	}
	if got := resp.Headers.Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary = %q, want %q", got, "Accept-Encoding")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "br-compressed-bytes" {
		t.Errorf("body = %q, want %q", string(body), "br-compressed-bytes")
	}
}

func TestStatic_Precompressed_NoSidecar_FallsThrough(t *testing.T) {
	root := setupRoot(t) // no .gz/.br files
	h := Handler(root, Options{Index: true, Precompressed: true})

	c := newContext(http.MethodGet, "/hello.txt")
	c.Request.Headers.Set("Accept-Encoding", "gzip")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if got := resp.Headers.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty (original file served)", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "hello world" {
		t.Errorf("body = %q, want %q", string(body), "hello world")
	}
}

func TestStatic_Precompressed_Disabled_SidecarIgnored(t *testing.T) {
	root := setupRootWithSidecars(t)
	h := Handler(root, Options{Index: true, Precompressed: false})

	c := newContext(http.MethodGet, "/app.js")
	c.Request.Headers.Set("Accept-Encoding", "gzip")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if got := resp.Headers.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty (Precompressed=false)", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "console.log('hello');" {
		t.Errorf("body = %q, want original JS content", string(body))
	}
}

func TestStatic_Precompressed_IdentityEncoding_NoSidecar(t *testing.T) {
	root := setupRootWithSidecars(t)
	h := Handler(root, Options{Index: true, Precompressed: true})

	c := newContext(http.MethodGet, "/app.js")
	c.Request.Headers.Set("Accept-Encoding", "identity")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if got := resp.Headers.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty (identity encoding)", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "console.log('hello');" {
		t.Errorf("body = %q, want original JS content", string(body))
	}
}
