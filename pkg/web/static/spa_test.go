package static

import (
	"errors"
	"io"
	"net/http"
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-sum/foundry/pkg/web"
)

// spaFS returns a minimal in-memory fstest.MapFS for SPA tests.
func spaFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data:    []byte("<html>SPA</html>"),
			Mode:    0o644,
			ModTime: time.Now(),
		},
	}
}

func TestSPAFallbackFS_UnmatchedPath_ServesIndex(t *testing.T) {
	fsys := spaFS()
	fallback := SPAFallbackFS(fsys, "index.html")

	// Request for an unknown path.
	c := newFSContext(http.MethodGet, "/some/deep/route")
	resp, err := fallback(c)
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
	if string(body) != "<html>SPA</html>" {
		t.Errorf("body = %q, want %q", string(body), "<html>SPA</html>")
	}
}

func TestSPAFallbackFS_MissingIndex_Returns404(t *testing.T) {
	// Empty FS — no index.html.
	fsys := fstest.MapFS{}
	fallback := SPAFallbackFS(fsys, "index.html")

	c := newFSContext(http.MethodGet, "/anything")
	_, err := fallback(c)
	if err == nil {
		t.Fatal("expected error for missing index file, got nil")
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T", err)
	}
	if webErr.Status != http.StatusNotFound {
		t.Errorf("status = %d, want %d", webErr.Status, http.StatusNotFound)
	}
}

func TestSPAFallbackFS_AsNotFoundOption(t *testing.T) {
	// The fallback is used as Options.NotFound on an FSHandler.
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data:    []byte("<html>SPA</html>"),
			Mode:    0o644,
			ModTime: time.Now(),
		},
		"static/app.js": &fstest.MapFile{
			Data:    []byte("console.log('hi');"),
			Mode:    0o644,
			ModTime: time.Now(),
		},
	}

	h := FSHandler(fsys, Options{
		Index:    true,
		NotFound: SPAFallbackFS(fsys, "index.html"),
	})

	// Known static file is served normally.
	c := newFSContext(http.MethodGet, "/static/app.js")
	resp, err := h(c)
	if err != nil {
		t.Fatalf("handler error for known file: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status for known file = %d, want %d", resp.Status, http.StatusOK)
	}

	// Unknown path falls through to SPA index.
	c2 := newFSContext(http.MethodGet, "/unknown/route")
	resp2, err := h(c2)
	if err != nil {
		t.Fatalf("handler error for SPA fallback: %v", err)
	}
	if resp2.Status != http.StatusOK {
		t.Fatalf("status for SPA fallback = %d, want %d", resp2.Status, http.StatusOK)
	}
	body, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "<html>SPA</html>" {
		t.Errorf("body = %q, want SPA index content", string(body))
	}
}
