package form

import "encoding/json"

const formErrorKey = "_"

// FormState tracks submission state and field/form-level errors for a single
// form POST. It is deliberately free of any concrete validator dependency —
// callers supply the validate function and extract field errors externally.
type FormState struct {
	submitted bool
	errors    map[string][]string
}

// NewFormState creates a FormState ready to accept a submission.
func NewFormState() *FormState {
	return &FormState{
		errors: make(map[string][]string),
	}
}

// Bind marks the state as submitted and runs validate(dest).
// Any returned error is stored as a form-level error under key "_".
// Callers who need per-field errors should call SetFieldErrors after Bind.
func (s *FormState) Bind(dest any, validate func(any) error) {
	s.submitted = true
	if err := validate(dest); err != nil {
		s.SetFormError(err.Error())
	}
}

// IsSubmitted reports whether Bind has been called.
func (s *FormState) IsSubmitted() bool { return s.submitted }

// IsValid reports whether Bind has been called and no errors have been accumulated.
func (s *FormState) IsValid() bool { return s.submitted && len(s.errors) == 0 }

// FieldHasErrors reports whether any errors have been recorded for field.
func (s *FormState) FieldHasErrors(field string) bool {
	return len(s.errors[field]) > 0
}

// GetFieldErrors returns all error messages recorded for field.
func (s *FormState) GetFieldErrors(field string) []string {
	return s.errors[field]
}

// SetFieldErrors merges a map of field → messages into the error state.
// Use this to store field-level validation errors extracted by the caller.
func (s *FormState) SetFieldErrors(errs map[string][]string) {
	for field, msgs := range errs {
		s.errors[field] = append(s.errors[field], msgs...)
	}
}

// SetFormError appends a form-level error message (stored under key "_").
func (s *FormState) SetFormError(msg string) {
	s.errors[formErrorKey] = append(s.errors[formErrorKey], msg)
}

// GetFormErrors returns all form-level error messages.
func (s *FormState) GetFormErrors() []string {
	return s.errors[formErrorKey]
}

// GetErrors returns the full error map.
func (s *FormState) GetErrors() map[string][]string {
	return s.errors
}

// SetFieldError appends a single error message for field.
func (s *FormState) SetFieldError(field, msg string) {
	s.errors[field] = append(s.errors[field], msg)
}

// MarshalJSON implements json.Marshaler so FormState can be round-tripped in tests.
func (s *FormState) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Submitted bool                `json:"submitted"`
		Errors    map[string][]string `json:"errors"`
	}{
		Submitted: s.submitted,
		Errors:    s.errors,
	})
}
