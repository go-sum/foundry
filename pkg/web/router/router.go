package router

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/go-sum/web"
	"github.com/go-sum/web/secure"
)

// Route describes a registered route.
type Route struct {
	Method  string
	Pattern string
	Name    string
	Handler web.Handler
}

// segment is a compiled path segment — either a literal string, a parameter name, or a wildcard.
type segment struct {
	literal  string
	param    string
	wildcard bool
}

type routeEntry struct {
	method   string
	segments []segment
	name     string
	handler  web.Handler
	pattern  string
}

// trieNode is a node in the dispatch trie. Each node represents one path segment.
type trieNode struct {
	// children keyed by literal segment value (highest priority).
	children map[string]*trieNode
	// single named-parameter child (e.g., {id}).
	paramChild *trieNode
	paramName  string
	// single wildcard child (e.g., {rest...}), always terminal.
	wildChild *trieNode
	wildName  string
	// handlers at this node, keyed by HTTP method.
	handlers map[string]*routeEntry
}

func newTrieNode() *trieNode {
	return &trieNode{children: make(map[string]*trieNode)}
}

// methodSet returns the set of HTTP methods registered at this node.
func (n *trieNode) methodSet() map[string]struct{} {
	out := make(map[string]struct{}, len(n.handlers))
	for m := range n.handlers {
		if m != "" {
			out[m] = struct{}{}
		}
	}
	return out
}

// walk descends the trie matching parts left-to-right, populating params.
// Returns the terminal node if the path matches any registered pattern, nil otherwise.
func (n *trieNode) walk(parts []string, params map[string]string) *trieNode {
	if len(parts) == 0 {
		return n
	}

	part := parts[0]
	rest := parts[1:]

	// 1. Literal match (highest priority).
	if child, ok := n.children[part]; ok {
		if node := child.walk(rest, params); node != nil {
			return node
		}
	}

	// 2. Named-parameter match.
	if n.paramChild != nil {
		params[n.paramName] = part
		if node := n.paramChild.walk(rest, params); node != nil {
			return node
		}
		delete(params, n.paramName)
	}

	// 3. Wildcard match — captures all remaining segments.
	if n.wildChild != nil {
		params[n.wildName] = strings.Join(parts, "/")
		return n.wildChild
	}

	return nil
}

// insertTrie inserts rt into the trie rooted at root.
func insertTrie(root *trieNode, rt *routeEntry) {
	node := root
	for _, seg := range rt.segments {
		switch {
		case seg.wildcard:
			if node.wildChild == nil {
				node.wildChild = newTrieNode()
			}
			node.wildName = seg.param
			node = node.wildChild
		case seg.param != "":
			if node.paramChild == nil {
				node.paramChild = newTrieNode()
			}
			node.paramName = seg.param
			node = node.paramChild
		default:
			child, ok := node.children[seg.literal]
			if !ok {
				child = newTrieNode()
				node.children[seg.literal] = child
			}
			node = child
		}
	}
	if node.handlers == nil {
		node.handlers = make(map[string]*routeEntry)
	}
	node.handlers[rt.method] = rt
}

// Router is a pattern-based HTTP router with named routes and middleware support.
type Router struct {
	routes     []*routeEntry
	middleware []web.Middleware
	names      map[string]*routeEntry
	trie       *trieNode
	mu         sync.RWMutex
	once       sync.Once
	frozen     bool
}

// New creates a Router with secure.SecureDefaults() installed as the first middleware.
// SecureDefaults provides: panic recovery, strict security headers (non-clobber),
// and per-request CSP nonces. Use NewWithoutSecureDefaults for full manual control.
func New() *Router {
	r := &Router{names: make(map[string]*routeEntry)}
	r.Use(secure.SecureDefaults())
	return r
}

// NewWithoutSecureDefaults creates a Router with no pre-installed middleware.
func NewWithoutSecureDefaults() *Router {
	return &Router{names: make(map[string]*routeEntry)}
}

// Freeze pre-compiles global middleware into all registered routes and builds
// the dispatch trie. Called automatically by Serve on the first request.
func (r *Router) Freeze() { r.freeze() }

func (r *Router) freeze() {
	r.once.Do(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, rt := range r.routes {
			if len(r.middleware) > 0 {
				rt.handler = web.Chain(rt.handler, r.middleware...)
			}
		}
		sort.SliceStable(r.routes, func(i, j int) bool {
			return routeSpecificity(r.routes[i]) > routeSpecificity(r.routes[j])
		})
		root := newTrieNode()
		for _, rt := range r.routes {
			insertTrie(root, rt)
		}
		r.trie = root
		r.frozen = true
	})
}

// IsFrozen reports whether the router has been frozen (i.e., the first request
// has been served or Freeze has been called). After freezing, new routes and
// middleware cannot be added.
func (r *Router) IsFrozen() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.frozen
}

// Use adds global middleware that applies to all routes.
func (r *Router) Use(mw ...web.Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		panic("router: cannot add middleware to a frozen router")
	}
	r.middleware = append(r.middleware, mw...)
}

func patternKey(method, pattern string) string { return method + "\x00" + pattern }

// Handle registers a route with the given method, pattern, name, and handler.
// The pattern uses {param} for named path parameters and {param...} for wildcards.
func (r *Router) Handle(method, pattern, name string, h web.Handler, mw ...web.Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		panic("router: cannot register routes on a frozen router")
	}

	key := patternKey(method, pattern)
	for _, rt := range r.routes {
		if patternKey(rt.method, rt.pattern) == key {
			panic(fmt.Sprintf("router: duplicate route %s %s", method, pattern))
		}
	}
	if name != "" {
		if _, exists := r.names[name]; exists {
			panic(fmt.Sprintf("router: duplicate route name %q", name))
		}
	}

	if len(mw) > 0 {
		h = web.Chain(h, mw...)
	}

	rt := &routeEntry{
		method:   method,
		segments: compile(pattern),
		name:     name,
		handler:  h,
		pattern:  pattern,
	}
	r.routes = append(r.routes, rt)
	if name != "" {
		r.names[name] = rt
	}
}

// GET registers a GET route.
func (r *Router) GET(pattern, name string, h web.Handler, mw ...web.Middleware) {
	r.Handle(http.MethodGet, pattern, name, h, mw...)
}

// POST registers a POST route.
func (r *Router) POST(pattern, name string, h web.Handler, mw ...web.Middleware) {
	r.Handle(http.MethodPost, pattern, name, h, mw...)
}

// PUT registers a PUT route.
func (r *Router) PUT(pattern, name string, h web.Handler, mw ...web.Middleware) {
	r.Handle(http.MethodPut, pattern, name, h, mw...)
}

// PATCH registers a PATCH route.
func (r *Router) PATCH(pattern, name string, h web.Handler, mw ...web.Middleware) {
	r.Handle(http.MethodPatch, pattern, name, h, mw...)
}

// DELETE registers a DELETE route.
func (r *Router) DELETE(pattern, name string, h web.Handler, mw ...web.Middleware) {
	r.Handle(http.MethodDelete, pattern, name, h, mw...)
}

// HEAD registers a HEAD route.
func (r *Router) HEAD(pattern, name string, h web.Handler, mw ...web.Middleware) {
	r.Handle(http.MethodHead, pattern, name, h, mw...)
}

// OPTIONS registers an OPTIONS route.
func (r *Router) OPTIONS(pattern, name string, h web.Handler, mw ...web.Middleware) {
	r.Handle(http.MethodOptions, pattern, name, h, mw...)
}

// standardMethods are the HTTP methods registered by Any.
var standardMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
}

// Any registers h for all standard HTTP methods (GET, HEAD, POST, PUT, PATCH, DELETE).
// If name is non-empty, each method's route name is suffixed: name + "." + lowercase(method).
func (r *Router) Any(pattern, name string, h web.Handler, mw ...web.Middleware) {
	for _, m := range standardMethods {
		routeName := ""
		if name != "" {
			routeName = name + "." + strings.ToLower(m)
		}
		r.Handle(m, pattern, routeName, h, mw...)
	}
}

// Match registers h for the specified HTTP methods.
// If name is non-empty, each method's route name is suffixed: name + "." + lowercase(method).
func (r *Router) Match(methods []string, pattern, name string, h web.Handler, mw ...web.Middleware) {
	for _, m := range methods {
		routeName := ""
		if name != "" {
			routeName = name + "." + strings.ToLower(m)
		}
		r.Handle(m, pattern, routeName, h, mw...)
	}
}

// RouteGroup is a set of routes sharing a common path prefix and middleware.
// Create one via Router.Group. Sub-groups are created via RouteGroup.Group.
type RouteGroup struct {
	router *Router
	prefix string
	mw     []web.Middleware
}

// Group creates a RouteGroup with the given path prefix and optional scoped middleware.
// Routes registered on the group have the prefix prepended and the group middleware
// applied before any route-level middleware. Panics if the router is frozen.
func (r *Router) Group(prefix string, mw ...web.Middleware) *RouteGroup {
	return &RouteGroup{router: r, prefix: prefix, mw: slices.Clone(mw)}
}

// Group creates a sub-group that inherits this group's prefix and middleware.
func (g *RouteGroup) Group(prefix string, mw ...web.Middleware) *RouteGroup {
	combined := make([]web.Middleware, 0, len(g.mw)+len(mw))
	combined = append(combined, g.mw...)
	combined = append(combined, mw...)
	return &RouteGroup{router: g.router, prefix: g.prefix + prefix, mw: combined}
}

// Use appends middleware to this group's scoped middleware chain.
func (g *RouteGroup) Use(mw ...web.Middleware) {
	g.mw = append(g.mw, mw...)
}

// Handle registers a route under this group's prefix with the group's middleware
// prepended before any route-level middleware.
func (g *RouteGroup) Handle(method, pattern, name string, h web.Handler, mw ...web.Middleware) {
	allMW := make([]web.Middleware, 0, len(g.mw)+len(mw))
	allMW = append(allMW, g.mw...)
	allMW = append(allMW, mw...)
	g.router.Handle(method, g.prefix+pattern, name, h, allMW...)
}

// GET registers a GET route on the group.
func (g *RouteGroup) GET(pattern, name string, h web.Handler, mw ...web.Middleware) {
	g.Handle(http.MethodGet, pattern, name, h, mw...)
}

// POST registers a POST route on the group.
func (g *RouteGroup) POST(pattern, name string, h web.Handler, mw ...web.Middleware) {
	g.Handle(http.MethodPost, pattern, name, h, mw...)
}

// PUT registers a PUT route on the group.
func (g *RouteGroup) PUT(pattern, name string, h web.Handler, mw ...web.Middleware) {
	g.Handle(http.MethodPut, pattern, name, h, mw...)
}

// PATCH registers a PATCH route on the group.
func (g *RouteGroup) PATCH(pattern, name string, h web.Handler, mw ...web.Middleware) {
	g.Handle(http.MethodPatch, pattern, name, h, mw...)
}

// DELETE registers a DELETE route on the group.
func (g *RouteGroup) DELETE(pattern, name string, h web.Handler, mw ...web.Middleware) {
	g.Handle(http.MethodDelete, pattern, name, h, mw...)
}

// HEAD registers a HEAD route on the group.
func (g *RouteGroup) HEAD(pattern, name string, h web.Handler, mw ...web.Middleware) {
	g.Handle(http.MethodHead, pattern, name, h, mw...)
}

// OPTIONS registers an OPTIONS route on the group.
func (g *RouteGroup) OPTIONS(pattern, name string, h web.Handler, mw ...web.Middleware) {
	g.Handle(http.MethodOptions, pattern, name, h, mw...)
}

// Any registers h for all standard HTTP methods on this group.
// If name is non-empty, each method's route name is suffixed: name + "." + lowercase(method).
func (g *RouteGroup) Any(pattern, name string, h web.Handler, mw ...web.Middleware) {
	for _, m := range standardMethods {
		routeName := ""
		if name != "" {
			routeName = name + "." + strings.ToLower(m)
		}
		g.Handle(m, pattern, routeName, h, mw...)
	}
}

// Match registers h for the specified HTTP methods on this group.
// If name is non-empty, each method's route name is suffixed: name + "." + lowercase(method).
func (g *RouteGroup) Match(methods []string, pattern, name string, h web.Handler, mw ...web.Middleware) {
	for _, m := range methods {
		routeName := ""
		if name != "" {
			routeName = name + "." + strings.ToLower(m)
		}
		g.Handle(m, pattern, routeName, h, mw...)
	}
}

// Serve dispatches the request to the matching route using an O(depth) trie walk.
// Returns 404 if no route matches the path, 405 if the path matches but the method does not.
func (r *Router) Serve(c *web.Context) (web.Response, error) {
	r.freeze()

	path := ""
	method := ""
	if c != nil {
		method = c.Method()
		if c.URL() != nil {
			path = c.URL().Path
		}
	}

	parts := splitPath(path)
	params := map[string]string{}
	node := r.trie.walk(parts, params)

	if node == nil || len(node.handlers) == 0 {
		return web.NotFoundHandler()(c)
	}

	// Exact method match.
	if rt, ok := node.handlers[method]; ok {
		if len(params) > 0 && c != nil {
			c.SetParams(params)
		}
		return rt.handler(c)
	}

	// HEAD → GET fallback.
	if method == http.MethodHead {
		if rt, ok := node.handlers[http.MethodGet]; ok {
			if len(params) > 0 && c != nil {
				c.SetParams(params)
			}
			return rt.handler(c)
		}
	}

	// OPTIONS auto-response.
	if method == http.MethodOptions {
		resp := web.Respond(http.StatusNoContent)
		resp.Headers.Set("Allow", buildAllowHeader(node.methodSet()))
		return resp, nil
	}

	// 405 — path matched but method not registered.
	return web.Response{}, web.ErrMethodNotAllowed("").WithHeader("Allow", buildAllowHeader(node.methodSet()))
}

// Reverse returns the URL path for a named route, substituting named
// parameters into {param} and {param...} placeholders.
func (r *Router) Reverse(name string, params map[string]string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rt, ok := r.names[name]
	if !ok {
		return "", fmt.Errorf("router: unknown route name %q", name)
	}

	var b strings.Builder
	for _, seg := range rt.segments {
		b.WriteByte('/')
		switch {
		case seg.wildcard:
			value, ok := params[seg.param]
			if !ok {
				return "", fmt.Errorf("router: route %q requires param %q", name, seg.param)
			}
			b.WriteString(escapeWildcard(value))
		case seg.param != "":
			value, ok := params[seg.param]
			if !ok {
				return "", fmt.Errorf("router: route %q requires param %q", name, seg.param)
			}
			b.WriteString(url.PathEscape(value))
		default:
			b.WriteString(seg.literal)
		}
	}
	if b.Len() == 0 {
		return "/", nil
	}
	return b.String(), nil
}

// MustReverse is Reverse but panics on error.
func (r *Router) MustReverse(name string, params map[string]string) string {
	path, err := r.Reverse(name, params)
	if err != nil {
		panic(err.Error())
	}
	return path
}

// Pattern returns the raw URL pattern for a named route.
func (r *Router) Pattern(name string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rt, ok := r.names[name]
	if !ok {
		return "", fmt.Errorf("router: unknown route name %q", name)
	}
	return rt.pattern, nil
}

// Routes returns all registered routes.
func (r *Router) Routes() []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Route, len(r.routes))
	for i, rt := range r.routes {
		out[i] = Route{
			Method:  rt.method,
			Pattern: rt.pattern,
			Name:    rt.name,
			Handler: rt.handler,
		}
	}
	return out
}

func routeSpecificity(rt *routeEntry) int {
	score := 0
	for _, seg := range rt.segments {
		switch {
		case seg.wildcard:
			score += 0
		case seg.literal != "":
			score += 2
		default:
			score += 1
		}
	}
	return score
}

func compile(pattern string) []segment {
	parts := splitPath(pattern)
	segments := make([]segment, len(parts))
	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			inner := part[1 : len(part)-1]
			if strings.HasSuffix(inner, "...") {
				if i != len(parts)-1 {
					panic("router: wildcard segment {" + inner + "} must be the last segment in the pattern")
				}
				segments[i] = segment{param: inner[:len(inner)-3], wildcard: true}
				continue
			}
			segments[i] = segment{param: inner}
			continue
		}
		segments[i] = segment{literal: part}
	}
	return segments
}

func splitPath(path string) []string {
	var parts []string
	for _, part := range strings.Split(path, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func escapeWildcard(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func buildAllowHeader(methods map[string]struct{}) string {
	if len(methods) == 0 {
		return http.MethodOptions
	}
	ordered := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}

	allow := make(map[string]struct{}, len(methods)+2)
	for method := range methods {
		allow[method] = struct{}{}
	}
	if _, ok := allow[http.MethodGet]; ok {
		allow[http.MethodHead] = struct{}{}
	}
	allow[http.MethodOptions] = struct{}{}

	out := make([]string, 0, len(allow))
	for _, method := range ordered {
		if _, ok := allow[method]; ok {
			out = append(out, method)
		}
	}
	for method := range allow {
		switch method {
		case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
			continue
		default:
			out = append(out, method)
		}
	}
	return strings.Join(out, ", ")
}
