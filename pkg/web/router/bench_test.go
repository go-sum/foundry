package router

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/web"
)

func benchRouter(n int) *Router {
	r := NewWithoutSecureDefaults()
	handler := func(c *web.Context) (web.Response, error) { return web.Respond(http.StatusOK), nil }
	for i := range n {
		pattern := fmt.Sprintf("/route/r%d/{param}", i)
		r.GET(pattern, "", handler)
	}
	r.freeze()
	return r
}

func BenchmarkRouter_1Route(b *testing.B) {
	r := benchRouter(1)
	c := benchContext(http.MethodGet, "/route/r0/value")
	b.ResetTimer()
	for range b.N {
		r.Serve(c)
	}
}

func BenchmarkRouter_10Routes(b *testing.B) {
	r := benchRouter(10)
	c := benchContext(http.MethodGet, "/route/r9/value") // last route
	b.ResetTimer()
	for range b.N {
		r.Serve(c)
	}
}

func BenchmarkRouter_100Routes(b *testing.B) {
	r := benchRouter(100)
	c := benchContext(http.MethodGet, "/route/r99/value") // last route
	b.ResetTimer()
	for range b.N {
		r.Serve(c)
	}
}

func BenchmarkRouter_500Routes(b *testing.B) {
	r := benchRouter(500)
	c := benchContext(http.MethodGet, "/route/r499/value") // last route
	b.ResetTimer()
	for range b.N {
		r.Serve(c)
	}
}

func BenchmarkRouter_1000Routes(b *testing.B) {
	r := benchRouter(1000)
	c := benchContext(http.MethodGet, "/route/r999/value") // last route
	b.ResetTimer()
	for range b.N {
		r.Serve(c)
	}
}

func BenchmarkRouter_Reverse(b *testing.B) {
	r := NewWithoutSecureDefaults()
	r.GET("/users/{id}/posts/{postID}", "user.post.show", func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})
	params := map[string]string{"id": "42", "postID": "99"}
	b.ResetTimer()
	for range b.N {
		if _, err := r.Reverse("user.post.show", params); err != nil {
			b.Fatal(err)
		}
	}
}

func benchContext(method, path string) *web.Context {
	u, _ := url.Parse(path)
	req := web.NewRequest(method, u)
	return web.NewContext(context.Background(), req)
}
