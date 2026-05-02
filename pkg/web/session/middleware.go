package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	validate "github.com/go-playground/validator/v10"
	"github.com/go-sum/foundry/pkg/web"
)

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
		return nil, classifyStoreError("load", err)
	}
	return sessionFromData(data, token, version), nil
}

func classifyStoreError(op string, err error) error {
	cause := fmt.Errorf("web/session: %s: %w", op, err)
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.Join(web.ErrDependencyTimeout, cause)
	}
	return errors.Join(web.ErrTransient, cause)
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
				return classifyStoreError("delete", err)
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
		// Save the replacement as a new record first so a failed save leaves the
		// current session intact. Old-token deletion happens after save succeeds.
		if oldToken != "" {
			saveToken = ""
			saveVersion = 0
		}
	}

	data, err := sess.marshalPayload()
	if err != nil {
		return fmt.Errorf("web/session: marshal: %w", err)
	}

	absolute := time.Now().Add(cfg.TTL)
	newToken, err := cfg.Store.Save(ctx, saveToken, data, absolute, cfg.IdleTTL, saveVersion)
	// Version conflicts are surfaced as transient errors but intentionally not
	// retried here: automatic replay would re-save a stale payload and risk
	// clobbering concurrent mutations from another request or browser tab.
	if err != nil {
		return classifyStoreError("save", err)
	}

	cookie := cfg.CookieTemplate
	cookie.Value = newToken
	if cookie.MaxAge == 0 && cookie.Expires.IsZero() && cfg.TTL > 0 {
		cookie.MaxAge = int(cfg.TTL.Seconds())
	}
	serialized := cookie.String()
	if len(serialized) > cfg.MaxCookieBytes {
		return fmt.Errorf("web/session: Set-Cookie size %d exceeds limit %d", len(serialized), cfg.MaxCookieBytes)
	}

	if regenerated && oldToken != "" {
		if err := cfg.Store.Delete(ctx, oldToken); err != nil {
			return classifyStoreError("delete", err)
		}
	}

	sess.mu.Lock()
	sess.token = newToken
	sess.oldToken = ""
	sess.mu.Unlock()

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
