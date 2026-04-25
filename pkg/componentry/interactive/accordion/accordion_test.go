package accordion_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/interactive/accordion"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestAccordion(t *testing.T) {
	tests := []struct {
		name string
		node g.Node
	}{
		{
			name: "root empty",
			node: accordion.Root(accordion.RootProps{}),
		},
		{
			name: "root with children",
			node: accordion.Root(accordion.RootProps{}, g.Text("items")),
		},
		{
			name: "root exclusive",
			node: accordion.Root(accordion.RootProps{Exclusive: true}, g.Text("items")),
		},
		{
			name: "item empty",
			node: accordion.Item(),
		},
		{
			name: "item with children",
			node: accordion.Item(g.Text("item-content")),
		},
		{
			name: "trigger with children",
			node: accordion.Trigger(nil, g.Text("Click me")),
		},
		{
			name: "trigger no children",
			node: accordion.Trigger(nil),
		},
		{
			name: "content with children",
			node: accordion.Content(g.Text("Details here")),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, tc.node)
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}
