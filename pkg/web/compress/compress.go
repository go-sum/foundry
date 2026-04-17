// Package compress provides a compression middleware for the web package.
// It supports gzip and deflate encoding negotiated via Accept-Encoding.
package compress

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-sum/web"
	"github.com/go-sum/web/headers"
)

// defaultAllowedTypes is the default set of MIME type prefixes that are eligible
// for compression.
var defaultAllowedTypes = []string{
	"text/",
	"application/json",
	"application/javascript",
	"application/xml",
	"application/xhtml+xml",
	"image/svg+xml",
}

// Config configures the compression middleware.
type Config struct {
	// Level is the gzip/deflate compression level. Defaults to -1 (default compression).
	Level int
	// MinSize is the minimum response body size (in bytes) to compress. Defaults to 1024.
	MinSize int
	// AllowedTypes is a list of MIME type prefixes/suffixes to compress.
	// Defaults to: "text/", "application/json", "application/javascript",
	// "application/xml", "application/xhtml+xml", "image/svg+xml".
	// Matching is prefix-based after stripping params.
	AllowedTypes []string
}

// gzipPool pools *gzip.Writer objects to reduce allocations.
var gzipPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

// Middleware returns a web.Middleware that compresses response bodies using gzip
// or deflate, negotiated via Accept-Encoding. Brotli is not supported (would
// require a third-party dependency). The middleware:
//   - Skips responses with a pre-set Content-Encoding header.
//   - Skips responses with status 206 (Partial Content).
//   - Skips responses whose Content-Type does not match AllowedTypes.
//   - Skips responses smaller than MinSize.
//   - Adds "Vary: Accept-Encoding" to all passing responses.
func Middleware(cfg Config) web.Middleware {
	if cfg.MinSize <= 0 {
		cfg.MinSize = 1024
	}
	if cfg.AllowedTypes == nil {
		cfg.AllowedTypes = defaultAllowedTypes
	}
	if cfg.Level == 0 {
		cfg.Level = gzip.DefaultCompression
	}

	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			resp, err := next(c)
			if err != nil {
				return resp, err
			}

			// Negotiate encoding from Accept-Encoding header.
			ae, _ := headers.ParseAcceptEncoding(c.Headers.Get("Accept-Encoding"))
			encoding := ae.Negotiate("gzip", "deflate", "identity")
			if encoding != "gzip" && encoding != "deflate" {
				return resp, nil
			}

			// Skip if Content-Encoding already set.
			if resp.Headers.Get("Content-Encoding") != "" {
				return resp, nil
			}

			// Skip 206 Partial Content.
			if resp.Status == http.StatusPartialContent {
				return resp, nil
			}

			// Skip if body is nil.
			if resp.Body == nil {
				return resp, nil
			}

			// Skip if Content-Type is not compressible.
			ct := resp.Headers.Get("Content-Type")
			if !isCompressible(ct, cfg.AllowedTypes) {
				return resp, nil
			}

			// Buffer up to MinSize bytes to decide whether to compress.
			buf := make([]byte, cfg.MinSize)
			n, readErr := io.ReadFull(resp.Body, buf)
			buf = buf[:n]

			if readErr != nil && readErr != io.ErrUnexpectedEOF {
				// Read error or EOF before MinSize — emit uncompressed.
				_ = resp.Body.Close()
				if n == 0 {
					resp.Body = nil
					return resp, nil
				}
				resp.Body = io.NopCloser(strings.NewReader(string(buf)))
				return resp, nil
			}

			if readErr == io.ErrUnexpectedEOF {
				// Body was smaller than MinSize — emit uncompressed.
				_ = resp.Body.Close()
				resp.Body = io.NopCloser(strings.NewReader(string(buf)))
				return resp, nil
			}

			// Body is >= MinSize — compress.
			originalBody := resp.Body
			pr, pw := io.Pipe()

			web.Go(nil, "compress", func() {
				var compressor io.WriteCloser
				var werr error

				switch encoding {
				case "gzip":
					gz := gzipPool.Get().(*gzip.Writer)
					gz.Reset(pw)
					compressor = gz
					defer func() {
						gz.Reset(io.Discard)
						gzipPool.Put(gz)
					}()
				case "deflate":
					compressor, werr = flate.NewWriter(pw, cfg.Level)
					if werr != nil {
						_ = pw.CloseWithError(werr)
						_ = originalBody.Close()
						return
					}
				}

				// Write buffered bytes first.
				if _, werr = compressor.Write(buf); werr != nil {
					_ = pw.CloseWithError(werr)
					_ = originalBody.Close()
					return
				}

				// Stream remainder.
				if _, werr = io.Copy(compressor, originalBody); werr != nil {
					_ = originalBody.Close()
					_ = pw.CloseWithError(werr)
					return
				}

				_ = originalBody.Close()
				if werr = compressor.Close(); werr != nil {
					_ = pw.CloseWithError(werr)
					return
				}
				_ = pw.Close()
			})

			resp.Body = pr
			resp.Headers.Set("Content-Encoding", encoding)
			resp.Headers.Delete("Content-Length")

			// Append Vary: Accept-Encoding without clobbering existing Vary.
			vary := headers.ParseVary(resp.Headers.Get("Vary"))
			vary = vary.Add("Accept-Encoding")
			resp.Headers.Set("Vary", vary.String())

			return resp, nil
		}
	}
}

// isCompressible reports whether the given Content-Type matches any of the
// allowed type prefixes. Params (e.g., "; charset=UTF-8") are stripped first.
func isCompressible(ct string, allowed []string) bool {
	// Strip params.
	if i := strings.Index(ct, ";"); i != -1 {
		ct = ct[:i]
	}
	ct = strings.TrimSpace(ct)
	if ct == "" {
		return false
	}
	ct = strings.ToLower(ct)

	for _, prefix := range allowed {
		prefix = strings.ToLower(prefix)
		if strings.HasPrefix(ct, prefix) || ct == prefix {
			return true
		}
	}
	return false
}
