package authui

import (
	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/htmx"
	"github.com/go-sum/foundry/pkg/web/render"
)

type renderer struct {
	cfg Config
}

// NewRenderer returns an auth.Renderer that builds views with componentry
// and delegates full-page layout to cfg.Page.
func NewRenderer(cfg Config) auth.Renderer {
	return &renderer{cfg: cfg}
}

// SigninPage renders the signin form.
func (r *renderer) SigninPage(c *web.Context, data auth.SigninPageData) (web.Response, error) {
	content := signinView(c, data)
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(content)
	}
	return r.cfg.Page(c, "Sign in", content)
}

// SignupPage renders the signup form.
func (r *renderer) SignupPage(c *web.Context, data auth.SignupPageData) (web.Response, error) {
	content := signupView(c, data)
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(content)
	}
	return r.cfg.Page(c, "Sign up", content)
}

// VerifyPage renders the verification code entry form.
func (r *renderer) VerifyPage(c *web.Context, data auth.VerifyPageData) (web.Response, error) {
	content := verifyView(c, data)
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(content)
	}
	return r.cfg.Page(c, "Verify", content)
}

// EmailChangePage renders the email change form.
func (r *renderer) EmailChangePage(c *web.Context, data auth.EmailChangePageData) (web.Response, error) {
	content := emailChangeView(c, data)
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(content)
	}
	return r.cfg.Page(c, "Change email", content)
}
