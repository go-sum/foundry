// Package feedback provides notification and status-indicator components.
package feedback

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// AlertVariant selects the visual style of an alert.
type AlertVariant string

const (
	AlertDefault     AlertVariant = "default"
	AlertDestructive AlertVariant = "destructive"
)

// AlertProps configures a single alert banner.
type AlertProps struct {
	ID          string
	Variant     AlertVariant
	Dismissible bool
	// Icon is an optional leading icon node.
	// When set, the layout switches to a two-column grid so the icon sits in
	// its own column alongside the children.
	Icon  g.Node
	Extra []g.Node
}

func alertVariantClasses(v AlertVariant) string {
	if v == AlertDestructive {
		return "backdrop-blur-sm border-destructive/30 bg-destructive/20 text-destructive [&_[data-alert-description]]:text-destructive/80"
	}
	return "backdrop-blur-sm border-primary/30 bg-primary/20 text-primary [&_[data-alert-description]]:text-muted-foreground"
}

func alertAriaLive(v AlertVariant) string {
	if v == AlertDestructive {
		return "assertive"
	}
	return "polite"
}

// dismissButton renders a dismiss <button> with data-dismiss="alert" for JS delegation.
func dismissButton() g.Node {
	return h.Button(
		g.Attr("data-dismiss", "alert"),
		h.Class("absolute top-3 right-3 opacity-70 hover:opacity-100 transition-opacity outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"),
		h.Type("button"),
		g.Attr("aria-label", "Dismiss"),
		g.Text("×"),
	)
}

type alertNS struct{}

// Alert groups alert sub-components under a namespace: Alert.Root, Alert.Title,
// Alert.Description, Alert.List.
var Alert alertNS

// Root renders a shadcn/ui-style alert with ARIA live-region support.
// When Dismissible is true, a dismiss button is added.
// When Icon is set, the layout switches to a two-column grid.
func (alertNS) Root(p AlertProps, children ...g.Node) g.Node {
	var cls string
	if p.Icon != nil {
		cls = "relative w-full rounded-lg border px-4 py-3 text-sm grid grid-cols-[auto_1fr] gap-x-3 items-start " + alertVariantClasses(p.Variant)
	} else {
		cls = "relative w-full rounded-lg border px-4 py-3 text-sm grid gap-1.5 items-start " + alertVariantClasses(p.Variant)
	}
	nodes := []g.Node{
		h.Class(cls),
		h.Role("alert"),
		g.Attr("aria-live", alertAriaLive(p.Variant)),
	}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	if p.Dismissible {
		nodes = append(nodes, g.Attr("data-dismissible", ""))
	}
	nodes = append(nodes, g.Group(p.Extra))
	if p.Icon != nil {
		nodes = append(nodes,
			h.Div(g.Attr("data-alert-icon", ""), h.Class("mt-0.5"), p.Icon),
			h.Div(h.Class("grid gap-1.5"), g.Group(children)),
		)
	} else {
		nodes = append(nodes, g.Group(children))
	}
	if p.Dismissible {
		nodes = append(nodes, dismissButton())
	}
	return h.Div(nodes...)
}

// Title renders the alert heading.
func (alertNS) Title(children ...g.Node) g.Node {
	return h.H5(
		h.Class("line-clamp-1 min-h-4 font-medium tracking-tight"),
		g.Group(children),
	)
}

// Description renders the alert body text.
func (alertNS) Description(children ...g.Node) g.Node {
	return h.Div(
		h.Class("grid justify-items-start gap-1 text-sm"),
		g.Attr("data-alert-description", ""),
		g.Group(children),
	)
}

// List renders multiple dismissible alerts from parallel type/text slices.
// Types that are not "destructive" or "error" fall back to AlertDefault.
func (alertNS) List(types []string, texts []string) g.Node {
	n := len(texts)
	if len(types) < n {
		n = len(types)
	}
	nodes := make([]g.Node, n)
	for i := range n {
		nodes[i] = Alert.Root(
			AlertProps{Variant: alertVariantForType(types[i]), Dismissible: true},
			Alert.Description(g.Text(texts[i])),
		)
	}
	return g.Group(nodes)
}

func alertVariantForType(kind string) AlertVariant {
	switch kind {
	case "destructive", "error":
		return AlertDestructive
	default:
		return AlertDefault
	}
}
