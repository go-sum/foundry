package router

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-sum/web"
)

func benchRouter(n int) *Router {
	r := New()
	handler := func(c *web.Context) (web.Response, error) { return web.Respond(http.StatusOK), nil }
	nodes := make([]Node, n)
	for i := range n {
		pattern := fmt.Sprintf("/route/r%d/{param}", i)
		nodes[i] = GET(pattern, "", handler)
	}
	Register(r, nodes...)
	r.freeze()
	return r
}

func BenchmarkRouter_1Route(b *testing.B) {
	r := benchRouter(1)
	c := testContext(http.MethodGet, "/route/r0/value")
	b.ResetTimer()
	for range b.N {
		r.Serve(c)
	}
}

func BenchmarkRouter_10Routes(b *testing.B) {
	r := benchRouter(10)
	c := testContext(http.MethodGet, "/route/r9/value") // last route
	b.ResetTimer()
	for range b.N {
		r.Serve(c)
	}
}

func BenchmarkRouter_100Routes(b *testing.B) {
	r := benchRouter(100)
	c := testContext(http.MethodGet, "/route/r99/value") // last route
	b.ResetTimer()
	for range b.N {
		r.Serve(c)
	}
}

func BenchmarkRouter_500Routes(b *testing.B) {
	r := benchRouter(500)
	c := testContext(http.MethodGet, "/route/r499/value") // last route
	b.ResetTimer()
	for range b.N {
		r.Serve(c)
	}
}

func BenchmarkRouter_1000Routes(b *testing.B) {
	r := benchRouter(1000)
	c := testContext(http.MethodGet, "/route/r999/value") // last route
	b.ResetTimer()
	for range b.N {
		r.Serve(c)
	}
}

func BenchmarkRouter_Reverse(b *testing.B) {
	r := New()
	Register(r, GET("/users/{id}/posts/{postID}", "user.post.show", func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	}))
	params := map[string]string{"id": "42", "postID": "99"}
	b.ResetTimer()
	for range b.N {
		if _, err := r.Reverse("user.post.show", params); err != nil {
			b.Fatal(err)
		}
	}
}


