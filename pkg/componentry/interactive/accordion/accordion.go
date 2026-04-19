// Package accordion provides a native HTML accordion component using <details>/<summary>.
// No JavaScript required — the browser handles open/close natively.
package accordion

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	icons "github.com/go-sum/componentry/icons"
	iconrender "github.com/go-sum/componentry/icons/render"
	core "github.com/go-sum/componentry/ui/core"
)

// Root renders the accordion container.
func Root(children ...g.Node) g.Node {
	return h.Div(
		h.Class("w-full divide-y divide-border rounded-lg border"),
		g.Group(children),
	)
}

// Item renders a single collapsible item as a native <details> element.
func Item(children ...g.Node) g.Node {
	return h.Details(
		h.Class("px-4"),
		g.Group(children),
	)
}

// Trigger renders the clickable <summary> header that toggles open/close.
func Trigger(children ...g.Node) g.Node {
	return h.Summary(
		h.Class("flex w-full items-center justify-between py-4 text-sm font-medium transition-all hover:underline text-left cursor-pointer"),
		g.Group(children),
		h.Span(
			h.Class("transition-transform duration-200 details-chevron"),
			core.Icon(iconrender.PropsFor(icons.ChevronDown, core.IconProps{Size: "size-4 shrink-0"})),
		),
	)
}

// Content renders the collapsible body of an accordion item.
func Content(children ...g.Node) g.Node {
	return h.Div(
		h.Class("pb-4 text-sm text-muted-foreground"),
		g.Group(children),
	)
}
