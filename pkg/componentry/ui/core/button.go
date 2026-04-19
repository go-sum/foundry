// Package core provides fundamental UI building blocks used across all pages.
package core

import (
	"cmp"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Variant selects the visual style of a button.
type Variant string

const (
	VariantDefault          Variant = "default"
	VariantDestructive      Variant = "destructive"
	VariantDestructiveGhost Variant = "destructive-ghost"
	VariantOutline          Variant = "outline"
	VariantSecondary        Variant = "secondary"
	VariantGhost            Variant = "ghost"
	VariantLink             Variant = "link"
)

// Size selects the size of a button.
type Size string

const (
	SizeDefault Size = ""
	SizeSm      Size = "sm"
	SizeLg      Size = "lg"
)

// ButtonProps configures a Button. Set Href to render an <a> instead of <button>.
type ButtonProps struct {
	ID    string
	Label string
	// Type defaults to "button" to avoid accidental form submission.
	Type      string
	Href      string
	Target    string
	Variant   Variant
	Size      Size
	Disabled  bool
	FullWidth bool
	// Children overrides Label for icon buttons or mixed content.
	Children []g.Node
	Extra    []g.Node
}

const baseClasses = "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] cursor-pointer"

var _variantClasses = map[Variant]string{
	VariantDestructive:      "bg-destructive text-white shadow-xs hover:bg-destructive/90",
	VariantDestructiveGhost: "text-destructive hover:bg-destructive/10 hover:text-destructive",
	VariantOutline:          "border bg-background text-foreground shadow-xs hover:bg-accent hover:text-accent-foreground",
	VariantSecondary:        "bg-secondary text-secondary-foreground shadow-xs hover:bg-secondary/80",
	VariantGhost:            "hover:bg-accent hover:text-accent-foreground",
	VariantLink:             "text-foreground underline-offset-4 hover:underline",
}

func variantClasses(v Variant) string {
	if c, ok := _variantClasses[v]; ok {
		return c
	}
	return "bg-primary text-primary-foreground shadow-xs hover:bg-primary/90"
}

var _sizeClasses = map[Size]string{
	SizeSm: "h-8 rounded-md gap-1.5 px-3",
	SizeLg: "h-10 rounded-md px-6",
}

func sizeClasses(s Size) string {
	if c, ok := _sizeClasses[s]; ok {
		return c
	}
	return "h-9 px-4 py-2"
}

func buttonClass(p ButtonProps) string {
	cls := baseClasses + " " + variantClasses(p.Variant) + " " + sizeClasses(p.Size)
	if p.FullWidth {
		cls += " w-full"
	}
	if p.Disabled {
		cls += " pointer-events-none opacity-50"
	}
	return cls
}

func buttonType(t string) string {
	return cmp.Or(t, "button")
}

func buttonContent(p ButtonProps) g.Node {
	if len(p.Children) > 0 {
		return g.Group(p.Children)
	}
	return g.Text(p.Label)
}

// Button renders a <button> or <a> with Tailwind v4 semantic-token styling.
// When Href is set, renders an <a> element. A disabled <a> omits href,
// sets aria-disabled="true", and tabindex="-1".
func Button(p ButtonProps) g.Node {
	if p.Href != "" {
		nodes := []g.Node{h.Class(buttonClass(p))}
		if p.Disabled {
			nodes = append(nodes, g.Attr("aria-disabled", "true"), g.Attr("tabindex", "-1"))
		} else {
			nodes = append(nodes, h.Href(p.Href))
		}
		if p.ID != "" {
			nodes = append(nodes, h.ID(p.ID))
		}
		if p.Target != "" && !p.Disabled {
			nodes = append(nodes, h.Target(p.Target))
		}
		nodes = append(nodes, g.Group(p.Extra))
		nodes = append(nodes, buttonContent(p))
		return h.A(nodes...)
	}
	nodes := []g.Node{
		h.Class(buttonClass(p)),
		h.Type(buttonType(p.Type)),
	}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	if p.Disabled {
		nodes = append(nodes, h.Disabled())
	}
	nodes = append(nodes, g.Group(p.Extra))
	nodes = append(nodes, buttonContent(p))
	return h.Button(nodes...)
}
