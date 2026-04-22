package static

import (
	"cmp"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-sum/web"
	"github.com/go-sum/web/file"
)

// FSHandler returns a web.Handler that serves static files from fsys.
// It supports the same Options as Handler: index file resolution, ETag,
// Cache-Control, pre-compressed sidecar files, and NotFound fallback.
// Works with embed.FS, os.DirFS, and any fs.FS implementation.
func FSHandler(fsys fs.FS, opts Options) web.Handler {
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

		// Clean the path.
		rel := path.Clean(strings.TrimPrefix(req.URL.Path, "/"))
		if rel == "." {
			rel = ""
		}

		src, err := openFSFile(fsys, rel)
		if err != nil {
			// Try index files for directory-like paths.
			for _, idx := range opts.IndexFiles {
				idxPath := rel
				if idxPath == "" {
					idxPath = idx
				} else {
					idxPath = rel + "/" + idx
				}
				src2, err2 := openFSFile(fsys, idxPath)
				if err2 == nil {
					return file.Serve(req, src2, file.ServeOptions{
						ETag: opts.ETag, CacheControl: opts.CacheControl,
						ContentType: mimeFor(path.Ext(idxPath), opts.MimeTypes),
					})
				}
			}
			if opts.NotFound != nil {
				return opts.NotFound(c)
			}
			return web.Response{}, web.ErrNotFound("")
		}

		return file.Serve(req, src, file.ServeOptions{
			ETag: opts.ETag, CacheControl: opts.CacheControl,
			ContentType: mimeFor(path.Ext(rel), opts.MimeTypes),
		})
	}
}

// openFSFile opens a file from fsys and returns it as a file.Source.
// Returns an error if the file does not exist, is a directory, or rel is empty.
func openFSFile(fsys fs.FS, rel string) (_ file.Source, err error) {
	if rel == "" {
		return nil, fs.ErrNotExist
	}

	f, err := fsys.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fs.ErrInvalid
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	ct := cmp.Or(mime.TypeByExtension(filepath.Ext(rel)), "application/octet-stream")

	return file.NewBytesSource(filepath.Base(rel), data, fi.ModTime(), ct), nil
}
