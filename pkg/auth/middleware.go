package auth

import (
	"errors"
	"net/url"
	"strings"

	"github.com/go-sum/web"
	"github.com/go-sum/web/htmx"
	"github.com/go-sum/web/session"
	"github.com/google/uuid"
)

// sanitizeReturnTo validates a return-to URL and returns it if safe, or "/".
// A safe URL must be relative (starts with "/"), not protocol-relative ("//"),
// and must not contain newlines (header injection guard).
func sanitizeReturnTo(returnTo string) string {
	if !strings.HasPrefix(returnTo, "/") {
		return "/"
	}
	if strings.HasPrefix(returnTo, "//") {
		return "/"
	}
	if strings.ContainsAny(returnTo, "\r\n") {
		return "/"
	}
	return returnTo
}

// LoadSession reads user identity from the session and sets context values.
// It is non-destructive: if no session or no user ID is present, the request
// proceeds without context values.
func LoadSession() web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			sess, ok := session.FromContext(c)
			if !ok {
				return next(c)
			}
			if userID, ok := getUserID(sess); ok {
				SetUserID(c, userID)
				if name, ok := getDisplayName(sess); ok {
					SetDisplayName(c, name)
				}
			}
			return next(c)
		}
	}
}

// RequireAuth rejects unauthenticated requests. For HTMX requests it returns
// a 401 with an HX-Redirect header; for full-page requests it returns a 303
// redirect to the signin path with a return_to query parameter so the user
// is sent back to the original URL after signin.
func RequireAuth(signinPath func() string) web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			if UserID(c) == "" {
				path := signinPath()
				returnTo := c.URL().RequestURI()
				sep := "?"
				if strings.Contains(path, "?") {
					sep = "&"
				}
				path = path + sep + "return_to=" + url.QueryEscape(returnTo)
				if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
					resp := web.Respond(401)
					htmx.SetRedirect(&resp, path)
					return resp, nil
				}
				return web.SeeOther(path), nil
			}
			return next(c)
		}
	}
}

// LoadUserRole resolves the authenticated user's role from the database and
// stores it in the request context for downstream authorization checks.
func LoadUserRole(users UserReader) web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			uid := UserID(c)
			if uid == "" {
				return next(c)
			}
			id, err := uuid.Parse(uid)
			if err != nil {
				return web.Response{}, web.ErrUnauthorized("Invalid session")
			}
			user, err := users.GetUserByID(c.Context(), id)
			if err != nil {
				if errors.Is(err, ErrUserNotFound) {
					return web.Response{}, web.ErrUnauthorized("Account not found")
				}
				return web.Response{}, web.ErrUnavailable("Unable to authorize", err)
			}
			SetUserRole(c, string(user.Role))
			return next(c)
		}
	}
}

// RequireAdmin ensures the authenticated user has the admin role.
func RequireAdmin() web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			if UserRole(c) != string(RoleAdmin) {
				return web.Response{}, web.ErrForbidden("Admin access required")
			}
			return next(c)
		}
	}
}
