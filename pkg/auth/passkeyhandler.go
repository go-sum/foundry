package auth

import (
	"strings"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/go-sum/foundry/pkg/web/validate"
	"github.com/google/uuid"
)

// PasskeyHandler handles HTTP requests for WebAuthn passkey operations.
type PasskeyHandler struct {
	svc       *PasskeyService
	router    *router.Router
	validator validate.Validator
}

// BeginAuthentication starts a WebAuthn discoverable authentication ceremony.
func (h *PasskeyHandler) BeginAuthentication(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)

	options, ceremony, err := h.svc.BeginAuthentication(c.Context())
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	if err := setPasskeyCeremony(sess, passkeyCeremonyState{
		Operation: "authenticate",
		Ceremony:  ceremony,
	}); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	return web.JSON(200, options), nil
}

// FinishAuthentication completes a WebAuthn discoverable authentication ceremony.
func (h *PasskeyHandler) FinishAuthentication(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)

	state, ok := getPasskeyCeremony(sess)
	if !ok || state.Operation != "authenticate" {
		return web.Response{}, web.ErrBadRequest("No authentication ceremony in progress")
	}
	clearPasskeyCeremony(sess)

	httpReq := toHTTPRequest(c)
	result, err := h.svc.FinishAuthentication(c.Context(), state.Ceremony, httpReq)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	sess.Regenerate()
	if err := SetAuth(sess, result.User.ID.String(), result.User.DisplayName, result.User.Verified); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	return web.JSON(200, map[string]string{"redirect": "/"}), nil
}

// BeginRegistration starts a WebAuthn credential registration ceremony for the
// authenticated user.
func (h *PasskeyHandler) BeginRegistration(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)
	uid := UserID(c)
	if uid == "" {
		return web.Response{}, web.ErrUnauthorized("Not authenticated")
	}
	userID, err := uuid.Parse(uid)
	if err != nil {
		return web.Response{}, web.ErrBadRequest("Invalid session")
	}

	options, ceremony, err := h.svc.BeginRegistration(c.Context(), userID)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	if err := setPasskeyCeremony(sess, passkeyCeremonyState{
		Operation: "register",
		Ceremony:  ceremony,
		UserID:    userID,
	}); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	return web.JSON(200, options), nil
}

// FinishRegistration completes a WebAuthn credential registration ceremony.
func (h *PasskeyHandler) FinishRegistration(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)

	state, ok := getPasskeyCeremony(sess)
	if !ok || state.Operation != "register" {
		return web.Response{}, web.ErrBadRequest("No registration ceremony in progress")
	}
	clearPasskeyCeremony(sess)

	// Parse and validate optional name from query param.
	name := sanitizePasskeyName(c.URL().Query().Get("name"))

	httpReq := toHTTPRequest(c)
	cred, err := h.svc.FinishRegistration(c.Context(), state.UserID, name, state.Ceremony, httpReq)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return web.JSON(200, cred), nil
}

// List returns the authenticated user's registered passkeys as JSON.
func (h *PasskeyHandler) List(c *web.Context) (web.Response, error) {
	uid := UserID(c)
	if uid == "" {
		return web.Response{}, web.ErrUnauthorized("Not authenticated")
	}
	userID, err := uuid.Parse(uid)
	if err != nil {
		return web.Response{}, web.ErrBadRequest("Invalid session")
	}

	creds, err := h.svc.ListPasskeys(c.Context(), userID)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return web.JSON(200, creds), nil
}

// Show returns details for a single passkey credential as JSON.
func (h *PasskeyHandler) Show(c *web.Context) (web.Response, error) {
	uid := UserID(c)
	userID, passkeyID, err := parseUserAndPasskeyID(uid, c.Param("id"))
	if err != nil {
		return web.Response{}, err
	}

	cred, err := h.svc.GetPasskey(c.Context(), userID, passkeyID)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return web.JSON(200, cred), nil
}

// RenameForm returns the passkey data for rendering an inline edit form.
func (h *PasskeyHandler) RenameForm(c *web.Context) (web.Response, error) {
	uid := UserID(c)
	userID, passkeyID, err := parseUserAndPasskeyID(uid, c.Param("id"))
	if err != nil {
		return web.Response{}, err
	}

	cred, err := h.svc.GetPasskey(c.Context(), userID, passkeyID)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return web.JSON(200, cred), nil
}

// Rename updates the display name of a passkey credential.
func (h *PasskeyHandler) Rename(c *web.Context) (web.Response, error) {
	uid := UserID(c)
	userID, passkeyID, err := parseUserAndPasskeyID(uid, c.Param("id"))
	if err != nil {
		return web.Response{}, err
	}

	var input struct {
		Name string `form:"name" json:"name" validate:"required,max=255"`
	}
	if err := validate.Bind(h.validator, c.Request, &input); err != nil {
		return web.Response{}, web.ErrValidation("Name is required")
	}

	cred, err := h.svc.RenamePasskey(c.Context(), userID, passkeyID, input.Name)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return web.JSON(200, cred), nil
}

// Delete removes a passkey credential.
func (h *PasskeyHandler) Delete(c *web.Context) (web.Response, error) {
	uid := UserID(c)
	userID, passkeyID, err := parseUserAndPasskeyID(uid, c.Param("id"))
	if err != nil {
		return web.Response{}, err
	}

	if err := h.svc.DeletePasskey(c.Context(), userID, passkeyID); err != nil {
		return web.Response{}, mapServiceError(err)
	}

	return web.Respond(200), nil
}

// sanitizePasskeyName trims whitespace from the provided name and enforces a
// 255-character maximum. Returns "Passkey" when the result is empty.
func sanitizePasskeyName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 255 {
		name = name[:255]
	}
	if name == "" {
		return "Passkey"
	}
	return name
}

// parseUserAndPasskeyID extracts and validates the user ID from the context and
// the passkey ID from a route parameter.
func parseUserAndPasskeyID(uid, rawID string) (uuid.UUID, uuid.UUID, error) {
	if uid == "" {
		return uuid.UUID{}, uuid.UUID{}, web.ErrUnauthorized("Not authenticated")
	}
	userID, err := uuid.Parse(uid)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, web.ErrBadRequest("Invalid session")
	}
	passkeyID, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, web.ErrBadRequest("Invalid passkey ID")
	}
	return userID, passkeyID, nil
}
