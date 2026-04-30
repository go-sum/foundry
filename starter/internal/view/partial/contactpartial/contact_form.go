// Package contactpartial provides the contact form HTMX partial.
package contactpartial

import (
	"github.com/go-sum/foundry/pkg/componentry/form"
	pform "github.com/go-sum/foundry/pkg/componentry/patterns/form"
	"github.com/go-sum/foundry/pkg/componentry/ui/core"
	"github.com/go-sum/foundry/pkg/componentry/ui/feedback"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// FormData carries view state for the contact form template.
type FormData struct {
	Name    string
	Email   string
	Message string
	Errors  map[string][]string
	Sent    bool
}

// ContactForm renders the contact form or success state inside a swap target div.
func ContactForm(req viewstate.Request, submitURL string, data FormData) g.Node {
	return h.Div(h.ID("contact-form"),
		g.If(data.Sent, successState()),
		g.If(!data.Sent, formState(req, submitURL, data)),
	)
}

func successState() g.Node {
	return feedback.Alert.Root(feedback.AlertProps{Variant: feedback.AlertDefault},
		feedback.Alert.Title(g.Text("Message sent!")),
		feedback.Alert.Description(g.Text("Thanks for reaching out. We'll be in touch soon.")),
	)
}

func formState(req viewstate.Request, submitURL string, data FormData) g.Node {
	return h.Form(
		h.ID("contact-form-inner"),
		g.Attr("hx-post", submitURL),
		g.Attr("hx-target", "#contact-form"),
		g.Attr("hx-swap", "outerHTML"),
		pform.CSRFHeaders(pform.CSRFProps{
			Token:      req.CSRFToken,
			HeaderName: req.CSRFHeaderName,
		}),
		h.Div(h.Class("grid gap-4"),
		form.Field(form.FieldProps{
			ID:       "name",
			Label:    "Name",
			Errors:   data.Errors["name"],
			Required: true,
			Control: form.Input(form.InputProps{
				ID:       "name",
				Name:     "name",
				Type:     form.TypeText,
				Value:    data.Name,
				HasError: len(data.Errors["name"]) > 0,
				Required: true,
			}),
		}),
		form.Field(form.FieldProps{
			ID:       "email",
			Label:    "Email",
			Errors:   data.Errors["email"],
			Required: true,
			Control: form.Input(form.InputProps{
				ID:       "email",
				Name:     "email",
				Type:     form.TypeEmail,
				Value:    data.Email,
				HasError: len(data.Errors["email"]) > 0,
				Required: true,
			}),
		}),
		form.Field(form.FieldProps{
			ID:       "message",
			Label:    "Message",
			Errors:   data.Errors["message"],
			Required: true,
			Control: form.Textarea(form.TextareaProps{
				ID:       "message",
				Name:     "message",
				Value:    data.Message,
				Rows:     5,
				HasError: len(data.Errors["message"]) > 0,
				Required: true,
			}),
		}),
		h.Div(
			core.Button(core.ButtonProps{
				Label:   "Send message",
				Variant: core.VariantDefault,
				Type:    "submit",
			}),
		),
		),
	)
}
