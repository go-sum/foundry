package form_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-sum/componentry/form"
)

type sampleForm struct {
	Name  string
	Email string
}

func noopValidate(_ any) error { return nil }

func errValidate(msg string) func(any) error {
	return func(_ any) error { return errors.New(msg) }
}

func TestFormState_NewIsUnsubmitted(t *testing.T) {
	s := form.NewFormState()
	if s.IsSubmitted() {
		t.Error("expected IsSubmitted=false before Bind")
	}
	if s.IsValid() {
		t.Error("expected IsValid=false before Bind")
	}
}

func TestFormState_BindValid(t *testing.T) {
	s := form.NewFormState()
	dest := &sampleForm{}
	s.Bind(dest, noopValidate)

	if !s.IsSubmitted() {
		t.Error("expected IsSubmitted=true after Bind")
	}
	if !s.IsValid() {
		t.Error("expected IsValid=true when validate returns nil")
	}
	if len(s.GetFormErrors()) != 0 {
		t.Errorf("expected no form errors, got: %v", s.GetFormErrors())
	}
}

func TestFormState_BindValidationError(t *testing.T) {
	s := form.NewFormState()
	dest := &sampleForm{}
	s.Bind(dest, errValidate("validation failed"))

	if !s.IsSubmitted() {
		t.Error("expected IsSubmitted=true after Bind")
	}
	if s.IsValid() {
		t.Error("expected IsValid=false when validate returns error")
	}
	errs := s.GetFormErrors()
	if len(errs) != 1 {
		t.Fatalf("expected 1 form error, got %d", len(errs))
	}
	if errs[0] != "validation failed" {
		t.Errorf("expected form error %q, got %q", "validation failed", errs[0])
	}
}

func TestFormState_SetFormError(t *testing.T) {
	s := form.NewFormState()
	s.SetFormError("first error")
	s.SetFormError("second error")

	errs := s.GetFormErrors()
	if len(errs) != 2 {
		t.Fatalf("expected 2 form errors, got %d", len(errs))
	}
	if errs[0] != "first error" {
		t.Errorf("expected %q, got %q", "first error", errs[0])
	}
	if errs[1] != "second error" {
		t.Errorf("expected %q, got %q", "second error", errs[1])
	}
}

func TestFormState_SetFieldErrors(t *testing.T) {
	s := form.NewFormState()
	s.SetFieldErrors(map[string][]string{
		"email": {"Email is required", "Must be a valid email"},
		"name":  {"Name is required"},
	})

	if !s.FieldHasErrors("email") {
		t.Error("expected email to have errors")
	}
	emailErrs := s.GetFieldErrors("email")
	if len(emailErrs) != 2 {
		t.Fatalf("expected 2 email errors, got %d", len(emailErrs))
	}
	if emailErrs[0] != "Email is required" {
		t.Errorf("expected %q, got %q", "Email is required", emailErrs[0])
	}
	if emailErrs[1] != "Must be a valid email" {
		t.Errorf("expected %q, got %q", "Must be a valid email", emailErrs[1])
	}

	if !s.FieldHasErrors("name") {
		t.Error("expected name to have errors")
	}
	if s.FieldHasErrors("password") {
		t.Error("expected password to have no errors")
	}
}

func TestFormState_FieldHasErrors_WithBindError(t *testing.T) {
	s := form.NewFormState()
	dest := &sampleForm{}
	s.Bind(dest, errValidate("bad input"))

	// The error goes to form-level ("_"), not a specific field
	if s.FieldHasErrors("name") {
		t.Error("expected no field errors on 'name'")
	}
	if !s.FieldHasErrors("_") {
		t.Error("expected form-level error under '_'")
	}
}

func TestFormState_GetFieldErrors_Empty(t *testing.T) {
	s := form.NewFormState()
	errs := s.GetFieldErrors("nonexistent")
	if len(errs) != 0 {
		t.Errorf("expected nil/empty for unknown field, got %v", errs)
	}
}

func TestFormState_MultipleErrors_PreserveOrder(t *testing.T) {
	s := form.NewFormState()
	s.SetFieldErrors(map[string][]string{
		"email": {"First", "Second", "Third"},
	})
	errs := s.GetFieldErrors("email")
	if len(errs) != 3 {
		t.Fatalf("expected 3 errors, got %d", len(errs))
	}
	for i, want := range []string{"First", "Second", "Third"} {
		if errs[i] != want {
			t.Errorf("index %d: expected %q, got %q", i, want, errs[i])
		}
	}
}

func TestFormState_IsValid_RequiresSubmitted(t *testing.T) {
	s := form.NewFormState()
	// Not yet submitted — IsValid must be false even with no errors
	if s.IsValid() {
		t.Error("IsValid should be false when not submitted")
	}
	s.Bind(&sampleForm{}, noopValidate)
	if !s.IsValid() {
		t.Error("IsValid should be true after successful Bind")
	}
}

func TestFormState_GetErrors(t *testing.T) {
	s := form.NewFormState()
	s.SetFieldErrors(map[string][]string{
		"email": {"required"},
		"name":  {"too short"},
	})
	errs := s.GetErrors()
	if errs == nil {
		t.Fatal("GetErrors: expected non-nil map")
	}
	if len(errs["email"]) != 1 || errs["email"][0] != "required" {
		t.Errorf("GetErrors email: got %v, want [required]", errs["email"])
	}
	if len(errs["name"]) != 1 || errs["name"][0] != "too short" {
		t.Errorf("GetErrors name: got %v, want [too short]", errs["name"])
	}
}

func TestFormState_GetErrors_empty(t *testing.T) {
	s := form.NewFormState()
	errs := s.GetErrors()
	if errs == nil {
		t.Fatal("GetErrors empty: expected non-nil map (empty)")
	}
	if len(errs) != 0 {
		t.Errorf("GetErrors empty: expected empty map, got %v", errs)
	}
}

func TestFormState_SetFieldError(t *testing.T) {
	s := form.NewFormState()
	s.SetFieldError("username", "too short")
	s.SetFieldError("username", "no spaces allowed")

	errs := s.GetFieldErrors("username")
	if len(errs) != 2 {
		t.Fatalf("SetFieldError: expected 2 errors, got %d: %v", len(errs), errs)
	}
	if errs[0] != "too short" {
		t.Errorf("SetFieldError[0]: got %q, want %q", errs[0], "too short")
	}
	if errs[1] != "no spaces allowed" {
		t.Errorf("SetFieldError[1]: got %q, want %q", errs[1], "no spaces allowed")
	}
}

func TestFormState_SetFieldError_differentFields(t *testing.T) {
	s := form.NewFormState()
	s.SetFieldError("email", "invalid format")
	s.SetFieldError("phone", "digits only")

	if !s.FieldHasErrors("email") {
		t.Error("SetFieldError: email should have errors")
	}
	if !s.FieldHasErrors("phone") {
		t.Error("SetFieldError: phone should have errors")
	}
	if s.FieldHasErrors("name") {
		t.Error("SetFieldError: name should not have errors")
	}
}

func TestFormState_MarshalJSON(t *testing.T) {
	s := form.NewFormState()
	s.Bind(&sampleForm{}, noopValidate)
	s.SetFieldErrors(map[string][]string{
		"email": {"invalid"},
	})
	s.SetFormError("form failure")

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := `{"submitted":true,"errors":{"_":["form failure"],"email":["invalid"]}}`
	if got := string(data); got != want {
		t.Fatalf("MarshalJSON = %q, want %q", got, want)
	}
}
