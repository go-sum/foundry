package router

import (
	"net/http"
	"slices"
	"strings"

	"github.com/go-sum/foundry/pkg/web"
)

type nodeKind int

const (
	nodeRoute  nodeKind = iota
	nodeGroup
	nodeLayout
	nodeUse
)

// Node is a single element in the declarative route tree.
// Build with constructor functions — never construct directly.
type Node struct {
	kind     nodeKind
	method   string
	pattern  string
	name     string
	handler  web.Handler
	mw       []web.Middleware
	children []Node
}

// RouteNode creates a leaf endpoint node.
func RouteNode(method, pattern, name string, h web.Handler) Node {
	return Node{
		kind:    nodeRoute,
		method:  method,
		pattern: pattern,
		name:    name,
		handler: h,
	}
}

// GET creates a GET leaf endpoint node.
func GET(pattern, name string, h web.Handler) Node {
	return RouteNode(http.MethodGet, pattern, name, h)
}

// POST creates a POST leaf endpoint node.
func POST(pattern, name string, h web.Handler) Node {
	return RouteNode(http.MethodPost, pattern, name, h)
}

// PUT creates a PUT leaf endpoint node.
func PUT(pattern, name string, h web.Handler) Node {
	return RouteNode(http.MethodPut, pattern, name, h)
}

// PATCH creates a PATCH leaf endpoint node.
func PATCH(pattern, name string, h web.Handler) Node {
	return RouteNode(http.MethodPatch, pattern, name, h)
}

// DELETE creates a DELETE leaf endpoint node.
func DELETE(pattern, name string, h web.Handler) Node {
	return RouteNode(http.MethodDelete, pattern, name, h)
}

// HEAD creates a HEAD leaf endpoint node.
func HEAD(pattern, name string, h web.Handler) Node {
	return RouteNode(http.MethodHead, pattern, name, h)
}

// OPTIONS creates an OPTIONS leaf endpoint node.
func OPTIONS(pattern, name string, h web.Handler) Node {
	return RouteNode(http.MethodOptions, pattern, name, h)
}

// Any returns route nodes for all standard HTTP methods at pattern/name.
// If name is non-empty, each method's route name is suffixed: name + "." + lowercase(method).
func Any(pattern, name string, h web.Handler) []Node {
	nodes := make([]Node, 0, len(standardMethods))
	for _, m := range standardMethods {
		routeName := ""
		if name != "" {
			routeName = name + "." + strings.ToLower(m)
		}
		nodes = append(nodes, RouteNode(m, pattern, routeName, h))
	}
	return nodes
}

// Match returns route nodes for the given HTTP methods at pattern/name.
// If name is non-empty, each method's route name is suffixed: name + "." + lowercase(method).
func Match(methods []string, pattern, name string, h web.Handler) []Node {
	nodes := make([]Node, 0, len(methods))
	for _, m := range methods {
		routeName := ""
		if name != "" {
			routeName = name + "." + strings.ToLower(m)
		}
		nodes = append(nodes, RouteNode(m, pattern, routeName, h))
	}
	return nodes
}

// Group creates a URL prefix + children node.
func Group(prefix string, children ...Node) Node {
	return Node{
		kind:     nodeGroup,
		pattern:  prefix,
		children: children,
	}
}

// Nodes flattens multiple slices of Node into a single slice.
// Use this to compose conditional route groups into a declarative tree.
func Nodes(groups ...[]Node) []Node {
	var out []Node
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// Layout scopes middleware to children without changing URL prefix.
func Layout(children ...Node) Node {
	return Node{
		kind:     nodeLayout,
		children: children,
	}
}

// Use declares middleware for the enclosing scope.
func Use(mw ...web.Middleware) Node {
	return Node{
		kind: nodeUse,
		mw:   mw,
	}
}

// Register walks the node tree and registers all routes on rt.
func Register(rt *Router, nodes ...Node) {
	walkNodes(rt, "", nil, nodes)
}

// walkNodes accumulates prefix + middleware as it descends, calling rt.Handle at each leaf.
func walkNodes(rt *Router, prefix string, mw []web.Middleware, nodes []Node) {
	// Pass 1: collect all nodeUse middleware in this scope.
	var scopeMW []web.Middleware
	for _, n := range nodes {
		if n.kind == nodeUse {
			scopeMW = append(scopeMW, n.mw...)
		}
	}

	allMW := append(slices.Clone(mw), scopeMW...)

	// Pass 2: process Route, Group, Layout nodes.
	for _, n := range nodes {
		switch n.kind {
		case nodeRoute:
			rt.Handle(n.method, prefix+n.pattern, n.name, n.handler, allMW...)
		case nodeGroup:
			walkNodes(rt, prefix+n.pattern, allMW, n.children)
		case nodeLayout:
			walkNodes(rt, prefix, allMW, n.children)
		case nodeUse:
			// already handled in pass 1
		}
	}
}
