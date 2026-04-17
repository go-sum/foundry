package file

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-sum/web"
	"github.com/go-sum/web/headers"
)

// ServeOptions configures file serving behavior.
type ServeOptions struct {
	ETag         ETagMode // default WeakETag
	AcceptRanges *bool    // nil = auto (false for compressible MIME)
	CacheControl string   // empty = no Cache-Control set
	ContentType  string   // empty = from Source.ContentType()
	// Filename, if non-empty, sets Content-Disposition: attachment; filename*=UTF-8''<encoded>
	Filename        string
	DispositionType string // "attachment" or "inline"; empty = omit header
}

// Serve writes a conditional HTTP response for the given Source.
// It handles If-Match, If-Unmodified-Since, If-None-Match, If-Modified-Since,
// If-Range, and Range headers in the correct RFC 7232/7233 precedence order.
func Serve(req *web.Request, src Source, opts ServeOptions) (web.Response, error) {
	etag := WeakETagFor(src)
	if opts.ETag == StrongETag {
		var err error
		etag, err = StrongETagFor(src)
		if err != nil {
			return web.Response{}, web.ErrInternal(err)
		}
	}

	ct := opts.ContentType
	if ct == "" {
		ct = src.ContentType()
	}

	acceptRanges := true
	if opts.AcceptRanges != nil {
		acceptRanges = *opts.AcceptRanges
	} else if isCompressibleMIME(ct) {
		acceptRanges = false
	}

	modTime := src.ModTime().Truncate(time.Second)

	// Step 1: If-Match (strong compare only) → 412 if fails or server ETag is weak
	if ifMatch := req.Headers.Get("If-Match"); ifMatch != "" {
		m, err := headers.ParseIfMatch(ifMatch)
		val, strong := strongETagValue(etag)
		if err == nil && (!strong || !m.Matches(val)) {
			return web.Response{Status: http.StatusPreconditionFailed}, nil
		}
	}

	// Step 2: If-Unmodified-Since (only if If-Match absent) → 412 if modified
	if req.Headers.Get("If-Match") == "" {
		if ius := req.Headers.Get("If-Unmodified-Since"); ius != "" {
			if t, err := http.ParseTime(ius); err == nil {
				if modTime.After(t) {
					return web.Response{Status: http.StatusPreconditionFailed}, nil
				}
			}
		}
	}

	// Step 3: If-None-Match → 304 (safe methods) or 412 (unsafe methods) if matched
	if inm := req.Headers.Get("If-None-Match"); inm != "" {
		m, err := headers.ParseIfNoneMatch(inm)
		if err == nil && m.Matches(stripWeakPrefix(etag), true) {
			if req.Method == http.MethodGet || req.Method == http.MethodHead {
				return buildNotModifiedResponse(etag, modTime, opts.CacheControl), nil
			}
			return web.Response{Status: http.StatusPreconditionFailed}, nil
		}
	}

	// Step 4: If-Modified-Since (only if If-None-Match absent) → 304
	if req.Headers.Get("If-None-Match") == "" {
		if ims := req.Headers.Get("If-Modified-Since"); ims != "" {
			if t, err := http.ParseTime(ims); err == nil {
				if !modTime.After(t) {
					return buildNotModifiedResponse(etag, modTime, opts.CacheControl), nil
				}
			}
		}
	}

	// Build base response headers
	h := web.NewHeaders()
	h.Set("Content-Type", ct)
	h.Set("ETag", etag)
	h.Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))
	if opts.CacheControl != "" {
		h.Set("Cache-Control", opts.CacheControl)
	}
	if acceptRanges {
		h.Set("Accept-Ranges", "bytes")
	} else {
		h.Set("Accept-Ranges", "none")
	}
	setContentDisposition(h, opts)

	// HEAD request: return headers only, no body
	if req.Method == http.MethodHead {
		h.Set("Content-Length", strconv.FormatInt(src.Size(), 10))
		return web.Response{Status: http.StatusOK, Headers: h}, nil
	}

	// Step 5: If-Range + Range
	if rangeHeader := req.Headers.Get("Range"); rangeHeader != "" && acceptRanges {
		rng, err := headers.ParseRange(rangeHeader)
		if err != nil || len(rng.Ranges) == 0 {
			// Malformed range: ignore and serve full content
			return buildFullResponse(src, h), nil
		}
		if len(rng.Ranges) > 1 {
			// Multi-range: return 416 (we don't support multipart/byteranges)
			r416 := web.NewHeaders()
			r416.Set("Content-Range", fmt.Sprintf("bytes */%d", src.Size()))
			return web.Response{Status: http.StatusRequestedRangeNotSatisfiable, Headers: r416}, nil
		}

		// Single range — check If-Range
		if ifRange := req.Headers.Get("If-Range"); ifRange != "" {
			ir, err := headers.ParseIfRange(ifRange)
			val, strong := strongETagValue(etag)
			if err != nil || !strong || !ir.Matches(val, modTime) {
				// If-Range failed (or weak server ETag): serve full content
				return buildFullResponse(src, h), nil
			}
		}

		start, end, ok := rng.Normalize(src.Size())
		if !ok {
			r416 := web.NewHeaders()
			r416.Set("Content-Range", fmt.Sprintf("bytes */%d", src.Size()))
			return web.Response{Status: http.StatusRequestedRangeNotSatisfiable, Headers: r416}, nil
		}

		// 206 Partial Content
		length := end - start + 1
		h.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, src.Size()))
		h.Set("Content-Length", strconv.FormatInt(length, 10))
		body := &sourceRangeReader{src: src, off: start, remaining: length}
		return web.Response{Status: http.StatusPartialContent, Headers: h, Body: io.NopCloser(body)}, nil
	}

	// Step 6: Full response
	return buildFullResponse(src, h), nil
}

func buildFullResponse(src Source, h web.Headers) web.Response {
	h.Set("Content-Length", strconv.FormatInt(src.Size(), 10))
	body := &sourceRangeReader{src: src, off: 0, remaining: src.Size()}
	return web.Response{Status: http.StatusOK, Headers: h, Body: io.NopCloser(body)}
}

func buildNotModifiedResponse(etag string, modTime time.Time, cacheControl string) web.Response {
	h := web.NewHeaders()
	h.Set("ETag", etag)
	h.Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))
	if cacheControl != "" {
		h.Set("Cache-Control", cacheControl)
	}
	return web.Response{Status: http.StatusNotModified, Headers: h}
}

// stripWeakPrefix removes the W/ prefix from a weak ETag for strong comparison.
// Returns the ETag value without quotes or prefix.
func stripWeakPrefix(etag string) string {
	etag = strings.TrimPrefix(etag, "W/")
	etag = strings.Trim(etag, `"`)
	return etag
}

// strongETagValue returns the unquoted ETag value and true if etag is a strong
// ETag (i.e. not prefixed with W/). Returns ("", false) for weak ETags.
// Used to enforce RFC 7232 3.1: weak ETags MUST NOT satisfy If-Match.
func strongETagValue(etag string) (string, bool) {
	if strings.HasPrefix(etag, `W/"`) {
		return "", false
	}
	return strings.Trim(etag, `"`), true
}

func setContentDisposition(h web.Headers, opts ServeOptions) {
	if opts.DispositionType == "" && opts.Filename == "" {
		return
	}
	dt := opts.DispositionType
	if dt == "" {
		dt = "attachment"
	}
	if opts.Filename == "" {
		h.Set("Content-Disposition", dt)
		return
	}
	// RFC 5987 encoding: filename*=UTF-8''percent-encoded
	encoded := rfc5987Encode(opts.Filename)
	h.Set("Content-Disposition",
		fmt.Sprintf(`%s; filename*=UTF-8''%s`, dt, encoded))
}

func rfc5987Encode(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if isAttrChar(c) {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// isAttrChar returns true for chars that don't need encoding in RFC 5987 attr-char.
func isAttrChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '!' || c == '#' || c == '$' || c == '&' || c == '+' ||
		c == '-' || c == '.' || c == '^' || c == '_' || c == '`' ||
		c == '|' || c == '~'
}

// sourceRangeReader reads from a Source at a specific offset range.
type sourceRangeReader struct {
	src       Source
	off       int64
	remaining int64
}

func (r *sourceRangeReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.src.ReadAt(p, r.off)
	r.off += int64(n)
	r.remaining -= int64(n)
	if r.remaining == 0 && err == nil {
		err = io.EOF
	}
	return n, err
}
