// Package data provides components for displaying structured data.
package data

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type cardNS struct{}

// Card groups card sub-components under a namespace: Card.Root, Card.Header, Card.Title, etc.
var Card cardNS

// Root renders a rounded card container.
func (cardNS) Root(children ...g.Node) g.Node {
	return h.Div(
		h.Class("w-full rounded-lg border bg-card text-card-foreground shadow-xs"),
		g.Group(children),
	)
}

// Header renders the card header area (contains title/description).
func (cardNS) Header(children ...g.Node) g.Node {
	return h.Div(
		h.Class("flex flex-col space-y-1.5 p-6 pb-0"),
		g.Group(children),
	)
}

// Title renders a card heading <h3>.
func (cardNS) Title(children ...g.Node) g.Node {
	return h.H3(
		h.Class("text-lg font-semibold leading-none tracking-tight"),
		g.Group(children),
	)
}

// Description renders a muted description paragraph.
func (cardNS) Description(children ...g.Node) g.Node {
	return h.P(
		h.Class("text-sm text-muted-foreground"),
		g.Group(children),
	)
}

// Content renders the main body area of a card.
func (cardNS) Content(children ...g.Node) g.Node {
	return h.Div(
		h.Class("p-6"),
		g.Group(children),
	)
}

// Footer renders the card footer area.
func (cardNS) Footer(children ...g.Node) g.Node {
	return h.Div(
		h.Class("flex items-center p-6 pt-0"),
		g.Group(children),
	)
}
