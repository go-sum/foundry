// Package form provides request binding and validation for HTML form submissions.
package form

// Form describes the contract for a form submission handler.
type Form interface {
	IsSubmitted() bool
	IsValid() bool
	FieldHasErrors(field string) bool
	GetFieldErrors(field string) []string
	SetFieldError(field, msg string)
	GetErrors() map[string][]string
}
