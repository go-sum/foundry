package provider

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	auth "github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/go-sum/foundry/pkg/web/validate"
)

const authzParamsSessionKey = "oauth.authz_params"

// AuthorizeHandler handles the /oauth/authorize endpoint.
type AuthorizeHandler struct {
	clients   ClientStore
	codes     CodeStore
	consents  ConsentStore
	renderer  ConsentRenderer
	config    Config
	validator validate.Validator
	logger    *slog.Logger
}

// Show handles GET /oauth/authorize.
// Validates the authorization request, checks existing consent, and either
// issues a code immediately or renders the consent screen.
func (h *AuthorizeHandler) Show(c *web.Context) (web.Response, error) {
	params, client, err := h.parseAndValidateAuthzParams(c)
	if err != nil {
		return web.Response{}, err
	}

	sess, ok := session.FromContext(c)
	if !ok {
		return web.Response{}, web.ErrInternal(fmt.Errorf("provider: no session in context"))
	}

	userIDStr := auth.UserID(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return web.Response{}, web.ErrUnauthorized("Not authenticated")
	}

	// Check if user has already consented to all requested scopes.
	if consent, err := h.consents.GetConsent(c.Context(), userID, params.ClientID); err == nil {
		if scopesGranted(consent.Scopes, params.Scopes) {
			return h.issueCodeAndRedirect(c, sess, params, userID)
		}
	}

	// First-party clients are auto-approved — skip the consent screen.
	// The full authorization code + PKCE flow still runs; only the UI step is omitted.
	if client.FirstParty {
		return h.issueCodeAndRedirect(c, sess, params, userID)
	}

	// Store params in session for the POST handler.
	if err := sess.Set(authzParamsSessionKey, params); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	return h.renderer.ConsentPage(c, ConsentPageData{
		Client:          client,
		RequestedScopes: params.Scopes,
		Params:          params,
	})
}

// Submit handles POST /oauth/authorize (consent form submission).
func (h *AuthorizeHandler) Submit(c *web.Context) (web.Response, error) {
	sess, ok := session.FromContext(c)
	if !ok {
		return web.Response{}, web.ErrInternal(fmt.Errorf("provider: no session in context"))
	}

	userIDStr := auth.UserID(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return web.Response{}, web.ErrUnauthorized("Not authenticated")
	}

	storedParams, ok, err := session.Get[AuthorizeParams](sess, authzParamsSessionKey)
	if err != nil || !ok {
		return web.Response{}, web.ErrBadRequest("No authorization request in progress")
	}

	var input struct {
		Action string `form:"action" validate:"required,oneof=approve deny"`
	}
	if err := validate.Bind(h.validator, c.Request, &input); err != nil {
		return web.Response{}, err
	}

	if input.Action != "approve" {
		sess.Unset(authzParamsSessionKey)
		return redirectError(storedParams.RedirectURI, storedParams.State, "access_denied", "User denied the request")
	}
	sess.Unset(authzParamsSessionKey)

	// Save consent for future requests.
	if err := h.consents.SaveConsent(c.Context(), Consent{
		ID:       uuid.New(),
		UserID:   userID,
		ClientID: storedParams.ClientID,
		Scopes:   storedParams.Scopes,
	}); err != nil {
		h.logger.WarnContext(c.Context(), "provider: failed to save consent", "error", err)
	}

	return h.issueCodeAndRedirect(c, sess, storedParams, userID)
}

func (h *AuthorizeHandler) parseAndValidateAuthzParams(c *web.Context) (AuthorizeParams, OAuthClient, error) {
	q := c.URL().Query()

	clientID := q.Get("client_id")
	if clientID == "" {
		return AuthorizeParams{}, OAuthClient{}, web.ErrBadRequest("client_id is required")
	}

	client, err := h.clients.GetClientByClientID(c.Context(), clientID)
	if err != nil {
		return AuthorizeParams{}, OAuthClient{}, web.ErrBadRequest("unknown client_id")
	}

	redirectURI := q.Get("redirect_uri")
	if !isAllowedRedirectURI(client.RedirectURIs, redirectURI) {
		return AuthorizeParams{}, OAuthClient{}, web.ErrBadRequest("redirect_uri not registered for this client")
	}

	if q.Get("response_type") != "code" {
		return AuthorizeParams{}, OAuthClient{}, web.ErrBadRequest("response_type must be code")
	}

	codeChallenge := q.Get("code_challenge")
	if h.config.RequirePKCE && codeChallenge == "" {
		return AuthorizeParams{}, OAuthClient{}, web.ErrBadRequest("code_challenge is required")
	}
	if codeChallenge != "" && q.Get("code_challenge_method") != "S256" {
		return AuthorizeParams{}, OAuthClient{}, web.ErrBadRequest("code_challenge_method must be S256")
	}

	rawScope := q.Get("scope")
	scopes := parseScopes(rawScope)
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}

	return AuthorizeParams{
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		Scopes:        scopes,
		State:         q.Get("state"),
		Nonce:         q.Get("nonce"),
		CodeChallenge: codeChallenge,
	}, client, nil
}

func (h *AuthorizeHandler) issueCodeAndRedirect(c *web.Context, sess *session.Session, params AuthorizeParams, userID uuid.UUID) (web.Response, error) {
	codeValue, err := generateToken()
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	code := AuthorizationCode{
		Code:          codeValue,
		ClientID:      params.ClientID,
		UserID:        userID,
		RedirectURI:   params.RedirectURI,
		Scopes:        params.Scopes,
		CodeChallenge: params.CodeChallenge,
		Nonce:         params.Nonce,
		ExpiresAt:     time.Now().UTC().Add(h.config.CodeTTL),
		CreatedAt:     time.Now().UTC(),
	}
	if err := h.codes.CreateCode(c.Context(), code); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	redirectURL := buildRedirectURL(params.RedirectURI, params.State, codeValue)
	return web.Redirect(302, redirectURL), nil
}

// redirectError builds an error redirect to the client's redirect_uri.
func redirectError(redirectURI, state, errCode, errDesc string) (web.Response, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return web.Response{}, fmt.Errorf("provider: invalid redirect URI: %w", err)
	}
	q := u.Query()
	q.Set("error", errCode)
	if errDesc != "" {
		q.Set("error_description", errDesc)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return web.Redirect(302, u.String()), nil
}

func buildRedirectURL(redirectURI, state, code string) string {
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func isAllowedRedirectURI(registered []string, requested string) bool {
	for _, r := range registered {
		if r == requested {
			return true
		}
	}
	return false
}

func parseScopes(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Fields(raw)
}

func scopesGranted(granted, requested []string) bool {
	grantedSet := make(map[string]struct{}, len(granted))
	for _, s := range granted {
		grantedSet[s] = struct{}{}
	}
	for _, s := range requested {
		if _, ok := grantedSet[s]; !ok {
			return false
		}
	}
	return true
}
