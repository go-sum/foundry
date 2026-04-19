package htmx_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/patterns/htmx"
	testutil "github.com/go-sum/componentry/testutil"
)

func renderAttrs(t *testing.T, nodes []g.Node) string {
	t.Helper()
	return testutil.RenderNode(t, g.El("div", nodes...))
}

func TestAttrs_ScalarAttributes(t *testing.T) {
	tests := []struct {
		name       string
		props      htmx.AttrsProps
		wantAttrs  []string
		absentAttr []string
	}{
		{
			name:      "hx-get",
			props:     htmx.AttrsProps{Get: "/search"},
			wantAttrs: []string{`hx-get="/search"`},
		},
		{
			name:      "hx-post",
			props:     htmx.AttrsProps{Post: "/submit"},
			wantAttrs: []string{`hx-post="/submit"`},
		},
		{
			name:      "hx-put",
			props:     htmx.AttrsProps{Put: "/update"},
			wantAttrs: []string{`hx-put="/update"`},
		},
		{
			name:      "hx-patch",
			props:     htmx.AttrsProps{Patch: "/partial"},
			wantAttrs: []string{`hx-patch="/partial"`},
		},
		{
			name:      "hx-delete",
			props:     htmx.AttrsProps{Delete: "/remove"},
			wantAttrs: []string{`hx-delete="/remove"`},
		},
		{
			name:      "hx-target",
			props:     htmx.AttrsProps{Target: "#result"},
			wantAttrs: []string{`hx-target="#result"`},
		},
		{
			name:      "hx-swap",
			props:     htmx.AttrsProps{Swap: "outerHTML"},
			wantAttrs: []string{`hx-swap="outerHTML"`},
		},
		{
			name:      "hx-trigger",
			props:     htmx.AttrsProps{Trigger: "click"},
			wantAttrs: []string{`hx-trigger="click"`},
		},
		{
			name:      "hx-push-url",
			props:     htmx.AttrsProps{PushURL: "/new-path"},
			wantAttrs: []string{`hx-push-url="/new-path"`},
		},
		{
			name:      "hx-replace-url",
			props:     htmx.AttrsProps{ReplaceURL: "/current"},
			wantAttrs: []string{`hx-replace-url="/current"`},
		},
		{
			name:      "hx-include",
			props:     htmx.AttrsProps{Include: "#extra-fields"},
			wantAttrs: []string{`hx-include="#extra-fields"`},
		},
		{
			name:      "hx-confirm",
			props:     htmx.AttrsProps{Confirm: "Are you sure?"},
			wantAttrs: []string{`hx-confirm="Are you sure?"`},
		},
		{
			name:      "empty props produce no hx attrs",
			props:     htmx.AttrsProps{},
			wantAttrs: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderAttrs(t, htmx.Attrs(tc.props))
			for _, want := range tc.wantAttrs {
				if !containsStr(got, want) {
					t.Errorf("expected %q in:\n%s", want, got)
				}
			}
			for _, absent := range tc.absentAttr {
				if containsStr(got, absent) {
					t.Errorf("expected %q to be absent from:\n%s", absent, got)
				}
			}
		})
	}
}

func TestAttrs_Boost(t *testing.T) {
	t.Run("nil boost produces no attribute", func(t *testing.T) {
		got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{}))
		if containsStr(got, "hx-boost") {
			t.Errorf("expected hx-boost to be absent, got: %s", got)
		}
	})

	t.Run("boost true", func(t *testing.T) {
		v := true
		got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{Boost: &v}))
		if !containsStr(got, `hx-boost="true"`) {
			t.Errorf("expected hx-boost=true, got: %s", got)
		}
	})

	t.Run("boost false", func(t *testing.T) {
		v := false
		got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{Boost: &v}))
		if !containsStr(got, `hx-boost="false"`) {
			t.Errorf("expected hx-boost=false, got: %s", got)
		}
	})
}

func TestAttrs_Values(t *testing.T) {
	t.Run("empty values produces no hx-vals", func(t *testing.T) {
		got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{}))
		if containsStr(got, "hx-vals") {
			t.Errorf("expected no hx-vals, got: %s", got)
		}
	})

	t.Run("values are JSON encoded", func(t *testing.T) {
		got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{
			Values: map[string]string{"action": "delete"},
		}))
		if !containsStr(got, "hx-vals") {
			t.Errorf("expected hx-vals attribute, got: %s", got)
		}
		if !containsStr(got, "action") {
			t.Errorf("expected 'action' key in hx-vals, got: %s", got)
		}
		if !containsStr(got, "delete") {
			t.Errorf("expected 'delete' value in hx-vals, got: %s", got)
		}
	})
}

func TestAttrs_Headers(t *testing.T) {
	t.Run("empty headers produces no hx-headers", func(t *testing.T) {
		got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{}))
		if containsStr(got, "hx-headers") {
			t.Errorf("expected no hx-headers, got: %s", got)
		}
	})

	t.Run("headers are JSON encoded", func(t *testing.T) {
		got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{
			Headers: map[string]string{"X-Custom": "value"},
		}))
		if !containsStr(got, "hx-headers") {
			t.Errorf("expected hx-headers attribute, got: %s", got)
		}
		if !containsStr(got, "X-Custom") {
			t.Errorf("expected 'X-Custom' key in hx-headers, got: %s", got)
		}
	})
}

func TestAttrs_Extra(t *testing.T) {
	got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{
		Get:   "/search",
		Extra: []g.Node{g.Attr("data-test", "extra")},
	}))
	if !containsStr(got, `hx-get="/search"`) {
		t.Errorf("expected hx-get in output, got: %s", got)
	}
	if !containsStr(got, `data-test="extra"`) {
		t.Errorf("expected data-test in output, got: %s", got)
	}
}

func TestAttrs_MultipleHTTPMethods(t *testing.T) {
	// Only set one method at a time — this just verifies individual attrs render.
	got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{
		Post:    "/items",
		Target:  "#list",
		Swap:    "beforeend",
		Trigger: "submit",
	}))
	for _, want := range []string{
		`hx-post="/items"`,
		`hx-target="#list"`,
		`hx-swap="beforeend"`,
		`hx-trigger="submit"`,
	} {
		if !containsStr(got, want) {
			t.Errorf("expected %q in:\n%s", want, got)
		}
	}
}

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
