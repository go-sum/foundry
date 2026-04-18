package etag

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/go-sum/web"
	"github.com/go-sum/web/headers"
)

// Config configures the ETag middleware.
type Config struct {
	// Weak, if true, emits a weak ETag (W/"..."). Default false = strong ETag.
	Weak bool
	// MinSize is the minimum body size in bytes before an ETag is computed.
	// Responses smaller than this always get an ETag. Default 0 = always compute.
	MinSize int
	// MaxBuffer is the maximum number of bytes to buffer for ETag computation.
	// Responses larger than MaxBuffer are streamed through without an ETag.
	// Default 1 MiB (1 << 20).
	MaxBuffer int64
}

// Middleware returns a web.Middleware that computes and sets an ETag header
// for responses where none is already set. It also handles conditional GET
// requests (If-None-Match → 304 Not Modified) for 200 OK responses.
//
// Responses are skipped when:
//   - resp.Status != 200
//   - resp.Headers already has an ETag header
//   - resp.Body is nil
//   - The buffered body exceeds MaxBuffer (streamed through as-is)
func Middleware(cfg Config) web.Middleware {
	cfg.MaxBuffer = cmp.Or(cfg.MaxBuffer, int64(1<<20))

	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			resp, err := next(c)
			if err != nil {
				return resp, err
			}

			if resp.Status != http.StatusOK {
				return resp, nil
			}
			if resp.Headers.Get("ETag") != "" {
				return resp, nil
			}
			if resp.Body == nil {
				return resp, nil
			}

			var buf bytes.Buffer
			// Read up to MaxBuffer+1 bytes to detect whether the body exceeds the limit.
			n, readErr := io.CopyN(&buf, resp.Body, cfg.MaxBuffer+1)

			if n > cfg.MaxBuffer {
				// Body exceeds MaxBuffer. Reconstruct the full body by prepending the
				// bytes already read to the remainder of the original reader.
				resp.Body = io.NopCloser(io.MultiReader(&buf, resp.Body))
				return resp, nil
			}

			// Body fits within MaxBuffer (or read completed with EOF).
			// Close the original body since we've fully consumed it.
			_ = resp.Body.Close()

			if readErr != nil && readErr != io.EOF {
				// Unreadable body — return what we buffered.
				resp.Body = io.NopCloser(&buf)
				return resp, nil
			}

			tag := computeETag(buf.Bytes(), cfg.Weak)

			// Conditional GET: check If-None-Match.
			ifNoneMatch := c.Headers().Get("If-None-Match")
			if ifNoneMatch != "" {
				parsed, parseErr := headers.ParseIfNoneMatch(ifNoneMatch)
				if parseErr == nil {
					// Extract the raw tag value for matching (strip quotes and W/ prefix).
					rawTag := extractTagValue(tag)
					// RFC 7232 6: If-None-Match uses weak comparison for GET.
					if parsed.Matches(rawTag, true) {
						return web.Respond(http.StatusNotModified), nil
					}
				}
			}

			resp.Headers.Set("ETag", tag)
			resp.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
			return resp, nil
		}
	}
}

// computeETag returns the ETag string for the given body bytes.
// Strong format: "abcdef..." Weak format: W/"abcdef..."
func computeETag(body []byte, weak bool) string {
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])[:32]
	if weak {
		return `W/"` + hash + `"`
	}
	return `"` + hash + `"`
}

// extractTagValue returns the unquoted tag value from an ETag string.
// For strong ETags ("abc") it returns "abc".
// For weak ETags (W/"abc") it returns "abc".
func extractTagValue(etag string) string {
	if len(etag) >= 2 && etag[:2] == `W/` {
		etag = etag[2:]
	}
	if len(etag) >= 2 && etag[0] == '"' && etag[len(etag)-1] == '"' {
		return etag[1 : len(etag)-1]
	}
	return etag
}
