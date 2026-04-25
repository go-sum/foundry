package router

import (
	"net/http"
	"testing"

	"github.com/go-sum/web"
)

func okHandler() web.Handler {
	return func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}
}

func TestResources_AllHandlers(t *testing.T) {
	r := New()
	Register(r, Resources("/users", "users", ResourceHandlers{
		Index:   okHandler(),
		New:     okHandler(),
		Create:  okHandler(),
		Show:    okHandler(),
		Edit:    okHandler(),
		Update:  okHandler(),
		Destroy: okHandler(),
	})...)

	want := []Route{
		{Method: http.MethodGet, Pattern: "/users", Name: "users.index"},
		{Method: http.MethodGet, Pattern: "/users/new", Name: "users.new"},
		{Method: http.MethodPost, Pattern: "/users", Name: "users.create"},
		{Method: http.MethodGet, Pattern: "/users/{id}", Name: "users.show"},
		{Method: http.MethodGet, Pattern: "/users/{id}/edit", Name: "users.edit"},
		{Method: http.MethodPatch, Pattern: "/users/{id}", Name: "users.update"},
		{Method: http.MethodDelete, Pattern: "/users/{id}", Name: "users.destroy"},
	}

	routes := r.Routes()
	if len(routes) != len(want) {
		t.Fatalf("len(Routes()) = %d, want %d", len(routes), len(want))
	}
	for i, rt := range routes {
		if rt.Method != want[i].Method || rt.Pattern != want[i].Pattern || rt.Name != want[i].Name {
			t.Errorf("Routes()[%d] = {%s %s %q}, want {%s %s %q}",
				i, rt.Method, rt.Pattern, rt.Name,
				want[i].Method, want[i].Pattern, want[i].Name)
		}
		if rt.Handler == nil {
			t.Errorf("Routes()[%d].Handler = nil", i)
		}
	}
}

func TestResources_PartialHandlers(t *testing.T) {
	// Only Index and Create — other routes must not be registered.
	r := New()
	Register(r, Resources("/posts", "posts", ResourceHandlers{
		Index:  okHandler(),
		Create: okHandler(),
	})...)

	routes := r.Routes()
	if len(routes) != 2 {
		t.Fatalf("len(Routes()) = %d, want 2", len(routes))
	}

	want := []Route{
		{Method: http.MethodGet, Pattern: "/posts", Name: "posts.index"},
		{Method: http.MethodPost, Pattern: "/posts", Name: "posts.create"},
	}
	for i, rt := range routes {
		if rt.Method != want[i].Method || rt.Pattern != want[i].Pattern || rt.Name != want[i].Name {
			t.Errorf("Routes()[%d] = {%s %s %q}, want {%s %s %q}",
				i, rt.Method, rt.Pattern, rt.Name,
				want[i].Method, want[i].Pattern, want[i].Name)
		}
	}
}

func TestResources_Dispatch(t *testing.T) {
	r := New()
	Register(r, Resources("/items", "items", ResourceHandlers{
		Index:   okHandler(),
		Show:    okHandler(),
		Update:  okHandler(),
		Destroy: okHandler(),
	})...)

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/items", http.StatusOK},
		{http.MethodGet, "/items/42", http.StatusOK},
		{http.MethodPatch, "/items/42", http.StatusOK},
		{http.MethodDelete, "/items/42", http.StatusOK},
		{http.MethodPost, "/items", http.StatusMethodNotAllowed},
		{http.MethodGet, "/items/42/edit", http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			got := serveStatus(t, r, tc.method, tc.path)
			if got != tc.want {
				t.Errorf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestResource_AllHandlers(t *testing.T) {
	r := New()
	Register(r, Resource("/profile", "profile", SingleResourceHandlers{
		Show:    okHandler(),
		New:     okHandler(),
		Create:  okHandler(),
		Edit:    okHandler(),
		Update:  okHandler(),
		Destroy: okHandler(),
	})...)

	want := []Route{
		{Method: http.MethodGet, Pattern: "/profile", Name: "profile.show"},
		{Method: http.MethodGet, Pattern: "/profile/new", Name: "profile.new"},
		{Method: http.MethodPost, Pattern: "/profile", Name: "profile.create"},
		{Method: http.MethodGet, Pattern: "/profile/edit", Name: "profile.edit"},
		{Method: http.MethodPatch, Pattern: "/profile", Name: "profile.update"},
		{Method: http.MethodDelete, Pattern: "/profile", Name: "profile.destroy"},
	}

	routes := r.Routes()
	if len(routes) != len(want) {
		t.Fatalf("len(Routes()) = %d, want %d", len(routes), len(want))
	}
	for i, rt := range routes {
		if rt.Method != want[i].Method || rt.Pattern != want[i].Pattern || rt.Name != want[i].Name {
			t.Errorf("Routes()[%d] = {%s %s %q}, want {%s %s %q}",
				i, rt.Method, rt.Pattern, rt.Name,
				want[i].Method, want[i].Pattern, want[i].Name)
		}
		if rt.Handler == nil {
			t.Errorf("Routes()[%d].Handler = nil", i)
		}
	}
}

func TestResource_PartialHandlers(t *testing.T) {
	r := New()
	Register(r, Resource("/settings", "settings", SingleResourceHandlers{
		Show:   okHandler(),
		Update: okHandler(),
	})...)

	routes := r.Routes()
	if len(routes) != 2 {
		t.Fatalf("len(Routes()) = %d, want 2", len(routes))
	}

	want := []Route{
		{Method: http.MethodGet, Pattern: "/settings", Name: "settings.show"},
		{Method: http.MethodPatch, Pattern: "/settings", Name: "settings.update"},
	}
	for i, rt := range routes {
		if rt.Method != want[i].Method || rt.Pattern != want[i].Pattern || rt.Name != want[i].Name {
			t.Errorf("Routes()[%d] = {%s %s %q}, want {%s %s %q}",
				i, rt.Method, rt.Pattern, rt.Name,
				want[i].Method, want[i].Pattern, want[i].Name)
		}
	}
}

func TestResource_Dispatch(t *testing.T) {
	r := New()
	Register(r, Resource("/profile", "profile", SingleResourceHandlers{
		Show:    okHandler(),
		Update:  okHandler(),
		Destroy: okHandler(),
	})...)

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/profile", http.StatusOK},
		{http.MethodPatch, "/profile", http.StatusOK},
		{http.MethodDelete, "/profile", http.StatusOK},
		{http.MethodPost, "/profile", http.StatusMethodNotAllowed},
		{http.MethodGet, "/profile/edit", http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			got := serveStatus(t, r, tc.method, tc.path)
			if got != tc.want {
				t.Errorf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestResources_ComposesWithGroup(t *testing.T) {
	r := New()
	Register(r,
		Group("/api/v1",
			Resources("/users", "api.users", ResourceHandlers{
				Index: okHandler(),
				Show:  okHandler(),
			})...,
		),
	)

	want := []Route{
		{Method: http.MethodGet, Pattern: "/api/v1/users", Name: "api.users.index"},
		{Method: http.MethodGet, Pattern: "/api/v1/users/{id}", Name: "api.users.show"},
	}

	routes := r.Routes()
	if len(routes) != len(want) {
		t.Fatalf("len(Routes()) = %d, want %d", len(routes), len(want))
	}
	for i, rt := range routes {
		if rt.Method != want[i].Method || rt.Pattern != want[i].Pattern || rt.Name != want[i].Name {
			t.Errorf("Routes()[%d] = {%s %s %q}, want {%s %s %q}",
				i, rt.Method, rt.Pattern, rt.Name,
				want[i].Method, want[i].Pattern, want[i].Name)
		}
	}
}

func TestResources_EmptyHandlers_RegistersNoRoutes(t *testing.T) {
	r := New()
	Register(r, Resources("/empty", "empty", ResourceHandlers{})...)
	if len(r.Routes()) != 0 {
		t.Fatalf("expected 0 routes for empty ResourceHandlers, got %d", len(r.Routes()))
	}
}
