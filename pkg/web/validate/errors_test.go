package validate

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/go-sum/web"
)

// compile-time assertion: Errors implements the error interface.
var _ error = Errors{}

func TestErrors_Error_FormatsCorrectly(t *testing.T) {
	errs := Errors{
		{Field: "name", Tag: "required", Message: "is required"},
		{Field: "email", Tag: "email", Message: "must be a valid email address"},
	}
	got := errs.Error()
	if !strings.Contains(got, "field=name") {
		t.Errorf("Error() missing field=name: %q", got)
	}
	if !strings.Contains(got, "tag=required") {
		t.Errorf("Error() missing tag=required: %q", got)
	}
	if !strings.Contains(got, "field=email") {
		t.Errorf("Error() missing field=email: %q", got)
	}
	if !strings.Contains(got, "tag=email") {
		t.Errorf("Error() missing tag=email: %q", got)
	}
	if !strings.Contains(got, "2 validation error(s)") {
		t.Errorf("Error() missing count prefix: %q", got)
	}
}

func TestErrors_ImplementsErrorInterface(t *testing.T) {
	var e Errors = Errors{{Field: "x", Tag: "required", Message: "is required"}}
	var _ error = e // compile-time
	if e.Error() == "" {
		t.Error("Error() returned empty string")
	}
}

func TestErrors_ToWebError_Status422(t *testing.T) {
	errs := Errors{{Field: "name", Tag: "required", Message: "is required"}}
	we := errs.ToWebError()
	if we == nil {
		t.Fatal("ToWebError() returned nil")
	}
	if we.Status != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want %d", we.Status, http.StatusUnprocessableEntity)
	}
}

func TestErrors_ToWebError_CodeValidation(t *testing.T) {
	errs := Errors{{Field: "name", Tag: "required", Message: "is required"}}
	we := errs.ToWebError()
	if we == nil {
		t.Fatal("ToWebError() returned nil")
	}
	if we.Code != web.CodeValidation {
		t.Errorf("Code = %q, want %q", we.Code, web.CodeValidation)
	}
}

func TestErrors_ToWebError_MetaFields(t *testing.T) {
	errs := Errors{
		{Field: "name", Tag: "required", Message: "is required"},
		{Field: "email", Tag: "email", Message: "must be a valid email address"},
	}
	we := errs.ToWebError()
	if we == nil {
		t.Fatal("ToWebError() returned nil")
	}
	if we.Meta == nil {
		t.Fatal("Meta is nil")
	}
	raw, ok := we.Meta["fields"]
	if !ok {
		t.Fatal("Meta[\"fields\"] missing")
	}
	got, ok := raw.(Errors)
	if !ok {
		t.Fatalf("Meta[\"fields\"] type = %T, want Errors", raw)
	}
	if !reflect.DeepEqual(got, errs) {
		t.Errorf("Meta[\"fields\"] = %v, want %v", got, errs)
	}
}

// buildValidationError creates a real validator.ValidationErrors by running
// validation against a struct with the given failing field.
func buildValidationError(t *testing.T, tag string) error {
	t.Helper()
	v := New()

	switch tag {
	case "required":
		type s struct {
			F string `json:"f" validate:"required"`
		}
		return v.Struct(s{})
	case "email":
		type s struct {
			F string `json:"f" validate:"required,email"`
		}
		return v.Struct(s{F: "not-an-email"})
	case "min":
		type s struct {
			F string `json:"f" validate:"min=5"`
		}
		return v.Struct(s{F: "ab"})
	case "max":
		type s struct {
			F string `json:"f" validate:"max=3"`
		}
		return v.Struct(s{F: "toolong"})
	case "len":
		type s struct {
			F string `json:"f" validate:"len=4"`
		}
		return v.Struct(s{F: "ab"})
	case "oneof":
		type s struct {
			F string `json:"f" validate:"oneof=a b c"`
		}
		return v.Struct(s{F: "z"})
	case "url":
		type s struct {
			F string `json:"f" validate:"url"`
		}
		return v.Struct(s{F: "not a url"})
	case "uuid":
		type s struct {
			F string `json:"f" validate:"uuid"`
		}
		return v.Struct(s{F: "not-a-uuid"})
	default:
		t.Fatalf("buildValidationError: unsupported tag %q", tag)
		return nil
	}
}

func TestFromValidator_ValidatorErrors(t *testing.T) {
	v := New()
	type s struct {
		Name string `json:"name" validate:"required"`
	}
	verr := v.Struct(s{})
	if verr == nil {
		t.Fatal("expected validation error, got nil")
	}

	errs, ok := FromValidator(verr)
	if !ok {
		t.Fatal("FromValidator returned ok=false for a validation error")
	}
	if len(errs) == 0 {
		t.Fatal("FromValidator returned empty Errors slice")
	}
	if errs[0].Field != "name" {
		t.Errorf("Field = %q, want %q", errs[0].Field, "name")
	}
	if errs[0].Tag != "required" {
		t.Errorf("Tag = %q, want %q", errs[0].Tag, "required")
	}
	if errs[0].Message == "" {
		t.Error("Message is empty")
	}
}

func TestFromValidator_NonValidationError(t *testing.T) {
	errs, ok := FromValidator(io.EOF)
	if ok {
		t.Error("FromValidator with io.EOF should return ok=false")
	}
	if errs != nil {
		t.Errorf("FromValidator with io.EOF should return nil Errors, got %v", errs)
	}
}

func TestToWebError_NilWhenNotValidation(t *testing.T) {
	if got := ToWebError(io.EOF); got != nil {
		t.Errorf("ToWebError(io.EOF) = %v, want nil", got)
	}
	if got := ToWebError(nil); got != nil {
		t.Errorf("ToWebError(nil) = %v, want nil", got)
	}
}

func TestToWebError_ReturnsWebErrorWhenValidation(t *testing.T) {
	v := New()
	type s struct {
		Email string `json:"email" validate:"required,email"`
	}
	verr := v.Struct(s{})
	if verr == nil {
		t.Fatal("expected validation error")
	}

	we := ToWebError(verr)
	if we == nil {
		t.Fatal("ToWebError returned nil for a validation error")
	}
	if we.Status != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want 422", we.Status)
	}
}

// TestPhraseTranslation verifies that each supported tag produces a human-
// readable phrase with parameter substitution where applicable.
func TestPhraseTranslation(t *testing.T) {
	tests := []struct {
		tag            string
		wantSubstrings []string
		notWant        string // the raw tag must not appear as the entire message
	}{
		{tag: "required", wantSubstrings: []string{"required"}},
		{tag: "email", wantSubstrings: []string{"email"}},
		{tag: "min", wantSubstrings: []string{"5"}},     // param is 5
		{tag: "max", wantSubstrings: []string{"3"}},     // param is 3
		{tag: "len", wantSubstrings: []string{"4"}},     // param is 4
		{tag: "oneof", wantSubstrings: []string{"a b c"}}, // param
		{tag: "url", wantSubstrings: []string{"URL"}},
		{tag: "uuid", wantSubstrings: []string{"UUID"}},
	}

	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			verr := buildValidationError(t, tc.tag)
			if verr == nil {
				t.Fatalf("tag %q: expected validation error, got nil", tc.tag)
			}
			errs, ok := FromValidator(verr)
			if !ok {
				t.Fatalf("tag %q: FromValidator returned ok=false", tc.tag)
			}
			if len(errs) == 0 {
				t.Fatalf("tag %q: FromValidator returned empty slice", tc.tag)
			}

			msg := errs[0].Message
			if msg == "" {
				t.Errorf("tag %q: Message is empty", tc.tag)
			}
			// The raw tag alone must not be the entire message (i.e., phrase was applied).
			if msg == tc.tag {
				t.Errorf("tag %q: Message equals raw tag — phrase was not applied", tc.tag)
			}
			for _, want := range tc.wantSubstrings {
				if !strings.Contains(msg, want) {
					t.Errorf("tag %q: Message %q does not contain %q", tc.tag, msg, want)
				}
			}
		})
	}
}

// TestErrors_Error_SingleError verifies formatting for a single field error.
func TestErrors_Error_SingleError(t *testing.T) {
	errs := Errors{{Field: "username", Tag: "required", Message: "is required"}}
	got := errs.Error()
	want := "1 validation error(s): field=username tag=required"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestFromValidator_NilError verifies FromValidator with nil returns (nil, false).
func TestFromValidator_NilError(t *testing.T) {
	errs, ok := FromValidator(nil)
	if ok {
		t.Error("FromValidator(nil) returned ok=true")
	}
	if errs != nil {
		t.Errorf("FromValidator(nil) returned non-nil Errors: %v", errs)
	}
}

// TestFromValidator_PlainError ensures a plain error (non-validation) returns (nil, false).
func TestFromValidator_PlainError(t *testing.T) {
	errs, ok := FromValidator(errors.New("plain error"))
	if ok {
		t.Error("FromValidator with plain error returned ok=true")
	}
	if errs != nil {
		t.Errorf("FromValidator with plain error returned non-nil Errors: %v", errs)
	}
}
