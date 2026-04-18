package static

import (
	"io/fs"
	"os"

	"github.com/go-sum/web"
	"github.com/go-sum/web/file"
)

// SPAFallback returns a NotFound handler for use in Options.NotFound that
// serves indexFile from root for any unmatched request path. This enables
// client-side routing in single-page applications.
//
// If indexFile does not exist in root, the handler returns 404.
func SPAFallback(root *os.Root, indexFile string) web.Handler {
	return func(c *web.Context) (web.Response, error) {
		src, err := file.OpenOSFile(root, indexFile)
		if err != nil {
			return web.Response{}, web.ErrNotFound("")
		}
		return file.Serve(&c.Request, src, file.ServeOptions{})
	}
}

// SPAFallbackFS is like SPAFallback but accepts any fs.FS implementation,
// enabling use with embed.FS, os.DirFS, and other fs.FS sources.
//
// If indexFile does not exist in fsys, the handler returns 404.
func SPAFallbackFS(fsys fs.FS, indexFile string) web.Handler {
	return func(c *web.Context) (web.Response, error) {
		src, err := openFSFile(fsys, indexFile)
		if err != nil {
			return web.Response{}, web.ErrNotFound("")
		}
		return file.Serve(&c.Request, src, file.ServeOptions{})
	}
}
