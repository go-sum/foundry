package form

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ToggleProps is the shared props type for Checkbox and Radio.
type ToggleProps struct {
	ID       string
	Name     string
	Value    string
	Checked  bool
	Disabled bool
	Required bool
	Extra    []g.Node
}

// CheckboxProps and RadioProps are aliases so existing callers compile unchanged.
type CheckboxProps = ToggleProps
type RadioProps = ToggleProps

// buildToggleInput builds the sr-only peer <input> nodes shared by Checkbox and Radio.
func buildToggleInput(inputType string, p ToggleProps) []g.Node {
	nodes := []g.Node{
		h.Class("sr-only peer"),
		h.Type(inputType),
	}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	if p.Name != "" {
		nodes = append(nodes, h.Name(p.Name))
	}
	if p.Value != "" {
		nodes = append(nodes, h.Value(p.Value))
	}
	if p.Checked {
		nodes = append(nodes, h.Checked())
	}
	if p.Disabled {
		nodes = append(nodes, h.Disabled())
	}
	if p.Required {
		nodes = append(nodes, g.Attr("required", ""))
	}
	nodes = append(nodes, g.Group(p.Extra))
	return nodes
}

// Checkbox renders a styled checkbox as a composite: a hidden peer <input> plus
// visual box and checkmark spans driven by peer-checked CSS.
func Checkbox(p CheckboxProps) g.Node {
	return h.Span(
		h.Class("relative inline-flex size-4 shrink-0 cursor-pointer"),
		h.Input(buildToggleInput("checkbox", p)...),
		h.Span(h.Class("absolute inset-0 rounded-[4px] border border-input bg-transparent transition-colors peer-checked:border-primary peer-checked:bg-primary peer-focus-visible:ring-[3px] peer-focus-visible:ring-ring/50 peer-disabled:opacity-50")),
		h.SVG(
			h.Class("absolute inset-0 m-auto size-3 hidden peer-checked:block text-primary-foreground"),
			g.Attr("viewBox", "0 0 10 10"),
			g.Attr("fill", "none"),
			g.Attr("stroke", "currentColor"),
			g.Attr("stroke-width", "1.5"),
			g.Attr("aria-hidden", "true"),
			g.El("path", g.Attr("d", "M2 6l3 3 5-5")),
		),
	)
}

// Radio renders a styled radio button as a composite: a hidden peer <input> plus
// visual ring and dot spans driven by peer-checked CSS.
func Radio(p RadioProps) g.Node {
	return h.Span(
		h.Class("relative inline-flex size-4 shrink-0 cursor-pointer"),
		h.Input(buildToggleInput("radio", p)...),
		h.Span(h.Class("absolute inset-0 rounded-full border border-input bg-transparent transition-colors peer-checked:border-primary peer-focus-visible:ring-[3px] peer-focus-visible:ring-ring/50 peer-disabled:opacity-50")),
		h.Span(h.Class("absolute inset-0 m-auto size-2 rounded-full bg-transparent transition-colors peer-checked:bg-primary")),
	)
}
