package core

import (
	"cmp"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type popoverNS struct{}

// Popover groups the CSS-first popover sub-components under a namespace.
// It uses native <details>/<summary> for click-toggle with no JavaScript.
var Popover popoverNS

// PopoverRootProps configures the popover root element.
type PopoverRootProps struct {
	ID    string
	Class string
	Extra []g.Node
}

// PopoverTriggerProps configures the <summary> trigger element.
type PopoverTriggerProps struct {
	Class string
	Extra []g.Node
}

// PopoverContentProps configures the floating panel.
type PopoverContentProps struct {
	Width string
	Align string
	Extra []g.Node
}

// Root renders <details data-popover class="relative inline-block [Class]">.
func (popoverNS) Root(p PopoverRootProps, children ...g.Node) g.Node {
	cls := cmp.Or(p.Class, "relative inline-block")
	nodes := []g.Node{
		g.Attr("data-popover", ""),
		h.Class(cls),
	}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	nodes = append(nodes, g.Group(p.Extra))
	nodes = append(nodes, g.Group(children))
	return h.Details(nodes...)
}

// Trigger renders <summary class="list-none cursor-pointer [Class]">.
func (popoverNS) Trigger(p PopoverTriggerProps, children ...g.Node) g.Node {
	cls := "list-none cursor-pointer outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
	if p.Class != "" {
		cls += " " + p.Class
	}
	nodes := []g.Node{
		h.Class(cls),
		g.Group(p.Extra),
		g.Group(children),
	}
	return h.Summary(nodes...)
}

// Content renders the positioned floating panel, shown when <details> is open.
func (popoverNS) Content(p PopoverContentProps, children ...g.Node) g.Node {
	width := cmp.Or(p.Width, "w-72")
	align := "left-0"
	switch p.Align {
	case "right":
		align = "right-0"
	case "center":
		align = "left-1/2 -translate-x-1/2"
	}
	cls := "absolute top-full z-50 mt-1 " + width + " " + align +
		" rounded-md border border-border bg-popover shadow-md"
	nodes := []g.Node{h.Class(cls)}
	nodes = append(nodes, g.Group(p.Extra))
	nodes = append(nodes, g.Group(children))
	return h.Div(nodes...)
}
