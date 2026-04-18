package validate

import (
	"errors"
	"testing"

	validator "github.com/go-playground/validator/v10"
)

// validStruct is a test struct with all required fields present.
type validStruct struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

// missingRequired has a required field left empty.
type missingRequired struct {
	Name string `json:"name" validate:"required"`
}

// jsonTagStruct exercises json tag name resolution.
type jsonTagStruct struct {
	EmailAddress string `json:"email_address" validate:"required"`
}

// omitemptyStruct exercises the tag stripping for "name,omitempty".
type omitemptyStruct struct {
	Username string `json:"username,omitempty" validate:"required"`
}

func TestNew_ValidStructReturnsNil(t *testing.T) {
	v := New()
	input := validStruct{Name: "Alice", Email: "alice@example.com"}
	if err := v.Struct(input); err != nil {
		t.Errorf("expected nil for valid struct, got: %v", err)
	}
}

func TestNew_InvalidStructReturnsErrors(t *testing.T) {
	v := New()
	input := missingRequired{} // Name is empty

	err := v.Struct(input)
	if err == nil {
		t.Fatal("expected error for invalid struct, got nil")
	}

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		t.Errorf("expected validator.ValidationErrors, got %T: %v", err, err)
	}
}

// TestNew_JSONTagFieldName verifies that the field name reported in errors uses
// the json tag, not the Go struct field name.
func TestNew_JSONTagFieldName(t *testing.T) {
	v := New()
	input := jsonTagStruct{} // EmailAddress empty, required

	err := v.Struct(input)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected validator.ValidationErrors, got %T", err)
	}

	if len(ve) == 0 {
		t.Fatal("expected at least one field error")
	}

	if ve[0].Field() != "email_address" {
		t.Errorf("field name = %q, want %q", ve[0].Field(), "email_address")
	}
}

// TestNew_OmitemptyStripped verifies that "name,omitempty" in a json tag
// produces a field name of "username", not "username,omitempty".
func TestNew_OmitemptyStripped(t *testing.T) {
	v := New()
	// With WithRequiredStructEnabled, a required field that is zero-value must
	// produce an error regardless of omitempty in the json tag.
	input := omitemptyStruct{} // Username is empty

	err := v.Struct(input)
	if err == nil {
		t.Fatal("expected validation error for empty required field, got nil")
	}

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected validator.ValidationErrors, got %T", err)
	}

	if len(ve) == 0 {
		t.Fatal("expected at least one field error")
	}

	if ve[0].Field() != "username" {
		t.Errorf("field name = %q, want %q (omitempty suffix must be stripped)", ve[0].Field(), "username")
	}
}
