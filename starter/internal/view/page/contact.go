package page

import (
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/partial/contactpartial"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ContactPage renders the full contact page.
func ContactPage(req view.Request, submitURL string, data contactpartial.FormData) g.Node {
	return req.Page("Contact Us", ContactContent(req, submitURL, data))
}

// ContactContent renders the contact page body (for HTMX partial).
func ContactContent(req view.Request, submitURL string, data contactpartial.FormData) g.Node {
	return h.Div(h.Class("max-w-md mx-auto py-16 px-4"),
		h.H1(h.Class("text-2xl font-bold leading-tight mb-2"), g.Text("Contact Us")),
		h.P(h.Class("text-sm text-muted-foreground mb-8"), g.Text("Send us a message and we'll get back to you.")),
		contactpartial.ContactForm(req, submitURL, data),
	)
}
