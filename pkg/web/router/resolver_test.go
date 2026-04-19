package router

import (
	"net/http"
	"testing"

	"github.com/go-sum/web"
)

func TestResolver_PathAndURL(t *testing.T) {
	r := New()
	r.GET("/users/{id}", "users.show", func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	resolver := NewResolver(r)
	if got, want := resolver.Path("users.show")(), "/users/{id}"; got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
	if got, want := resolver.URL("https://example.com", "users.show", map[string]string{"id": "42"})(), "https://example.com/users/42"; got != want {
		t.Fatalf("URL() = %q, want %q", got, want)
	}
}

func TestResolver_PathPanicsForUnknownRoute(t *testing.T) {
	resolver := NewResolver(New())

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unknown route")
		}
	}()
	_ = resolver.Path("missing")()
}
