package authweb

import (
	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web"
	webauth "github.com/go-sum/foundry/pkg/web/auth"
	"github.com/go-sum/foundry/pkg/web/htmx"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/go-sum/foundry/pkg/web/validate"
	authn "github.com/go-sum/foundry/pkg/web/authn"
	"github.com/google/uuid"
)

// Renderer produces full-page and partial HTML responses for auth views.
// The host application implements this interface to control layout and styling.
type Renderer interface {
	SigninPage(c *web.Context, data SigninPageData) (web.Response, error)
	SignupPage(c *web.Context, data SignupPageData) (web.Response, error)
	VerifyPage(c *web.Context, data VerifyPageData) (web.Response, error)
	EmailChangePage(c *web.Context, data EmailChangePageData) (web.Response, error)
}

// SigninPageData carries state for rendering the signin view.
type SigninPageData struct {
	Input      auth.BeginSigninInput
	Errors     map[string]string
	FormErrors []string
	Config     auth.Config
	ReturnTo   string // URL to return to after successful signin; empty means "/"
}

// SignupPageData carries state for rendering the signup view.
type SignupPageData struct {
	Input      auth.BeginSignupInput
	Errors     map[string]string
	FormErrors []string
	Config     auth.Config
	ReturnTo   string // URL to return to after successful signup; empty means "/"
}

// VerifyPageData carries state for rendering the verification view.
type VerifyPageData struct {
	Input      auth.VerifyInput
	Errors     map[string]string
	FormErrors []string
	State      auth.VerifyPageState
}

// EmailChangePageData carries state for rendering the email change view.
type EmailChangePageData struct {
	Input      auth.BeginEmailChangeInput
	Errors     map[string]string
	FormErrors []string
}

// AuthHandler handles HTTP requests for email-TOTP authentication flows.
type AuthHandler struct {
	svc         *auth.AuthService
	router      *router.Router
	validator   validate.Validator
	renderer    Renderer
	config      auth.Config
	signoutPath func() string
	routes      RouteConfig
}

// ShowSignin renders the signin form.
func (h *AuthHandler) ShowSignin(c *web.Context) (web.Response, error) {
	returnTo := webauth.SanitizeReturnTo(c.URL().Query().Get("return_to"))
	return h.renderer.SigninPage(c, SigninPageData{Config: h.config, ReturnTo: returnTo})
}

// BeginSignin starts a passwordless signin verification flow.
func (h *AuthHandler) BeginSignin(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)

	var input auth.BeginSigninInput
	if err := validate.Bind(h.validator, c.Request, &input); err != nil {
		return h.renderer.SigninPage(c, SigninPageData{
			Input:      input,
			FormErrors: []string{"Please enter a valid email address."},
			Config:     h.config,
			ReturnTo:   webauth.SanitizeReturnTo(input.ReturnTo),
		})
	}

	returnTo := webauth.SanitizeReturnTo(input.ReturnTo)
	verifyPath := h.router.MustReverse(h.routes.Verify.Name, nil)
	flow, err := h.svc.BeginSignin(c.Context(), input, verifyPath)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	flow.ReturnTo = returnTo
	if err := setPendingFlow(sess, flow); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	redirectURL := h.router.MustReverse(h.routes.Verify.Name, nil)
	return htmxRedirect(c, redirectURL)
}

// ShowSignup renders the signup form.
func (h *AuthHandler) ShowSignup(c *web.Context) (web.Response, error) {
	returnTo := webauth.SanitizeReturnTo(c.URL().Query().Get("return_to"))
	return h.renderer.SignupPage(c, SignupPageData{Config: h.config, ReturnTo: returnTo})
}

// BeginSignup starts a signup verification flow.
func (h *AuthHandler) BeginSignup(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)

	var input auth.BeginSignupInput
	if err := validate.Bind(h.validator, c.Request, &input); err != nil {
		return h.renderer.SignupPage(c, SignupPageData{
			Input:      input,
			FormErrors: []string{"Please correct the errors below."},
			Config:     h.config,
			ReturnTo:   webauth.SanitizeReturnTo(input.ReturnTo),
		})
	}

	returnTo := webauth.SanitizeReturnTo(input.ReturnTo)
	verifyPath := h.router.MustReverse(h.routes.Verify.Name, nil)
	flow, err := h.svc.BeginSignup(c.Context(), input, verifyPath)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	flow.ReturnTo = returnTo
	if err := setPendingFlow(sess, flow); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	redirectURL := h.router.MustReverse(h.routes.Verify.Name, nil)
	return htmxRedirect(c, redirectURL)
}

// ShowVerify renders the verification page. Supports both session-based flows
// and token-based flows from emailed verify links.
func (h *AuthHandler) ShowVerify(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)

	token := c.URL().Query().Get("token")
	if token != "" {
		state, err := h.svc.VerifyPageState(token)
		if err != nil {
			return h.renderer.VerifyPage(c, VerifyPageData{
				FormErrors: []string{"This verification link is invalid or has expired."},
			})
		}
		return h.renderer.VerifyPage(c, VerifyPageData{
			State: state,
			Input: auth.VerifyInput{Token: token},
		})
	}

	flow, ok := getPendingFlow(sess)
	if !ok {
		signinURL := h.router.MustReverse(h.routes.Signin.Name, nil)
		return web.SeeOther(signinURL), nil
	}

	return h.renderer.VerifyPage(c, VerifyPageData{
		State: auth.VerifyPageState{
			Purpose:   flow.Purpose,
			Email:     flow.Email,
			CanResend: true,
		},
	})
}

// Verify validates the submitted verification code against either a pending
// session flow or a self-contained token.
func (h *AuthHandler) Verify(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)

	var input auth.VerifyInput
	if err := validate.Bind(h.validator, c.Request, &input); err != nil {
		return h.renderer.VerifyPage(c, VerifyPageData{
			Input:      input,
			FormErrors: []string{"Please enter the 6-digit verification code."},
		})
	}

	var result auth.VerifyResult
	var err error
	var returnTo string

	if input.Token != "" {
		result, err = h.svc.VerifyToken(c.Context(), input.Token, input)
	} else {
		flow, ok := getPendingFlow(sess)
		if !ok {
			return web.Response{}, web.ErrBadRequest("No pending verification flow")
		}
		returnTo = flow.ReturnTo
		var updatedFlow auth.PendingFlow
		result, updatedFlow, err = h.svc.VerifyPendingFlow(c.Context(), flow, input)
		if err != nil {
			// Persist the attempt count back to the session so brute-force
			// attempts are counted across requests.
			_ = setPendingFlow(sess, updatedFlow)
			return web.Response{}, mapServiceError(err)
		}
	}
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	sess.Regenerate()
	if err := authn.SetAuth(sess, result.User.ID.String(), result.User.DisplayName, result.User.Verified); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	if returnTo == "" {
		returnTo = "/"
	}
	return web.SeeOther(returnTo), nil
}

// Resend re-sends the verification code for the current pending flow.
func (h *AuthHandler) Resend(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)

	flow, ok := getPendingFlow(sess)
	if !ok {
		return web.Response{}, web.ErrBadRequest("No pending verification flow")
	}

	verifyPath := h.router.MustReverse(h.routes.Verify.Name, nil)
	newFlow, err := h.svc.ResendPendingFlow(c.Context(), flow, verifyPath)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	if err := setPendingFlow(sess, newFlow); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	redirectURL := h.router.MustReverse(h.routes.Verify.Name, nil)
	return htmxRedirect(c, redirectURL)
}

// Signout destroys the session and redirects to the configured signout path.
func (h *AuthHandler) Signout(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)
	sess.Destroy()
	return web.SeeOther(h.signoutPath()), nil
}

// ShowEmailChange renders the email change form for the authenticated user.
func (h *AuthHandler) ShowEmailChange(c *web.Context) (web.Response, error) {
	return h.renderer.EmailChangePage(c, EmailChangePageData{})
}

// BeginEmailChange starts an email-change verification flow for the authenticated user.
func (h *AuthHandler) BeginEmailChange(c *web.Context) (web.Response, error) {
	sess, _ := session.FromContext(c)
	uid := authn.UserID(c)
	if uid == "" {
		return web.Response{}, web.ErrUnauthorized("Not authenticated")
	}
	userID, err := uuid.Parse(uid)
	if err != nil {
		return web.Response{}, web.ErrBadRequest("Invalid session")
	}

	var input auth.BeginEmailChangeInput
	if err := validate.Bind(h.validator, c.Request, &input); err != nil {
		return h.renderer.EmailChangePage(c, EmailChangePageData{
			Input:      input,
			FormErrors: []string{"Please enter a valid email address."},
		})
	}

	verifyPath := h.router.MustReverse(h.routes.Verify.Name, nil)
	flow, err := h.svc.BeginEmailChange(c.Context(), userID, input, verifyPath)
	if err != nil {
		return web.Response{}, mapServiceError(err)
	}

	if err := setPendingFlow(sess, flow); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	redirectURL := h.router.MustReverse(h.routes.Verify.Name, nil)
	return htmxRedirect(c, redirectURL)
}

// htmxRedirect returns the appropriate redirect response for HTMX vs full-page requests.
func htmxRedirect(c *web.Context, url string) (web.Response, error) {
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		resp := web.Respond(200)
		htmx.SetRedirect(&resp, url)
		return resp, nil
	}
	return web.SeeOther(url), nil
}
