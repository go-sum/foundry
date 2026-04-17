package page

import (
	"testing"

	"github.com/go-sum/web/render"
	"github.com/go-sum/foundry/internal/view"
)

func TestHomePage(t *testing.T) {
	req := view.Request{}
	got := render.RenderNode(t, HomePage(req))
	want := render.RenderNode(t, HomePage(req))

	if got != want {
		t.Errorf("HomePage output is not stable\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHomeContent(t *testing.T) {
	got := render.RenderNode(t, HomeContent())
	want := render.RenderNode(t, HomeContent())

	if got != want {
		t.Errorf("HomeContent output is not stable\ngot:  %s\nwant: %s", got, want)
	}
}
