package validate

import (
	"errors"
	"fmt"
	"strings"

	validator "github.com/go-playground/validator/v10"

	"github.com/go-sum/foundry/pkg/web"
)

// FieldError describes a single field-level validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// Errors is a slice of FieldError that implements the error interface.
type Errors []FieldError

// Error formats the validation errors as a semicolon-separated summary.
func (e Errors) Error() string {
	parts := make([]string, len(e))
	for i, fe := range e {
		parts[i] = fmt.Sprintf("field=%s tag=%s", fe.Field, fe.Tag)
	}
	return fmt.Sprintf("%d validation error(s): %s", len(e), strings.Join(parts, "; "))
}

// ToWebError converts e into a *web.Error with a "fields" meta entry.
// The original Errors value is set as the cause so callers can use errors.As
// to extract the field-level details from the returned *web.Error.
func (e Errors) ToWebError() *web.Error {
	return web.NewError(
		422, web.CodeValidation, "Validation Failed", e.Error(), e,
	).WithMeta("fields", e)
}

// phrases maps validator tag names to human-readable message templates.
// {param} is replaced with the field's constraint parameter.
var phrases = map[string]string{
	"required": "is required",
	"email":    "must be a valid email address",
	"min":      "must be at least {param}",
	"max":      "must be at most {param}",
	"len":      "must be exactly {param} characters",
	"oneof":    "must be one of {param}",
	"url":      "must be a valid URL",
	"uuid":     "must be a valid UUID",
}

func renderPhrase(tag, param string) string {
	phrase, ok := phrases[tag]
	if !ok {
		return tag
	}
	return strings.ReplaceAll(phrase, "{param}", param)
}

// FromValidator extracts Errors from a validator.ValidationErrors chain.
// Returns (nil, false) if err is not a validator.ValidationErrors.
func FromValidator(verr error) (Errors, bool) {
	var ve validator.ValidationErrors
	if !errors.As(verr, &ve) {
		return nil, false
	}
	errs := make(Errors, len(ve))
	for i, fe := range ve {
		errs[i] = FieldError{
			Field:   fe.Field(),
			Tag:     fe.Tag(),
			Message: renderPhrase(fe.Tag(), fe.Param()),
		}
	}
	return errs, true
}

// ToWebError maps err to a *web.Error when err contains validation errors.
// Returns nil if err is nil or does not contain validator.ValidationErrors.
func ToWebError(err error) *web.Error {
	if err == nil {
		return nil
	}
	errs, ok := FromValidator(err)
	if !ok {
		return nil
	}
	return errs.ToWebError()
}
