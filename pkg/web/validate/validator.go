package validate

import (
	"reflect"
	"strings"

	validator "github.com/go-playground/validator/v10"
)

// Validator validates struct values.
type Validator interface {
	Struct(any) error
}

// Option is a functional option applied to the underlying *validator.Validate.
type Option func(*validator.Validate)

// New returns a Validator with json-tag field-name resolution and
// RequiredStructEnabled. Additional options are applied in order.
func New(opts ...Option) Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		tag := fld.Tag.Get("json")
		if tag == "" {
			return ""
		}
		name, _, _ := strings.Cut(tag, ",")
		return name
	})
	for _, opt := range opts {
		opt(v)
	}
	return v
}
