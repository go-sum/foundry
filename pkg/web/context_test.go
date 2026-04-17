package web

import (
	"context"
	"net/url"
	"testing"
)

func TestContextParams(t *testing.T) {
	c := NewContext(context.Background(), Request{})
	c.SetParam("id", "123")
	c.SetParams(map[string]string{"slug": "hello"})

	if got := c.Param("id"); got != "123" {
		t.Fatalf("Param(id) = %q, want %q", got, "123")
	}
	if got := c.Param("slug"); got != "hello" {
		t.Fatalf("Param(slug) = %q, want %q", got, "hello")
	}

	params := c.Params()
	params["id"] = "mutated"
	if got := c.Param("id"); got != "123" {
		t.Fatalf("Params() should return a copy, got %q", got)
	}
}

func TestContextValues(t *testing.T) {
	type key struct{}

	c := NewContext(context.Background(), Request{})
	c.Set(key{}, "hello")
	got, ok := Get[string](c, key{})
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got != "hello" {
		t.Fatalf("Get = %q, want %q", got, "hello")
	}
}

func BenchmarkAcquireReleaseContext(b *testing.B) {
	req := NewRequest("GET", &url.URL{Path: "/"})
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := AcquireContext(context.Background(), req)
			ReleaseContext(c)
		}
	})
}
