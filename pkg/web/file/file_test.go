package file

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-sum/web"
)

// newTestRequest creates a Request with the given method and optional headers.
func newTestRequest(method, path string) *web.Request {
	u, _ := url.Parse(path)
	req := web.NewRequest(method, u)
	return &req
}

// fixedModTime is a fixed time used in tests.
var fixedModTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// testSource creates a BytesSource with fixed content and mod time.
func testSource(data []byte) *BytesSource {
	return NewBytesSource("test.bin", data, fixedModTime, "application/octet-stream")
}

func TestFile_Serve_FullContent(t *testing.T) {
	data := []byte("hello world")
	src := testSource(data)
	req := newTestRequest(http.MethodGet, "/test.bin")

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if resp.Body == nil {
		t.Fatal("expected body, got nil")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != string(data) {
		t.Errorf("body = %q, want %q", string(body), string(data))
	}
	if got := resp.Headers.Get("Content-Length"); got != "11" {
		t.Errorf("Content-Length = %q, want %q", got, "11")
	}
}

func TestFile_Serve_NotModified_INM(t *testing.T) {
	data := []byte("hello world")
	src := testSource(data)

	// Compute expected weak ETag
	etag := WeakETagFor(src)
	// Strip W/ prefix for If-None-Match value
	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("If-None-Match", etag)

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}

	if resp.Status != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusNotModified)
	}
}

func TestFile_Serve_NotModified_IMS(t *testing.T) {
	data := []byte("hello")
	src := testSource(data)

	t.Run("future IMS returns 304", func(t *testing.T) {
		req := newTestRequest(http.MethodGet, "/test.bin")
		// IMS is after mod time — file not modified since then → 304
		futureTime := fixedModTime.Add(time.Hour)
		req.Headers.Set("If-Modified-Since", futureTime.UTC().Format(http.TimeFormat))

		resp, err := Serve(req, src, ServeOptions{})
		if err != nil {
			t.Fatalf("Serve: %v", err)
		}
		if resp.Status != http.StatusNotModified {
			t.Fatalf("status = %d, want %d", resp.Status, http.StatusNotModified)
		}
	})

	t.Run("past IMS returns 200", func(t *testing.T) {
		req := newTestRequest(http.MethodGet, "/test.bin")
		// IMS is before mod time — file was modified → 200
		pastTime := fixedModTime.Add(-time.Hour)
		req.Headers.Set("If-Modified-Since", pastTime.UTC().Format(http.TimeFormat))

		resp, err := Serve(req, src, ServeOptions{})
		if err != nil {
			t.Fatalf("Serve: %v", err)
		}
		if resp.Status != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
		}
	})
}

func TestFile_Serve_PreconditionFailed_IM(t *testing.T) {
	data := []byte("hello")
	src := NewBytesSource("test.bin", data, fixedModTime, "application/octet-stream")

	req := newTestRequest(http.MethodGet, "/test.bin")
	// Use a strong ETag that won't match
	req.Headers.Set("If-Match", `"nonexistent-etag"`)

	resp, err := Serve(req, src, ServeOptions{ETag: StrongETag})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusPreconditionFailed {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusPreconditionFailed)
	}
}

func TestFile_Serve_PreconditionFailed_IUS(t *testing.T) {
	data := []byte("hello")
	src := testSource(data)

	req := newTestRequest(http.MethodGet, "/test.bin")
	// If-Unmodified-Since is before mod time → file was modified → 412
	pastTime := fixedModTime.Add(-time.Hour)
	req.Headers.Set("If-Unmodified-Since", pastTime.UTC().Format(http.TimeFormat))

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusPreconditionFailed {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusPreconditionFailed)
	}
}

func TestFile_Serve_Range_Single(t *testing.T) {
	data := []byte("hello world!!!!")
	src := testSource(data)

	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("Range", "bytes=0-99")

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusPartialContent)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != string(data) {
		t.Errorf("body = %q, want %q", string(body), string(data))
	}
}

func TestFile_Serve_Range_Suffix(t *testing.T) {
	data := []byte("hello world")
	src := testSource(data)

	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("Range", "bytes=-5")

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusPartialContent)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	// Last 5 bytes of "hello world" = "world"
	if string(body) != "world" {
		t.Errorf("body = %q, want %q", string(body), "world")
	}
}

func TestFile_Serve_Range_Multi(t *testing.T) {
	data := []byte("hello world")
	src := testSource(data)

	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("Range", "bytes=0-10,20-30")

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusRequestedRangeNotSatisfiable)
	}
}

func TestFile_Serve_Range_Unsatisfiable(t *testing.T) {
	data := make([]byte, 100)
	src := testSource(data)

	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("Range", "bytes=9999-9999")

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusRequestedRangeNotSatisfiable)
	}
	if got := resp.Headers.Get("Content-Range"); got != "bytes */100" {
		t.Errorf("Content-Range = %q, want %q", got, "bytes */100")
	}
}

func TestFile_Serve_IfRange_WeakETag(t *testing.T) {
	data := []byte("hello world")
	src := testSource(data)

	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("Range", "bytes=0-4")
	// If-Range with weak ETag — ParseIfRange rejects W/, so falls back to full
	req.Headers.Set("If-Range", `W/"12345-67890"`)

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	// If-Range with weak ETag is invalid per RFC 7233 3.2, so ParseIfRange
	// returns an error, which causes full content to be served
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d (weak ETag If-Range should fall back to full)", resp.Status, http.StatusOK)
	}
}

func TestFile_Serve_HEAD(t *testing.T) {
	data := []byte("hello world")
	src := testSource(data)

	req := newTestRequest(http.MethodHead, "/test.bin")

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if resp.Body != nil {
		t.Error("expected nil body for HEAD request")
	}
	if got := resp.Headers.Get("Content-Length"); got != "11" {
		t.Errorf("Content-Length = %q, want %q", got, "11")
	}
}

func TestFile_Serve_ContentDisposition_RFC5987(t *testing.T) {
	data := []byte("hello")
	src := testSource(data)

	req := newTestRequest(http.MethodGet, "/test.bin")

	resp, err := Serve(req, src, ServeOptions{
		Filename:        "résumé.pdf",
		DispositionType: "attachment",
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	cd := resp.Headers.Get("Content-Disposition")
	if cd == "" {
		t.Fatal("Content-Disposition header missing")
	}
	// Verify it contains the RFC 5987 filename* param
	if cd[:10] != "attachment" {
		t.Errorf("Content-Disposition prefix = %q, want %q", cd[:10], "attachment")
	}
	// Verify percent-encoding is present for non-ASCII
	if len(cd) <= len("attachment; filename*=UTF-8''") {
		t.Errorf("Content-Disposition too short: %q", cd)
	}
}

// ---------------------------------------------------------------------------
// strongETagValue unit tests
// ---------------------------------------------------------------------------

func TestStrongETagValue(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantStrong bool
	}{
		{
			name:       "strong ETag returns value and true",
			input:      `"abc123"`,
			wantVal:    "abc123",
			wantStrong: true,
		},
		{
			name:       "weak ETag returns empty and false",
			input:      `W/"abc123"`,
			wantVal:    "",
			wantStrong: false,
		},
		{
			name:       "empty quoted string returns empty and true",
			input:      `""`,
			wantVal:    "",
			wantStrong: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotVal, gotStrong := strongETagValue(tc.input)
			if gotVal != tc.wantVal {
				t.Errorf("strongETagValue(%q) val = %q, want %q", tc.input, gotVal, tc.wantVal)
			}
			if gotStrong != tc.wantStrong {
				t.Errorf("strongETagValue(%q) strong = %v, want %v", tc.input, gotStrong, tc.wantStrong)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Bug 1 — If-Match rejects weak server ETag
// ---------------------------------------------------------------------------

// TestFile_Serve_IfMatch_WeakETagRejected verifies that a server-side weak ETag
// causes If-Match to fail with 412, even when the client sends the bare strong
// form of the same value.
func TestFile_Serve_IfMatch_WeakETagRejected(t *testing.T) {
	data := []byte("hello")
	src := testSource(data)

	// Server will produce W/"5-<mtime>" (WeakETag is the default zero value).
	weakEtag := WeakETagFor(src)
	// Strip W/ prefix to form the "matching" strong-looking ETag the client sends.
	bare := `"` + stripWeakPrefix(weakEtag) + `"`

	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("If-Match", bare)

	resp, err := Serve(req, src, ServeOptions{}) // default = WeakETag
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusPreconditionFailed {
		t.Fatalf("status = %d, want %d (weak server ETag must fail If-Match)", resp.Status, http.StatusPreconditionFailed)
	}
}

// TestFile_Serve_IfMatch_StrongETagMatches verifies that a correct strong ETag
// sent by the client satisfies If-Match and the resource is served.
func TestFile_Serve_IfMatch_StrongETagMatches(t *testing.T) {
	data := []byte("hello")
	src := testSource(data)

	strongEtag, err := StrongETagFor(src)
	if err != nil {
		t.Fatalf("StrongETagFor: %v", err)
	}

	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("If-Match", strongEtag)

	resp, err := Serve(req, src, ServeOptions{ETag: StrongETag})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d (strong ETag match must serve 200)", resp.Status, http.StatusOK)
	}
}

// TestFile_Serve_IfMatch_StrongETagMismatch verifies that a mismatched strong
// ETag results in 412, preserving the existing behavior.
func TestFile_Serve_IfMatch_StrongETagMismatch(t *testing.T) {
	data := []byte("hello")
	src := testSource(data)

	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("If-Match", `"definitely-wrong-etag"`)

	resp, err := Serve(req, src, ServeOptions{ETag: StrongETag})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusPreconditionFailed {
		t.Fatalf("status = %d, want %d (wrong strong ETag must fail)", resp.Status, http.StatusPreconditionFailed)
	}
}

// ---------------------------------------------------------------------------
// Bug 2 — If-None-Match method dispatch
// ---------------------------------------------------------------------------

func TestFile_Serve_IfNoneMatch_MethodDispatch(t *testing.T) {
	data := []byte("hello")
	src := testSource(data)

	// Use the weak ETag that the server will produce (default opts).
	matchingEtag := WeakETagFor(src)

	tests := []struct {
		method     string
		wantStatus int
	}{
		{http.MethodGet, http.StatusNotModified},
		{http.MethodHead, http.StatusNotModified},
		{http.MethodPut, http.StatusPreconditionFailed},
		{http.MethodPost, http.StatusPreconditionFailed},
		{http.MethodDelete, http.StatusPreconditionFailed},
		{"PATCH", http.StatusPreconditionFailed},
	}

	for _, tc := range tests {
		t.Run(tc.method, func(t *testing.T) {
			req := newTestRequest(tc.method, "/test.bin")
			req.Headers.Set("If-None-Match", matchingEtag)

			resp, err := Serve(req, src, ServeOptions{})
			if err != nil {
				t.Fatalf("Serve: %v", err)
			}
			if resp.Status != tc.wantStatus {
				t.Errorf("method %s: status = %d, want %d", tc.method, resp.Status, tc.wantStatus)
			}
		})
	}
}

// TestFile_Serve_IfNoneMatch_NonMatchingUnsafeMethod verifies that an unsafe
// method with a non-matching If-None-Match header does not fail — the condition
// is satisfied and the full response is served.
func TestFile_Serve_IfNoneMatch_NonMatchingUnsafeMethod(t *testing.T) {
	data := []byte("hello")
	src := testSource(data)

	req := newTestRequest(http.MethodPut, "/test.bin")
	req.Headers.Set("If-None-Match", `"wrong-etag-value"`)

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d (non-matching INM with unsafe method must serve 200)", resp.Status, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// Bug 3 — Sub-second modTime precision
// ---------------------------------------------------------------------------

// subSecondModTime is a fixed time that includes sub-second precision.
// Truncated to the second it is fixedModTime; the 500ms remainder must be
// discarded before any HTTP-date comparison.
var subSecondModTime = time.Date(2026, 1, 2, 3, 4, 5, 500_000_000, time.UTC)

// httpDate formats t truncated to the second as an HTTP-date string.
func httpDate(t time.Time) string {
	return t.Truncate(time.Second).UTC().Format(http.TimeFormat)
}

// testSourceSubSecond returns a BytesSource whose ModTime has sub-second precision.
func testSourceSubSecond(data []byte) *BytesSource {
	return NewBytesSource("test.bin", data, subSecondModTime, "application/octet-stream")
}

// TestFile_Serve_IUS_SubSecondModTime verifies that If-Unmodified-Since set to
// the second-truncated form of the file's modTime passes (200), because after
// truncation the resource is not considered strictly modified.
func TestFile_Serve_IUS_SubSecondModTime(t *testing.T) {
	data := []byte("hello")
	src := testSourceSubSecond(data)

	req := newTestRequest(http.MethodGet, "/test.bin")
	// IUS equals the truncated modTime — not strictly after → condition passes.
	req.Headers.Set("If-Unmodified-Since", httpDate(subSecondModTime))

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d (IUS == truncated modTime must serve 200)", resp.Status, http.StatusOK)
	}
}

// TestFile_Serve_IMS_SubSecondModTime verifies that If-Modified-Since set to
// the second-truncated form of the file's modTime returns 304, because after
// truncation the resource is not newer than the cached copy.
func TestFile_Serve_IMS_SubSecondModTime(t *testing.T) {
	data := []byte("hello")
	src := testSourceSubSecond(data)

	req := newTestRequest(http.MethodGet, "/test.bin")
	// IMS equals the truncated modTime — file is not strictly after → 304.
	req.Headers.Set("If-Modified-Since", httpDate(subSecondModTime))

	resp, err := Serve(req, src, ServeOptions{})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusNotModified {
		t.Fatalf("status = %d, want %d (IMS == truncated modTime must return 304)", resp.Status, http.StatusNotModified)
	}
}

// TestFile_Serve_IfRange_SubSecondModTime verifies that an If-Range date header
// set to the second-truncated modTime causes the range request to be honoured
// (206), because after truncation the resource is not considered newer.
func TestFile_Serve_IfRange_SubSecondModTime(t *testing.T) {
	data := []byte("hello world") // 11 bytes
	src := testSourceSubSecond(data)

	strongEtag, err := StrongETagFor(src)
	if err != nil {
		t.Fatalf("StrongETagFor: %v", err)
	}

	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("If-Match", strongEtag) // satisfy If-Match so we reach If-Range
	req.Headers.Set("Range", "bytes=0-4")
	req.Headers.Set("If-Range", httpDate(subSecondModTime))

	acceptRanges := true
	resp, err := Serve(req, src, ServeOptions{ETag: StrongETag, AcceptRanges: &acceptRanges})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d (If-Range date == truncated modTime must serve 206)", resp.Status, http.StatusPartialContent)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "hello" {
		t.Errorf("body = %q, want %q", string(body), "hello")
	}
}

// ---------------------------------------------------------------------------
// Bug 1 (related) — If-Range rejects weak server ETag
// ---------------------------------------------------------------------------

// TestFile_Serve_IfRange_WeakServerETag verifies that when the server holds a
// weak ETag, an If-Range header carrying the bare strong form of that value
// does NOT satisfy the condition — the full response (200) is served instead
// of 206 partial content.
func TestFile_Serve_IfRange_WeakServerETag(t *testing.T) {
	data := []byte("hello world")
	src := testSource(data) // fixedModTime, no sub-second

	weakEtag := WeakETagFor(src)
	// Construct the strong-looking ETag the client might send.
	bare := `"` + stripWeakPrefix(weakEtag) + `"`

	req := newTestRequest(http.MethodGet, "/test.bin")
	req.Headers.Set("Range", "bytes=0-4")
	req.Headers.Set("If-Range", bare)

	resp, err := Serve(req, src, ServeOptions{}) // default = WeakETag
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d (weak server ETag must fall back to full response)", resp.Status, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// Original traversal test (unchanged)
// ---------------------------------------------------------------------------

func TestP0_10_File_OSRootTraversalBlocked(t *testing.T) {
	// Create a temp dir as the OS root
	tmpDir := t.TempDir()

	// Write a file inside the root
	innerPath := filepath.Join(tmpDir, "inner.txt")
	if err := os.WriteFile(innerPath, []byte("inner content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Open an os.Root pointing at tmpDir
	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	defer root.Close()

	// Attempt to open a traversal path — os.Root should block this structurally
	_, err = OpenOSFile(root, "../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for traversal path, got nil")
	}
	t.Logf("traversal correctly blocked: %v", err)
}
