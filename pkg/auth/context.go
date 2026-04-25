package auth

import "github.com/go-sum/web"

type userIDKey struct{}
type userRoleKey struct{}
type displayNameKey struct{}

// SetUserID stores the authenticated user's ID in the request context.
func SetUserID(c *web.Context, id string) { c.Set(userIDKey{}, id) }

// UserID returns the authenticated user's ID from the request context.
func UserID(c *web.Context) string { v, _ := web.Get[string](c, userIDKey{}); return v }

// SetUserRole stores the authenticated user's role in the request context.
func SetUserRole(c *web.Context, role string) { c.Set(userRoleKey{}, role) }

// UserRole returns the authenticated user's role from the request context.
func UserRole(c *web.Context) string { v, _ := web.Get[string](c, userRoleKey{}); return v }

// SetDisplayName stores the authenticated user's display name in the request context.
func SetDisplayName(c *web.Context, n string) { c.Set(displayNameKey{}, n) }

// DisplayName returns the authenticated user's display name from the request context.
func DisplayName(c *web.Context) string { v, _ := web.Get[string](c, displayNameKey{}); return v }
