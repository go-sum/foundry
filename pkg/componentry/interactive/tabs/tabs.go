// Package tabs provides an accessible tabs component using ARIA roles and data attributes.
// The initial active state is rendered server-side (SSR-ready, works without JS).
package tabs

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func triggerID(rootID, value string) string {
	return rootID + "-tab-" + value
}

func panelID(rootID, value string) string {
	return rootID + "-panel-" + value
}

// Root renders the root tabs container. defaultTab sets the initially active tab value.
func Root(id, defaultTab string, children ...g.Node) g.Node {
	return h.Div(
		h.ID(id),
		g.Attr("data-tabs", defaultTab),
		h.Class("w-full"),
		g.Group(children),
	)
}

// List renders the tab button bar.
func List(children ...g.Node) g.Node {
	return h.Div(
		h.Role("tablist"),
		g.Attr("aria-orientation", "horizontal"),
		h.Class("inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground"),
		g.Group(children),
	)
}

// Trigger renders a single tab button. isDefault marks the initially active tab.
func Trigger(rootID, value string, isDefault bool, children ...g.Node) g.Node {
	cls := "inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium transition-all focus-visible:outline-none focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50"
	ariaSelected := "false"
	tabIndex := "-1"
	if isDefault {
		ariaSelected = "true"
		tabIndex = "0"
		cls += " bg-background text-foreground shadow"
	}
	return h.Button(
		h.ID(triggerID(rootID, value)),
		h.Type("button"),
		h.Role("tab"),
		g.Attr("data-tab", value),
		g.Attr("aria-selected", ariaSelected),
		g.Attr("aria-controls", panelID(rootID, value)),
		g.Attr("tabindex", tabIndex),
		h.Class(cls),
		g.Group(children),
	)
}

// Content renders the panel for a specific tab value. isDefault controls
// whether the panel is visible on initial render.
func Content(rootID, value string, isDefault bool, children ...g.Node) g.Node {
	nodes := []g.Node{
		h.ID(panelID(rootID, value)),
		h.Role("tabpanel"),
		g.Attr("data-tab", value),
		g.Attr("aria-labelledby", triggerID(rootID, value)),
		g.Attr("tabindex", "0"),
		h.Class("mt-2"),
	}
	if !isDefault {
		nodes = append(nodes, g.Attr("hidden", ""))
	}
	nodes = append(nodes, g.Group(children))
	return h.Div(nodes...)
}
