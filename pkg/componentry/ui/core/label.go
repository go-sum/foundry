package core

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// LabelProps configures a form label.
type LabelProps struct {
	For      string
	Required bool
	// Error, when non-empty, adds destructive colour styling.
	Error string
	Extra []g.Node
}

// Label renders a <label> element with optional required marker and error styling.
func Label(p LabelProps, children ...g.Node) g.Node {
	cls := "text-sm font-medium leading-none inline-block"
	if p.Error != "" {
		cls += " text-destructive"
	}
	nodes := []g.Node{h.Class(cls)}
	if p.For != "" {
		nodes = append(nodes, h.For(p.For))
	}
	nodes = append(nodes, g.Group(p.Extra))
	nodes = append(nodes, g.Group(children))
	if p.Required {
		nodes = append(nodes, h.Span(
			h.Class("ml-0.5 text-destructive"),
			g.Attr("aria-hidden", "true"),
			g.Text("*"),
		))
	}
	return h.Label(nodes...)
}
