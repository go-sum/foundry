package page

import (
	"testing"

	"github.com/go-sum/web/render"
	"github.com/go-sum/foundry/internal/view"
)

func TestHelloPage(t *testing.T) {
	req := view.Request{}
	got := render.RenderNode(t, HelloPage(req, "World"))
	want := render.RenderNode(t, HelloPage(req, "World"))

	if got != want {
		t.Errorf("HelloPage output is not stable\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHelloPartial(t *testing.T) {
	got := render.RenderNode(t, HelloPartial("World"))
	want := render.RenderNode(t, HelloPartial("World"))

	if got != want {
		t.Errorf("HelloPartial output is not stable\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHelloPartial_HTMLEntities(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"apostrophe", "O'Brien"},
		{"ampersand", "AT&T"},
		{"angle brackets", "<script>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := render.RenderNode(t, HelloPartial(tt.input))
			want := render.RenderNode(t, HelloPartial(tt.input))
			if got != want {
				t.Errorf("HelloPartial(%q) output is not stable\ngot:  %s\nwant: %s", tt.input, got, want)
			}
		})
	}
}
