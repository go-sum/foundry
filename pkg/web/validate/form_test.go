package validate

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
)

// loginInput is used for JSON and form binding tests.
type loginInput struct {
	Email    string `json:"email"    form:"email"    validate:"required,email"`
	Password string `json:"password" form:"password" validate:"required"`
}

// countInput has an int64 field to exercise conversion errors.
type countInput struct {
	Count int64 `form:"count" validate:"required"`
}

// buildJSONRequest creates a web.Request with application/json content.
func buildJSONRequest(body string) web.Request {
	u := &url.URL{Path: "/"}
	req := web.NewRequest("POST", u)
	req.SetBody(io.NopCloser(strings.NewReader(body)))
	req.Headers.Set("Content-Type", "application/json")
	return req
}

// buildFormRequest creates a web.Request with application/x-www-form-urlencoded content.
func buildFormRequest(values url.Values) web.Request {
	u := &url.URL{Path: "/"}
	req := web.NewRequest("POST", u)
	body := values.Encode()
	req.SetBody(io.NopCloser(strings.NewReader(body)))
	req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// buildMultipartRequest creates a web.Request with multipart/form-data content.
func buildMultipartRequest(fields map[string]string) web.Request {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	_ = w.Close()

	u := &url.URL{Path: "/"}
	req := web.NewRequest("POST", u)
	req.SetBody(io.NopCloser(bytes.NewReader(buf.Bytes())))
	req.Headers.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestBind_JSONHappy(t *testing.T) {
	v := New()
	req := buildJSONRequest(`{"email":"user@example.com","password":"secret"}`)

	var dest loginInput
	err := Bind(v, req, &dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", dest.Email, "user@example.com")
	}
	if dest.Password != "secret" {
		t.Errorf("Password = %q, want %q", dest.Password, "secret")
	}
}

func TestBind_JSONValidationFailure(t *testing.T) {
	v := New()
	// Valid JSON but missing required "password" field.
	req := buildJSONRequest(`{"email":"user@example.com"}`)

	var dest loginInput
	err := Bind(v, req, &dest)
	if err == nil {
		t.Fatal("expected error for missing required field, got nil")
	}

	var we *web.Error
	if !errors.As(err, &we) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if we.Status != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want 422", we.Status)
	}
	if we.Meta == nil || we.Meta["fields"] == nil {
		t.Error("expected Meta[\"fields\"] to be set")
	}
}

func TestBind_JSONMalformed(t *testing.T) {
	v := New()
	req := buildJSONRequest(`{not valid json}`)

	var dest loginInput
	err := Bind(v, req, &dest)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestBind_FormHappy(t *testing.T) {
	v := New()
	req := buildFormRequest(url.Values{
		"email":    {"user@example.com"},
		"password": {"secret"},
	})

	var dest loginInput
	err := Bind(v, req, &dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", dest.Email, "user@example.com")
	}
	if dest.Password != "secret" {
		t.Errorf("Password = %q, want %q", dest.Password, "secret")
	}
}

func TestBind_FormUnknownKeysIgnored(t *testing.T) {
	v := New()
	req := buildFormRequest(url.Values{
		"email":    {"user@example.com"},
		"password": {"secret"},
		"extra":    {"ignored"},
	})

	var dest loginInput
	err := Bind(v, req, &dest)
	if err != nil {
		t.Fatalf("unexpected error for extra form keys: %v", err)
	}
}

func TestBind_FormConversionError(t *testing.T) {
	v := New()
	req := buildFormRequest(url.Values{
		"count": {"abc"}, // cannot convert to int64
	})

	var dest countInput
	err := Bind(v, req, &dest)
	if err == nil {
		t.Fatal("expected error for type conversion failure, got nil")
	}
}

func TestBind_FormValidationFailure(t *testing.T) {
	v := New()
	// Parses fine but missing required "password".
	req := buildFormRequest(url.Values{
		"email": {"user@example.com"},
	})

	var dest loginInput
	err := Bind(v, req, &dest)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var we *web.Error
	if !errors.As(err, &we) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if we.Status != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want 422", we.Status)
	}
	if we.Meta == nil || we.Meta["fields"] == nil {
		t.Error("expected Meta[\"fields\"] to be set")
	}
}

func TestBind_MultipartHappy(t *testing.T) {
	v := New()
	req := buildMultipartRequest(map[string]string{
		"email":    "user@example.com",
		"password": "secret",
	})

	var dest loginInput
	err := Bind(v, req, &dest)
	if err != nil {
		t.Fatalf("unexpected error for multipart request: %v", err)
	}
	if dest.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", dest.Email, "user@example.com")
	}
	if dest.Password != "secret" {
		t.Errorf("Password = %q, want %q", dest.Password, "secret")
	}
}

func TestBind_UnsupportedMedia(t *testing.T) {
	v := New()
	u := &url.URL{Path: "/"}
	req := web.NewRequest("POST", u)
	req.SetBody(io.NopCloser(strings.NewReader("some body")))
	req.Headers.Set("Content-Type", "text/plain")

	var dest loginInput
	err := Bind(v, req, &dest)
	if err == nil {
		t.Fatal("expected error for unsupported media type, got nil")
	}

	var we *web.Error
	if !errors.As(err, &we) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if we.Status != http.StatusUnsupportedMediaType {
		t.Errorf("Status = %d, want 415", we.Status)
	}
}

func TestBind_MissingContentType(t *testing.T) {
	v := New()
	u := &url.URL{Path: "/"}
	req := web.NewRequest("POST", u)
	req.SetBody(io.NopCloser(strings.NewReader("body")))
	// No Content-Type header set.

	var dest loginInput
	err := Bind(v, req, &dest)
	if err == nil {
		t.Fatal("expected error for missing Content-Type, got nil")
	}

	var we *web.Error
	if !errors.As(err, &we) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if we.Status != http.StatusUnsupportedMediaType {
		t.Errorf("Status = %d, want 415", we.Status)
	}
}

func TestBind_BodyAlreadyConsumed(t *testing.T) {
	v := New()
	req := buildJSONRequest(`{"email":"user@example.com","password":"secret"}`)

	// Consume the body first.
	var other loginInput
	_ = req.JSON(&other)

	// Now Bind should fail because body is already consumed.
	var dest loginInput
	err := Bind(v, req, &dest)
	if err == nil {
		t.Fatal("expected error for already-consumed body, got nil")
	}
	if !errors.Is(err, web.ErrBodyConsumed) {
		t.Errorf("expected ErrBodyConsumed, got: %v", err)
	}
}

