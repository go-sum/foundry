package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	validate "github.com/go-playground/validator/v10"
	"github.com/go-sum/web"
)

const (
	defaultTTL     = 24 * time.Hour
	defaultMaxSize = 4096
)

// Settings is the env-facing shape for session configuration.
type Settings struct {
	CookieName   string        `validate:"required"`
	IdleTTL      time.Duration
	AbsoluteTTL  time.Duration
	CookieSecure bool
}

// DefaultSettings returns production-grade session defaults.
func DefaultSettings() Settings {
	return Settings{
		CookieName:   "session",
		IdleTTL:      30 * time.Minute,
		AbsoluteTTL:  24 * time.Hour,
		CookieSecure: true,
	}
}

// NewConfig builds a session Config from Settings and a Store.
func NewConfig(s Settings, store Store) Config {
	return Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     s.CookieName,
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
			Secure:   s.CookieSecure,
		},
		TTL:     s.AbsoluteTTL,
		IdleTTL: s.IdleTTL,
	}
}

// Config configures the session Middleware.
type Config struct {
	// Store handles session persistence. Required.
	// Use NewMemoryStore for server-side sessions.
	// Use NewCookieStore for client-side AEAD-encrypted sessions.
	Store Store `validate:"required"`

	// CookieTemplate defines the attributes of the session cookie.
	// CookieTemplate.Name is required.
	CookieTemplate web.Cookie

	// TTL is the absolute session lifetime. Defaults to 24 hours.
	TTL time.Duration

	// IdleTTL is the idle-inactivity timeout. Zero disables idle expiry.
	IdleTTL time.Duration

	// MaxCookieBytes is the maximum serialized Set-Cookie size. Defaults to 4096.
	MaxCookieBytes int
}

// Middleware loads the session before calling next and commits it to the response
// after the handler returns normally. If the handler panics, the deferred
// commit does not run — session mutations made during a panicking handler
// (Destroy, Regenerate, flash writes) are not persisted.
func Middleware(cfg Config) web.Middleware {
	if err := validate.New().Struct(&cfg); err != nil {
		panic(err)
	}
	if cfg.CookieTemplate.Name == "" {
		panic("web/session: CookieTemplate.Name must not be empty")
	}
	if cfg.TTL <= 0 {
		cfg.TTL = defaultTTL
	}
	if cfg.MaxCookieBytes <= 0 {
		cfg.MaxCookieBytes = defaultMaxSize
	}

	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (resp web.Response, herr error) {
			sess, err := loadSession(c.Context(), c.Headers().Get("Cookie"), cfg)
			if err != nil {
				return web.Response{}, web.ErrInternal(err)
			}
			c.Set(contextKey, sess)

			committed := false
			defer func() {
				if !committed {
					return
				}
				if cerr := commit(c.Context(), &resp, cfg, sess); cerr != nil {
					resp = web.Response{}
					herr = web.ErrInternal(cerr)
				}
			}()

			resp, herr = next(c)
			committed = true
			return
		}
	}
}

func loadSession(ctx context.Context, cookieHeader string, cfg Config) (*Session, error) {
	token := tokenFromCookie(cookieHeader, cfg.CookieTemplate.Name)
	if token == "" {
		return newSession(), nil
	}
	data, version, err := cfg.Store.Read(ctx, token)
	if errors.Is(err, ErrSessionNotFound) {
		return newSession(), nil
	}
	if err != nil {
		cause := fmt.Errorf("web/session: load: %w", err)
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errors.Join(web.ErrDependencyTimeout, cause)
		}
		return nil, errors.Join(web.ErrTransient, cause)
	}
	return sessionFromData(data, token, version), nil
}

func commit(ctx context.Context, resp *web.Response, cfg Config, sess *Session) error {
	if sess == nil {
		return nil
	}

	sess.mu.Lock()
	destroyed := sess.destroyed
	regenerated := sess.regenerated
	dirty := sess.dirty
	fresh := sess.fresh
	_ = fresh // retained for readability; save decision is driven by dirty alone
	token := sess.token
	oldToken := sess.oldToken
	version := sess.version
	sess.mu.Unlock()

	if destroyed {
		if token != "" {
			if err := cfg.Store.Delete(ctx, token); err != nil {
				cause := fmt.Errorf("web/session: delete: %w", err)
				if errors.Is(err, context.DeadlineExceeded) {
					return errors.Join(web.ErrDependencyTimeout, cause)
				}
				return errors.Join(web.ErrTransient, cause)
			}
		}
		web.SetCookie(resp, expiredCookie(cfg))
		resp.Headers.Set("Clear-Site-Data", `"cookies", "storage"`)
		return nil
	}

	if !dirty {
		return nil
	}

	saveToken := token
	saveVersion := version
	if regenerated {
		// Delete old token from store; new session starts at version 0.
		if oldToken != "" {
			if err := cfg.Store.Delete(ctx, oldToken); err != nil {
				slog.Debug("web/session: delete old token on regenerate", "err", err)
			}
		}
		saveToken = ""
		saveVersion = 0
	}

	data, err := sess.marshalPayload()
	if err != nil {
		return fmt.Errorf("web/session: marshal: %w", err)
	}

	absolute := time.Now().Add(cfg.TTL)
	newToken, err := cfg.Store.Save(ctx, saveToken, data, absolute, cfg.IdleTTL, saveVersion)
	if err != nil {
		cause := fmt.Errorf("web/session: save: %w", err)
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.Join(web.ErrDependencyTimeout, cause)
		}
		return cause
	}

	sess.mu.Lock()
	sess.token = newToken
	sess.mu.Unlock()

	cookie := cfg.CookieTemplate
	cookie.Value = newToken
	if cookie.MaxAge == 0 && cookie.Expires.IsZero() && cfg.TTL > 0 {
		cookie.MaxAge = int(cfg.TTL.Seconds())
	}
	serialized := cookie.String()
	if len(serialized) > cfg.MaxCookieBytes {
		return fmt.Errorf("web/session: Set-Cookie size %d exceeds limit %d", len(serialized), cfg.MaxCookieBytes)
	}
	web.SetCookie(resp, cookie)
	return nil
}

func tokenFromCookie(header, name string) string {
	for _, c := range web.ParseCookies(header) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

func expiredCookie(cfg Config) web.Cookie {
	c := cfg.CookieTemplate
	c.Value = ""
	c.MaxAge = -1
	return c
}
