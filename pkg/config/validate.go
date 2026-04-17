package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

func Validate[T any](cfg T, registrars ...func(*validator.Validate)) error {
	v := validator.New(validator.WithRequiredStructEnabled())
	for _, r := range registrars {
		r(v)
	}

	err := v.Struct(cfg)
	if err == nil {
		return nil
	}

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return err
	}

	rootType := reflect.TypeOf(cfg)
	var b strings.Builder
	b.WriteString("configuration validation failed:")
	for _, fe := range ve {
		ns := fe.StructNamespace()
		parts := strings.Split(ns, ".")
		if len(parts) > 1 {
			parts = parts[1:]
		}
		b.WriteString("\n  - ")
		b.WriteString(ns)
		b.WriteString(" ")
		b.WriteString(phraseFor(fe))
		if help := findFieldTag(rootType, parts, "help"); help != "" {
			b.WriteString("\n    help: ")
			b.WriteString(help)
		}
	}
	return errors.New(b.String())
}

func phraseFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("must have length %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of [%s]", fe.Param())
	default:
		return fmt.Sprintf("failed validation %q", fe.Tag())
	}
}

func findFieldTag(t reflect.Type, parts []string, tag string) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	var last reflect.StructField
	for _, name := range parts {
		if t.Kind() != reflect.Struct {
			return ""
		}
		sf, ok := t.FieldByName(name)
		if !ok {
			return ""
		}
		last = sf
		ft := sf.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		t = ft
	}
	return last.Tag.Get(tag)
}
