package auth

import (
	"errors"
	"net/url"
	"strings"

	"github.com/go-sum/foundry/pkg/web"
	webauth "github.com/go-sum/foundry/pkg/web/auth"
	"github.com/go-sum/foundry/pkg/web/htmx"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/google/uuid"
)

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
			ensureIdentityFromSession(c, sess)
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
			sess, ok := session.FromContext(c)
			if !ok {
				panic("auth: RequireAuth called without session middleware")
			}
			if !IsAuthenticated(sess) {
				path := signinPath()
				returnTo := webauth.SanitizeReturnTo(c.URL().RequestURI())
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
			ensureIdentityFromSession(c, sess)
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

func ensureIdentityFromSession(c *web.Context, sess *session.Session) bool {
	userID, ok := getUserID(sess)
	if !ok {
		return false
	}

	name, _ := getDisplayName(sess)
	verified := getVerified(sess)
	identity := GetIdentity(c)
	if UserID(c) == userID &&
		identity.IsAuthenticated &&
		identity.IsVerified == verified &&
		identity.DisplayName == name {
		return false
	}

	SetUserID(c, userID)
	SetIdentity(c, Identity{
		IsAuthenticated: true,
		IsVerified:      verified,
		DisplayName:     name,
	})
	return true
}
