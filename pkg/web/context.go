package web

import (
	"context"
	"net/url"
)

// Context is the request-scoped runtime context passed to handlers and middleware.
// It owns the request, the underlying standard context for cancellation/deadlines,
// and all request-scoped params and values.
type Context struct {
	ctx     context.Context
	Request Request
	URL     *url.URL
	Method  string
	Headers Headers
	params  map[string]string
	values  map[any]any
}

// NewContext creates a request context from a parent context and Request.
//
// Callers should pass a non-nil parent context. If parent is nil, NewContext
// falls back to context.Background() defensively.
func NewContext(parent context.Context, req Request) *Context {
	if parent == nil {
		parent = context.Background()
	}
	return &Context{
		ctx:     parent,
		Request: req,
		URL:     req.URL,
		Method:  req.Method,
		Headers: req.Headers,
		params:  make(map[string]string),
		values:  make(map[any]any),
	}
}

// Context returns the underlying standard context carrying cancellation and deadlines.
func (c *Context) Context() context.Context {
	if c == nil || c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// SetContext replaces the underlying standard context. Use this to propagate
// a new context (e.g., one containing a span) into the request scope without
// replacing the full Context struct. The caller must not pass nil.
func (c *Context) SetContext(ctx context.Context) {
	if c == nil || ctx == nil {
		return
	}
	c.ctx = ctx
}

// Param returns the named route parameter from the request context.
// Returns "" if the parameter does not exist.
func (c *Context) Param(name string) string {
	if c == nil {
		return ""
	}
	return c.params[name]
}

// Params returns a copy of all route parameters.
func (c *Context) Params() map[string]string {
	if c == nil || len(c.params) == 0 {
		return nil
	}
	out := make(map[string]string, len(c.params))
	for k, v := range c.params {
		out[k] = v
	}
	return out
}

// SetParam stores a single named route parameter in the request context.
func (c *Context) SetParam(name, value string) {
	if c == nil {
		return
	}
	if c.params == nil {
		c.params = make(map[string]string)
	}
	c.params[name] = value
}

// SetParams stores multiple route parameters in the request context.
func (c *Context) SetParams(params map[string]string) {
	if c == nil || len(params) == 0 {
		return
	}
	if c.params == nil {
		c.params = make(map[string]string, len(params))
	}
	for k, v := range params {
		c.params[k] = v
	}
}

// Set stores an arbitrary value in the request context.
func (c *Context) Set(key, value any) {
	if c == nil {
		return
	}
	if c.values == nil {
		c.values = make(map[any]any)
	}
	c.values[key] = value
}

// Get retrieves an untyped value from the request context.
func (c *Context) Get(key any) (any, bool) {
	if c == nil || c.values == nil {
		return nil, false
	}
	v, ok := c.values[key]
	return v, ok
}

// Get retrieves a typed value from the request context.
// Returns the zero value and false if the key is missing or the type does not match.
func Get[T any](c *Context, key any) (T, bool) {
	var zero T
	if c == nil {
		return zero, false
	}
	v, ok := c.Get(key)
	if !ok {
		return zero, false
	}
	typed, ok := v.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}

// AcquireContext returns a pooled *Context initialized with the given parent
// context and request. It is more efficient than NewContext for hot paths.
//
// Callers should pass a non-nil parent context. If parent is nil,
// AcquireContext falls back to context.Background() defensively.
// The returned context must be released via ReleaseContext after the response
// is fully written — callers must not retain it beyond that point.
func AcquireContext(parent context.Context, req Request) *Context {
	c := contextPool.Get().(*Context)
	if parent == nil {
		parent = context.Background()
	}
	c.ctx = parent
	c.Request = req
	c.URL = req.URL
	c.Method = req.Method
	c.Headers = req.Headers
	// params and values stay nil until first write (lazy init already exists)
	return c
}

// ReleaseContext zeroes the context and returns it to the pool.
// Must be called after WriteHTTPResponse completes. The context must
// not be used after this call.
func ReleaseContext(c *Context) {
	if c == nil {
		return
	}
	c.ctx = nil
	c.Request = Request{}
	c.URL = nil
	c.Method = ""
	c.Headers = Headers{}
	c.params = nil
	c.values = nil
	contextPool.Put(c)
}
