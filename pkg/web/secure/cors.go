package secure

import (
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/go-sum/foundry/pkg/web"
)

// CORSConfig configures CORS middleware.
type CORSConfig struct {
	// AllowOrigins is the list of allowed origins. Use "*" for wildcard
	// (wildcard responses are automatically reflected when credentials are enabled).
	// When AllowOrigins, AllowOriginPatterns, and AllowOriginFunc are all empty,
	// no cross-origin requests are permitted (deny by default).
	AllowOrigins []string

	// AllowOriginPatterns are regular expressions evaluated against the request origin.
	AllowOriginPatterns []*regexp.Regexp

	// AllowOriginFunc resolves the allowed origin dynamically.
	AllowOriginFunc func(origin string, c *web.Context) (string, bool)

	// AllowMethods is the list of allowed HTTP methods.
	// Defaults to GET, HEAD, PUT, PATCH, POST, DELETE.
	AllowMethods []string

	// AllowHeaders is the list of allowed request headers. If empty, the
	// preflight request headers are reflected.
	AllowHeaders []string

	// AllowHeadersFunc resolves the allowed preflight headers dynamically.
	AllowHeadersFunc func(c *web.Context) []string

	// ExposeHeaders is the list of headers safe to expose to the browser.
	ExposeHeaders []string

	// AllowCredentials indicates whether the request can include credentials.
	AllowCredentials bool

	// AllowPrivateNetwork controls Access-Control-Allow-Private-Network responses.
	AllowPrivateNetwork bool

	// MaxAge is the maximum time (in seconds) to cache preflight results.
	MaxAge int

	// PreflightContinue controls whether OPTIONS preflight requests continue to
	// downstream handlers after headers are added.
	PreflightContinue bool

	// PreflightStatus sets the status code used when preflight requests are
	// short-circuited. Defaults to 204.
	PreflightStatus int

	// Skipper returns true to skip CORS handling for the request.
	Skipper func(c *web.Context) bool
}

// CORS returns middleware that handles Cross-Origin Resource Sharing.
func CORS(cfg CORSConfig) web.Middleware {
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = []string{
			http.MethodGet, http.MethodHead, http.MethodPut,
			http.MethodPatch, http.MethodPost, http.MethodDelete,
		}
	}
	if cfg.PreflightStatus == 0 {
		cfg.PreflightStatus = http.StatusNoContent
	}

	allowMethods := normalizeHeaderList(cfg.AllowMethods, true)
	exposeHeaders := normalizeHeaderList(cfg.ExposeHeaders, false)
	configuredAllowHeaders := normalizeHeaderList(cfg.AllowHeaders, false)

	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}

			origin := strings.TrimSpace(c.Headers().Get("Origin"))
			preflight := isPreflight(c)
			if origin == "" {
				if preflight && !cfg.PreflightContinue {
					return web.Respond(cfg.PreflightStatus), nil
				}
				return next(c)
			}

			allowedOrigin, ok := resolveAllowedOrigin(origin, c, cfg)
			if !ok {
				if preflight && !cfg.PreflightContinue {
					return web.Response{}, web.ErrForbidden("CORS origin denied")
				}
				return next(c)
			}

			baseHeaders := web.NewHeaders()
			baseHeaders.Set("Access-Control-Allow-Origin", allowOriginHeader(allowedOrigin, origin, cfg.AllowCredentials))
			if cfg.AllowCredentials {
				baseHeaders.Set("Access-Control-Allow-Credentials", "true")
			}
			if cfg.MaxAge > 0 {
				baseHeaders.Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
			}

			vary := newVarySet()
			if baseHeaders.Get("Access-Control-Allow-Origin") != "*" {
				vary.Add("Origin")
			}

			if preflight {
				baseHeaders.Set("Access-Control-Allow-Methods", allowMethods)
				vary.Add("Access-Control-Request-Method")

				allowHeaders := configuredAllowHeaders
				if cfg.AllowHeadersFunc != nil {
					allowHeaders = normalizeHeaderList(cfg.AllowHeadersFunc(c), false)
				}
				if allowHeaders == "" {
					if requested := strings.TrimSpace(c.Headers().Get("Access-Control-Request-Headers")); requested != "" {
						baseHeaders.Set("Access-Control-Allow-Headers", requested)
						vary.Add("Access-Control-Request-Headers")
					}
				} else {
					baseHeaders.Set("Access-Control-Allow-Headers", allowHeaders)
				}

				if cfg.AllowPrivateNetwork &&
					strings.EqualFold(c.Headers().Get("Access-Control-Request-Private-Network"), "true") {
					baseHeaders.Set("Access-Control-Allow-Private-Network", "true")
					vary.Add("Access-Control-Request-Private-Network")
				}

				if !cfg.PreflightContinue {
					resp := web.Respond(cfg.PreflightStatus)
					mergeHeaders(&resp.Headers, baseHeaders)
					if varyHeader := vary.String(); varyHeader != "" {
						resp.Headers.Set("Vary", varyHeader)
					}
					return resp, nil
				}
			}

			resp, err := next(c)
			mergeHeaders(&resp.Headers, baseHeaders)
			if !preflight && exposeHeaders != "" {
				resp.Headers.Set("Access-Control-Expose-Headers", exposeHeaders)
			}
			appendVaryHeader(&resp.Headers, vary)
			return resp, err
		}
	}
}

func resolveAllowedOrigin(origin string, c *web.Context, cfg CORSConfig) (string, bool) {
	if cfg.AllowOriginFunc != nil {
		return cfg.AllowOriginFunc(origin, c)
	}

	for _, allowed := range cfg.AllowOrigins {
		if allowed == "*" {
			return "*", true
		}
		if strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return origin, true
		}
	}
	for _, pattern := range cfg.AllowOriginPatterns {
		if pattern != nil && pattern.MatchString(origin) {
			return origin, true
		}
	}
	return "", false
}

func allowOriginHeader(allowedOrigin string, requestOrigin string, allowCredentials bool) string {
	if allowedOrigin == "*" && !allowCredentials {
		return "*"
	}
	if allowedOrigin == "*" {
		return requestOrigin
	}
	return allowedOrigin
}

func isPreflight(c *web.Context) bool {
	return c.Method() == http.MethodOptions && c.Headers().Has("Access-Control-Request-Method")
}

func normalizeHeaderList(values []string, upper bool) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if upper {
			value = strings.ToUpper(value)
		}
		if slices.ContainsFunc(out, func(existing string) bool {
			return strings.EqualFold(existing, value)
		}) {
			continue
		}
		out = append(out, value)
	}
	return strings.Join(out, ", ")
}

func mergeHeaders(dst *web.Headers, src web.Headers) {
	src.ForEach(func(name string, values []string) {
		dst.Delete(name)
		for _, value := range values {
			dst.Append(name, value)
		}
	})
}

type varySet map[string]struct{}

func newVarySet() varySet {
	return make(varySet)
}

func (v varySet) Add(value string) {
	if value == "" {
		return
	}
	v[strings.ToLower(value)] = struct{}{}
}

func (v varySet) String() string {
	if len(v) == 0 {
		return ""
	}
	preferred := []string{
		"origin",
		"access-control-request-method",
		"access-control-request-headers",
		"access-control-request-private-network",
	}
	parts := make([]string, 0, len(v))
	seen := make(map[string]struct{}, len(v))
	for _, value := range preferred {
		if _, ok := v[value]; ok {
			parts = append(parts, canonicalHeaderToken(value))
			seen[value] = struct{}{}
		}
	}
	extra := make([]string, 0, len(v))
	for value := range v {
		if _, ok := seen[value]; ok {
			continue
		}
		extra = append(extra, canonicalHeaderToken(value))
	}
	slices.Sort(extra)
	parts = append(parts, extra...)
	return strings.Join(parts, ", ")
}

func appendVaryHeader(headers *web.Headers, values varySet) {
	for _, part := range strings.Split(headers.Get("Vary"), ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			values.Add(part)
		}
	}
	if varyHeader := values.String(); varyHeader != "" {
		headers.Set("Vary", varyHeader)
	}
}

func canonicalHeaderToken(value string) string {
	parts := strings.Split(strings.ToLower(value), "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "-")
}
