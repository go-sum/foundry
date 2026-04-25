// Package dropdown provides a native HTML dropdown using <details>/<summary>.
// Open/close is handled natively by the browser.
package dropdown

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	icons "github.com/go-sum/componentry/icons"
	iconrender "github.com/go-sum/componentry/icons/render"
	core "github.com/go-sum/componentry/ui/core"
)

// Props configures a dropdown root.
type Props struct {
	ID    string
	Extra []g.Node
}

// TriggerProps configures the native <summary> trigger.
type TriggerProps struct {
	Icons *icons.Registry
	Extra []g.Node
}

// ItemProps configures a single menu entry.
type ItemProps struct {
	Label    string
	Href     string
	Disabled bool
	Extra    []g.Node
}

// Root renders a dropdown root using core.Popover.Root.
func Root(p Props, children ...g.Node) g.Node {
	extra := append([]g.Node{g.Attr("data-controller", "dropdown")}, p.Extra...)
	return core.Popover.Root(core.PopoverRootProps{
		ID:    p.ID,
		Extra: extra,
	}, children...)
}

// Trigger renders a styled <summary> via core.Popover.Trigger.
func Trigger(p TriggerProps, children ...g.Node) g.Node {
	chevron := core.Icon(iconrender.PropsForRegistry(p.Icons, icons.ChevronDown, core.IconProps{Size: "size-4 shrink-0 text-muted-foreground"}))
	return core.Popover.Trigger(core.PopoverTriggerProps{
		Class: "flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground",
		Extra: p.Extra,
	}, g.Group(children), chevron)
}

// Content renders the dropdown panel, visible when <details> is open.
func Content(children ...g.Node) g.Node {
	return h.Div(
		h.Class("absolute z-50 mt-1 min-w-[8rem] rounded-md border border-border bg-popover p-1 shadow-md"),
		g.Group(children),
	)
}

// Item renders a single menu entry as <a> (href set) or <button>.
func Item(p ItemProps) g.Node {
	cls := "flex w-full items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors focus-visible:bg-accent focus-visible:text-accent-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50"
	if p.Disabled {
		cls += " opacity-50"
	} else {
		cls += " cursor-default hover:bg-accent hover:text-accent-foreground"
	}
	if p.Href != "" {
		nodes := []g.Node{h.Class(cls)}
		if p.Disabled {
			nodes = append(nodes, g.Attr("aria-disabled", "true"), g.Attr("tabindex", "-1"))
		} else {
			nodes = append(nodes, h.Href(p.Href))
		}
		nodes = append(nodes, g.Group(p.Extra), g.Text(p.Label))
		return h.A(nodes...)
	}
	nodes := []g.Node{
		h.Class(cls),
		h.Type("button"),
	}
	if p.Disabled {
		nodes = append(nodes, h.Disabled())
	}
	nodes = append(nodes, g.Group(p.Extra), g.Text(p.Label))
	return h.Button(nodes...)
}

// Separator renders a horizontal rule between menu sections.
func Separator() g.Node {
	return h.Div(h.Class("-mx-1 my-1 h-px bg-muted"), h.Role("separator"))
}

// Label renders a non-interactive section heading.
func Label(label string) g.Node {
	return h.Div(h.Class("px-2 py-1.5 text-sm font-semibold"), g.Text(label))
}
