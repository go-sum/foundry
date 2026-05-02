package secure

import (
	"cmp"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	validator "github.com/go-playground/validator/v10"
	"github.com/go-sum/foundry/pkg/web"
	websession "github.com/go-sum/foundry/pkg/web/session"
)

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
	c.FormField = cmp.Or(c.FormField, "_csrf")
	c.CookieName = cmp.Or(c.CookieName, "csrf")

	if len(c.SafeMethods) == 0 {
		c.SafeMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	}
}


// CSRFConfigFromHex returns a CSRFConfig with defaults and the decoded key.
// On empty, invalid hex, or short key input the Key field is left nil;
// validation catches this via the required,min=32 struct tag on Key.
func CSRFConfigFromHex(keyHex string) CSRFConfig {
	cfg := InitialCSRFConfig()
	if keyHex == "" {
		return cfg
	}
	k, err := decodeHexKey(keyHex)
	if err != nil {
		return cfg
	}
	cfg.Key = k
	return cfg
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
					if err := VerifyToken(cfg.Key, cfg.ContextKey, submittedToken); err != nil {
						return web.Response{}, web.ErrForbidden("CSRF token invalid")
					}
					cookie, ok := web.GetCookie(c.Request, cfg.CookieName)
					if !ok || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(submittedToken)) != 1 {
						return web.Response{}, web.ErrForbidden("CSRF token mismatch")
					}
				}
			}

			// In stateless (cookie double-submit) mode, a fresh token is issued on
			// every request. This is safe: verification is MAC-based so outstanding
			// tokens remain valid for their full TTL even after rotation. Session-backed
			// mode reuses the stable session token instead.
			token := sessionToken
			if !hasSession {
				token, err = IssueToken(cfg.Key, cfg.ContextKey, cfg.TokenTTL)
				if err != nil {
					return web.Response{}, web.ErrInternal(err)
				}
			}

			advertisedHeader := cfg.HeaderName
			c.Set(csrfContextKey{}, csrfContextData{
				token:      token,
				fieldName:  cfg.FormField,
				headerName: advertisedHeader,
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
//  3. Form body field (FormField) — for application/x-www-form-urlencoded and
//     multipart/form-data. The body is cloned so the downstream handler still
//     receives the full original body.
func submittedCSRFToken(c *web.Context, cfg CSRFConfig) string {
	if cfg.TokenLookup != nil {
		return strings.TrimSpace(cfg.TokenLookup(c))
	}

	if token := strings.TrimSpace(c.Headers().Get(cfg.HeaderName)); token != "" {
		return token
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

	requestOrigin := cfg.ServerOrigin
	if requestOrigin == "" {
		requestOrigin = sameOriginBase(c)
	}
	origin := strings.TrimSpace(c.Headers().Get("Origin"))
	if origin == "" {
		referer := strings.TrimSpace(c.Headers().Get("Referer"))
		if referer == "" {
			return cfg.AllowMissingOrigin
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
