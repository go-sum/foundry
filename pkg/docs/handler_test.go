package docs

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
)

// testContext builds a *web.Context for the given method and raw URL.
func testContext(method, rawURL string) *web.Context {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return web.NewContext(context.Background(), web.NewRequest(method, u))
}

// readBody drains and closes the response body, returning the string.
func readBody(t *testing.T, resp web.Response) string {
	t.Helper()
	if resp.Body == nil {
		return ""
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll body: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("Close body: %v", err)
	}
	return string(b)
}

// TestResolvePath covers all branches of resolvePath.
func TestResolvePath(t *testing.T) {
	root := filepath.Join("public", "docs")

	tests := []struct {
		name      string
		root      string
		rel       string
		wantPath  string
		wantAsset bool
		wantErr   bool
	}{
		{
			name:    "empty root",
			root:    "",
			rel:     "",
			wantErr: true,
		},
		{
			name:     "home (empty rel)",
			root:     root,
			rel:      "",
			wantPath: filepath.Join(root, "index.html"),
		},
		{
			name:     "slash rel",
			root:     root,
			rel:      "/",
			wantPath: filepath.Join(root, "index.html"),
		},
		{
			name:     "nested page",
			root:     root,
			rel:      "architecture/design_guide",
			wantPath: filepath.Join(root, "architecture", "design_guide", "index.html"),
		},
		{
			name:      "asset",
			root:      root,
			rel:       "css/main.css",
			wantPath:  filepath.Join(root, "css", "main.css"),
			wantAsset: true,
		},
		{
			name:    "path traversal",
			root:    root,
			rel:     "../../etc/passwd",
			wantErr: true,
		},
		{
			name:      "double-dot in filename (not traversal)",
			root:      root,
			rel:       "foo..bar",
			wantPath:  filepath.Join(root, "foo..bar"),
			wantAsset: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotAsset, err := resolvePath(tc.root, tc.rel)
			if tc.wantErr {
				if err == nil {
					t.Fatal("resolvePath() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolvePath() error = %v", err)
			}
			if gotPath != tc.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tc.wantPath)
			}
			if gotAsset != tc.wantAsset {
				t.Fatalf("isAsset = %v, want %v", gotAsset, tc.wantAsset)
			}
		})
	}
}

// buildTempDocsDir creates a temp directory with the standard doc/ structure
// used by multiple integration tests. Returns the tmpDir (parent of doc/).
func buildTempDocsDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	docsRoot := filepath.Join(tmpDir, "docs")

	dirs := []string{
		filepath.Join(docsRoot, "architecture", "api-rules"),
		filepath.Join(docsRoot, "css"),
		filepath.Join(docsRoot, "js"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}

	files := map[string]string{
		filepath.Join(docsRoot, "index.html"):                              "<h1>Docs</h1>",
		filepath.Join(docsRoot, "architecture", "api-rules", "index.html"): "<h1>API Rules</h1>",
		filepath.Join(docsRoot, "css", "main.css"):                         "body{color:#000;}",
		filepath.Join(docsRoot, "js", "darkmode.js"):                       "console.log('theme');",
		filepath.Join(docsRoot, "404.html"):                                "<h1>Document not found</h1>",
	}
	for name, contents := range files {
		if err := os.WriteFile(name, []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	return tmpDir
}

// TestHandlerServesPagesAssetsAndDocs404 exercises the full routing path via
// a real temp directory and the router.
func TestHandlerServesPagesAssetsAndDocs404(t *testing.T) {
	tmpDir := buildTempDocsDir(t)

	rt := router.New()
	router.Register(rt, Routes(DefaultConfig(tmpDir))...)

	tests := []struct {
		url        string
		wantStatus int
		wantBody   string
		wantType   string
		wantErr    bool
		errStatus  int
	}{
		{
			url:        "/docs",
			wantStatus: http.StatusOK,
			wantBody:   "<h1>Docs</h1>",
			wantType:   "text/html; charset=utf-8",
		},
		{
			url:        "/docs/architecture/api-rules",
			wantStatus: http.StatusOK,
			wantBody:   "<h1>API Rules</h1>",
			wantType:   "text/html; charset=utf-8",
		},
		{
			url:        "/docs/css/main.css",
			wantStatus: http.StatusOK,
			wantBody:   "body{color:#000;}",
			wantType:   "text/css; charset=utf-8",
		},
		{
			url:        "/docs/js/darkmode.js",
			wantStatus: http.StatusOK,
			wantBody:   "console.log('theme');",
			wantType:   "text/javascript; charset=utf-8",
		},
		{
			url:        "/docs/missing",
			wantStatus: http.StatusNotFound,
			wantBody:   "<h1>Document not found</h1>",
			wantType:   "text/html; charset=utf-8",
		},
		{
			url:       "/docs/css/missing.css",
			wantErr:   true,
			errStatus: http.StatusNotFound,
		},
		{
			url:       "/docs/../../etc/passwd",
			wantErr:   true,
			errStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			resp, err := rt.Serve(testContext(http.MethodGet, tc.url))

			if tc.wantErr {
				if err == nil {
					t.Fatal("Serve() error = nil, want non-nil")
				}
				classified := web.Classify(err)
				if classified.Status != tc.errStatus {
					t.Fatalf("error status = %d, want %d", classified.Status, tc.errStatus)
				}
				return
			}

			if err != nil {
				t.Fatalf("Serve() error = %v", err)
			}
			if resp.Status != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Status, tc.wantStatus)
			}
			body := readBody(t, resp)
			if body != tc.wantBody {
				t.Fatalf("body = %q, want %q", body, tc.wantBody)
			}
			if got := resp.Headers.Get("Content-Type"); got != tc.wantType {
				t.Fatalf("Content-Type = %q, want %q", got, tc.wantType)
			}
			wantLen := strconv.Itoa(len(tc.wantBody))
			if got := resp.Headers.Get("Content-Length"); got != wantLen {
				t.Fatalf("Content-Length = %q, want %q", got, wantLen)
			}
		})
	}
}

// TestHandlerCacheControlHeaders verifies that pages receive PageCacheControl
// and assets receive AssetCacheControl.
func TestHandlerCacheControlHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	docsRoot := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(filepath.Join(docsRoot, "css"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	files := map[string]string{
		filepath.Join(docsRoot, "index.html"):      "<h1>Docs</h1>",
		filepath.Join(docsRoot, "css", "main.css"): "body{}",
	}
	for name, contents := range files {
		if err := os.WriteFile(name, []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	rt := router.New()
	router.Register(rt, Routes(DefaultConfig(tmpDir))...)

	tests := []struct {
		url           string
		wantCacheCtrl string
	}{
		{url: "/docs", wantCacheCtrl: "no-cache"},
		{url: "/docs/css/main.css", wantCacheCtrl: "public, max-age=3600"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			resp, err := rt.Serve(testContext(http.MethodGet, tc.url))
			if err != nil {
				t.Fatalf("Serve() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if got := resp.Headers.Get("Cache-Control"); got != tc.wantCacheCtrl {
				t.Fatalf("Cache-Control = %q, want %q", got, tc.wantCacheCtrl)
			}
		})
	}
}

// TestHandlerMissing404Page verifies that a missing page without a 404.html
// still returns a 404 error (not a 500).
func TestHandlerMissing404Page(t *testing.T) {
	tmpDir := t.TempDir()
	// Create doc/ directory but without 404.html or index files.
	if err := os.MkdirAll(filepath.Join(tmpDir, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	rt := router.New()
	router.Register(rt, Routes(DefaultConfig(tmpDir))...)

	_, err := rt.Serve(testContext(http.MethodGet, "/docs/missing"))
	if err == nil {
		t.Fatal("Serve() error = nil, want non-nil for missing page without 404.html")
	}
	classified := web.Classify(err)
	if classified.Status != http.StatusNotFound {
		t.Fatalf("error status = %d, want %d", classified.Status, http.StatusNotFound)
	}
}

// TestHandlerCustomConfig verifies that Config.BasePath, Config.PageCacheControl,
// and Config.AssetCacheControl are all respected.
func TestHandlerCustomConfig(t *testing.T) {
	tmpDir := t.TempDir()
	docsRoot := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(filepath.Join(docsRoot, "css"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	files := map[string]string{
		filepath.Join(docsRoot, "index.html"):      "<h1>API Docs</h1>",
		filepath.Join(docsRoot, "css", "main.css"): "body{}",
	}
	for name, contents := range files {
		if err := os.WriteFile(name, []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	cfg := Config{
		PublicDir:         tmpDir,
		BasePath:          "/api-docs",
		AssetCacheControl: "public, max-age=86400",
		PageCacheControl:  "no-store",
	}

	rt := router.New()
	router.Register(rt, Routes(cfg)...)

	t.Run("custom base path — page cache control", func(t *testing.T) {
		resp, err := rt.Serve(testContext(http.MethodGet, "/api-docs"))
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.Status != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
		}
		if got, want := resp.Headers.Get("Cache-Control"), "no-store"; got != want {
			t.Fatalf("Cache-Control = %q, want %q", got, want)
		}
	})

	t.Run("custom base path — asset cache control", func(t *testing.T) {
		resp, err := rt.Serve(testContext(http.MethodGet, "/api-docs/css/main.css"))
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.Status != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
		}
		if got, want := resp.Headers.Get("Cache-Control"), "public, max-age=86400"; got != want {
			t.Fatalf("Cache-Control = %q, want %q", got, want)
		}
	})

	t.Run("old base path not registered", func(t *testing.T) {
		_, err := rt.Serve(testContext(http.MethodGet, "/docs"))
		if err == nil {
			t.Fatal("Serve() error = nil, want non-nil for unregistered /docs path")
		}
	})
}
