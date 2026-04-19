// Package form provides shadcn/ui-styled form field components and form state management.
package form

import (
	"strings"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/go-sum/componentry/ui/core"
)

const (
	fieldDescriptionSuffix = "-description"
	fieldHintSuffix        = "-hint"
	fieldErrorSuffix       = "-error"
)

// FieldProps configures a label, control, and assistive text as one field block.
type FieldProps struct {
	ID          string
	Label       string
	Description string
	Hint        string
	Errors      []string
	Required    bool
	Control     g.Node
	Extra       []g.Node
}

func fieldMessageID(controlID, suffix string) string {
	if controlID == "" {
		return ""
	}
	return controlID + suffix
}

func descriptionID(controlID string) string {
	return fieldMessageID(controlID, fieldDescriptionSuffix)
}

func hintID(controlID string) string {
	return fieldMessageID(controlID, fieldHintSuffix)
}

func errorID(controlID string) string {
	return fieldMessageID(controlID, fieldErrorSuffix)
}

// FieldControlAttrs returns ARIA attributes wiring a control to the description,
// hint, and error blocks rendered by Field. Returns nil when controlID is empty
// or no assistive text is present.
func FieldControlAttrs(controlID, description, hint string, errors []string) []g.Node {
	if controlID == "" {
		return nil
	}

	ids := make([]string, 0, 3)
	if description != "" {
		ids = append(ids, descriptionID(controlID))
	}
	if hint != "" {
		ids = append(ids, hintID(controlID))
	}
	if len(errors) > 0 {
		ids = append(ids, errorID(controlID))
	}
	if len(ids) == 0 {
		return nil
	}

	nodes := []g.Node{g.Attr("aria-describedby", strings.Join(ids, " "))}
	if len(errors) > 0 {
		nodes = append(nodes,
			g.Attr("aria-errormessage", errorID(controlID)),
			g.Attr("aria-invalid", "true"),
		)
	}
	return nodes
}

// Field renders a field wrapper with consistent label, description, hint, and error output.
func Field(p FieldProps) g.Node {
	nodes := []g.Node{h.Class("grid gap-2")}
	nodes = append(nodes, g.Group(p.Extra))
	if p.Label != "" {
		nodes = append(nodes, core.Label(core.LabelProps{
			For:      p.ID,
			Required: p.Required,
			Error:    firstError(p.Errors),
		}, g.Text(p.Label)))
	}
	if p.Control != nil {
		nodes = append(nodes, p.Control)
	}
	if p.Description != "" {
		nodes = append(nodes, Description(p.ID, p.Description))
	}
	if p.Hint != "" {
		nodes = append(nodes, Hint(p.ID, p.Hint))
	}
	if len(p.Errors) > 0 {
		nodes = append(nodes, ErrorMessage(p.ID, p.Errors...))
	}
	return h.Div(nodes...)
}

// Description renders descriptive text associated with a form control.
func Description(controlID, text string) g.Node {
	if text == "" {
		return g.Group(nil)
	}
	nodes := []g.Node{h.Class("text-sm text-muted-foreground")}
	if id := descriptionID(controlID); id != "" {
		nodes = append(nodes, h.ID(id))
	}
	nodes = append(nodes, g.Text(text))
	return h.P(nodes...)
}

// Hint renders secondary guidance associated with a form control.
func Hint(controlID, text string) g.Node {
	if text == "" {
		return g.Group(nil)
	}
	nodes := []g.Node{h.Class("text-xs text-muted-foreground")}
	if id := hintID(controlID); id != "" {
		nodes = append(nodes, h.ID(id))
	}
	nodes = append(nodes, g.Text(text))
	return h.P(nodes...)
}

// ErrorMessage renders one or more validation messages associated with a form control.
func ErrorMessage(controlID string, errors ...string) g.Node {
	if len(errors) == 0 {
		return g.Group(nil)
	}
	messageNodes := make([]g.Node, 0, len(errors)+2)
	messageNodes = append(messageNodes, h.Class("grid gap-1"))
	if id := errorID(controlID); id != "" {
		messageNodes = append(messageNodes, h.ID(id))
	}
	for _, errText := range errors {
		messageNodes = append(messageNodes, h.P(h.Class("text-xs text-destructive"), g.Text(errText)))
	}
	return h.Div(messageNodes...)
}

func firstError(errors []string) string {
	if len(errors) == 0 {
		return ""
	}
	return errors[0]
}
