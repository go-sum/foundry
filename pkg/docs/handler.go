package docs

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-sum/web"
)

// DefaultDocsDir is the subdirectory under PublicDir where Hugo output is stored.
// The CLI build command and the handler both use this value so they stay in sync.
const DefaultDocsDir = "docs"

// Config holds configuration for the docs handler.
type Config struct {
	PublicDir         string
	BasePath          string
	AssetCacheControl string
	PageCacheControl  string
}

// DefaultConfig returns a Config with sensible defaults for the given public directory.
func DefaultConfig(publicDir string) Config {
	return Config{
		PublicDir:         publicDir,
		BasePath:          "/docs",
		AssetCacheControl: "public, max-age=3600",
		PageCacheControl:  "no-cache",
	}
}

// Handler serves pre-built Hugo documentation from the filesystem.
type Handler struct {
	cfg Config
}

// NewHandler creates a Handler with the given configuration.
func NewHandler(cfg Config) *Handler {
	return &Handler{cfg: cfg}
}

// Serve is a web.Handler that serves documentation files from the public directory.
func (h *Handler) Serve(c *web.Context) (web.Response, error) {
	rel := c.Param("path")
	root := filepath.Join(h.cfg.PublicDir, DefaultDocsDir)

	target, isAsset, err := resolvePath(root, rel)
	if err != nil {
		return web.Response{}, web.ErrNotFound("")
	}

	_, statErr := os.Stat(target)
	if statErr == nil {
		cacheControl := h.cfg.PageCacheControl
		if isAsset {
			cacheControl = h.cfg.AssetCacheControl
		}
		return serveFile(http.StatusOK, target, cacheControl)
	}

	if !os.IsNotExist(statErr) {
		return web.Response{}, statErr
	}

	if isAsset {
		return web.Response{}, web.ErrNotFound("")
	}

	notFoundPath := filepath.Join(root, "404.html")
	if _, err := os.Stat(notFoundPath); err != nil {
		return web.Response{}, web.ErrNotFound("")
	}
	return serveFile(http.StatusNotFound, notFoundPath, h.cfg.PageCacheControl)
}

// resolvePath maps a URL-relative path to a filesystem path under root.
// It returns the target file path, whether it is a static asset, and any error.
func resolvePath(root, rel string) (string, bool, error) {
	if root == "" {
		return "", false, errEmptyRoot
	}
	if rel == "" {
		return filepath.Join(root, "index.html"), false, nil
	}
	if strings.Contains(rel, "..") {
		return "", false, errPathTraversal
	}

	cleanRel := strings.TrimPrefix(path.Clean("/"+rel), "/")
	if cleanRel == "" || cleanRel == "." {
		return filepath.Join(root, "index.html"), false, nil
	}

	if path.Ext(cleanRel) != "" {
		return filepath.Join(root, filepath.FromSlash(cleanRel)), true, nil
	}

	return filepath.Join(root, filepath.FromSlash(cleanRel), "index.html"), false, nil
}

// serveFile reads filename from disk and returns a Response with the given status and cache headers.
func serveFile(status int, filename, cacheControl string) (web.Response, error) {
	body, err := os.ReadFile(filename)
	if err != nil {
		return web.Response{}, err
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	hdrs := web.NewHeaders()
	hdrs.Set("Content-Type", contentType)
	hdrs.Set("Cache-Control", cacheControl)

	return web.Response{
		Status:  status,
		Headers: hdrs,
		Body:    io.NopCloser(bytes.NewReader(body)),
	}, nil
}

var (
	errEmptyRoot     = pathError("root is empty")
	errPathTraversal = pathError("path traversal")
)

type pathError string

func (e pathError) Error() string { return string(e) }
