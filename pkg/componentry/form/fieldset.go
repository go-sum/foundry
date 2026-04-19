package form

import (
	"strings"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// FieldSetProps configures a <fieldset> grouping for related controls.
type FieldSetProps struct {
	ID          string
	Legend      string
	Description string
	Hint        string
	Errors      []string
	Disabled    bool
	Extra       []g.Node
}

// FieldSet renders a <fieldset> with optional legend, description, hint, and error output.
func FieldSet(p FieldSetProps, children ...g.Node) g.Node {
	nodes := []g.Node{h.Class("grid gap-2")}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
		ids := make([]string, 0, 3)
		if p.Description != "" {
			ids = append(ids, descriptionID(p.ID))
		}
		if p.Hint != "" {
			ids = append(ids, hintID(p.ID))
		}
		if len(p.Errors) > 0 {
			ids = append(ids, errorID(p.ID))
		}
		if len(ids) > 0 {
			nodes = append(nodes, g.Attr("aria-describedby", strings.Join(ids, " ")))
		}
		if len(p.Errors) > 0 {
			nodes = append(nodes, g.Attr("aria-errormessage", errorID(p.ID)))
		}
	}
	if p.Disabled {
		nodes = append(nodes, h.Disabled())
	}
	nodes = append(nodes, g.Group(p.Extra))
	if p.Legend != "" {
		nodes = append(nodes, Legend(g.Text(p.Legend)))
	}
	nodes = append(nodes, Description(p.ID, p.Description))
	nodes = append(nodes, g.Group(children))
	nodes = append(nodes, Hint(p.ID, p.Hint))
	nodes = append(nodes, ErrorMessage(p.ID, p.Errors...))
	return h.FieldSet(nodes...)
}

// Legend renders a styled <legend> element.
func Legend(children ...g.Node) g.Node {
	return h.Legend(
		h.Class("text-sm font-medium leading-none"),
		g.Group(children),
	)
}
