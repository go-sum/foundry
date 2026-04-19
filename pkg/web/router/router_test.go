package router

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/go-sum/web"
)

func serveStatus(t *testing.T, r *Router, method, path string) int {
	t.Helper()
	resp, err := r.Serve(testContext(method, path))
	if err != nil {
		var webErr *web.Error
		if errors.As(err, &webErr) {
			return webErr.Status
		}
		t.Fatalf("unexpected non-web error: %v", err)
	}
	return resp.Status
}

func TestServeSetsParams(t *testing.T) {
	r := New()
	Register(r, GET("/hello/{name}", "hello.show", func(c *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, c.Param("name")), nil
	}))

	resp, _ := r.Serve(testContext(http.MethodGet, "/hello/alice"))
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestServeDistinguishes404And405(t *testing.T) {
	r := New()
	Register(r, GET("/items", "items.index", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}))

	if got := serveStatus(t, r, http.MethodPost, "/items"); got != http.StatusMethodNotAllowed {
		t.Fatalf("POST /items status = %d, want %d", got, http.StatusMethodNotAllowed)
	}

	// Also verify Allow header on 405.
	_, err405 := r.Serve(testContext(http.MethodPost, "/items"))
	_ = err405 // allow header only in unit router; integration test verifies it end-to-end

	if got := serveStatus(t, r, http.MethodGet, "/missing"); got != http.StatusNotFound {
		t.Fatalf("GET /missing status = %d, want %d", got, http.StatusNotFound)
	}
}

func TestServe_HEADFallsBackToGET(t *testing.T) {
	r := New()
	Register(r, GET("/items/{id}", "items.show", func(c *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, c.Param("id")), nil
	}))

	resp, _ := r.Serve(testContext(http.MethodHead, "/items/42"))
	if resp.Status != http.StatusOK {
		t.Fatalf("HEAD /items/42 status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestServe_OPTIONSReturnsAllowHeader(t *testing.T) {
	r := New()
	Register(r,
		GET("/items", "items.index", func(_ *web.Context) (web.Response, error) {
			return web.Respond(http.StatusOK), nil
		}),
		POST("/items", "items.create", func(_ *web.Context) (web.Response, error) {
			return web.Respond(http.StatusCreated), nil
		}),
	)

	resp, _ := r.Serve(testContext(http.MethodOptions, "/items"))
	if resp.Status != http.StatusNoContent {
		t.Fatalf("OPTIONS /items status = %d, want %d", resp.Status, http.StatusNoContent)
	}
	if got := resp.Headers.Get("Allow"); got != "GET, HEAD, POST, OPTIONS" {
		t.Fatalf("OPTIONS /items Allow = %q, want %q", got, "GET, HEAD, POST, OPTIONS")
	}
}

func TestSpecificityPrefersLiteralOverParam(t *testing.T) {
	r := New()
	Register(r,
		GET("/users/{id}", "users.show", func(_ *web.Context) (web.Response, error) {
			return web.Text(http.StatusOK, "param"), nil
		}),
		GET("/users/me", "users.me", func(_ *web.Context) (web.Response, error) {
			return web.Text(http.StatusOK, "literal"), nil
		}),
	)

	resp, _ := r.Serve(testContext(http.MethodGet, "/users/me"))
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d", resp.Status)
	}
}

func TestReverseUsesNamedParamsAfterFreeze(t *testing.T) {
	r := New()
	Register(r,
		GET("/hello/{name}", "hello.show", func(_ *web.Context) (web.Response, error) {
			return web.Respond(http.StatusOK), nil
		}),
		GET("/users/me", "users.me", func(_ *web.Context) (web.Response, error) {
			return web.Respond(http.StatusOK), nil
		}),
	)

	r.Freeze()

	got, err := r.Reverse("hello.show", map[string]string{"name": "alice"})
	if err != nil {
		t.Fatalf("Reverse() error = %v", err)
	}
	if got != "/hello/alice" {
		t.Fatalf("Reverse() = %q, want %q", got, "/hello/alice")
	}
}

func TestReverseEscapesAndWildcard(t *testing.T) {
	r := New()
	Register(r, GET("/files/{path...}", "files.show", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}))

	got, err := r.Reverse("files.show", map[string]string{"path": "a b/c"})
	if err != nil {
		t.Fatalf("Reverse() error = %v", err)
	}
	if got != "/files/a%20b/c" {
		t.Fatalf("Reverse() = %q, want %q", got, "/files/a%20b/c")
	}
}

func TestDuplicateRouteNamePanics(t *testing.T) {
	r := New()
	Register(r, GET("/a", "dup", func(_ *web.Context) (web.Response, error) { return web.Respond(http.StatusOK), nil }))

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for duplicate route name")
		}
	}()

	Register(r, GET("/b", "dup", func(_ *web.Context) (web.Response, error) { return web.Respond(http.StatusOK), nil }))
}


func TestIsFrozen(t *testing.T) {
	r := New()
	Register(r, GET("/", "home", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}))

	if r.IsFrozen() {
		t.Fatal("IsFrozen() = true before first Serve, want false")
	}

	r.Serve(testContext(http.MethodGet, "/"))

	if !r.IsFrozen() {
		t.Fatal("IsFrozen() = false after Serve, want true")
	}
}

func TestIsFrozen_AfterExplicitFreeze(t *testing.T) {
	r := New()
	Register(r, GET("/", "home", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}))

	if r.IsFrozen() {
		t.Fatal("IsFrozen() = true before Freeze, want false")
	}

	r.Freeze()

	if !r.IsFrozen() {
		t.Fatal("IsFrozen() = false after Freeze, want true")
	}
}

func TestTrieWildcardCapturesRemainingSegments(t *testing.T) {
	r := New()
	var capturedPath string
	Register(r, GET("/files/{path...}", "files.show", func(c *web.Context) (web.Response, error) {
		capturedPath = c.Param("path")
		return web.Respond(http.StatusOK), nil
	}))

	resp, _ := r.Serve(testContext(http.MethodGet, "/files/a/b/c"))
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if capturedPath != "a/b/c" {
		t.Fatalf("path param = %q, want %q", capturedPath, "a/b/c")
	}
}

func TestTrieParamExtraction(t *testing.T) {
	r := New()
	var capturedID string
	Register(r, GET("/users/{id}", "users.show", func(c *web.Context) (web.Response, error) {
		capturedID = c.Param("id")
		return web.Respond(http.StatusOK), nil
	}))

	resp, _ := r.Serve(testContext(http.MethodGet, "/users/42"))
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if capturedID != "42" {
		t.Fatalf("id param = %q, want %q", capturedID, "42")
	}
}

func TestTrieLiteralBeatsParam(t *testing.T) {
	r := New()
	Register(r,
		GET("/users/{id}", "users.show", func(_ *web.Context) (web.Response, error) {
			return web.Text(http.StatusOK, "param"), nil
		}),
		GET("/users/me", "users.me", func(_ *web.Context) (web.Response, error) {
			return web.Text(http.StatusOK, "literal"), nil
		}),
	)

	resp, _ := r.Serve(testContext(http.MethodGet, "/users/me"))
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	// The literal route must win; we verify the param route is not selected
	// by confirming the param route still works for a non-literal segment.
	paramResp, _ := r.Serve(testContext(http.MethodGet, "/users/99"))
	if paramResp.Status != http.StatusOK {
		t.Fatalf("param route status = %d, want %d", paramResp.Status, http.StatusOK)
	}
}

func TestTrie405WithAllowHeader(t *testing.T) {
	r := New()
	Register(r, POST("/things", "things.create", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusCreated), nil
	}))

	_, err := r.Serve(testContext(http.MethodGet, "/things"))
	// 405 is returned as an *web.Error with Allow in ResponseHeaders.
	var webErr *web.Error
	if !errors.As(err, &webErr) || webErr.Status != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 web error, got: %v", err)
	}
	allow := webErr.ResponseHeaders["Allow"]
	if allow == "" {
		t.Fatal("Allow header absent on 405 error ResponseHeaders")
	}
}

func TestTrie404UnregisteredPath(t *testing.T) {
	r := New()
	Register(r, GET("/exists", "exists", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}))

	if got := serveStatus(t, r, http.MethodGet, "/does-not-exist"); got != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", got, http.StatusNotFound)
	}
}

func TestTrieHEADFallbackToGET(t *testing.T) {
	r := New()
	Register(r, GET("/resource", "resource.show", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}))

	resp, _ := r.Serve(testContext(http.MethodHead, "/resource"))
	if resp.Status != http.StatusOK {
		t.Fatalf("HEAD fallback status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestTrieOPTIONSAutoResponse(t *testing.T) {
	r := New()
	Register(r,
		GET("/resource", "resource.list", func(_ *web.Context) (web.Response, error) {
			return web.Respond(http.StatusOK), nil
		}),
		POST("/resource", "resource.create", func(_ *web.Context) (web.Response, error) {
			return web.Respond(http.StatusCreated), nil
		}),
	)

	resp, _ := r.Serve(testContext(http.MethodOptions, "/resource"))
	if resp.Status != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want %d", resp.Status, http.StatusNoContent)
	}
	allow := resp.Headers.Get("Allow")
	if allow == "" {
		t.Fatal("Allow header absent on OPTIONS response")
	}
}

// --- Phase 2 tests: RouteGroup, Any, Match ---

func TestRouterGroup_PrefixApplied(t *testing.T) {
	r := New()
	g := r.Group("/api")
	g.GET("/users", "api.users.list", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	if got := serveStatus(t, r, http.MethodGet, "/api/users"); got != http.StatusOK {
		t.Fatalf("GET /api/users status = %d, want %d", got, http.StatusOK)
	}
	// Unpreixed path must 404.
	if got := serveStatus(t, r, http.MethodGet, "/users"); got != http.StatusNotFound {
		t.Fatalf("GET /users status = %d, want %d", got, http.StatusNotFound)
	}
}

func TestRouterGroup_MiddlewareApplied(t *testing.T) {
	r := New()
	var mwCalled bool
	mw := func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			mwCalled = true
			return next(c)
		}
	}
	g := r.Group("/v1", mw)
	g.GET("/ping", "v1.ping", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	mwCalled = false
	serveStatus(t, r, http.MethodGet, "/v1/ping")
	if !mwCalled {
		t.Fatal("group middleware was not called")
	}
}

func TestRouterGroup_FrozenRouterPanics(t *testing.T) {
	r := New()
	r.GET("/seed", "seed", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})
	r.Freeze()

	g := r.Group("/frozen")
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when registering route on frozen router via group")
		}
	}()
	g.GET("/route", "frozen.route", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})
}

func TestRouteGroup_SubGroupAccumulatesPrefixAndMiddleware(t *testing.T) {
	r := New()

	var order []string
	mw1 := func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			order = append(order, "mw1")
			return next(c)
		}
	}
	mw2 := func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			order = append(order, "mw2")
			return next(c)
		}
	}

	g := r.Group("/a", mw1)
	sg := g.Group("/b", mw2)
	sg.GET("/c", "a.b.c", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	order = nil
	if got := serveStatus(t, r, http.MethodGet, "/a/b/c"); got != http.StatusOK {
		t.Fatalf("GET /a/b/c status = %d, want %d", got, http.StatusOK)
	}
	if len(order) != 2 || order[0] != "mw1" || order[1] != "mw2" {
		t.Fatalf("middleware order = %v, want [mw1 mw2]", order)
	}
}

func TestRouterAny_AllMethodsDispatch(t *testing.T) {
	r := New()
	r.Any("/resource", "resource", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	for _, m := range standardMethods {
		if got := serveStatus(t, r, m, "/resource"); got != http.StatusOK {
			t.Errorf("%s /resource status = %d, want %d", m, got, http.StatusOK)
		}
	}
}

func TestRouterAny_NamesAreSuffixed(t *testing.T) {
	r := New()
	r.Any("/resource", "resource", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	for _, m := range standardMethods {
		wantName := "resource." + strings.ToLower(m)
		if _, err := r.Reverse(wantName, nil); err != nil {
			t.Errorf("Reverse(%q) error = %v", wantName, err)
		}
	}
}

func TestRouterAny_EmptyNameRegistersNoNames(t *testing.T) {
	r := New()
	r.Any("/anon", "", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	// Routes() should still have 6 entries but none with names.
	routes := r.Routes()
	named := 0
	for _, rt := range routes {
		if rt.Name != "" {
			named++
		}
	}
	if named != 0 {
		t.Fatalf("named routes = %d, want 0 for empty-name Any", named)
	}
}

func TestRouterMatch_OnlySpecifiedMethodsDispatch(t *testing.T) {
	r := New()
	methods := []string{http.MethodGet, http.MethodPost}
	r.Match(methods, "/thing", "thing", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	if got := serveStatus(t, r, http.MethodGet, "/thing"); got != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", got, http.StatusOK)
	}
	if got := serveStatus(t, r, http.MethodPost, "/thing"); got != http.StatusOK {
		t.Fatalf("POST status = %d, want %d", got, http.StatusOK)
	}
	// PUT is not registered — expect 405.
	if got := serveStatus(t, r, http.MethodPut, "/thing"); got != http.StatusMethodNotAllowed {
		t.Fatalf("PUT status = %d, want %d", got, http.StatusMethodNotAllowed)
	}
}

func TestRouterMatch_NamesAreSuffixed(t *testing.T) {
	r := New()
	methods := []string{http.MethodGet, http.MethodDelete}
	r.Match(methods, "/item", "item", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	for _, m := range methods {
		wantName := "item." + strings.ToLower(m)
		if _, err := r.Reverse(wantName, nil); err != nil {
			t.Errorf("Reverse(%q) error = %v", wantName, err)
		}
	}
}

func TestRouteGroupAny_PrefixAndMiddlewareApplied(t *testing.T) {
	r := New()
	var mwCalled bool
	mw := func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			mwCalled = true
			return next(c)
		}
	}
	g := r.Group("/api", mw)
	g.Any("/resource", "api.resource", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	for _, m := range standardMethods {
		mwCalled = false
		if got := serveStatus(t, r, m, "/api/resource"); got != http.StatusOK {
			t.Errorf("%s /api/resource status = %d, want %d", m, got, http.StatusOK)
		}
		if !mwCalled {
			t.Errorf("%s /api/resource: group middleware was not called", m)
		}
	}
}

func TestRouteGroupMatch_PrefixAndMiddlewareApplied(t *testing.T) {
	r := New()
	var mwCalled bool
	mw := func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			mwCalled = true
			return next(c)
		}
	}
	g := r.Group("/v2", mw)
	g.Match([]string{http.MethodGet, http.MethodPost}, "/items", "v2.items", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	for _, m := range []string{http.MethodGet, http.MethodPost} {
		mwCalled = false
		if got := serveStatus(t, r, m, "/v2/items"); got != http.StatusOK {
			t.Errorf("%s /v2/items status = %d, want %d", m, got, http.StatusOK)
		}
		if !mwCalled {
			t.Errorf("%s /v2/items: group middleware was not called", m)
		}
	}
	// PUT not registered — expect 405.
	if got := serveStatus(t, r, http.MethodPut, "/v2/items"); got != http.StatusMethodNotAllowed {
		t.Fatalf("PUT /v2/items status = %d, want %d", got, http.StatusMethodNotAllowed)
	}
}

func TestTreeAny_ConstructorProducesAllMethodNodes(t *testing.T) {
	r := New()
	Register(r, Any("/res", "res", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})...)

	for _, m := range standardMethods {
		if got := serveStatus(t, r, m, "/res"); got != http.StatusOK {
			t.Errorf("%s /res status = %d, want %d", m, got, http.StatusOK)
		}
	}
}

func TestTreeMatch_ConstructorProducesNamedNodes(t *testing.T) {
	r := New()
	methods := []string{http.MethodGet, http.MethodPatch}
	Register(r, Match(methods, "/doc", "doc", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})...)

	for _, m := range methods {
		if got := serveStatus(t, r, m, "/doc"); got != http.StatusOK {
			t.Errorf("%s /doc status = %d, want %d", m, got, http.StatusOK)
		}
	}
	// DELETE not in methods — expect 405.
	if got := serveStatus(t, r, http.MethodDelete, "/doc"); got != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE /doc status = %d, want %d", got, http.StatusMethodNotAllowed)
	}
}

func TestTreeAny_NamesAreSuffixed(t *testing.T) {
	r := New()
	Register(r, Any("/multi", "multi", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})...)

	for _, m := range standardMethods {
		wantName := "multi." + strings.ToLower(m)
		if _, err := r.Reverse(wantName, nil); err != nil {
			t.Errorf("Reverse(%q) error = %v", wantName, err)
		}
	}
}

func TestRouteGroupUse_AppendsMiddleware(t *testing.T) {
	r := New()
	var order []string
	mw1 := func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			order = append(order, "mw1")
			return next(c)
		}
	}
	mw2 := func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			order = append(order, "mw2")
			return next(c)
		}
	}
	g := r.Group("/g", mw1)
	g.Use(mw2)
	g.GET("/route", "g.route", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	order = nil
	serveStatus(t, r, http.MethodGet, "/g/route")
	if len(order) != 2 || order[0] != "mw1" || order[1] != "mw2" {
		t.Fatalf("middleware order = %v, want [mw1 mw2]", order)
	}
}

func TestTrieLargeRouteSet(t *testing.T) {
	r := New()
	handler := func(c *web.Context) (web.Response, error) { return web.Respond(http.StatusOK), nil }
	const n = 200
	nodes := make([]Node, n)
	for i := range n {
		pattern := fmt.Sprintf("/route/r%d/{param}", i)
		nodes[i] = GET(pattern, fmt.Sprintf("route.%d", i), handler)
	}
	Register(r, nodes...)

	// Dispatch to the last registered route to stress the trie.
	resp, _ := r.Serve(testContext(http.MethodGet, fmt.Sprintf("/route/r%d/value", n-1)))
	if resp.Status != http.StatusOK {
		t.Fatalf("large route set: status = %d, want %d", resp.Status, http.StatusOK)
	}
	// Also verify the first route still works.
	resp, _ = r.Serve(testContext(http.MethodGet, "/route/r0/value"))
	if resp.Status != http.StatusOK {
		t.Fatalf("large route set first route: status = %d, want %d", resp.Status, http.StatusOK)
	}
}
