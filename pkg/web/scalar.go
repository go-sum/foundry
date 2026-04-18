package web

import (
	"fmt"
	"strconv"
	"strings"
)

// Scalar is the type constraint for generic parameter binders.
// Covers the common scalar types: string, int, int64, float64, bool.
type Scalar interface {
	~string | ~int | ~int64 | ~float64 | ~bool
}

// parseScalar converts a raw string to type T. Returns a *Error with status 400 on failure.
// source and name are used in error messages (e.g., "query parameter", "page").
func parseScalar[T Scalar](raw, source, name string) (T, error) {
	var zero T
	switch any(zero).(type) {
	case string:
		return any(raw).(T), nil
	case int:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return zero, scalarErr(source, name, "integer", raw)
		}
		return any(n).(T), nil
	case int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return zero, scalarErr(source, name, "integer", raw)
		}
		return any(n).(T), nil
	case float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return zero, scalarErr(source, name, "number", raw)
		}
		return any(f).(T), nil
	case bool:
		b, err := parseBoolLenient(raw)
		if err != nil {
			return zero, scalarErr(source, name, "boolean", raw)
		}
		return any(b).(T), nil
	}
	return zero, fmt.Errorf("web: parseScalar: unsupported type for %s %q", source, name)
}

func scalarErr(source, name, expected, got string) *Error {
	return ErrBadRequest(fmt.Sprintf("%s %q must be a valid %s, got %q", source, name, expected, got))
}

func parseBoolLenient(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	}
	return false, fmt.Errorf("unrecognized boolean value %q", s)
}
