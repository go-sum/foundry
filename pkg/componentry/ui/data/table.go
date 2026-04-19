package data

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type tableNS struct{}

// Table groups table sub-components under a namespace: Table.Root, Table.Header, Table.Row, etc.
var Table tableNS

// Root renders a responsive table wrapper.
func (tableNS) Root(children ...g.Node) g.Node {
	return h.Div(
		h.Class("relative w-full overflow-auto"),
		h.Table(
			h.Class("w-full caption-bottom text-sm"),
			g.Group(children),
		),
	)
}

// Header renders a <thead> section.
func (tableNS) Header(children ...g.Node) g.Node {
	return h.THead(
		h.Class("[&_tr]:border-b"),
		g.Group(children),
	)
}

// BodyProps configures a <tbody> section.
type BodyProps struct {
	ID    string
	Extra []g.Node
}

// Body renders a <tbody> section.
func (tableNS) Body(p BodyProps, children ...g.Node) g.Node {
	nodes := []g.Node{h.Class("[&_tr:last-child]:border-0")}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	nodes = append(nodes, g.Group(p.Extra))
	nodes = append(nodes, g.Group(children))
	return h.TBody(nodes...)
}

// Footer renders a <tfoot> section.
func (tableNS) Footer(children ...g.Node) g.Node {
	return h.TFoot(
		h.Class("border-t bg-muted/50 font-medium [&>tr]:last:border-b-0"),
		g.Group(children),
	)
}

// RowProps configures a table row.
type RowProps struct {
	Selected bool
	Extra    []g.Node
}

// Row renders a <tr>. Pass RowProps{Selected: true} for a selected highlight.
func (tableNS) Row(p RowProps, children ...g.Node) g.Node {
	cls := "border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted"
	if p.Selected {
		cls += " bg-muted"
	}
	nodes := []g.Node{h.Class(cls)}
	nodes = append(nodes, g.Group(p.Extra))
	nodes = append(nodes, g.Group(children))
	return h.Tr(nodes...)
}

// Head renders a <th> header cell.
func (tableNS) Head(children ...g.Node) g.Node {
	return h.Th(
		h.Class("h-10 px-2 text-left align-middle font-medium text-muted-foreground [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]"),
		g.Group(children),
	)
}

// Cell renders a <td> data cell.
func (tableNS) Cell(children ...g.Node) g.Node {
	return h.Td(
		h.Class("p-2 align-middle [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]"),
		g.Group(children),
	)
}

// Caption renders a <caption>.
func (tableNS) Caption(children ...g.Node) g.Node {
	return h.Caption(
		h.Class("mt-4 text-sm text-muted-foreground"),
		g.Group(children),
	)
}
