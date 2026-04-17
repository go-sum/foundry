package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSentinelConstructors(t *testing.T) {
	tests := []struct {
		name       string
		err        *Error
		wantStatus int
		wantCode   Code
		wantTitle  string
	}{
		{
			name:       "ErrBadRequest",
			err:        ErrBadRequest("invalid input"),
			wantStatus: http.StatusBadRequest,
			wantCode:   CodeBadRequest,
			wantTitle:  "Bad Request",
		},
		{
			name:       "ErrNotFound",
			err:        ErrNotFound("user not found"),
			wantStatus: http.StatusNotFound,
			wantCode:   CodeNotFound,
			wantTitle:  "Not Found",
		},
		{
			name:       "ErrForbidden",
			err:        ErrForbidden("access denied"),
			wantStatus: http.StatusForbidden,
			wantCode:   CodeForbidden,
			wantTitle:  "Forbidden",
		},
		{
			name:       "ErrUnauthorized",
			err:        ErrUnauthorized("not logged in"),
			wantStatus: http.StatusUnauthorized,
			wantCode:   CodeUnauthorized,
			wantTitle:  "Unauthorized",
		},
		{
			name:       "ErrConflict",
			err:        ErrConflict("duplicate entry"),
			wantStatus: http.StatusConflict,
			wantCode:   CodeConflict,
			wantTitle:  "Conflict",
		},
		{
			name:       "ErrValidation",
			err:        ErrValidation("email is required"),
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   CodeValidation,
			wantTitle:  "Validation Failed",
		},
		{
			name:       "ErrInternal",
			err:        ErrInternal(errors.New("db connection lost")),
			wantStatus: http.StatusInternalServerError,
			wantCode:   CodeInternal,
			wantTitle:  "Internal Server Error",
		},
		{
			name:       "ErrUnavailable",
			err:        ErrUnavailable("try again later", errors.New("upstream timeout")),
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   CodeUnavailable,
			wantTitle:  "Service Unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", tt.err.Status, tt.wantStatus)
			}
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.wantCode)
			}
			if tt.err.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", tt.err.Title, tt.wantTitle)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "returns Message when present, even when cause is set",
			err:  ErrUnavailable("service down", errors.New("connection refused")),
			want: "service down",
		},
		{
			name: "returns Message when no cause",
			err:  ErrBadRequest("invalid email"),
			want: "invalid email",
		},
		{
			name: "falls back to Title when no cause and no message",
			err:  NewError(http.StatusTeapot, "teapot", "I Am A Teapot", "", nil),
			want: "I Am A Teapot",
		},
		{
			name: "falls back to Title when cause is set but Message is empty",
			err:  ErrInternal(errors.New("db failed")),
			want: "Internal Server Error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	t.Run("returns cause when present", func(t *testing.T) {
		cause := errors.New("root cause")
		e := ErrInternal(cause)
		if e.Unwrap() != cause {
			t.Errorf("Unwrap() = %v, want %v", e.Unwrap(), cause)
		}
	})

	t.Run("returns nil when no cause", func(t *testing.T) {
		e := ErrBadRequest("bad")
		if e.Unwrap() != nil {
			t.Errorf("Unwrap() = %v, want nil", e.Unwrap())
		}
	})

	t.Run("errors.Is works through wrapping", func(t *testing.T) {
		cause := errors.New("sentinel")
		e := ErrInternal(cause)
		wrapped := fmt.Errorf("handler: %w", e)
		if !errors.Is(wrapped, cause) {
			t.Error("errors.Is could not find cause through wrapped Error")
		}
	})
}

func TestErrorFrom(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantNil  bool
		wantCode Code
	}{
		{
			name:     "extracts Error from direct value",
			err:      ErrNotFound("missing"),
			wantNil:  false,
			wantCode: CodeNotFound,
		},
		{
			name:     "extracts Error from wrapped chain",
			err:      fmt.Errorf("outer: %w", ErrForbidden("no access")),
			wantNil:  false,
			wantCode: CodeForbidden,
		},
		{
			name:     "extracts Error from doubly wrapped chain",
			err:      fmt.Errorf("a: %w", fmt.Errorf("b: %w", ErrConflict("dup"))),
			wantNil:  false,
			wantCode: CodeConflict,
		},
		{
			name:    "returns nil for plain error",
			err:     errors.New("plain error"),
			wantNil: true,
		},
		{
			name:    "returns nil for nil error",
			err:     nil,
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrorFrom(tt.err)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ErrorFrom() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("ErrorFrom() = nil, want non-nil")
			}
			if got.Code != tt.wantCode {
				t.Errorf("ErrorFrom().Code = %q, want %q", got.Code, tt.wantCode)
			}
		})
	}
}

func TestError_PublicMessage(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "returns Message when present",
			err:  ErrBadRequest("invalid email format"),
			want: "invalid email format",
		},
		{
			name: "falls back to Title when Message is empty",
			err:  ErrInternal(errors.New("db crash")),
			want: "Internal Server Error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.PublicMessage()
			if got != tt.want {
				t.Errorf("PublicMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	cause := errors.New("root")
	e := NewError(418, "teapot", "I Am A Teapot", "short and stout", cause)

	if e.Status != 418 {
		t.Errorf("Status = %d, want 418", e.Status)
	}
	if e.Code != "teapot" {
		t.Errorf("Code = %q, want %q", e.Code, "teapot")
	}
	if e.Title != "I Am A Teapot" {
		t.Errorf("Title = %q, want %q", e.Title, "I Am A Teapot")
	}
	if e.Message != "short and stout" {
		t.Errorf("Message = %q, want %q", e.Message, "short and stout")
	}
	if e.Cause != cause {
		t.Errorf("Cause = %v, want %v", e.Cause, cause)
	}
}

func TestError_ImplementsErrorInterface(t *testing.T) {
	var err error = ErrBadRequest("test")
	if err.Error() != "test" {
		t.Errorf("error interface Error() = %q, want %q", err.Error(), "test")
	}
	// Cause does not leak through Error().
	var errWithCause error = ErrInternal(errors.New("secret cause"))
	if errWithCause.Error() == "secret cause" {
		t.Errorf("Error() leaked cause text: %q", errWithCause.Error())
	}
}

// ---------------------------------------------------------------------------
// Classify
// ---------------------------------------------------------------------------

func TestClassify(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		got := Classify(nil)
		if got != nil {
			t.Errorf("Classify(nil) = %+v, want nil", got)
		}
	})

	t.Run("*Error is returned as-is via ErrorFrom", func(t *testing.T) {
		orig := ErrBadRequest("x")
		got := Classify(orig)
		if got != orig {
			t.Errorf("Classify(*Error) returned a different pointer")
		}
		if got.Status != http.StatusBadRequest {
			t.Errorf("Status = %d, want %d", got.Status, http.StatusBadRequest)
		}
	})

	t.Run("wrapped ErrBodyTooLarge returns 413", func(t *testing.T) {
		got := Classify(fmt.Errorf("wrapped: %w", ErrBodyTooLarge))
		if got == nil {
			t.Fatal("Classify returned nil")
		}
		if got.Status != http.StatusRequestEntityTooLarge {
			t.Errorf("Status = %d, want %d", got.Status, http.StatusRequestEntityTooLarge)
		}
	})

	t.Run("wrapped context.DeadlineExceeded returns 499", func(t *testing.T) {
		got := Classify(fmt.Errorf("wrapped: %w", context.DeadlineExceeded))
		if got == nil {
			t.Fatal("Classify returned nil")
		}
		if got.Status != 499 {
			t.Errorf("Status = %d, want 499", got.Status)
		}
	})

	t.Run("context.Canceled wrapped returns 499", func(t *testing.T) {
		got := Classify(fmt.Errorf("wrapped: %w", context.Canceled))
		if got == nil {
			t.Fatal("Classify returned nil")
		}
		if got.Status != 499 {
			t.Errorf("Status = %d, want 499", got.Status)
		}
	})

	t.Run("unknown error returns 500", func(t *testing.T) {
		got := Classify(errors.New("unknown"))
		if got == nil {
			t.Fatal("Classify returned nil")
		}
		if got.Status != http.StatusInternalServerError {
			t.Errorf("Status = %d, want %d", got.Status, http.StatusInternalServerError)
		}
	})
}

// ---------------------------------------------------------------------------
// Fluent methods
// ---------------------------------------------------------------------------

func TestError_WithInstance(t *testing.T) {
	e := ErrNotFound("missing")
	got := e.WithInstance("/users/123")
	if got != e {
		t.Error("WithInstance did not return same pointer")
	}
	if e.Instance != "/users/123" {
		t.Errorf("Instance = %q, want %q", e.Instance, "/users/123")
	}
}

func TestError_WithMeta(t *testing.T) {
	e := ErrBadRequest("bad")
	got := e.WithMeta("field", "email")
	if got != e {
		t.Error("WithMeta did not return same pointer")
	}
	if e.Meta == nil {
		t.Fatal("Meta is nil after WithMeta")
	}
	if e.Meta["field"] != "email" {
		t.Errorf("Meta[field] = %v, want %q", e.Meta["field"], "email")
	}
}

func TestError_WithRetryAfter(t *testing.T) {
	e := ErrTooManyRequests(0)
	d := 3 * time.Second
	got := e.WithRetryAfter(d)
	if got != e {
		t.Error("WithRetryAfter did not return same pointer")
	}
	if e.RetryAfter != d {
		t.Errorf("RetryAfter = %v, want %v", e.RetryAfter, d)
	}
}

// ---------------------------------------------------------------------------
// ErrTooManyRequests
// ---------------------------------------------------------------------------

func TestErrTooManyRequests(t *testing.T) {
	e := ErrTooManyRequests(3 * time.Second)
	if e.Status != http.StatusTooManyRequests {
		t.Errorf("Status = %d, want %d", e.Status, http.StatusTooManyRequests)
	}
	if e.Code != CodeTooManyRequests {
		t.Errorf("Code = %q, want %q", e.Code, CodeTooManyRequests)
	}
	if e.RetryAfter != 3*time.Second {
		t.Errorf("RetryAfter = %v, want %v", e.RetryAfter, 3*time.Second)
	}
}

// ---------------------------------------------------------------------------
// ErrBadGateway
// ---------------------------------------------------------------------------

func TestErrBadGateway(t *testing.T) {
	cause := errors.New("upstream error")
	e := ErrBadGateway(cause)
	if e.Status != http.StatusBadGateway {
		t.Errorf("Status = %d, want %d", e.Status, http.StatusBadGateway)
	}
	if e.Code != CodeBadGateway {
		t.Errorf("Code = %q, want %q", e.Code, CodeBadGateway)
	}
	if e.Cause != cause {
		t.Errorf("Cause = %v, want %v", e.Cause, cause)
	}
}

// ---------------------------------------------------------------------------
// G11 — Error() returns Message, not Cause
// ---------------------------------------------------------------------------

func TestError_Error_G11(t *testing.T) {
	tests := []struct {
		name           string
		err            *Error
		wantText       string
		mustNotContain string
	}{
		{
			name:           "with cause set, Error() returns Message not cause text",
			err:            &Error{Message: "safe msg", Cause: errors.New("secret internal detail")},
			wantText:       "safe msg",
			mustNotContain: "secret internal detail",
		},
		{
			name:     "with cause but no Message, Error() returns Title",
			err:      &Error{Title: "Not Found", Cause: errors.New("db detail")},
			wantText: "Not Found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantText {
				t.Errorf("Error() = %q, want %q", got, tt.wantText)
			}
			if tt.mustNotContain != "" && strings.Contains(got, tt.mustNotContain) {
				t.Errorf("Error() leaked internal detail %q in output %q", tt.mustNotContain, got)
			}
		})
	}
}

func TestError_Error_G11_UnwrapPreservesCause(t *testing.T) {
	cause := errors.New("db detail")
	e := &Error{Title: "Not Found", Cause: cause}
	if !errors.Is(e, cause) {
		t.Error("errors.Is(e, cause) = false, want true — Unwrap must expose Cause")
	}
}

// ---------------------------------------------------------------------------
// G6 — New Code constants: Classify table extensions
// ---------------------------------------------------------------------------

func TestClassify_G6(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   Code
	}{
		{
			name:       "413 maps to CodePayloadTooLarge",
			err:        fmt.Errorf("outer: %w", ErrBodyTooLarge),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   CodePayloadTooLarge,
		},
		{
			name:       "415 maps to CodeUnsupportedMedia",
			err:        fmt.Errorf("outer: %w", ErrContentTypeMismatch),
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   CodeUnsupportedMedia,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.err)
			if got == nil {
				t.Fatal("Classify returned nil")
			}
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", got.Status, tt.wantStatus)
			}
			if got.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", got.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// G6 — New constructor functions: ErrMethodNotAllowed, ErrPayloadTooLarge,
//      ErrUnsupportedMedia
// ---------------------------------------------------------------------------

func TestErrConstructors_NewCodes(t *testing.T) {
	tests := []struct {
		name       string
		err        *Error
		wantStatus int
		wantCode   Code
	}{
		{
			name:       "ErrMethodNotAllowed",
			err:        ErrMethodNotAllowed("test"),
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   CodeMethodNotAllowed,
		},
		{
			name:       "ErrPayloadTooLarge",
			err:        ErrPayloadTooLarge("test"),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   CodePayloadTooLarge,
		},
		{
			name:       "ErrUnsupportedMedia",
			err:        ErrUnsupportedMedia("test"),
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   CodeUnsupportedMedia,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", tt.err.Status, tt.wantStatus)
			}
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.wantCode)
			}
			if tt.err.Message != "test" {
				t.Errorf("Message = %q, want %q", tt.err.Message, "test")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// G3 — ErrDependencyTimeout
// ---------------------------------------------------------------------------

func TestErrDependencyTimeout_Classify504(t *testing.T) {
	// ErrDependencyTimeout joined with a causal error wrapping context.DeadlineExceeded.
	// The ErrDependencyTimeout branch must fire BEFORE the bare context.DeadlineExceeded branch.
	cause := fmt.Errorf("store: read: %w", context.DeadlineExceeded)
	joined := errors.Join(ErrDependencyTimeout, cause)

	got := Classify(joined)
	if got == nil {
		t.Fatal("Classify returned nil")
	}
	if got.Status != http.StatusGatewayTimeout {
		t.Errorf("Status = %d, want %d (504)", got.Status, http.StatusGatewayTimeout)
	}
	if got.Code != CodeUnavailable {
		t.Errorf("Code = %q, want %q", got.Code, CodeUnavailable)
	}
}

func TestErrDependencyTimeout_IsUnwrappable(t *testing.T) {
	cause := fmt.Errorf("store: read: %w", context.DeadlineExceeded)
	joined := errors.Join(ErrDependencyTimeout, cause)

	if !errors.Is(joined, ErrDependencyTimeout) {
		t.Error("errors.Is(joined, ErrDependencyTimeout) = false, want true")
	}
	if !errors.Is(joined, context.DeadlineExceeded) {
		t.Error("errors.Is(joined, context.DeadlineExceeded) = false, want true — causal chain must be preserved")
	}
}

// ---------------------------------------------------------------------------
// G4 — ErrTransient
// ---------------------------------------------------------------------------

func TestErrTransient_IsUnwrappable(t *testing.T) {
	cause := fmt.Errorf("cache: get: %w", io.ErrUnexpectedEOF)
	joined := errors.Join(ErrTransient, cause)

	if !errors.Is(joined, ErrTransient) {
		t.Error("errors.Is(joined, ErrTransient) = false, want true")
	}
	if !errors.Is(joined, io.ErrUnexpectedEOF) {
		t.Error("errors.Is(joined, io.ErrUnexpectedEOF) = false, want true — causal chain must be preserved")
	}
}

func TestErrTransient_IsNotDependencyTimeout(t *testing.T) {
	if errors.Is(ErrTransient, ErrDependencyTimeout) {
		t.Error("errors.Is(ErrTransient, ErrDependencyTimeout) = true, want false — sentinels must be distinct")
	}
}
