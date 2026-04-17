// Package web provides Go types that directly model W3C Web API primitives
// (Request, Response, Headers) for building HTTP handlers that compile to both
// native Go and WebAssembly.
package web

import (
	"log/slog"
	"strings"
)

// Headers models the W3C Headers API. Keys are case-insensitive.
// The zero value is ready to use.
type Headers struct {
	values map[string][]string
}

// NewHeaders returns an initialized Headers.
func NewHeaders() Headers {
	return Headers{values: make(map[string][]string)}
}

// init lazily allocates the backing map on first write.
func (h *Headers) init() {
	if h.values == nil {
		h.values = make(map[string][]string)
	}
}

// key normalizes header names to lowercase per the W3C Headers spec.
func key(name string) string {
	return strings.ToLower(name)
}

// sanitizeHeaderValue strips all \r and \n characters from value.
// It logs a WARN via slog if any were found, since CRLF in header values
// is a header injection vector.
func sanitizeHeaderValue(name, value string) string {
	if !strings.ContainsAny(value, "\r\n") {
		return value
	}
	slog.Warn("web: stripped CRLF from header value",
		"header", name,
		"original_len", len(value))
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, value)
}

// sanitizeHeaderName strips all \r and \n characters from name.
// It logs a WARN via slog if any were found.
func sanitizeHeaderName(name string) string {
	if !strings.ContainsAny(name, "\r\n") {
		return name
	}
	slog.Warn("web: stripped CRLF from header name",
		"original", name)
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, name)
}

// forbiddenResponseHeaders are headers that should not be set by handlers
// because they are controlled by the transport layer.
var forbiddenResponseHeaders = map[string]bool{
	"transfer-encoding": true,
	"connection":        true,
	"keep-alive":        true,
	"upgrade":           true,
	"trailer":           true,
}

// IsForbiddenResponseHeader reports whether name is a transport-controlled
// header that handlers should not set.
func IsForbiddenResponseHeader(name string) bool {
	return forbiddenResponseHeaders[strings.ToLower(name)]
}

// IsForbiddenResponseHeaderForStatus returns true if the named header must not
// be set on a response with the given status code. For 101 Switching Protocols,
// Upgrade and Connection are allowed (WebSocket handshake requires them).
func IsForbiddenResponseHeaderForStatus(name string, status int) bool {
	if status == 101 {
		lower := strings.ToLower(name)
		// Only block genuine transport-control headers that are always illegal
		// for application code to set; Upgrade/Connection are needed for WS.
		switch lower {
		case "transfer-encoding", "keep-alive", "trailer", "te":
			return true
		}
		return false
	}
	return IsForbiddenResponseHeader(name)
}

// Get returns the first value for the given header name.
// Returns "" if the header does not exist.
func (h Headers) Get(name string) string {
	if h.values == nil {
		return ""
	}
	vals := h.values[key(name)]
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

// Set replaces all values for the given header name with a single value.
// CR and LF characters are stripped from both name and value.
func (h *Headers) Set(name, value string) {
	h.init()
	name = sanitizeHeaderName(name)
	value = sanitizeHeaderValue(name, value)
	h.values[key(name)] = []string{value}
}

// Append adds a value to the given header name without replacing existing values.
// CR and LF characters are stripped from both name and value.
func (h *Headers) Append(name, value string) {
	h.init()
	name = sanitizeHeaderName(name)
	value = sanitizeHeaderValue(name, value)
	k := key(name)
	h.values[k] = append(h.values[k], value)
}

// Delete removes all values for the given header name.
func (h *Headers) Delete(name string) {
	if h.values == nil {
		return
	}
	delete(h.values, key(name))
}

// Has reports whether the header exists.
func (h Headers) Has(name string) bool {
	if h.values == nil {
		return false
	}
	_, ok := h.values[key(name)]
	return ok
}

// Values returns all values for the given header name.
// Returns nil if the header does not exist.
func (h Headers) Values(name string) []string {
	if h.values == nil {
		return nil
	}
	values := h.values[key(name)]
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

// GetSetCookie returns all Set-Cookie header values as a slice.
// Unlike Get, this returns every value without joining, preserving semantics.
func (h Headers) GetSetCookie() []string {
	return h.Values("Set-Cookie")
}

// Entries returns a copy of all header entries.
func (h Headers) Entries() map[string][]string {
	if h.values == nil {
		return nil
	}
	out := make(map[string][]string, len(h.values))
	for k, v := range h.values {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// Clone returns a deep copy of the headers.
func (h Headers) Clone() Headers {
	entries := h.Entries()
	if entries == nil {
		return Headers{}
	}
	return Headers{values: entries}
}

// ForEach calls fn for each header entry.
func (h Headers) ForEach(fn func(name string, values []string)) {
	if h.values == nil {
		return
	}
	for k, v := range h.values {
		cp := make([]string, len(v))
		copy(cp, v)
		fn(k, cp)
	}
}
