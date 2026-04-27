package auth

import "github.com/go-sum/foundry/pkg/web"

type userIDKey struct{}
type userRoleKey struct{}
type identityKey struct{}

// Identity holds the request-scoped auth state for the current user,
// populated by the LoadSession middleware on every request.
type Identity struct {
	IsAuthenticated bool
	IsVerified      bool
	DisplayName     string
}

// SetUserID stores the authenticated user's ID in the request context.
func SetUserID(c *web.Context, id string) { c.Set(userIDKey{}, id) }

// UserID returns the authenticated user's ID from the request context.
func UserID(c *web.Context) string { v, _ := web.Get[string](c, userIDKey{}); return v }

// SetUserRole stores the authenticated user's role in the request context.
func SetUserRole(c *web.Context, role string) { c.Set(userRoleKey{}, role) }

// UserRole returns the authenticated user's role from the request context.
func UserRole(c *web.Context) string { v, _ := web.Get[string](c, userRoleKey{}); return v }

// SetIdentity stores the current user's identity in the request context.
func SetIdentity(c *web.Context, id Identity) { c.Set(identityKey{}, id) }

// GetIdentity returns the current user's identity from the request context.
// Returns a zero Identity (unauthenticated) if none has been set.
func GetIdentity(c *web.Context) Identity { v, _ := web.Get[Identity](c, identityKey{}); return v }
