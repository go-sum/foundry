package layout

import (
	"testing"

	"github.com/go-sum/web/render"

	g "maragu.dev/gomponents"
)

func TestPage(t *testing.T) {
	props := Props{
		Title:    "Test Page",
		Children: []g.Node{g.Text("content")},
	}
	got := render.RenderNode(t, Page(props))
	want := render.RenderNode(t, Page(props))

	if got != want {
		t.Errorf("Page output is not stable\ngot:  %s\nwant: %s", got, want)
	}
}
