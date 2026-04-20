// Package page provides full-page gomponent views.
package page

import (
	"github.com/go-sum/componentry/ui/core"
	"github.com/go-sum/foundry/internal/view"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// HomePage renders the home page.
func HomePage(req view.Request, helloURL string) g.Node {
	content := HomeContent(req, helloURL)
	return req.Page("Home", content)
}

// HomeContent renders the home page content (for HTMX partial).
func HomeContent(req view.Request, helloURL string) g.Node {
	return h.Div(h.Class("max-w-2xl mx-auto py-16 px-4"),
		h.H1(h.Class("text-2xl font-bold text-foreground mb-4"),
			g.Text("Welcome to Foundry"),
		),
		h.P(h.Class("text-muted-foreground mb-8"),
			g.Text("A Go web application built on W3C Web API primitives."),
		),
		core.Button(core.ButtonProps{
			Href:    helloURL,
			Variant: core.VariantLink,
			Label:   "Say hello to World",
		}),
	)
}
