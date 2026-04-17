// Package page provides full-page gomponent views.
package page

import (
	"github.com/go-sum/foundry/internal/view"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// HomePage renders the home page.
func HomePage(req view.Request) g.Node {
	content := HomeContent()
	return req.Page("Home", content)
}

// HomeContent renders the home page content (for HTMX partial).
func HomeContent() g.Node {
	return h.Div(h.Class("max-w-2xl mx-auto py-16 px-4"),
		h.H1(h.Class("text-3xl font-bold text-gray-900 mb-4"),
			g.Text("Welcome to Foundry"),
		),
		h.P(h.Class("text-gray-600 mb-8"),
			g.Text("A Go web application built on W3C Web API primitives."),
		),
		h.A(h.Href("/hello/World"), h.Class("text-blue-600 hover:underline"),
			g.Text("Say hello to World"),
		),
	)
}
