package form

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// TextareaProps configures a multi-line text field.
type TextareaProps struct {
	ID          string
	Name        string
	Placeholder string
	Value       string
	Rows        int
	Disabled    bool
	Readonly    bool
	Required    bool
	HasError    bool
	Extra       []g.Node
}

func textareaClass(hasError bool) string {
	base := inputBaseClass + " min-h-[60px] px-3 py-2"
	if hasError {
		base += inputErrorClass
	}
	return base
}

// Textarea renders a multi-line <textarea>.
func Textarea(p TextareaProps) g.Node {
	nodes := []g.Node{h.Class(textareaClass(p.HasError))}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	if p.Name != "" {
		nodes = append(nodes, h.Name(p.Name))
	}
	if p.Placeholder != "" {
		nodes = append(nodes, h.Placeholder(p.Placeholder))
	}
	if p.Rows > 0 {
		nodes = append(nodes, h.Rows(strconv.Itoa(p.Rows)))
	}
	if p.Disabled {
		nodes = append(nodes, h.Disabled())
	}
	if p.Readonly {
		nodes = append(nodes, h.ReadOnly())
	}
	if p.Required {
		nodes = append(nodes, h.Required())
	}
	if p.HasError {
		nodes = append(nodes, g.Attr("aria-invalid", "true"))
	}
	nodes = append(nodes, g.Group(p.Extra))
	nodes = append(nodes, g.Text(p.Value))
	return h.Textarea(nodes...)
}
