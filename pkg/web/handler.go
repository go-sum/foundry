package web

import (
	"crypto/rand"
	"fmt"
	"net/http"
)

// Handler is the core request-handling function. It receives a context and
// returns a response value and an optional error. This is the foundational
// contract for all HTTP handling in the web package.
type Handler func(c *Context) (Response, error)

// Middleware wraps a Handler to add cross-cutting behavior.
// Middleware is applied outermost-first: Chain(h, A, B) calls A(B(h)).
type Middleware func(Handler) Handler

// Chain composes middleware around a handler. Middleware is applied such that
// the first middleware in the list is the outermost (runs first on request,
// last on response).
func Chain(h Handler, mw ...Middleware) Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// NotFoundHandler returns a Handler that always responds with 404 Not Found.
func NotFoundHandler() Handler {
	return func(_ *Context) (Response, error) {
		return Response{}, ErrNotFound("")
	}
}

// MethodNotAllowedHandler returns a Handler that responds with 405 Method Not Allowed.
func MethodNotAllowedHandler() Handler {
	return func(_ *Context) (Response, error) {
		return Response{}, &Error{Status: http.StatusMethodNotAllowed, Code: CodeMethodNotAllowed, Title: "Method Not Allowed"}
	}
}

// maxBodyKey is the context key used by WithMaxBody to store the per-route limit.
type maxBodyKey struct{}

// WithMaxBody returns a Middleware that marks a per-route body size limit on
// the context. The adapter already wraps Body with MaxBytesReader at 32 MiB;
// this middleware records a tighter per-route ceiling for body.go to check.
func WithMaxBody(maxBytes int64) Middleware {
	return func(next Handler) Handler {
		return func(c *Context) (Response, error) {
			if c.Request.Body != nil {
				c.Set(maxBodyKey{}, maxBytes)
			}
			return next(c)
		}
	}
}

// RequestIDKey is the context key for the request ID set by WithRequestID.
type RequestIDKey struct{}

// WithRequestID returns a Middleware that generates a unique request ID,
// stores it in the context for logging and tracing, and sets the X-Request-Id
// response header.
func WithRequestID() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) (Response, error) {
			id := generateRequestID()
			c.Set(RequestIDKey{}, id)
			resp, err := next(c)
			resp.Headers.Set("X-Request-Id", id)
			return resp, err
		}
	}
}

// RequestID returns the request ID from the context, or "" if not set.
func RequestID(c *Context) string {
	v, ok := c.Get(RequestIDKey{})
	if !ok {
		return ""
	}
	id, ok := v.(string)
	if !ok {
		return ""
	}
	return id
}

// CheckCancellation returns a Middleware that short-circuits the handler chain
// when the request context has been cancelled or its deadline exceeded. Place it
// at any position in the chain to guard the subsequent handlers against
// continued work after a client disconnect.
func CheckCancellation() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) (Response, error) {
			if err := c.Context().Err(); err != nil {
				return Response{}, err
			}
			return next(c)
		}
	}
}

func generateRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%x", b)
}
