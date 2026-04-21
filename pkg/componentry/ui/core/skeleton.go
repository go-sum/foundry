package core

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Skeleton renders an animated placeholder element used while content loads.
// class is appended to the base classes to control sizing and shape.
func Skeleton(class string, extra ...g.Node) g.Node {
	nodes := []g.Node{h.Class("animate-pulse rounded-md bg-muted " + class)}
	nodes = append(nodes, extra...)
	return h.Div(nodes...)
}
