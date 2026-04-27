package form_test

import (
	"testing"

	pform "github.com/go-sum/foundry/pkg/componentry/patterns/form"
	componentryform "github.com/go-sum/foundry/pkg/componentry/form"
)

// Compile-time assertion: *FormState satisfies the patterns/form.Form interface.
var _ pform.Form = (*componentryform.FormState)(nil)

func TestFormInterface_FormStateSatisfies(t *testing.T) {
	// This test documents that FormState satisfies the Form interface contract.
	// The real check is the compile-time var _ above.
	s := componentryform.NewFormState()

	// Verify each method on the interface is callable.
	_ = s.IsSubmitted()
	_ = s.IsValid()
	_ = s.FieldHasErrors("name")
	_ = s.GetFieldErrors("name")
	s.SetFieldError("name", "required")
	errs := s.GetErrors()
	if errs == nil {
		t.Error("GetErrors: expected non-nil map")
	}
}
