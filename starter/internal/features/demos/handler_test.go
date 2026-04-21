package demos

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-sum/componentry/showcase"
	"github.com/go-sum/componentry/showcase/demo"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
)

func TestHandlerShow(t *testing.T) {
	t.Parallel()

	h := NewHandler(nil)

	u, _ := url.Parse("/demos/")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	vr := view.NewRequest(c)
	want := render.RenderNode(t, vr.Page("Component Showcase", showcase.Showcase()))
	if string(body) != want {
		t.Fatalf("full page body mismatch\ngot:  %s\nwant: %s", string(body), want)
	}
}

func TestHandlerShow_HTMXReturnsFullPage(t *testing.T) {
	// nil partial means HTMX requests receive the full page instead of a fragment.
	t.Parallel()

	h := NewHandler(nil)

	u, _ := url.Parse("/demos/")
	req := web.NewRequest(http.MethodGet, u)
	req.Headers.Set("HX-Request", "true")
	c := web.NewContext(context.Background(), req)
	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	vr := view.NewRequest(c)
	want := render.RenderNode(t, vr.Page("Component Showcase", showcase.Showcase()))
	if string(body) != want {
		t.Fatalf("HTMX body mismatch\ngot:  %s\nwant: %s", string(body), want)
	}
}

func TestHandlerSearch(t *testing.T) {
	t.Parallel()

	h := NewHandler(nil)

	tests := []struct {
		query   string
		contain string
	}{
		{"", "Alice Johnson"},
		{"alice", "Alice Johnson"},
		{"zzz", "No results found"},
	}
	for _, tc := range tests {
		u, _ := url.Parse("/componentry/demo/search?q=" + tc.query)
		req := web.NewRequest(http.MethodGet, u)
		c := web.NewContext(context.Background(), req)
		resp, err := h.Search(c)
		if err != nil {
			t.Fatalf("query=%q unexpected error: %v", tc.query, err)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), tc.contain) {
			t.Fatalf("query=%q: expected body to contain %q\ngot: %s", tc.query, tc.contain, string(body))
		}
	}
}

func TestHandlerValidate(t *testing.T) {
	t.Parallel()

	h := NewHandler(nil)

	tests := []struct {
		field   string
		value   string
		contain string
	}{
		{"email", "bad", "valid email"},
		{"email", "good@example.com", "Looks good"},
		{"username", "ab", "3 characters"},
		{"username", "alice", "Looks good"},
	}
	for _, tc := range tests {
		u, _ := url.Parse("/componentry/demo/validate?field=" + tc.field + "&value=" + tc.value)
		req := web.NewRequest(http.MethodGet, u)
		c := web.NewContext(context.Background(), req)
		resp, err := h.Validate(c)
		if err != nil {
			t.Fatalf("field=%q value=%q unexpected error: %v", tc.field, tc.value, err)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), tc.contain) {
			t.Fatalf("field=%q value=%q: expected body to contain %q\ngot: %s", tc.field, tc.value, tc.contain, string(body))
		}
	}
}

func TestHandlerPaginate(t *testing.T) {
	t.Parallel()

	h := NewHandler(nil)

	u, _ := url.Parse("/componentry/demo/paginate?page=1&per_page=10")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	resp, err := h.Paginate(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "paginate-region") {
		t.Fatalf("expected paginate-region in body\ngot: %s", string(body))
	}
}

func TestHandlerRegion(t *testing.T) {
	t.Parallel()

	h := NewHandler(nil)

	tests := []struct {
		id      string
		contain string
	}{
		{"se", "Stockholm"},
		{"us", "California"},
		{"xx", "No regions available"},
	}
	for _, tc := range tests {
		u, _ := url.Parse("/componentry/demo/region/" + tc.id)
		req := web.NewRequest(http.MethodGet, u)
		c := web.NewContext(context.Background(), req)
		c.SetParam("id", tc.id)
		resp, err := h.Region(c)
		if err != nil {
			t.Fatalf("id=%q unexpected error: %v", tc.id, err)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), tc.contain) {
			t.Fatalf("id=%q: expected body to contain %q\ngot: %s", tc.id, tc.contain, string(body))
		}
	}
}

func TestDemoPathConstants(t *testing.T) {
	// Ensure path constants match the expected route prefix.
	if !strings.HasPrefix(demo.PathSearch, "/componentry/") {
		t.Errorf("PathSearch has unexpected prefix: %s", demo.PathSearch)
	}
	if !strings.HasPrefix(demo.PathRegion, "/componentry/") {
		t.Errorf("PathRegion has unexpected prefix: %s", demo.PathRegion)
	}
}
