package core

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Skeleton renders an animated placeholder element used while content loads.
func Skeleton(extra ...g.Node) g.Node {
	nodes := []g.Node{h.Class("animate-pulse rounded-md bg-muted")}
	nodes = append(nodes, extra...)
	return h.Div(nodes...)
}
