package web

import (
	"net/url"
	"strings"
)

// MethodOverrideConfig configures the MethodOverride middleware.
type MethodOverrideConfig struct {
	// FormField is the form field name used to override the method.
	// Defaults to "_method".
	FormField string
	// Header is the HTTP header used to override the method.
	// Defaults to "X-HTTP-Method-Override".
	Header string
	// AllowedMethods lists which override values are permitted.
	// Defaults to ["PUT", "PATCH", "DELETE"].
	AllowedMethods []string
}

// MethodOverride returns a Middleware that allows HTML forms (which only support
// GET and POST) to tunnel PUT, PATCH, and DELETE by embedding a hidden field
// (default: _method) or setting a header (default: X-HTTP-Method-Override).
//
// Override is only applied on POST requests. The original method is never
// overridden for other methods. The request body is NOT consumed by this
// middleware — it clones the body to peek at form fields and the downstream
// handler still receives the full body.
func MethodOverride(cfg MethodOverrideConfig) Middleware {
	if cfg.FormField == "" {
		cfg.FormField = "_method"
	}
	if cfg.Header == "" {
		cfg.Header = "X-HTTP-Method-Override"
	}
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = []string{"PUT", "PATCH", "DELETE"}
	}

	allowed := make(map[string]bool, len(cfg.AllowedMethods))
	for _, m := range cfg.AllowedMethods {
		allowed[strings.ToUpper(m)] = true
	}

	return func(next Handler) Handler {
		return func(c *Context) (Response, error) {
			if c.Method != "POST" {
				return next(c)
			}

			// Header check first — no body read needed.
			if headerVal := strings.TrimSpace(c.Headers.Get(cfg.Header)); headerVal != "" {
				target := strings.ToUpper(headerVal)
				if allowed[target] {
					c.Method = target
					c.Request.Method = target
					return next(c)
				}
			}

			// Form field check: clone the body so the downstream handler
			// still receives the full original body.
			if peek, err := c.Request.Clone(); err == nil {
				if bodyText, err := peek.Text(); err == nil {
					values, err := url.ParseQuery(bodyText)
					if err == nil {
						if fieldVal := strings.TrimSpace(values.Get(cfg.FormField)); fieldVal != "" {
							target := strings.ToUpper(fieldVal)
							if allowed[target] {
								c.Method = target
								c.Request.Method = target
							}
						}
					}
				}
			}

			return next(c)
		}
	}
}
