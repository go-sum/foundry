package form

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SwitchProps configures a toggle switch (styled checkbox).
type SwitchProps struct {
	ID       string
	Name     string
	Value    string
	Checked  bool
	Disabled bool
	Extra    []g.Node
}

// Switch renders a toggle switch as a composite: a hidden peer <input> plus
// a visual track and moving thumb span.
func Switch(p SwitchProps) g.Node {
	inputNodes := []g.Node{
		h.Class("sr-only peer"),
		h.Type("checkbox"),
		h.Role("switch"),
	}
	if p.ID != "" {
		inputNodes = append(inputNodes, h.ID(p.ID))
	}
	if p.Name != "" {
		inputNodes = append(inputNodes, h.Name(p.Name))
	}
	if p.Value != "" {
		inputNodes = append(inputNodes, h.Value(p.Value))
	}
	if p.Checked {
		inputNodes = append(inputNodes, h.Checked())
	}
	if p.Disabled {
		inputNodes = append(inputNodes, h.Disabled())
	}
	inputNodes = append(inputNodes, g.Group(p.Extra))
	return h.Span(
		h.Class("relative inline-flex h-5 w-9 shrink-0 cursor-pointer"),
		h.Input(inputNodes...),
		h.Span(h.Class("pointer-events-none absolute inset-0 rounded-full bg-input transition-colors peer-checked:bg-primary peer-focus-visible:ring-[3px] peer-focus-visible:ring-ring/50 peer-disabled:opacity-50")),
		h.Span(h.Class("pointer-events-none absolute left-0.5 top-0.5 size-4 rounded-full bg-white peer-checked:bg-primary-foreground shadow-xs transition-transform peer-checked:translate-x-4")),
	)
}
