package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Response models the W3C Response API. It is a value type returned from handlers.
type Response struct {
	Status  int
	Headers Headers
	Body    io.ReadCloser
}

// Respond returns a Response with the given status and no body.
func Respond(status int) Response {
	return Response{
		Status:  status,
		Headers: NewHeaders(),
	}
}

// Text returns a plain text response.
func Text(status int, body string) Response {
	h := NewHeaders()
	h.Set("Content-Type", "text/plain; charset=UTF-8")
	return Response{
		Status:  status,
		Headers: h,
		Body:    io.NopCloser(strings.NewReader(body)),
	}
}

// HTML returns an HTML response from a string.
func HTML(status int, body string) Response {
	h := NewHeaders()
	h.Set("Content-Type", "text/html; charset=UTF-8")
	return Response{
		Status:  status,
		Headers: h,
		Body:    io.NopCloser(strings.NewReader(body)),
	}
}

// HTMLBytes returns an HTML response from a byte slice.
func HTMLBytes(status int, body []byte) Response {
	h := NewHeaders()
	h.Set("Content-Type", "text/html; charset=UTF-8")
	return Response{
		Status:  status,
		Headers: h,
		Body:    io.NopCloser(bytes.NewReader(body)),
	}
}

// HTMLReader returns an HTML response from a reader.
func HTMLReader(status int, body io.ReadCloser) Response {
	h := NewHeaders()
	h.Set("Content-Type", "text/html; charset=UTF-8")
	return Response{
		Status:  status,
		Headers: h,
		Body:    body,
	}
}

// JSON returns a JSON response that encodes v lazily in a goroutine.
// The response body is a streaming io.Pipe reader; encoding errors surface
// as an abrupt body close after headers have already been sent.
func JSON(status int, v any) Response {
	pr, pw := io.Pipe()
	Go(nil, "response.json", func() { pw.CloseWithError(json.NewEncoder(pw).Encode(v)) })
	h := NewHeaders()
	h.Set("Content-Type", "application/json")
	return Response{Status: status, Headers: h, Body: pr}
}

// StreamJSON returns a JSON response that encodes v lazily in a goroutine.
// The response body is a streaming io.Pipe reader; encoding errors surface
// as an abrupt body close after headers have already been sent.
// Use this when v is large or when avoiding a full marshal-to-buffer allocation matters.
// For small, error-safe payloads, JSON is identical in behavior.
func StreamJSON(status int, v any) Response {
	return JSON(status, v)
}

// Problem renders an RFC 7807 application/problem+json response from the
// given *Error. The context c may be nil (e.g., in tests). When c is non-nil,
// the request path is used as the "instance" fallback and the request ID is
// included in the document.
//
// Key behaviors:
//   - "type" is e.TypeURI or DefaultTypeBase when empty.
//   - "instance" is e.Instance, or c.URL.Path when e.Instance is "".
//   - "detail" is included only for status < 500 and only when e.Message != "".
//   - All e.Meta entries are merged at the top level.
//   - Retry-After header is set when e.RetryAfter > 0.
//   - Body is buffered; no io.Pipe.
func Problem(c *Context, e *Error) Response {
	doc := map[string]any{
		"type":   e.TypeURI,
		"title":  e.Title,
		"status": e.Status,
	}
	if doc["type"] == "" {
		doc["type"] = DefaultTypeBase
	}

	instance := e.Instance
	if instance == "" && c != nil && c.URL != nil {
		instance = c.URL.Path
	}
	if instance != "" {
		doc["instance"] = instance
	}

	if c != nil {
		if rid := RequestID(c); rid != "" {
			doc["request_id"] = rid
		}
	}

	if e.Code != "" {
		doc["code"] = string(e.Code)
	}

	// Only expose detail for client errors; never leak cause for 5xx.
	if e.Status < 500 && e.Message != "" {
		doc["detail"] = e.Message
	}

	for k, v := range e.Meta {
		doc[k] = v
	}

	if e.RetryAfter > 0 {
		doc["retry_after"] = int(e.RetryAfter.Seconds())
	}

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(doc)

	h := NewHeaders()
	h.Set("Content-Type", "application/problem+json")
	if e.RetryAfter > 0 {
		h.Set("Retry-After", strconv.Itoa(int(e.RetryAfter.Seconds())))
	}
	return Response{
		Status:  e.Status,
		Headers: h,
		Body:    io.NopCloser(&buf),
	}
}

// Redirect returns a redirect response with the given status and location.
func Redirect(status int, url string) Response {
	h := NewHeaders()
	h.Set("Location", url)
	return Response{
		Status:  status,
		Headers: h,
	}
}

// SeeOther returns a 303 See Other redirect response.
func SeeOther(url string) Response {
	return Redirect(http.StatusSeeOther, url)
}
