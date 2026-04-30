// Package page provides full-page gomponent views.
package page

import (
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ServiceStatus holds the name and health of a monitored service.
type ServiceStatus struct {
	Name    string
	Healthy bool
}

// HomePage renders the home page.
func HomePage(req viewstate.Request, statuses []ServiceStatus) g.Node {
	return req.Page("Home", HomeContent(req, statuses))
}

// HomeContent renders the home page content (for HTMX partial).
func HomeContent(_ viewstate.Request, statuses []ServiceStatus) g.Node {
	nodes := []g.Node{
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
	}
	if len(statuses) > 0 {
		nodes = append(nodes, serviceHealthCards(statuses))
	}
	return h.Div(h.Class("mx-auto flex max-w-3xl flex-col items-center justify-center gap-8 py-24 text-center"), g.Group(nodes))
}

func serviceHealthCards(statuses []ServiceStatus) g.Node {
	cards := make([]g.Node, len(statuses))
	for i, s := range statuses {
		cards[i] = serviceHealthCard(s)
	}
	return h.Div(h.Class("flex flex-wrap gap-3 justify-center"), g.Group(cards))
}

func serviceHealthCard(s ServiceStatus) g.Node {
	label := "Healthy"
	dot := "bg-green-500"
	if !s.Healthy {
		label = "Unavailable"
		dot = "bg-red-500"
	}
	return h.Div(
		h.Class("flex items-center gap-2 rounded-lg border bg-card px-4 py-3 text-sm shadow-sm"),
		h.Span(h.Class("size-2 rounded-full "+dot)),
		h.Span(h.Class("font-medium text-foreground"), g.Text(s.Name)),
		h.Span(h.Class("text-muted-foreground"), g.Text(label)),
	)
}
