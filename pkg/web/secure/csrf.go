package secure

import (
	"cmp"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	validator "github.com/go-playground/validator/v10"
	"github.com/go-sum/web"
	websession "github.com/go-sum/web/session"
)

// ErrCSRFPreviousKeys is returned by NewCSRFConfigFromHex when the
// comma-separated previous-keys hex string contains an invalid entry.
// Callers can use errors.Is to distinguish this from a primary-key error.
var ErrCSRFPreviousKeys = errors.New("secure: csrf previous keys invalid")

// CSRFConfig configures CSRF protection middleware.
type CSRFConfig struct {
	// Key is the HMAC-SHA256 signing key for stateless fallback mode.
	Key []byte `validate:"required,min=32" help:"set SECURITY_CSRF_KEY — generate with 'openssl rand -hex 32' and place in starter/.env"`

	// PreviousKeys are additional HMAC keys accepted during verification only.
	// During a key rotation, place the retiring key here for at least TokenTTL
	PreviousKeys [][]byte `validate:"dive,min=32" help:"set SECURITY_CSRF_KEY_PREVIOUS (comma-separated hex) during rotation; keep for at least TokenTTL"`

	// TokenTTL is the stateless token lifetime.
	TokenTTL time.Duration

	// ContextKey is the context/session key under which the token is stored.
	// Defaults to "csrf".
	ContextKey string

	// HeaderName is the legacy single header name checked for the token.
	// Defaults to "X-CSRF-Token".
	HeaderName string

	// HeaderNames is the ordered list of header names checked for the token.
	// If empty, defaults to ["X-CSRF-Token", "X-XSRF-Token", "csrf-token"].
	HeaderNames []string

	// QueryField is the URL query parameter checked for the token.
	// Defaults to "_csrf".
	QueryField string

	// FormField is the form body field checked for the token on
	// application/x-www-form-urlencoded and multipart/form-data requests.
	// The body is peeked via Clone so the downstream handler still receives
	// the full original body. Defaults to "_csrf".
	// This matches the field name emitted by render.CSRFField.
	FormField string

	// SafeMethods are HTTP methods that do not require CSRF validation.
	// Defaults to GET, HEAD, OPTIONS.
	SafeMethods []string

	// AllowMissingOrigin controls whether unsafe requests without Origin or
	// Referer are allowed. Defaults to true.
	AllowMissingOrigin bool

	// AllowedOrigins are additional trusted origins for unsafe requests.
	AllowedOrigins []string

	// AllowedOriginFunc performs dynamic origin checks for unsafe requests.
	AllowedOriginFunc func(origin string, c *web.Context) bool

	// TokenLookup overrides default submitted-token extraction.
	TokenLookup func(c *web.Context) string

	// Skipper returns true to skip CSRF validation for the request.
	Skipper func(c *web.Context) bool

	// CookieName is the name of the double-submit CSRF cookie used in
	// stateless mode. Defaults to "csrf". For production over HTTPS, use
	// "__Host-csrf" with CookieSecure set to true.
	CookieName string

	// CookieSecure sets the Secure attribute on the stateless CSRF cookie.
	// Set to true in production (HTTPS-only). Defaults to false.
	CookieSecure bool

	// CookieSameSite controls the SameSite attribute of the CSRF cookie.
	// Accepted values: "Lax" (default), "Strict", "None". Empty string defaults to "Lax".
	CookieSameSite string
}

type csrfContextKey struct{}

type csrfContextData struct {
	token      string
	fieldName  string
	headerName string
}

var csrfSameSiteDefaults = map[string]string{
	"Strict": "Strict",
	"None":   "None",
	"Lax":    "Lax",
}

// CSRFToken retrieves the CSRF token from the request context.
// Returns "" if no token is available.
func CSRFToken(c *web.Context) string {
	data, _ := web.Get[csrfContextData](c, csrfContextKey{})
	return data.token
}

// CSRFFieldName retrieves the configured CSRF form field name from the request context.
// Returns "" if the CSRF middleware has not run.
func CSRFFieldName(c *web.Context) string {
	data, _ := web.Get[csrfContextData](c, csrfContextKey{})
	return data.fieldName
}

// CSRFHeaderName retrieves the configured CSRF header name from the request context.
// Returns "" if the CSRF middleware has not run.
func CSRFHeaderName(c *web.Context) string {
	data, _ := web.Get[csrfContextData](c, csrfContextKey{})
	return data.headerName
}

func applyCSRFDefaults(c *CSRFConfig) {
	c.TokenTTL = cmp.Or(c.TokenTTL, time.Hour)
	c.ContextKey = cmp.Or(c.ContextKey, "csrf")
	c.HeaderName = cmp.Or(c.HeaderName, "X-CSRF-Token")
	c.QueryField = cmp.Or(c.QueryField, "_csrf")
	c.FormField = cmp.Or(c.FormField, "_csrf")
	c.CookieName = cmp.Or(c.CookieName, "csrf")
	if len(c.HeaderNames) == 0 {
		c.HeaderNames = []string{"X-CSRF-Token", "X-XSRF-Token", "csrf-token"}
	}
	if len(c.SafeMethods) == 0 {
		c.SafeMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	}
}

// DefaultCSRFConfig returns a CSRFConfig with all defaults applied.
// Key is zero-length; the caller must supply a key before use.
func DefaultCSRFConfig() CSRFConfig {
	var c CSRFConfig
	applyCSRFDefaults(&c)
	return c
}

// NewCSRFConfigFromHex returns a CSRFConfig populated with defaults plus the
// supplied hex-encoded keys. An empty keyHex leaves Key zero-length (the
// caller or validator decides whether that is acceptable). previousKeysHex is
// a comma-separated list; empty entries are skipped; empty string is a no-op.
//
// Errors are namespaced: "csrf key: ..." or "csrf previous keys: ...".
func NewCSRFConfigFromHex(keyHex, previousKeysHex string) (CSRFConfig, error) {
	cfg := DefaultCSRFConfig()
	if keyHex != "" {
		k, err := decodeHexKey(keyHex)
		if err != nil {
			return CSRFConfig{}, fmt.Errorf("csrf key: %w", err)
		}
		cfg.Key = k
	}
	if previousKeysHex != "" {
		keys, err := decodeHexKeys(previousKeysHex)
		if err != nil {
			return CSRFConfig{}, errors.Join(ErrCSRFPreviousKeys, fmt.Errorf("csrf previous keys: %w", err))
		}
		cfg.PreviousKeys = keys
	}
	return cfg, nil
}

// CSRF returns middleware that validates CSRF tokens on unsafe HTTP methods
// and issues fresh tokens on all requests. If session middleware is present,
// session-backed tokens are used automatically. Otherwise the middleware falls
// back to signed double-submit cookies.
func CSRF(cfg CSRFConfig) web.Middleware {
	if err := validator.New().Struct(&cfg); err != nil {
		panic(err)
	}
	applyCSRFDefaults(&cfg)

	verifyKeys := make([][]byte, 0, 1+len(cfg.PreviousKeys))
	verifyKeys = append(verifyKeys, cfg.Key)
	verifyKeys = append(verifyKeys, cfg.PreviousKeys...)

	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}

			sessionToken, hasSession, err := ensureSessionToken(c, cfg.ContextKey)
			if err != nil {
				return web.Response{}, web.ErrInternal(err)
			}

			if isUnsafeMethod(c.Method(), cfg.SafeMethods) {
				if !validOrigin(c, cfg) {
					return web.Response{}, web.ErrForbidden("CSRF origin invalid")
				}

				submittedToken := submittedCSRFToken(c, cfg)
				if submittedToken == "" {
					return web.Response{}, web.ErrForbidden("CSRF token missing")
				}

				if hasSession {
					if subtle.ConstantTimeCompare([]byte(submittedToken), []byte(sessionToken)) != 1 {
						return web.Response{}, web.ErrForbidden("CSRF token invalid")
					}
				} else {
					if err := verifyTokenAny(verifyKeys, cfg.ContextKey, submittedToken); err != nil {
						return web.Response{}, web.ErrForbidden("CSRF token invalid")
					}
					cookie, ok := web.GetCookie(c.Request, cfg.CookieName)
					if !ok || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(submittedToken)) != 1 {
						return web.Response{}, web.ErrForbidden("CSRF token mismatch")
					}
				}
			}

			token := sessionToken
			if !hasSession {
				token, err = IssueToken(cfg.Key, cfg.ContextKey, cfg.TokenTTL)
				if err != nil {
					return web.Response{}, web.ErrInternal(err)
				}
			}

			c.Set(csrfContextKey{}, csrfContextData{
				token:      token,
				fieldName:  cfg.FormField,
				headerName: cfg.HeaderName,
			})
			resp, herr := next(c)
			if !hasSession {
				sameSite := resolveSameSite(cfg.CookieSameSite)
				secure := cfg.CookieSecure
				if sameSite == "None" {
					secure = true
				}
				web.SetCookie(&resp, web.Cookie{
					Name:     cfg.CookieName,
					Value:    token,
					Path:     "/",
					MaxAge:   int(cfg.TokenTTL.Seconds()),
					Secure:   secure,
					HTTPOnly: false,
					SameSite: sameSite,
				})
			}
			return resp, herr
		}
	}
}

func ensureSessionToken(c *web.Context, key string) (string, bool, error) {
	sess, ok := websession.FromContext(c)
	if !ok || sess == nil {
		return "", false, nil
	}

	token, found, err := websession.Get[string](sess, key)
	if err == nil && found && token != "" {
		return token, true, nil
	}

	token, err = newSessionCSRFToken()
	if err != nil {
		return "", true, err
	}
	if err := sess.Set(key, token); err != nil {
		return "", true, err
	}
	return token, true, nil
}

func newSessionCSRFToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// submittedCSRFToken extracts a CSRF token from the request. The lookup order is:
//  1. TokenLookup override (if configured)
//  2. Request headers (HeaderNames, in order)
//  3. URL query parameter (QueryField)
//  4. Form body field (FormField) — for application/x-www-form-urlencoded and
//     multipart/form-data. The body is cloned so the downstream handler still
//     receives the full original body.
func submittedCSRFToken(c *web.Context, cfg CSRFConfig) string {
	if cfg.TokenLookup != nil {
		return strings.TrimSpace(cfg.TokenLookup(c))
	}

	for _, headerName := range cfg.HeaderNames {
		if token := strings.TrimSpace(c.Headers().Get(headerName)); token != "" {
			return token
		}
	}

	if c.URL() != nil {
		if token := strings.TrimSpace(c.URL().Query().Get(cfg.QueryField)); token != "" {
			return token
		}
	}

	// Form body: only for urlencoded/multipart. Clone the body so the handler
	// can still read it after we peek.
	ct := strings.ToLower(strings.TrimSpace(c.Request.Headers.Get("Content-Type")))
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(ct, "multipart/form-data") {
		if peek, err := c.Request.Clone(); err == nil {
			if fd, err := peek.FormData(); err == nil {
				return strings.TrimSpace(fd.Values.Get(cfg.FormField))
			}
		}
	}
	return ""
}

func validOrigin(c *web.Context, cfg CSRFConfig) bool {
	// Sec-Fetch-Site is tamper-proof in modern browsers; use it first.
	switch strings.ToLower(strings.TrimSpace(c.Headers().Get("Sec-Fetch-Site"))) {
	case "same-origin", "none":
		return true
	case "cross-site":
		return false
	}

	requestOrigin := sameOriginBase(c)
	origin := strings.TrimSpace(c.Headers().Get("Origin"))
	if origin == "" {
		referer := strings.TrimSpace(c.Headers().Get("Referer"))
		if referer == "" {
			return true
		}
		parsed, err := neturl.Parse(referer)
		if err != nil {
			return false
		}
		origin = parsed.Scheme + "://" + parsed.Host
	}

	if requestOrigin != "" && sameOrigin(origin, requestOrigin) {
		return true
	}
	for _, allowed := range cfg.AllowedOrigins {
		if sameOrigin(origin, allowed) {
			return true
		}
	}
	if cfg.AllowedOriginFunc != nil {
		return cfg.AllowedOriginFunc(origin, c)
	}
	return false
}

func sameOriginBase(c *web.Context) string {
	if c == nil {
		return ""
	}
	if c.URL() != nil && c.URL().Scheme != "" && c.URL().Host != "" {
		return c.URL().Scheme + "://" + c.URL().Host
	}

	host := c.Request.Host()
	if host == "" && c.URL() != nil {
		host = c.URL().Host
	}
	if host == "" {
		return ""
	}

	scheme := "http"
	if c.URL() != nil && c.URL().Scheme != "" {
		scheme = c.URL().Scheme
	} else if strings.EqualFold(c.Headers().Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + host
}

func sameOrigin(left string, right string) bool {
	leftURL, err := neturl.Parse(left)
	if err != nil {
		return false
	}
	rightURL, err := neturl.Parse(right)
	if err != nil {
		return false
	}
	return strings.EqualFold(leftURL.Scheme, rightURL.Scheme) &&
		strings.EqualFold(leftURL.Host, rightURL.Host)
}

func isUnsafeMethod(method string, safeMethods []string) bool {
	for _, safeMethod := range safeMethods {
		if method == safeMethod {
			return false
		}
	}
	return true
}

// resolveSameSite maps a caller-supplied SameSite string to a canonical value.
// Accepted inputs: "Lax", "Strict", "None". Any other value (including empty)
// defaults to "Lax".
func resolveSameSite(s string) string {
	if sameSite, ok := csrfSameSiteDefaults[s]; ok {
		return sameSite
	}
	return "Lax"
}
