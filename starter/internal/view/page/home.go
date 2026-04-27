// Package page provides full-page gomponent views.
package page

import (
	"github.com/go-sum/foundry/pkg/componentry/ui/core"
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
	return h.Div(h.Class("mx-auto flex max-w-3xl flex-col items-center justify-center gap-8 py-24 text-center"),
		h.Div(
			h.Class("space-y-4"),
			h.P(
				h.Class("text-sm font-medium uppercase tracking-[0.2em] text-muted-foreground"),
				g.Text("Go Web Starter"),
			),
			h.H1(h.Class("text-2xl font-bold"),
				g.Text("Welcome to Foundry"),
			),
			h.P(h.Class("mx-auto max-w-2xl text-sm text-muted-foreground"),
				g.Text("A Go web application built on W3C Web API primitives."),
			),
		),
		h.Div(
			h.Class("flex flex-col gap-3 sm:flex-row"),
			core.Button(core.ButtonProps{
				Href:    helloURL,
				Variant: core.VariantDefault,
				Label:   "Get Started",
			}),
		),
	)
}
