// Package breadcrumb provides navigation breadcrumb components.
package breadcrumb

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Root renders a <nav aria-label="breadcrumb"> container.
func Root(children ...g.Node) g.Node {
	return h.Nav(
		g.Attr("aria-label", "breadcrumb"),
		h.Class("flex"),
		g.Group(children),
	)
}

// List renders the <ol> list.
func List(children ...g.Node) g.Node {
	return h.Ol(
		h.Class("flex items-center flex-wrap gap-1 text-sm"),
		g.Group(children),
	)
}

// Item renders a <li> item.
func Item(children ...g.Node) g.Node {
	return h.Li(
		h.Class("flex items-center"),
		g.Group(children),
	)
}

// Link renders an anchor link for a non-current breadcrumb.
func Link(href string, children ...g.Node) g.Node {
	return h.A(
		h.Href(href),
		h.Class("text-muted-foreground hover:text-foreground hover:underline flex items-center gap-1.5 transition-colors"),
		g.Group(children),
	)
}

// Separator renders a "/" separator between items.
// Pass children to use a custom separator symbol.
func Separator(children ...g.Node) g.Node {
	content := g.Node(g.Text("/"))
	if len(children) > 0 {
		content = g.Group(children)
	}
	return h.Span(
		h.Class("mx-2 text-muted-foreground"),
		g.Attr("aria-hidden", "true"),
		content,
	)
}

// Page renders the current page indicator (non-link).
func Page(children ...g.Node) g.Node {
	return h.Span(
		h.Class("font-medium text-foreground flex items-center gap-1.5"),
		g.Attr("aria-current", "page"),
		g.Group(children),
	)
}
