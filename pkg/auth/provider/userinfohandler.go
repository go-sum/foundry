package provider

import (
	"context"
	"log/slog"
	"strings"
	"time"

	auth "github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/google/uuid"
)

// UserinfoClaims is the JSON response body for GET /oauth/userinfo.
type UserinfoClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
}

// UserinfoUserReader fetches a user by ID for the userinfo endpoint.
// The authpgstore.Store satisfies this interface.
type UserinfoUserReader interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (auth.User, error)
}

// UserinfoHandler serves GET /oauth/userinfo. It validates a bearer access
// token and returns standard OIDC claims for the token's owner.
type UserinfoHandler struct {
	tokens TokenStore
	users  UserinfoUserReader
	logger *slog.Logger
}

// Serve handles GET /oauth/userinfo.
func (h *UserinfoHandler) Serve(c *web.Context) (web.Response, error) {
	raw, err := h.extractBearerToken(c)
	if err != nil {
		return web.Response{}, err
	}

	hash := HashToken(raw)
	token, err := h.tokens.GetTokenByHash(c.Context(), hash)
	if err != nil {
		return web.Response{}, web.ErrUnauthorized("invalid or unknown token")
	}
	if token.Revoked {
		return web.Response{}, web.ErrUnauthorized("token has been revoked")
	}
	if time.Now().UTC().After(token.ExpiresAt) {
		return web.Response{}, web.ErrUnauthorized("token has expired")
	}
	if token.TokenType != "access" {
		return web.Response{}, web.ErrUnauthorized("not an access token")
	}

	user, err := h.users.GetUserByID(c.Context(), token.UserID)
	if err != nil {
		h.logger.WarnContext(c.Context(), "provider: userinfo: user not found for token", "user_id", token.UserID, "error", err)
		return web.Response{}, web.ErrUnauthorized("user not found")
	}

	return web.JSON(200, UserinfoClaims{
		Sub:           user.ID.String(),
		Email:         user.Email,
		EmailVerified: user.Verified,
		Name:          user.DisplayName,
	}), nil
}

func (h *UserinfoHandler) extractBearerToken(c *web.Context) (string, error) {
	hdr := c.Request.Headers.Get("Authorization")
	if hdr == "" {
		return "", web.ErrUnauthorized("Authorization header required")
	}
	parts := strings.SplitN(hdr, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", web.ErrUnauthorized("malformed Authorization header — expected Bearer <token>")
	}
	if parts[1] == "" {
		return "", web.ErrUnauthorized("empty bearer token")
	}
	return parts[1], nil
}
