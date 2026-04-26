package provider

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/go-sum/web"
	webauth "github.com/go-sum/web/auth"
	"github.com/go-sum/web/validate"
)

// tokenResponseJSON is the JSON body returned by the token endpoint.
type tokenResponseJSON struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// TokenHandler handles POST /oauth/token.
type TokenHandler struct {
	clients   ClientStore
	codes     CodeStore
	tokens    TokenStore
	config    Config
	validator validate.Validator
	logger    *slog.Logger
}

// tokenExchangeInput holds all fields that may appear in a token endpoint request.
// Parsed once in Exchange so the body is consumed only once.
type tokenExchangeInput struct {
	GrantType    string `form:"grant_type"`
	Code         string `form:"code"`
	RedirectURI  string `form:"redirect_uri"`
	ClientID     string `form:"client_id"`
	CodeVerifier string `form:"code_verifier"`
	RefreshToken string `form:"refresh_token"`
}

// Exchange handles POST /oauth/token.
// Supports grant_type=authorization_code and grant_type=refresh_token.
// The entire form body is parsed once here; sub-handlers receive pre-parsed values.
func (h *TokenHandler) Exchange(c *web.Context) (web.Response, error) {
	var input tokenExchangeInput
	if err := validate.Bind(h.validator, c.Request, &input); err != nil || input.GrantType == "" {
		return web.JSON(400, oauthErrorJSON("invalid_request", "grant_type is required")), nil
	}

	switch input.GrantType {
	case "authorization_code":
		return h.handleAuthorizationCode(c, input.Code, input.RedirectURI, input.ClientID, input.CodeVerifier)
	case "refresh_token":
		return h.handleRefreshToken(c, input.RefreshToken, input.ClientID)
	default:
		return web.JSON(400, oauthErrorJSON("unsupported_grant_type", fmt.Sprintf("unsupported grant_type: %s", input.GrantType))), nil
	}
}

func (h *TokenHandler) handleAuthorizationCode(c *web.Context, codeValue, redirectURI, clientID, codeVerifier string) (web.Response, error) {
	if codeValue == "" || redirectURI == "" || clientID == "" {
		return web.JSON(400, oauthErrorJSON("invalid_request", "missing required parameters")), nil
	}

	code, err := h.codes.GetCode(c.Context(), codeValue)
	if err != nil {
		return web.JSON(400, oauthErrorJSON("invalid_grant", "authorization code not found")), nil
	}
	if code.Used {
		return web.JSON(400, oauthErrorJSON("invalid_grant", "authorization code already used")), nil
	}
	if time.Now().UTC().After(code.ExpiresAt) {
		return web.JSON(400, oauthErrorJSON("invalid_grant", "authorization code expired")), nil
	}
	if code.ClientID != clientID {
		return web.JSON(400, oauthErrorJSON("invalid_grant", "client_id mismatch")), nil
	}
	if code.RedirectURI != redirectURI {
		return web.JSON(400, oauthErrorJSON("invalid_grant", "redirect_uri mismatch")), nil
	}

	// PKCE verification.
	if code.CodeChallenge != "" {
		if codeVerifier == "" {
			return web.JSON(400, oauthErrorJSON("invalid_grant", "code_verifier is required")), nil
		}
		if err := webauth.VerifyChallenge(codeVerifier, code.CodeChallenge); err != nil {
			return web.JSON(400, oauthErrorJSON("invalid_grant", "PKCE verification failed")), nil
		}
	}

	// Mark code as used (single-use enforcement).
	if err := h.codes.MarkCodeUsed(c.Context(), codeValue); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	return h.issueTokenPair(c, code.ClientID, code.UserID, code.Scopes, nil)
}

func (h *TokenHandler) handleRefreshToken(c *web.Context, refreshTokenValue, clientID string) (web.Response, error) {
	if refreshTokenValue == "" || clientID == "" {
		return web.JSON(400, oauthErrorJSON("invalid_request", "missing required parameters")), nil
	}

	hash := HashToken(refreshTokenValue)
	token, err := h.tokens.GetTokenByHash(c.Context(), hash)
	if err != nil {
		return web.JSON(400, oauthErrorJSON("invalid_grant", "refresh token not found")), nil
	}
	if token.TokenType != "refresh" {
		return web.JSON(400, oauthErrorJSON("invalid_grant", "not a refresh token")), nil
	}
	if token.Revoked {
		return web.JSON(400, oauthErrorJSON("invalid_grant", "refresh token revoked")), nil
	}
	if time.Now().UTC().After(token.ExpiresAt) {
		return web.JSON(400, oauthErrorJSON("invalid_grant", "refresh token expired")), nil
	}
	if token.ClientID != clientID {
		return web.JSON(400, oauthErrorJSON("invalid_grant", "client_id mismatch")), nil
	}

	// Revoke the old refresh token (rotation).
	if err := h.tokens.RevokeToken(c.Context(), token.ID); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	return h.issueTokenPair(c, token.ClientID, token.UserID, token.Scopes, &token.ID)
}

func (h *TokenHandler) issueTokenPair(c *web.Context, clientID string, userID uuid.UUID, scopes []string, parentID *uuid.UUID) (web.Response, error) {
	now := time.Now().UTC()

	// Issue access token.
	accessTokenValue, err := generateToken()
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	accessToken := OAuthToken{
		ID:        uuid.New(),
		TokenHash: HashToken(accessTokenValue),
		TokenType: "access",
		ClientID:  clientID,
		UserID:    userID,
		Scopes:    scopes,
		ExpiresAt: now.Add(h.config.AccessTokenTTL),
		CreatedAt: now,
	}
	if err := h.tokens.CreateToken(c.Context(), accessToken); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	// Issue refresh token.
	refreshTokenValue, err := generateToken()
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	refreshToken := OAuthToken{
		ID:        uuid.New(),
		TokenHash: HashToken(refreshTokenValue),
		TokenType: "refresh",
		ClientID:  clientID,
		UserID:    userID,
		Scopes:    scopes,
		ParentID:  parentID,
		ExpiresAt: now.Add(h.config.RefreshTokenTTL),
		CreatedAt: now,
	}
	if err := h.tokens.CreateToken(c.Context(), refreshToken); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	return web.JSON(200, tokenResponseJSON{
		AccessToken:  accessTokenValue,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.config.AccessTokenTTL.Seconds()),
		RefreshToken: refreshTokenValue,
		Scope:        joinScopes(scopes),
	}), nil
}

func oauthErrorJSON(code, description string) map[string]string {
	m := map[string]string{"error": code}
	if description != "" {
		m["error_description"] = description
	}
	return m
}

func joinScopes(scopes []string) string {
	return strings.Join(scopes, " ")
}
