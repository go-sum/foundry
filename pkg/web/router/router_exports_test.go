package router

import (
	"net/http"
	"testing"

	"github.com/go-sum/web"
)

func TestConvenienceMethods_RegisterExpectedRoutes(t *testing.T) {
	r := New()
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}

	r.GET("/get", "get.route", handler)
	r.POST("/post", "post.route", handler)
	r.PUT("/put", "put.route", handler)
	r.PATCH("/patch", "patch.route", handler)
	r.DELETE("/delete", "delete.route", handler)
	r.HEAD("/head", "head.route", handler)

	routes := r.Routes()
	if len(routes) != 6 {
		t.Fatalf("len(Routes()) = %d, want 6", len(routes))
	}

	want := []Route{
		{Method: http.MethodGet, Pattern: "/get", Name: "get.route"},
		{Method: http.MethodPost, Pattern: "/post", Name: "post.route"},
		{Method: http.MethodPut, Pattern: "/put", Name: "put.route"},
		{Method: http.MethodPatch, Pattern: "/patch", Name: "patch.route"},
		{Method: http.MethodDelete, Pattern: "/delete", Name: "delete.route"},
		{Method: http.MethodHead, Pattern: "/head", Name: "head.route"},
	}

	for i, rt := range routes {
		if rt.Method != want[i].Method || rt.Pattern != want[i].Pattern || rt.Name != want[i].Name {
			t.Fatalf("Routes()[%d] = %#v, want method=%q pattern=%q name=%q", i, rt, want[i].Method, want[i].Pattern, want[i].Name)
		}
		if rt.Handler == nil {
			t.Fatalf("Routes()[%d].Handler = nil, want handler", i)
		}
	}
}

func TestMustReverseAndPattern(t *testing.T) {
	r := New()
	r.GET("/users/{id}", "users.show", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	if got, want := r.MustReverse("users.show", map[string]string{"id": "42"}), "/users/42"; got != want {
		t.Fatalf("MustReverse() = %q, want %q", got, want)
	}
	got, err := r.Pattern("users.show")
	if err != nil {
		t.Fatalf("Pattern() error = %v", err)
	}
	if want := "/users/{id}"; got != want {
		t.Fatalf("Pattern() = %q, want %q", got, want)
	}
	if _, err := r.Pattern("missing"); err == nil {
		t.Fatal("Pattern(missing) error = nil, want non-nil")
	}
}

func TestMustReverse_PanicsOnMissingRoute(t *testing.T) {
	r := New()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unknown route")
		}
	}()
	_ = r.MustReverse("missing", nil)
}

func TestRouteGroupConvenienceMethods_RegisterExpectedRoutes(t *testing.T) {
	r := New()
	g := r.Group("/api")
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}

	g.POST("/post", "post.route", handler)
	g.PUT("/put", "put.route", handler)
	g.PATCH("/patch", "patch.route", handler)
	g.DELETE("/delete", "delete.route", handler)
	g.HEAD("/head", "head.route", handler)
	g.OPTIONS("/options", "options.route", handler)

	routes := r.Routes()
	if len(routes) != 6 {
		t.Fatalf("len(Routes()) = %d, want 6", len(routes))
	}

	want := []Route{
		{Method: http.MethodPost, Pattern: "/api/post", Name: "post.route"},
		{Method: http.MethodPut, Pattern: "/api/put", Name: "put.route"},
		{Method: http.MethodPatch, Pattern: "/api/patch", Name: "patch.route"},
		{Method: http.MethodDelete, Pattern: "/api/delete", Name: "delete.route"},
		{Method: http.MethodHead, Pattern: "/api/head", Name: "head.route"},
		{Method: http.MethodOptions, Pattern: "/api/options", Name: "options.route"},
	}

	for i, rt := range routes {
		if rt.Method != want[i].Method || rt.Pattern != want[i].Pattern || rt.Name != want[i].Name {
			t.Fatalf("Routes()[%d] = %#v, want method=%q pattern=%q name=%q", i, rt, want[i].Method, want[i].Pattern, want[i].Name)
		}
	}
}
