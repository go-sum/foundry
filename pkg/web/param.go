package web

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParamInt returns the route parameter named `name` parsed as int.
// Returns a *Error with Status 400 if the parameter is missing or not a valid integer.
func ParamInt(c *Context, name string) (int, error) {
	raw := c.Param(name)
	if raw == "" {
		return 0, &Error{
			Status:  400,
			Code:    CodeBadRequest,
			Title:   "Bad Request",
			Message: fmt.Sprintf("route parameter %q is missing", name),
		}
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, &Error{
			Status:  400,
			Code:    CodeBadRequest,
			Title:   "Bad Request",
			Message: fmt.Sprintf("route parameter %q must be an integer, got %q", name, raw),
			Cause:   err,
		}
	}
	return n, nil
}

// Param returns the named path parameter from the context, or "" if missing.
// This is a package-level helper mirroring c.Param.
func Param(c *Context, name string) string {
	return c.Param(name)
}

// ParamUUID reads a named path parameter and validates it is a well-formed
// UUID (any version, RFC 4122 v1..v8 or nil UUID). It returns a *Error with
// Status 400 if the parameter is missing or malformed.
func ParamUUID(c *Context, name string) (string, error) {
	raw := c.Param(name)
	if raw == "" {
		return "", &Error{
			Status:  400,
			Code:    CodeBadRequest,
			Title:   "Bad Request",
			Message: fmt.Sprintf("route parameter %q is missing", name),
		}
	}
	lower := strings.ToLower(raw)
	if !isValidUUID(lower) {
		return "", &Error{
			Status:  400,
			Code:    CodeBadRequest,
			Title:   "Bad Request",
			Message: fmt.Sprintf("route parameter %q is not a valid UUID, got %q", name, raw),
		}
	}
	return lower, nil
}

// isValidUUID reports whether s is a well-formed UUID string.
// It accepts any RFC 4122 variant (v1..v8) and the nil UUID.
func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, b := range []byte(s) {
		switch i {
		case 8, 13, 18, 23:
			if b != '-' {
				return false
			}
		default:
			if !isHexByte(b) {
				return false
			}
		}
	}
	return true
}

func isHexByte(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// ParamBool reads a named path parameter as a boolean ("true"/"false"/"1"/"0"/"yes"/"no").
// Returns a *Error with Status 400 if the parameter is missing or unrecognized.
func ParamBool(c *Context, name string) (bool, error) {
	raw := c.Param(name)
	if raw == "" {
		return false, &Error{
			Status:  400,
			Code:    CodeBadRequest,
			Title:   "Bad Request",
			Message: fmt.Sprintf("route parameter %q is missing", name),
		}
	}
	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, &Error{
			Status:  400,
			Code:    CodeBadRequest,
			Title:   "Bad Request",
			Message: fmt.Sprintf("route parameter %q must be a boolean, got %q", name, raw),
		}
	}
}

// ParamTime reads a named path parameter as an RFC 3339 timestamp.
// Returns a *Error with Status 400 if the parameter is missing or malformed.
func ParamTime(c *Context, name string) (time.Time, error) {
	raw := c.Param(name)
	if raw == "" {
		return time.Time{}, &Error{
			Status:  400,
			Code:    CodeBadRequest,
			Title:   "Bad Request",
			Message: fmt.Sprintf("route parameter %q is missing", name),
		}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, &Error{
			Status:  400,
			Code:    CodeBadRequest,
			Title:   "Bad Request",
			Message: fmt.Sprintf("route parameter %q must be an RFC 3339 timestamp, got %q", name, raw),
			Cause:   err,
		}
	}
	return t, nil
}

// ParamEnum reads a named path parameter and validates it is one of the
// allowed values. Comparison is case-insensitive. Returns the matching allowed
// value (preserving its original case) on success.
// Returns a *Error with Status 400 if the parameter is missing or not allowed.
func ParamEnum(c *Context, name string, allowed ...string) (string, error) {
	raw := c.Param(name)
	if raw == "" {
		return "", &Error{
			Status:  400,
			Code:    CodeBadRequest,
			Title:   "Bad Request",
			Message: fmt.Sprintf("route parameter %q is missing", name),
		}
	}
	lower := strings.ToLower(raw)
	for _, a := range allowed {
		if strings.ToLower(a) == lower {
			return a, nil
		}
	}
	return "", &Error{
		Status:  400,
		Code:    CodeBadRequest,
		Title:   "Bad Request",
		Message: fmt.Sprintf("route parameter %q has invalid value %q", name, raw),
	}
}

// PathParam extracts and parses a named route parameter into type T.
// T must be one of: string, int, int64, float64, bool.
// Returns a *Error with status 400 if the parameter is missing or cannot be parsed.
// For richer types (UUID validation, time parsing, enum sets), use the typed Param* functions.
func PathParam[T Scalar](c *Context, name string) (T, error) {
	var zero T
	raw := c.Param(name)
	if raw == "" {
		return zero, ErrBadRequest(fmt.Sprintf("route parameter %q is missing or empty", name))
	}
	return parseScalar[T](raw, "route parameter", name)
}
