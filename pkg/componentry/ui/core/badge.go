package core

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// BadgeVariant selects the visual style of a badge.
type BadgeVariant string

const (
	BadgeDefault     BadgeVariant = "default"
	BadgeSecondary   BadgeVariant = "secondary"
	BadgeDestructive BadgeVariant = "destructive"
	BadgeOutline     BadgeVariant = "outline"
)

// BadgeProps configures a Badge.
// Set Class instead of Variant to apply arbitrary Tailwind color utilities.
type BadgeProps struct {
	ID       string
	Variant  BadgeVariant
	Class    string
	Children []g.Node
	Extra    []g.Node
}

var _badgeVariantClasses = map[BadgeVariant]string{
	BadgeDestructive: "border-transparent bg-destructive text-white",
	BadgeOutline:     "text-foreground",
	BadgeSecondary:   "border-transparent bg-secondary text-secondary-foreground",
}

func badgeVariantClasses(v BadgeVariant) string {
	if c, ok := _badgeVariantClasses[v]; ok {
		return c
	}
	return "border-transparent bg-primary text-primary-foreground"
}

// Badge renders a small status indicator <span>.
func Badge(p BadgeProps) g.Node {
	variantCls := p.Class
	if variantCls == "" {
		variantCls = badgeVariantClasses(p.Variant)
	}
	cls := "inline-flex items-center justify-center rounded-md border px-2 py-0.5 text-xs font-medium w-fit whitespace-nowrap shrink-0 transition-colors " + variantCls
	nodes := []g.Node{h.Class(cls)}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	nodes = append(nodes, g.Group(p.Extra))
	nodes = append(nodes, g.Group(p.Children))
	return h.Span(nodes...)
}
