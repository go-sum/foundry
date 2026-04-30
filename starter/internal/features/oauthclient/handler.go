// Package oauthclient implements the first-party OAuth 2.1 connect and callback
// handlers. The app acts as its own OAuth client, running the full
// authorization code + PKCE flow against its own Authorization Server.
package oauthclient

import (
	"net/http"
	"time"

	webauth "github.com/go-sum/foundry/pkg/web/auth"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/session"
)

// Handler handles the first-party OAuth connect and callback routes.
type Handler struct {
	provider webauth.ProviderConfig
	client   *http.Client
}

// New creates a Handler for the given first-party ProviderConfig.
func New(provider webauth.ProviderConfig) *Handler {
	return &Handler{
		provider: provider,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Connect initiates the first-party OAuth 2.1 flow.
// GET /auth/connect?return_to=...
//
// Generates a fresh OAuthTransaction (state, nonce, PKCE verifier), stores it
// in the session, and redirects to the Authorization Server's /oauth/authorize.
func (h *Handler) Connect(c *web.Context) (web.Response, error) {
	returnTo := c.URL().Query().Get("return_to")
	tx, authURL, err := webauth.BeginOAuth(h.provider, returnTo)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	sess, ok := session.FromContext(c)
	if !ok {
		return web.Response{}, web.ErrInternal(nil)
	}
	if err := sess.Set(webauth.SessionKey, tx); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	return web.SeeOther(authURL), nil
}

// Callback handles the Authorization Server's redirect after authorization.
// GET /auth/callback?code=...&state=...
//
// Flow:
//  1. Retrieve the OAuthTransaction from the session.
//  2. Verify the state parameter (CSRF guard).
//  3. Exchange the authorization code for tokens via POST to /oauth/token.
//  4. Fetch user claims from /oauth/userinfo with the access token.
//  5. Complete auth: regenerate session, clear transaction.
//  6. Establish the app session from the resolved identity claims.
//  7. Redirect to the original return_to URL.
func (h *Handler) Callback(c *web.Context) (web.Response, error) {
	sess, ok := session.FromContext(c)
	if !ok {
		return web.Response{}, web.ErrInternal(nil)
	}

	tx, hasTx, err := session.Get[webauth.OAuthTransaction](sess, webauth.SessionKey)
	if err != nil || !hasTx {
		return web.Response{}, web.ErrBadRequest("No OAuth transaction in progress")
	}

	q := c.URL().Query()
	if err := webauth.VerifyState(q.Get("state"), tx.State); err != nil {
		return web.Response{}, web.ErrBadRequest("Invalid state parameter")
	}

	tokens, err := webauth.ExchangeCode(c.Context(), h.client, webauth.ExchangeParams{
		TokenEndpoint: h.provider.TokenEndpoint,
		ClientID:      h.provider.ClientID,
		ClientSecret:  h.provider.ClientSecret,
		Code:          q.Get("code"),
		RedirectURI:   h.provider.RedirectURL,
		CodeVerifier:  tx.Verifier,
	})
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	claims, err := webauth.FetchUserinfo(c.Context(), h.client, h.provider.UserinfoEndpoint, tokens.AccessToken)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	returnTo, err := webauth.CompleteAuth(c.Context(), sess, tx)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	if err := authn.SetAuth(sess, claims.Sub, claims.Name, claims.EmailVerified); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	return web.SeeOther(returnTo), nil
}
