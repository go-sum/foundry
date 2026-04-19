package router

import (
	"net/http"
	"testing"

	"github.com/go-sum/web"
)

func TestTreeBuilderConstructors(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}

	route := RouteNode(http.MethodGet, "/users", "users.index", handler)
	if route.kind != nodeRoute {
		t.Fatalf("RouteNode kind = %v, want %v", route.kind, nodeRoute)
	}
	if route.method != http.MethodGet || route.pattern != "/users" || route.name != "users.index" || route.handler == nil {
		t.Fatalf("RouteNode = %#v, want GET /users users.index with handler", route)
	}

	if got := GET("/get", "get.route", handler); got.method != http.MethodGet {
		t.Fatalf("GET method = %q, want %q", got.method, http.MethodGet)
	}
	if got := POST("/post", "post.route", handler); got.method != http.MethodPost {
		t.Fatalf("POST method = %q, want %q", got.method, http.MethodPost)
	}
	if got := PUT("/put", "put.route", handler); got.method != http.MethodPut {
		t.Fatalf("PUT method = %q, want %q", got.method, http.MethodPut)
	}
	if got := PATCH("/patch", "patch.route", handler); got.method != http.MethodPatch {
		t.Fatalf("PATCH method = %q, want %q", got.method, http.MethodPatch)
	}
	if got := DELETE("/delete", "delete.route", handler); got.method != http.MethodDelete {
		t.Fatalf("DELETE method = %q, want %q", got.method, http.MethodDelete)
	}
	if got := HEAD("/head", "head.route", handler); got.method != http.MethodHead {
		t.Fatalf("HEAD method = %q, want %q", got.method, http.MethodHead)
	}
	if got := OPTIONS("/options", "options.route", handler); got.method != http.MethodOptions {
		t.Fatalf("OPTIONS method = %q, want %q", got.method, http.MethodOptions)
	}
}

func TestTreeBuilderScopes(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}
	mw := func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			return next(c)
		}
	}

	any := Any("/resource", "resource", handler)
	if len(any) != len(standardMethods) {
		t.Fatalf("len(Any()) = %d, want %d", len(any), len(standardMethods))
	}
	if any[0].name != "resource.get" {
		t.Fatalf("Any()[0].name = %q, want %q", any[0].name, "resource.get")
	}

	match := Match([]string{http.MethodGet, http.MethodDelete}, "/resource", "resource", handler)
	if len(match) != 2 {
		t.Fatalf("len(Match()) = %d, want 2", len(match))
	}
	if match[1].method != http.MethodDelete || match[1].name != "resource.delete" {
		t.Fatalf("Match()[1] = %#v, want DELETE resource.delete", match[1])
	}

	group := GroupNode("/api", GET("/users", "users.index", handler))
	if group.kind != nodeGroup || group.pattern != "/api" || len(group.children) != 1 {
		t.Fatalf("GroupNode() = %#v, want group /api with one child", group)
	}

	layout := Layout(GET("/", "home.show", handler))
	if layout.kind != nodeLayout || len(layout.children) != 1 {
		t.Fatalf("Layout() = %#v, want layout with one child", layout)
	}

	use := Use(mw)
	if use.kind != nodeUse || len(use.mw) != 1 {
		t.Fatalf("Use() = %#v, want one middleware", use)
	}
}
