package web

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var (
	ErrBodyConsumed       = errors.New("web: body already consumed")
	ErrEmptyBody          = errors.New("web: body is empty")
	ErrBodyTooLarge       = errors.New("web: request body too large")
	ErrFormFilesExceeded  = errors.New("web: too many uploaded files")
	ErrFormFieldsExceeded = errors.New("web: too many form fields")
	ErrFormFileTooLarge   = errors.New("web: uploaded file too large")
	ErrFormValueTooLarge  = errors.New("web: form value too large")

	// ErrContentTypeMismatch is returned when the request Content-Type does not
	// match what the handler expects.
	ErrContentTypeMismatch = errors.New("web: content type mismatch")

	// ErrMultipartExceeded is returned when a multipart upload exceeds a size limit.
	ErrMultipartExceeded = errors.New("web: multipart size limit exceeded")

	// ErrTrailerHeader is returned when a caller attempts to set a forbidden
	// trailer header.
	ErrTrailerHeader = errors.New("web: cannot set trailer header")

	// ErrSlowLoris is returned when a connection read is taking too long.
	ErrSlowLoris = errors.New("web: read timeout (possible slow-loris)")

	// ErrDependencyTimeout marks a timeout on a server-owned deadline against an
	// upstream or dependency call (DB, queue, external service). Wrap with errors.Join
	// so callers can distinguish server-owned timeouts from client-set deadlines.
	//
	//   cause := fmt.Errorf("store: read: %w", ctx.Err())
	//   return errors.Join(ErrDependencyTimeout, cause)
	ErrDependencyTimeout = errors.New("web: dependency timeout")

	// ErrTransient marks a failure as safely retryable at the calling boundary.
	// Attach alongside the causal error using errors.Join:
	//
	//   cause := fmt.Errorf("cache: get: %w", err)
	//   return errors.Join(ErrTransient, cause)
	ErrTransient = errors.New("web: transient failure")

	// ErrBreakerOpen marks a failure as originating from an open circuit breaker.
	// Attach alongside the causal error: errors.Join(ErrTransient, ErrBreakerOpen, cause).
	ErrBreakerOpen = errors.New("web: breaker open")
)

// Code identifies the category of a transport error.
type Code string

const (
	CodeBadRequest        Code = "bad_request"
	CodeNotFound          Code = "not_found"
	CodeForbidden         Code = "forbidden"
	CodeUnauthorized      Code = "unauthorized"
	CodeConflict          Code = "conflict"
	CodeValidation        Code = "validation_failed"
	CodeInternal          Code = "internal_error"
	CodeUnavailable       Code = "service_unavailable"
	CodeTooManyRequests   Code = "too_many_requests"
	CodeBadGateway        Code = "bad_gateway"
	CodeMethodNotAllowed    Code = "method_not_allowed"
	CodePayloadTooLarge     Code = "payload_too_large"
	CodeUnsupportedMedia    Code = "unsupported_media_type"
	CodeMisdirectedRequest  Code = "misdirected_request"
)

// DefaultTypeBase is the RFC 7807 "type" URI used when no TypeURI is set on an Error.
var DefaultTypeBase = "about:blank"

// Error is a transport-facing error with an HTTP status code, machine-readable
// code, human-readable title and message, and an optional wrapped cause.
type Error struct {
	Status     int
	Code       Code
	Title      string
	Message    string
	Cause      error
	Instance   string         // RFC 7807 "instance"; fallback to c.URL.Path
	TypeURI    string         // RFC 7807 "type"; fallback to DefaultTypeBase
	RetryAfter      time.Duration     // 0 means omit Retry-After header
	Meta            map[string]any    // extension members merged into the problem document
	// ResponseHeaders contains HTTP headers that must be set on the error response
	// (e.g., Allow for 405, WWW-Authenticate for 401). Not included in the problem document body.
	ResponseHeaders map[string]string
}

// Error returns the user-safe message. It prefers Message, then Title.
// Use Unwrap to access the causal error chain.
func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Title
}

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error {
	return e.Cause
}

// PublicMessage returns the user-safe message string.
func (e *Error) PublicMessage() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Title
}

// NewError constructs an Error with all fields.
func NewError(status int, code Code, title, message string, cause error) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Title:   title,
		Message: message,
		Cause:   cause,
	}
}

// ErrBadRequest returns a 400 Bad Request error.
func ErrBadRequest(message string) *Error {
	return NewError(http.StatusBadRequest, CodeBadRequest, "Bad Request", message, nil)
}

// ErrNotFound returns a 404 Not Found error.
func ErrNotFound(message string) *Error {
	return NewError(http.StatusNotFound, CodeNotFound, "Not Found", message, nil)
}

// ErrForbidden returns a 403 Forbidden error.
func ErrForbidden(message string) *Error {
	return NewError(http.StatusForbidden, CodeForbidden, "Forbidden", message, nil)
}

// ErrUnauthorized returns a 401 Unauthorized error.
func ErrUnauthorized(message string) *Error {
	return NewError(http.StatusUnauthorized, CodeUnauthorized, "Unauthorized", message, nil)
}

// ErrConflict returns a 409 Conflict error.
func ErrConflict(message string) *Error {
	return NewError(http.StatusConflict, CodeConflict, "Conflict", message, nil)
}

// ErrValidation returns a 422 Unprocessable Entity error.
func ErrValidation(message string) *Error {
	return NewError(http.StatusUnprocessableEntity, CodeValidation, "Validation Failed", message, nil)
}

// ErrInternal returns a 500 Internal Server Error wrapping the given cause.
func ErrInternal(cause error) *Error {
	return NewError(http.StatusInternalServerError, CodeInternal, "Internal Server Error", "", cause)
}

// ErrUnavailable returns a 503 Service Unavailable error.
func ErrUnavailable(message string, cause error) *Error {
	return NewError(http.StatusServiceUnavailable, CodeUnavailable, "Service Unavailable", message, cause)
}

// UnavailableHandler returns a Handler that always returns a 503 Service Unavailable error.
// Use it as a placeholder for features that are not yet available in the current configuration.
func UnavailableHandler(feature string) Handler {
	return func(*Context) (Response, error) {
		return Response{}, ErrUnavailable(feature+" feature unavailable", nil)
	}
}

// ErrBreakerOpenResponse returns a 503 Service Unavailable error indicating
// that the circuit breaker for an upstream is open.
func ErrBreakerOpenResponse(retryAfter time.Duration) *Error {
	e := NewError(http.StatusServiceUnavailable, CodeUnavailable, "Service Unavailable",
		"A dependency is temporarily unavailable. Please retry later.", nil)
	e.RetryAfter = retryAfter
	return e
}

// ErrorFrom extracts an *Error from an error chain using errors.As.
// Returns nil if no *Error is found.
func ErrorFrom(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// WithInstance sets the RFC 7807 "instance" field and returns the same *Error.
func (e *Error) WithInstance(s string) *Error {
	e.Instance = s
	return e
}

// WithMeta adds a single extension member to the problem document and returns
// the same *Error.
func (e *Error) WithMeta(k string, v any) *Error {
	if e.Meta == nil {
		e.Meta = make(map[string]any)
	}
	e.Meta[k] = v
	return e
}

// WithRetryAfter sets the Retry-After duration and returns the same *Error.
func (e *Error) WithRetryAfter(d time.Duration) *Error {
	e.RetryAfter = d
	return e
}

// WithHeader adds a response header to the error and returns the same *Error.
func (e *Error) WithHeader(name, value string) *Error {
	if e.ResponseHeaders == nil {
		e.ResponseHeaders = make(map[string]string)
	}
	e.ResponseHeaders[name] = value
	return e
}

// ErrTooManyRequests returns a 429 Too Many Requests error with the given
// retry-after duration.
func ErrTooManyRequests(retryAfter time.Duration) *Error {
	e := NewError(http.StatusTooManyRequests, CodeTooManyRequests, "Too Many Requests", "", nil)
	e.RetryAfter = retryAfter
	return e
}

// ErrBadGateway returns a 502 Bad Gateway error wrapping the given cause.
func ErrBadGateway(cause error) *Error {
	return NewError(http.StatusBadGateway, CodeBadGateway, "Bad Gateway", "", cause)
}

// ErrMethodNotAllowed returns a 405 Method Not Allowed error.
func ErrMethodNotAllowed(msg string) *Error {
	return NewError(http.StatusMethodNotAllowed, CodeMethodNotAllowed, "Method Not Allowed", msg, nil)
}

// ErrPayloadTooLarge returns a 413 Payload Too Large error.
func ErrPayloadTooLarge(msg string) *Error {
	return NewError(http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "Payload Too Large", msg, nil)
}

// ErrUnsupportedMedia returns a 415 Unsupported Media Type error.
func ErrUnsupportedMedia(msg string) *Error {
	return NewError(http.StatusUnsupportedMediaType, CodeUnsupportedMedia, "Unsupported Media Type", msg, nil)
}

// ErrMisdirectedRequest returns a 421 Misdirected Request error.
func ErrMisdirectedRequest(message string) *Error {
	return NewError(http.StatusMisdirectedRequest, CodeMisdirectedRequest, "Misdirected Request", message, nil)
}

// Classify maps a generic error to an *Error. The mapping order is:
//  1. Already an *Error — returned as-is.
//  2. errors.Is(ErrBodyTooLarge) → 413 Payload Too Large.
//  3. errors.Is(ErrContentTypeMismatch) → 415 Unsupported Media Type.
//  4. errors.Is(ErrBreakerOpen) → 503 Service Unavailable.
//  5. errors.Is(ErrDependencyTimeout) → 504 Gateway Timeout (server-owned deadline).
//  6. errors.Is(context.DeadlineExceeded) → 499 Request Canceled (non-fault client-context timeout).
//  7. errors.Is(context.Canceled) → 499 Request Canceled.
//  8. Anything else → 500 Internal Server Error.
func Classify(err error) *Error {
	if err == nil {
		return nil
	}
	if e := ErrorFrom(err); e != nil {
		return e
	}
	switch {
	case errors.Is(err, ErrBodyTooLarge):
		return NewError(http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "Payload Too Large", "", err)
	case errors.Is(err, ErrContentTypeMismatch):
		return NewError(http.StatusUnsupportedMediaType, CodeUnsupportedMedia, "Unsupported Media Type", "", err)
	case errors.Is(err, ErrBreakerOpen):
		return ErrBreakerOpenResponse(0)
	case errors.Is(err, ErrDependencyTimeout):
		return NewError(http.StatusGatewayTimeout, CodeUnavailable,
			"Gateway Timeout", "A dependency timed out. Please try again.", err)
	case errors.Is(err, context.DeadlineExceeded):
		return NewError(499, CodeUnavailable, "Request Canceled", "", err)
	case errors.Is(err, context.Canceled):
		return NewError(499, CodeUnavailable, "Request Canceled", "", err)
	default:
		return ErrInternal(err)
	}
}
