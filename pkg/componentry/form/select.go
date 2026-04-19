package form

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Option is a single <option> value/label pair.
type Option struct {
	Value string
	Label string
}

// OptGroup is a labeled group of options within a <select>.
type OptGroup struct {
	Label    string
	Disabled bool
	Options  []Option
}

// SelectProps configures a <select> element.
type SelectProps struct {
	ID       string
	Name     string
	Multiple bool
	Disabled bool
	Required bool
	HasError bool
	Options  []Option
	// Groups renders <optgroup> sections after flat Options.
	Groups []OptGroup
	// Selected remains a shorthand for single-select call sites.
	Selected string
	// SelectedValues is used when Multiple is true.
	SelectedValues []string
	Extra          []g.Node
}

func selectClass(hasError bool) string {
	base := inputBaseClass + " h-9 min-w-0 px-3 py-1"
	if hasError {
		base += inputErrorClass
	}
	return base
}

// Select renders a native <select> dropdown.
func Select(p SelectProps) g.Node {
	nodes := []g.Node{h.Class(selectClass(p.HasError))}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	if p.Name != "" {
		nodes = append(nodes, h.Name(p.Name))
	}
	if p.Disabled {
		nodes = append(nodes, h.Disabled())
	}
	if p.Required {
		nodes = append(nodes, g.Attr("required", ""))
	}
	if p.Multiple {
		nodes = append(nodes, g.Attr("multiple", ""))
	}
	if p.HasError {
		nodes = append(nodes, g.Attr("aria-invalid", "true"))
	}
	nodes = append(nodes, g.Group(p.Extra))

	selected := make(map[string]struct{}, len(p.SelectedValues))
	for _, value := range p.SelectedValues {
		selected[value] = struct{}{}
	}

	for _, opt := range p.Options {
		optNodes := []g.Node{h.Value(opt.Value), g.Text(opt.Label)}
		if _, ok := selected[opt.Value]; ok || opt.Value == p.Selected {
			optNodes = append([]g.Node{h.Selected()}, optNodes...)
		}
		nodes = append(nodes, h.Option(optNodes...))
	}
	for _, grp := range p.Groups {
		grpNodes := []g.Node{g.Attr("label", grp.Label)}
		if grp.Disabled {
			grpNodes = append(grpNodes, h.Disabled())
		}
		for _, opt := range grp.Options {
			optNodes := []g.Node{h.Value(opt.Value), g.Text(opt.Label)}
			if _, ok := selected[opt.Value]; ok || opt.Value == p.Selected {
				optNodes = append([]g.Node{h.Selected()}, optNodes...)
			}
			grpNodes = append(grpNodes, h.Option(optNodes...))
		}
		nodes = append(nodes, h.OptGroup(grpNodes...))
	}
	return h.Select(nodes...)
}
