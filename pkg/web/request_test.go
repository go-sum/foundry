package web

import (
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
)

type cloneErrReader struct {
	closed bool
	read   bool
}

func (r *cloneErrReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, io.ErrUnexpectedEOF
	}
	r.read = true
	copy(p, "bad")
	return 3, io.ErrUnexpectedEOF
}

func (r *cloneErrReader) Close() error {
	r.closed = true
	return nil
}

func TestNewRequest(t *testing.T) {
	u, _ := url.Parse("https://example.com/path?q=1")
	r := NewRequest("POST", u)

	if r.Method != "POST" {
		t.Errorf("Method = %q, want %q", r.Method, "POST")
	}
	if r.URL != u {
		t.Errorf("URL = %v, want %v", r.URL, u)
	}
	// Headers should be initialized and usable
	r.Headers.Set("X-Test", "val")
	if got := r.Headers.Get("x-test"); got != "val" {
		t.Errorf("Headers.Get = %q, want %q", got, "val")
	}
}

func TestRequest_HostAndSetHost(t *testing.T) {
	u, _ := url.Parse("/")
	r := NewRequest("GET", u)

	if r.Host() != "" {
		t.Errorf("Host() = %q, want empty", r.Host())
	}

	r.SetHost("api.example.com")
	if r.Host() != "api.example.com" {
		t.Errorf("Host() = %q, want %q", r.Host(), "api.example.com")
	}
}

func TestRequest_RemoteAddrAndSetRemoteAddr(t *testing.T) {
	u, _ := url.Parse("/")
	r := NewRequest("GET", u)

	if r.RemoteAddr() != "" {
		t.Errorf("RemoteAddr() = %q, want empty", r.RemoteAddr())
	}

	r.SetRemoteAddr("192.168.1.1:8080")
	if r.RemoteAddr() != "192.168.1.1:8080" {
		t.Errorf("RemoteAddr() = %q, want %q", r.RemoteAddr(), "192.168.1.1:8080")
	}
}

func TestRequest_SetBody_TracksRead(t *testing.T) {
	req := NewRequest("POST", &url.URL{Path: "/"})
	req.SetBody(io.NopCloser(strings.NewReader("data")))

	_, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}
	if !req.BodyUsed() {
		t.Error("BodyUsed() = false after io.ReadAll via SetBody tracker, want true")
	}
}

func TestRequest_Clone_IndependentReaders(t *testing.T) {
	req := NewRequest("POST", &url.URL{Path: "/"})
	req.SetBody(io.NopCloser(strings.NewReader("payload")))

	clone, err := req.Clone()
	if err != nil {
		t.Fatalf("Clone() error = %v, want nil", err)
	}

	// Read clone first.
	cloneData, err := clone.Bytes()
	if err != nil {
		t.Fatalf("clone.Bytes() error = %v", err)
	}
	if string(cloneData) != "payload" {
		t.Errorf("clone.Bytes() = %q, want %q", string(cloneData), "payload")
	}

	// Original must still be unread.
	if req.BodyUsed() {
		t.Error("req.BodyUsed() = true after clone was read, want false")
	}

	// Read original — must get same content.
	origData, err := req.Bytes()
	if err != nil {
		t.Fatalf("req.Bytes() error = %v", err)
	}
	if string(origData) != "payload" {
		t.Errorf("req.Bytes() = %q, want %q", string(origData), "payload")
	}
}

func TestRequest_Clone_IndependentMetadata(t *testing.T) {
	u, _ := url.Parse("https://example.com/original?q=1")
	req := NewRequest("POST", u)
	req.Headers.Set("X-Test", "original")
	req.SetHost("api.example.com")
	req.SetRemoteAddr("192.168.1.1:8080")
	req.SetBody(io.NopCloser(strings.NewReader("payload")))

	clone, err := req.Clone()
	if err != nil {
		t.Fatalf("Clone() error = %v, want nil", err)
	}

	if clone.Host() != req.Host() {
		t.Errorf("clone.Host() = %q, want %q", clone.Host(), req.Host())
	}
	if clone.RemoteAddr() != req.RemoteAddr() {
		t.Errorf("clone.RemoteAddr() = %q, want %q", clone.RemoteAddr(), req.RemoteAddr())
	}
	if clone.URL == req.URL {
		t.Fatal("clone.URL shares the original pointer")
	}

	clone.Headers.Set("X-Test", "clone")
	clone.URL.Path = "/clone"
	cloneQuery := clone.URL.Query()
	cloneQuery.Set("q", "2")
	clone.URL.RawQuery = cloneQuery.Encode()

	if got := req.Headers.Get("X-Test"); got != "original" {
		t.Errorf("req.Headers.Get(X-Test) = %q, want %q", got, "original")
	}
	if req.URL.Path != "/original" {
		t.Errorf("req.URL.Path = %q, want %q", req.URL.Path, "/original")
	}
	if req.URL.RawQuery != "q=1" {
		t.Errorf("req.URL.RawQuery = %q, want %q", req.URL.RawQuery, "q=1")
	}

	req.Headers.Set("X-Test", "updated-original")
	req.URL.Path = "/updated-original"

	if got := clone.Headers.Get("X-Test"); got != "clone" {
		t.Errorf("clone.Headers.Get(X-Test) = %q, want %q", got, "clone")
	}
	if clone.URL.Path != "/clone" {
		t.Errorf("clone.URL.Path = %q, want %q", clone.URL.Path, "/clone")
	}
}

func TestRequest_Clone_AfterDisturbed_Fails(t *testing.T) {
	req := NewRequest("POST", &url.URL{Path: "/"})
	req.SetBody(io.NopCloser(strings.NewReader("payload")))

	_, _ = req.Bytes()

	_, err := req.Clone()
	if !errors.Is(err, ErrBodyConsumed) {
		t.Errorf("Clone() after Bytes() error = %v, want ErrBodyConsumed", err)
	}
}

func TestRequest_Clone_ReadErrorMarksOriginalConsumed(t *testing.T) {
	req := NewRequest("POST", &url.URL{Path: "/"})
	req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")

	reader := &cloneErrReader{}
	req.SetBody(reader)

	_, err := req.Clone()
	if err == nil {
		t.Fatal("Clone() error = nil, want wrapped io.ErrUnexpectedEOF")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("Clone() error = %v, want errors.Is(err, io.ErrUnexpectedEOF)", err)
	}
	if !reader.closed {
		t.Error("Clone() did not close the source body on failure")
	}
	if !req.BodyUsed() {
		t.Fatal("req.BodyUsed() = false after failed Clone(), want true")
	}

	_, err = req.Bytes()
	if !errors.Is(err, ErrBodyConsumed) {
		t.Errorf("req.Bytes() after failed Clone() error = %v, want ErrBodyConsumed", err)
	}

	_, err = req.FormData()
	if !errors.Is(err, ErrBodyConsumed) {
		t.Errorf("req.FormData() after failed Clone() error = %v, want ErrBodyConsumed", err)
	}
}

func TestRequest_Clone_NilBody(t *testing.T) {
	req := NewRequest("GET", &url.URL{Path: "/"})
	// No SetBody — Body is nil.

	clone, err := req.Clone()
	if err != nil {
		t.Fatalf("Clone() error = %v, want nil", err)
	}

	if req.BodyUsed() {
		t.Error("req.BodyUsed() = true after Clone() on nil body, want false")
	}
	if clone.BodyUsed() {
		t.Error("clone.BodyUsed() = true after Clone() on nil body, want false")
	}

	data, err := clone.Bytes()
	if err != nil {
		t.Fatalf("clone.Bytes() error = %v, want nil", err)
	}
	if len(data) != 0 {
		t.Errorf("clone.Bytes() = %v, want empty", data)
	}
}

func TestRequest_Clone_BoundedBody(t *testing.T) {
	t.Run("body within clone limit succeeds", func(t *testing.T) {
		// Use a body that fits within both Clone and Bytes limits (1 MiB).
		const size = 1 * 1024 * 1024
		body := strings.Repeat("x", size)
		req := NewRequest("POST", &url.URL{Path: "/"})
		req.SetBody(io.NopCloser(strings.NewReader(body)))

		clone, err := req.Clone()
		if err != nil {
			t.Fatalf("Clone() error = %v, want nil", err)
		}

		data, err := clone.Bytes()
		if err != nil {
			t.Fatalf("clone.Bytes() error = %v", err)
		}
		if len(data) != size {
			t.Errorf("clone.Bytes() len = %d, want %d", len(data), size)
		}
	})

	t.Run("body over clone limit returns ErrBodyTooLarge", func(t *testing.T) {
		body := strings.Repeat("x", defaultMaxCloneBytes+1)
		req := NewRequest("POST", &url.URL{Path: "/"})
		req.SetBody(io.NopCloser(strings.NewReader(body)))

		_, err := req.Clone()
		if !errors.Is(err, ErrBodyTooLarge) {
			t.Errorf("Clone() error = %v, want ErrBodyTooLarge", err)
		}
	})
}
