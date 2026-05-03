package authweb

import (
	"github.com/go-sum/foundry/pkg/componentry/form"
	"github.com/go-sum/foundry/pkg/componentry/ui/core"
	"github.com/go-sum/foundry/pkg/componentry/ui/data"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/render"
	"github.com/go-sum/foundry/pkg/web/secure"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// signinView builds the signin form component.
func signinView(c *web.Context, d SigninPageData) g.Node {
	return data.Card.Root(
		data.Card.Header(
			data.Card.Title(g.Text("Sign in")),
			data.Card.Description(g.Text("Enter your email to receive a verification code.")),
		),
		data.Card.Content(
			formErrors(d.FormErrors),
			h.Form(
				h.Method("POST"),
				h.Action("/auth/signin"),
				csrfField(c),
				h.Div(h.Class("grid gap-4"),
					form.Field(form.FieldProps{
						ID:       "email",
						Label:    "Email",
						Required: true,
						Errors:   fieldErrors(d.Errors, "email"),
						Control: form.Input(form.InputProps{
							ID:          "email",
							Name:        "email",
							Type:        form.TypeEmail,
							Placeholder: "you@example.com",
							Value:       d.Input.Email,
							Required:    true,
							HasError:    hasFieldError(d.Errors, "email"),
						}),
					}),
					core.Button(core.ButtonProps{
						Type:      "submit",
						Label:     "Continue",
						FullWidth: true,
					}),
				),
			),
		),
		data.Card.Footer(
			h.P(h.Class("text-sm text-muted-foreground"),
				g.Text("Don't have an account? "),
				h.A(
					h.Class("text-primary underline-offset-4 hover:underline"),
					h.Href("/auth/signup"),
					g.Text("Sign up"),
				),
			),
		),
	)
}

// signupView builds the signup form component.
func signupView(c *web.Context, d SignupPageData) g.Node {
	return data.Card.Root(
		data.Card.Header(
			data.Card.Title(g.Text("Sign up")),
			data.Card.Description(g.Text("Create an account to get started.")),
		),
		data.Card.Content(
			formErrors(d.FormErrors),
			h.Form(
				h.Method("POST"),
				h.Action("/auth/signup"),
				csrfField(c),
				h.Div(h.Class("grid gap-4"),
					form.Field(form.FieldProps{
						ID:       "email",
						Label:    "Email",
						Required: true,
						Errors:   fieldErrors(d.Errors, "email"),
						Control: form.Input(form.InputProps{
							ID:          "email",
							Name:        "email",
							Type:        form.TypeEmail,
							Placeholder: "you@example.com",
							Value:       d.Input.Email,
							Required:    true,
							HasError:    hasFieldError(d.Errors, "email"),
						}),
					}),
					form.Field(form.FieldProps{
						ID:       "display_name",
						Label:    "Display name",
						Required: true,
						Errors:   fieldErrors(d.Errors, "display_name"),
						Control: form.Input(form.InputProps{
							ID:       "display_name",
							Name:     "display_name",
							Type:     form.TypeText,
							Value:    d.Input.DisplayName,
							Required: true,
							HasError: hasFieldError(d.Errors, "display_name"),
						}),
					}),
					core.Button(core.ButtonProps{
						Type:      "submit",
						Label:     "Create account",
						FullWidth: true,
					}),
				),
			),
		),
		data.Card.Footer(
			h.P(h.Class("text-sm text-muted-foreground"),
				g.Text("Already have an account? "),
				h.A(
					h.Class("text-primary underline-offset-4 hover:underline"),
					h.Href("/auth/signin"),
					g.Text("Sign in"),
				),
			),
		),
	)
}

// verifyView builds the verification code entry form.
func verifyView(c *web.Context, d VerifyPageData) g.Node {
	var headerDesc g.Node
	if d.State.Email != "" {
		headerDesc = data.Card.Description(
			g.Text("A verification code was sent to "),
			h.Strong(g.Text(d.State.Email)),
			g.Text("."),
		)
	} else {
		headerDesc = data.Card.Description(g.Text("Enter the 6-digit verification code."))
	}

	var resendNode g.Node
	if d.State.CanResend {
		resendNode = h.Form(
			h.Method("POST"),
			h.Action("/auth/verify/resend"),
			csrfField(c),
			h.Div(h.Class("text-center"),
				core.Button(core.ButtonProps{
					Type:    "submit",
					Variant: core.VariantLink,
					Label:   "Resend code",
				}),
			),
		)
	}

	return data.Card.Root(
		data.Card.Header(
			data.Card.Title(g.Text("Verify")),
			headerDesc,
		),
		data.Card.Content(
			formErrors(d.FormErrors),
			h.Form(
				h.Method("POST"),
				h.Action("/auth/verify"),
				csrfField(c),
				g.If(d.State.Token != "", h.Input(
					h.Type("hidden"),
					h.Name("token"),
					h.Value(d.State.Token),
				)),
				h.Div(h.Class("grid gap-4"),
					form.Field(form.FieldProps{
						ID:       "code",
						Label:    "Verification code",
						Required: true,
						Errors:   fieldErrors(d.Errors, "code"),
						Control: form.Input(form.InputProps{
							ID:          "code",
							Name:        "code",
							Type:        form.TypeText,
							Placeholder: "000000",
							Value:       d.Input.Code,
							Required:    true,
							HasError:    hasFieldError(d.Errors, "code"),
							Extra: []g.Node{
								g.Attr("inputmode", "numeric"),
								g.Attr("autocomplete", "one-time-code"),
								g.Attr("maxlength", "6"),
								g.Attr("pattern", "[0-9]{6}"),
							},
						}),
					}),
					core.Button(core.ButtonProps{
						Type:      "submit",
						Label:     "Verify",
						FullWidth: true,
					}),
				),
			),
			resendNode,
		),
	)
}

// emailChangeView builds the email change form.
func emailChangeView(c *web.Context, d EmailChangePageData) g.Node {
	return data.Card.Root(
		data.Card.Header(
			data.Card.Title(g.Text("Change email")),
			data.Card.Description(g.Text("Enter your new email address. A verification code will be sent to confirm.")),
		),
		data.Card.Content(
			formErrors(d.FormErrors),
			h.Form(
				h.Method("POST"),
				h.Action("/account/email-change"),
				csrfField(c),
				h.Div(h.Class("grid gap-4"),
					form.Field(form.FieldProps{
						ID:       "email",
						Label:    "New email",
						Required: true,
						Errors:   fieldErrors(d.Errors, "email"),
						Control: form.Input(form.InputProps{
							ID:          "email",
							Name:        "email",
							Type:        form.TypeEmail,
							Placeholder: "new@example.com",
							Value:       d.Input.Email,
							Required:    true,
							HasError:    hasFieldError(d.Errors, "email"),
						}),
					}),
					core.Button(core.ButtonProps{
						Type:      "submit",
						Label:     "Send verification code",
						FullWidth: true,
					}),
				),
			),
		),
	)
}

// csrfField returns a hidden CSRF token input for the current request.
func csrfField(c *web.Context) g.Node {
	return render.CSRFField(render.CSRFProps{
		Token:     secure.CSRFToken(c),
		FieldName: secure.CSRFFieldName(c),
	})
}

// formErrors renders form-level error messages as an alert list.
func formErrors(errs []string) g.Node {
	if len(errs) == 0 {
		return nil
	}
	items := make([]g.Node, len(errs))
	for i, e := range errs {
		items[i] = h.Li(g.Text(e))
	}
	return h.Div(
		h.Class("rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"),
		h.Ul(h.Class("list-disc pl-4 space-y-1"), g.Group(items)),
	)
}

// fieldErrors returns per-field error strings as a slice for FieldProps.Errors.
func fieldErrors(errs map[string]string, field string) []string {
	if errs == nil {
		return nil
	}
	msg, ok := errs[field]
	if !ok || msg == "" {
		return nil
	}
	return []string{msg}
}

// hasFieldError reports whether a field has a validation error.
func hasFieldError(errs map[string]string, field string) bool {
	if errs == nil {
		return false
	}
	_, ok := errs[field]
	return ok
}
