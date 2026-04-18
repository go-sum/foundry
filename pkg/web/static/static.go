package static

import (
	"mime"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-sum/web"
	"github.com/go-sum/web/file"
	"github.com/go-sum/web/headers"
)

// AssetsConfig is the env-facing shape for static asset serving.
type AssetsConfig struct {
	PublicDir string `validate:"required"`
	URLPrefix string `validate:"required"`
}

// DefaultAssetsConfig returns generic static asset defaults.
func DefaultAssetsConfig() AssetsConfig {
	return AssetsConfig{
		PublicDir: "public/static",
		URLPrefix: "/static",
	}
}

// Options configures the Static handler.
type Options struct {
	// IndexFiles is the list of filenames to serve for directory requests.
	// Default: ["index.html", "index.htm"]
	IndexFiles []string
	// Index enables index-file resolution. Default: true.
	Index bool
	// Filter is an optional function called for each file path before serving.
	// Return false to skip (fall through to next handler).
	Filter func(string) bool
	// ETag controls ETag mode for served files. Default: WeakETag.
	ETag file.ETagMode
	// CacheControl is the Cache-Control header value. Default: none.
	CacheControl string
	// NotFound is an optional handler invoked when no file matches the request.
	// When nil (the default), the handler returns a 404 response. Set this to
	// chain with a downstream handler — for example to implement an SPA
	// fallback that serves index.html for unknown paths.
	NotFound web.Handler
	// Precompressed, if true, attempts to serve pre-compressed sidecar files
	// when the client accepts the encoding. For a requested path "/app.js",
	// the handler tries "/app.js.br" (Brotli) and "/app.js.gz" (gzip) in
	// that order, falling back to the original file. Only activates when the
	// sidecar file exists in the root and the client Accept-Encoding allows
	// the encoding. Defaults to false.
	Precompressed bool
}

var defaultIndexFiles = []string{"index.html", "index.htm"}

// Handler returns a web.Handler that serves static files from root.
// It serves GET and HEAD requests only. On any error (not found, permission
// denied, directory without index), it returns 404.
func Handler(root *os.Root, opts Options) web.Handler {
	if opts.IndexFiles == nil {
		opts.IndexFiles = defaultIndexFiles
	}
	if !opts.Index {
		opts.IndexFiles = nil
	}

	return func(c *web.Context) (web.Response, error) {
		req := &c.Request
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			return web.Response{}, web.ErrMethodNotAllowed("")
		}

		if opts.Filter != nil && !opts.Filter(req.URL.Path) {
			if opts.NotFound != nil {
				return opts.NotFound(c)
			}
			return web.Response{}, web.ErrNotFound("")
		}

		// Clean the path
		rel := path.Clean(strings.TrimPrefix(req.URL.Path, "/"))
		if rel == "." {
			rel = ""
		}

		src, err := file.OpenOSFile(root, rel)
		if err != nil {
			// Try index files for directory-like paths
			for _, idx := range opts.IndexFiles {
				idxPath := rel
				if idxPath == "" {
					idxPath = idx
				} else {
					idxPath = rel + "/" + idx
				}
				src2, err2 := file.OpenOSFile(root, idxPath)
				if err2 == nil {
					return file.Serve(req, src2, file.ServeOptions{
						ETag: opts.ETag, CacheControl: opts.CacheControl,
					})
				}
			}
			if opts.NotFound != nil {
				return opts.NotFound(c)
			}
			return web.Response{}, web.ErrNotFound("")
		}

		if opts.Precompressed {
			ae, _ := headers.ParseAcceptEncoding(req.Headers.Get("Accept-Encoding"))
			encoding := ae.Negotiate("br", "gzip")
			if encoding == "br" || encoding == "gzip" {
				suffix := ".gz"
				if encoding == "br" {
					suffix = ".br"
				}
				sidecarPath := rel + suffix
				sidecar, sidecarErr := file.OpenOSFile(root, sidecarPath)
				if sidecarErr == nil {
					ct := mime.TypeByExtension(path.Ext(rel))
					if ct == "" {
						ct = "application/octet-stream"
					}
					resp, err := file.Serve(req, sidecar, file.ServeOptions{
						ETag:         opts.ETag,
						CacheControl: opts.CacheControl,
						ContentType:  ct,
					})
					if err != nil {
						return resp, err
					}
					resp.Headers.Set("Content-Encoding", encoding)
					resp.Headers.Set("Vary", "Accept-Encoding")
					return resp, nil
				}
			}
		}

		return file.Serve(req, src, file.ServeOptions{
			ETag: opts.ETag, CacheControl: opts.CacheControl,
		})
	}
}
