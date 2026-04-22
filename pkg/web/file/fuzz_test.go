package file

import (
	"io"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-sum/web"
)

// FuzzServeHeaders exercises the full RFC 7232/7233 conditional-request
// matrix (If-Match / If-Unmodified-Since / If-None-Match / If-Modified-Since /
// If-Range / Range) with adversarial header values.
//
// Invariants:
//  1. Serve never panics.
//  2. The returned Response always has a valid HTTP status code.
//  3. If the response body is non-nil, it must be readable without panic.
func FuzzServeHeaders(f *testing.F) {
	// Seed: normal GET
	f.Add("GET", "", "", "", "", "", "")
	// Seed: simple byte range
	f.Add("GET", "bytes=0-4", "", "", "", "", "")
	// Seed: If-None-Match wildcard
	f.Add("GET", "", "*", "", "", "", "")
	// Seed: If-Match
	f.Add("GET", "", "", `"someetag"`, "", "", "")
	// Seed: conditional 304
	f.Add("GET", "", `W/"abc"`, "", "", "Thu, 01 Jan 2026 00:00:00 GMT", "")
	// Seed: If-Range with ETag
	f.Add("GET", "bytes=0-4", "", "", `"abc"`, "", "")
	// Seed: multi-range → 416
	f.Add("GET", "bytes=0-1,3-4", "", "", "", "", "")
	// Seed: huge range
	f.Add("GET", "bytes=-9999999999999", "", "", "", "", "")
	// Seed: malformed range
	f.Add("GET", "notbytes=abc", "", "", "", "", "")
	// Seed: HEAD
	f.Add("HEAD", "", "", "", "", "", "")
	// Seed: CRLF injection attempt in Range header
	f.Add("GET", "bytes=0-4\r\nX-Injected: yes", "", "", "", "", "")

	fixedTime := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	src := NewBytesSource("fuzz.bin", []byte("hello world fuzz content"), fixedTime, "application/octet-stream")

	f.Fuzz(func(t *testing.T,
		method, rangeHdr, ifNoneMatch, ifMatch, ifRange, ifModifiedSince, ifUnmodifiedSince string,
	) {
		u, _ := url.Parse("/fuzz.bin")
		req := web.NewRequest(method, u)

		if rangeHdr != "" {
			req.Headers.Set("Range", rangeHdr)
		}
		if ifNoneMatch != "" {
			req.Headers.Set("If-None-Match", ifNoneMatch)
		}
		if ifMatch != "" {
			req.Headers.Set("If-Match", ifMatch)
		}
		if ifRange != "" {
			req.Headers.Set("If-Range", ifRange)
		}
		if ifModifiedSince != "" {
			req.Headers.Set("If-Modified-Since", ifModifiedSince)
		}
		if ifUnmodifiedSince != "" {
			req.Headers.Set("If-Unmodified-Since", ifUnmodifiedSince)
		}

		resp, _ := Serve(&req, src, ServeOptions{})

		// Status must be a valid HTTP code (zero for error path is acceptable)
		if resp.Status != 0 && (resp.Status < 100 || resp.Status > 599) {
			t.Errorf("invalid status %d", resp.Status)
		}

		// Body must be fully readable without panic
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close() //nolint:errcheck
		}
	})
}

// FuzzOSFilePath exercises OpenOSFile with adversarial relative paths.
// os.Root guarantees traversal is impossible, but we confirm it never panics
// and always returns an error (or a valid source) without touching files
// outside the root.
func FuzzOSFilePath(f *testing.F) {
	// Seed: normal file
	f.Add("file.txt")
	// Seed: traversal attempts
	f.Add("../secret")
	f.Add("../../etc/passwd")
	f.Add("a/../../../etc/shadow")
	f.Add("%2e%2e/secret")
	f.Add("a/%2F../b")
	// Seed: null byte injection
	f.Add("file\x00.txt")
	// Seed: absolute path
	f.Add("/etc/passwd")
	// Seed: windows-style
	f.Add(`..\..\windows\system32`)
	// Seed: deep nesting
	f.Add("a/b/c/d/e/f/g/h/i/j/k/l/m/n.txt")
	// Seed: empty
	f.Add("")

	// Create a temp root directory with one file.
	dir := t_tempDir(f)
	if err := os.WriteFile(dir+"/file.txt", []byte("hello"), 0o644); err != nil {
		f.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		f.Fatal(err)
	}
	f.Cleanup(func() { root.Close() }) //nolint:errcheck

	f.Fuzz(func(t *testing.T, rel string) {
		// Must not panic; error is expected for traversal / missing paths.
		src, err := OpenOSFile(root, rel)
		if err != nil {
			// Expected — traversal attempts, non-existent files, etc.
			return
		}
		// If it succeeded, it must be "file.txt" (the only file in root).
		// Verify we can read it without panic.
		buf := make([]byte, src.Size())
		_, _ = src.ReadAt(buf, 0)
	})
}

// t_tempDir creates a temporary directory and registers cleanup.
// Named to avoid collision with the testing.T.TempDir method.
func t_tempDir(f *testing.F) string {
	f.Helper()
	dir, err := os.MkdirTemp("", "fuzz-file-*")
	if err != nil {
		f.Fatal(err)
	}
	f.Cleanup(func() { os.RemoveAll(dir) }) //nolint:errcheck
	return dir
}
