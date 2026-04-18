package web

import "fmt"

// QueryParam extracts and parses a query parameter into type T.
// Returns a *Error with status 400 if the parameter is absent or unparseable.
func QueryParam[T Scalar](c *Context, name string) (T, error) {
	var zero T
	raw := c.URL().Query().Get(name)
	if raw == "" {
		return zero, ErrBadRequest(fmt.Sprintf("query parameter %q is missing or empty", name))
	}
	return parseScalar[T](raw, "query parameter", name)
}

// QueryParamOr extracts a query parameter with a default value when absent.
// Returns *Error only if the parameter is present but cannot be parsed.
func QueryParamOr[T Scalar](c *Context, name string, fallback T) (T, error) {
	raw := c.URL().Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	return parseScalar[T](raw, "query parameter", name)
}

// QueryParams returns all values for a query parameter key as a slice of T.
// Returns nil and no error if the key is absent.
func QueryParams[T Scalar](c *Context, name string) ([]T, error) {
	values := c.URL().Query()[name]
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]T, len(values))
	for i, v := range values {
		parsed, err := parseScalar[T](v, "query parameter", name)
		if err != nil {
			return nil, err
		}
		out[i] = parsed
	}
	return out, nil
}

// HeaderParam extracts and parses a request header value into type T.
// Returns a *Error with status 400 if the header is absent or unparseable.
func HeaderParam[T Scalar](c *Context, name string) (T, error) {
	var zero T
	raw := c.Headers().Get(name)
	if raw == "" {
		return zero, ErrBadRequest(fmt.Sprintf("header %q is missing or empty", name))
	}
	return parseScalar[T](raw, "header", name)
}

// HeaderParamOr extracts a request header value with a default when absent.
func HeaderParamOr[T Scalar](c *Context, name string, fallback T) (T, error) {
	raw := c.Headers().Get(name)
	if raw == "" {
		return fallback, nil
	}
	return parseScalar[T](raw, "header", name)
}
