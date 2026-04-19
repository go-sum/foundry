// Package dialog provides a native HTML <dialog> modal component.
// Trigger and content are linked by a shared ID string.
package dialog

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func titleID(dialogID string) string {
	return dialogID + "-title"
}

func descriptionID(dialogID string) string {
	return dialogID + "-description"
}

// Root is a fragment wrapper; callers place Trigger and Content
// as siblings anywhere in the tree. They are linked by dialogID, not by DOM nesting.
func Root(children ...g.Node) g.Node {
	return h.Div(h.Class("contents"), g.Group(children))
}

// Trigger renders a wrapper that opens the <dialog> with the given ID when clicked.
func Trigger(dialogID string, children ...g.Node) g.Node {
	return h.Div(
		g.Attr("data-dialog-open", dialogID),
		g.Attr("aria-haspopup", "dialog"),
		g.Attr("aria-controls", dialogID),
		h.Class("contents cursor-pointer"),
		g.Group(children),
	)
}

// Content renders a native <dialog> element with the given ID.
func Content(id string, children ...g.Node) g.Node {
	return h.Dialog(
		h.ID(id),
		g.Attr("aria-labelledby", titleID(id)),
		g.Attr("aria-describedby", descriptionID(id)),
		h.Class("w-full max-w-lg rounded-lg border bg-background p-6 shadow-lg backdrop:bg-black/50"),
		g.Group(children),
	)
}

// Header renders the dialog header container.
func Header(children ...g.Node) g.Node {
	return h.Div(
		h.Class("flex flex-col gap-2 text-center sm:text-left mb-4"),
		g.Group(children),
	)
}

// Footer renders the dialog footer container.
func Footer(children ...g.Node) g.Node {
	return h.Div(
		h.Class("flex flex-col-reverse gap-2 sm:flex-row sm:justify-end mt-4"),
		g.Group(children),
	)
}

// Title renders the dialog heading.
func Title(dialogID string, children ...g.Node) g.Node {
	return h.H2(
		h.ID(titleID(dialogID)),
		h.Class("text-lg leading-none font-semibold"),
		g.Group(children),
	)
}

// Description renders a muted description paragraph.
func Description(dialogID string, children ...g.Node) g.Node {
	return h.P(
		h.ID(descriptionID(dialogID)),
		h.Class("text-muted-foreground text-sm"),
		g.Group(children),
	)
}

// Close renders a wrapper that closes the nearest parent <dialog> on click.
func Close(children ...g.Node) g.Node {
	return h.Div(
		g.Attr("data-dialog-close", ""),
		h.Class("contents cursor-pointer"),
		g.Group(children),
	)
}
