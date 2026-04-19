// Package pagination provides navigation pagination components.
package pagination

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	icons "github.com/go-sum/componentry/icons"
	iconrender "github.com/go-sum/componentry/icons/render"
	core "github.com/go-sum/componentry/ui/core"
)

// Root renders a <nav role="navigation"> wrapper.
func Root(children ...g.Node) g.Node {
	return h.Nav(
		h.Role("navigation"),
		g.Attr("aria-label", "pagination"),
		h.Class("flex flex-wrap justify-center"),
		g.Group(children),
	)
}

// Content renders the <ul> flex row.
func Content(children ...g.Node) g.Node {
	return h.Ul(
		h.Class("flex flex-row items-center gap-1"),
		g.Group(children),
	)
}

// Item renders a <li> wrapper.
func Item(children ...g.Node) g.Node {
	return h.Li(g.Group(children))
}

func paginationLinkBase(isActive bool) string {
	base := "inline-flex items-center justify-center size-9 rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
	if isActive {
		return base + " border border-border bg-background shadow-xs"
	}
	return base + " hover:bg-accent hover:text-accent-foreground"
}

// Link renders a page number link.
func Link(href string, isActive bool, children ...g.Node) g.Node {
	nodes := []g.Node{
		h.Class(paginationLinkBase(isActive)),
		h.Href(href),
	}
	if isActive {
		nodes = append(nodes, g.Attr("aria-current", "page"))
	}
	nodes = append(nodes, g.Group(children))
	return h.A(nodes...)
}

// Previous renders a "previous" navigation button.
// Renders a <span> when disabled to carry correct semantics for assistive technology.
func Previous(href string, disabled bool, extra ...g.Node) g.Node {
	baseCls := "inline-flex items-center gap-1 px-2.5 h-9 rounded-md text-sm font-medium outline-none transition-colors focus-visible:ring-[3px] focus-visible:ring-ring/50"
	icon := core.Icon(iconrender.PropsFor(icons.ChevronLeft, core.IconProps{}))
	ariaLabel := g.Attr("aria-label", "Go to previous page")
	if disabled {
		return h.Span(
			h.Class(baseCls+" pointer-events-none opacity-50"),
			g.Attr("aria-disabled", "true"),
			ariaLabel,
			g.Group(extra),
			icon,
			h.Span(g.Text("Previous")),
		)
	}
	return h.A(
		h.Class(baseCls+" hover:bg-accent hover:text-accent-foreground"),
		h.Href(href),
		ariaLabel,
		g.Group(extra),
		icon,
		h.Span(g.Text("Previous")),
	)
}

// Next renders a "next" navigation button.
// Renders a <span> when disabled to carry correct semantics for assistive technology.
func Next(href string, disabled bool, extra ...g.Node) g.Node {
	baseCls := "inline-flex items-center gap-1 px-2.5 h-9 rounded-md text-sm font-medium outline-none transition-colors focus-visible:ring-[3px] focus-visible:ring-ring/50"
	icon := core.Icon(iconrender.PropsFor(icons.ChevronRight, core.IconProps{}))
	ariaLabel := g.Attr("aria-label", "Go to next page")
	if disabled {
		return h.Span(
			h.Class(baseCls+" pointer-events-none opacity-50"),
			g.Attr("aria-disabled", "true"),
			ariaLabel,
			g.Group(extra),
			h.Span(g.Text("Next")),
			icon,
		)
	}
	return h.A(
		h.Class(baseCls+" hover:bg-accent hover:text-accent-foreground"),
		h.Href(href),
		ariaLabel,
		g.Group(extra),
		h.Span(g.Text("Next")),
		icon,
	)
}

// Ellipsis renders a "…" placeholder for skipped page ranges.
func Ellipsis() g.Node {
	return h.Span(
		h.Class("flex size-9 items-center justify-center"),
		g.Attr("aria-hidden", "true"),
		g.Text("…"),
	)
}
