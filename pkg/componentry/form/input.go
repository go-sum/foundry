package form

import (
	"cmp"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// InputType selects the HTML input type attribute.
type InputType string

const (
	TypeText     InputType = "text"
	TypePassword InputType = "password"
	TypeEmail    InputType = "email"
	TypeNumber   InputType = "number"
	TypeTel      InputType = "tel"
	TypeURL      InputType = "url"
	TypeSearch   InputType = "search"
	TypeDate     InputType = "date"
	TypeFile     InputType = "file"
	TypeColor    InputType = "color"
)

// InputProps configures a bare <input> element.
type InputProps struct {
	ID          string
	Name        string
	Type        InputType
	Placeholder string
	Value       string
	Disabled    bool
	Readonly    bool
	Required    bool
	HasError    bool
	Extra       []g.Node
}

const inputBaseClass = "flex w-full rounded-md border border-input bg-transparent text-base shadow-xs transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50 md:text-sm"

const inputErrorClass = " border-destructive ring-destructive/20"

func inputClass(hasError bool) string {
	base := inputBaseClass + " h-9 min-w-0 px-3 py-1"
	if hasError {
		base += inputErrorClass
	}
	return base
}

// Input renders a bare <input>. Wrap in Field for a full fieldset layout.
func Input(p InputProps) g.Node {
	t := cmp.Or(string(p.Type), "text")
	nodes := []g.Node{
		h.Class(inputClass(p.HasError)),
		h.Type(t),
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
	if p.Placeholder != "" {
		nodes = append(nodes, h.Placeholder(p.Placeholder))
	}
	if p.Required {
		nodes = append(nodes, h.Required())
	}
	if p.Disabled {
		nodes = append(nodes, h.Disabled())
	}
	if p.Readonly {
		nodes = append(nodes, h.ReadOnly())
	}
	if p.HasError {
		nodes = append(nodes, g.Attr("aria-invalid", "true"))
	}
	nodes = append(nodes, g.Group(p.Extra))
	return h.Input(nodes...)
}
