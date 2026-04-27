// Package webtest provides test helper utilities for pkg/web.
package webtest

import (
	"io"
	"net/url"
	"strings"

	"github.com/go-sum/foundry/pkg/web"
)

// BuildRequest constructs a web.Request for use in handler tests.
func BuildRequest(method, rawURL string, opts ...func(*web.Request)) web.Request {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic("webtest.BuildRequest: invalid URL: " + err.Error())
	}
	req := web.NewRequest(method, u)
	for _, o := range opts {
		o(&req)
	}
	return req
}

// WithBody sets a string body on a request.
func WithBody(contentType, body string) func(*web.Request) {
	return func(r *web.Request) {
		r.SetBody(io.NopCloser(strings.NewReader(body)))
		r.Headers.Set("Content-Type", contentType)
	}
}

// WithHeader sets a header on a request.
func WithHeader(name, value string) func(*web.Request) {
	return func(r *web.Request) {
		r.Headers.Set(name, value)
	}
}
