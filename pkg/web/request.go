package web

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
)

// defaultMaxCloneBytes is the maximum body size allowed for Clone().
// Bodies exceeding this limit cause Clone to return ErrBodyTooLarge.
const defaultMaxCloneBytes = 32 * 1024 * 1024 // 32 MiB

// Request models the W3C Fetch Request API. It represents an incoming HTTP request.
//
// Context is not embedded. It flows through the Handler signature
// func(c *Context) Response. In W3C terms, c.Context() is the Go
// equivalent of Request.signal (AbortSignal): cancellation propagates through
// c.Context().Done().
//
// Body is one-shot. The first call to Bytes, Text, JSON, FormData, or any
// direct read of Body disturbs the body; subsequent typed-method calls return
// ErrBodyConsumed. Use Clone to obtain an independent copy for peek-ahead
// middleware (e.g. CSRF token extraction before the handler reads the form).
// Clone isolates the body, headers, and URL so downstream inspection cannot
// mutate the original request.
type Request struct {
	Method  string
	URL     *url.URL
	Headers Headers
	Body    io.ReadCloser

	host       string
	remoteAddr string
	state      *bodyState
}

// NewRequest creates a Request with the given method and URL.
func NewRequest(method string, u *url.URL) Request {
	return Request{
		Method:  method,
		URL:     u,
		Headers: NewHeaders(),
		state:   &bodyState{},
	}
}

// SetBody sets the underlying body reader and installs body-use tracking.
// Intended for use by adapters. The body is wrapped in a tracker that flips
// BodyUsed on the first Read, matching the behaviour of the typed body methods.
func (r *Request) SetBody(body io.ReadCloser) {
	if r.state == nil {
		r.state = &bodyState{}
	}
	if body == nil {
		r.Body = nil
		return
	}
	r.Body = &trackedBody{rc: body, state: r.state}
}

// Clone returns an independent copy of the request with a fresh body.
// The body is buffered into memory; both the original and the clone can be
// read independently. Headers and URL are deep-copied so the clone can be
// mutated without affecting the original. Returns ErrBodyConsumed if the body
// has already been disturbed.
//
// If cloning fails while buffering the body, the original body is treated as
// disturbed and subsequent typed body reads return ErrBodyConsumed.
//
// Use Clone in peek-ahead middleware that must inspect the body before the
// downstream handler reads it:
//
//	peek, err := req.Clone()
//	if err == nil {
//	    fd, _ := peek.FormData()
//	    token := fd.Values.Get("csrf_token")
//	}
//	// req is untouched — downstream handler can read it normally.
func (r *Request) Clone() (Request, error) {
	if r.state != nil && r.state.bodyUsed {
		return Request{}, ErrBodyConsumed
	}

	hadBody, data, err := r.cloneBody()
	if err != nil {
		return Request{}, err
	}

	clone := Request{
		Method:     r.Method,
		URL:        cloneURL(r.URL),
		Headers:    r.Headers.Clone(),
		host:       r.host,
		remoteAddr: r.remoteAddr,
		state:      &bodyState{},
	}

	if hadBody {
		r.SetBody(io.NopCloser(bytes.NewReader(data)))
		clone.SetBody(io.NopCloser(bytes.NewReader(data)))
	}

	return clone, nil
}

func (r *Request) cloneBody() (bool, []byte, error) {
	if r.Body == nil {
		return false, nil, nil
	}
	if r.state == nil {
		r.state = &bodyState{}
	}

	raw := r.Body
	if tb, ok := raw.(*trackedBody); ok {
		raw = tb.rc
	}

	// Bounded read: detect overflow via LimitedReader.
	lr := &io.LimitedReader{R: raw, N: defaultMaxCloneBytes + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		r.state.bodyUsed = true
		_ = raw.Close()
		return true, nil, wrapBodyReadError("cloning body", err)
	}
	_ = raw.Close()
	if lr.N == 0 {
		r.state.bodyUsed = true
		return true, nil, fmt.Errorf("web: Clone: read body: %w", ErrBodyTooLarge)
	}

	return true, data, nil
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	clone := *u
	return &clone
}

// BodyUsed reports whether the body has been disturbed — either by a typed
// body method (Bytes, Text, JSON, FormData) or by a direct read of Body.
// Mirrors the W3C Fetch Request.bodyUsed property.
func (r Request) BodyUsed() bool {
	if r.state == nil {
		return false
	}
	return r.state.bodyUsed
}

// Host returns the host from the request, as set by the adapter.
func (r Request) Host() string {
	return r.host
}

// RemoteAddr returns the remote address, as set by the adapter.
func (r Request) RemoteAddr() string {
	return r.remoteAddr
}

// SetHost sets the host field. Intended for use by adapters.
func (r *Request) SetHost(host string) {
	r.host = host
}

// SetRemoteAddr sets the remote address field. Intended for use by adapters.
func (r *Request) SetRemoteAddr(addr string) {
	r.remoteAddr = addr
}

// Query returns the parsed query parameters from the request URL.
// Equivalent to req.URL.Query() — provided for convenience so handlers
// do not need to dereference URL directly.
func (r Request) Query() url.Values {
	if r.URL == nil {
		return url.Values{}
	}
	return r.URL.Query()
}
