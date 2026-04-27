package breadcrumb_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/interactive/breadcrumb"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestBreadcrumb(t *testing.T) {
	tests := []struct {
		name string
		node g.Node
	}{
		{
			name: "root empty",
			node: breadcrumb.Root(),
		},
		{
			name: "root with children",
			node: breadcrumb.Root(g.Text("content")),
		},
		{
			name: "list empty",
			node: breadcrumb.List(),
		},
		{
			name: "item with children",
			node: breadcrumb.Item(g.Text("Home")),
		},
		{
			name: "link",
			node: breadcrumb.Link("/home", g.Text("Home")),
		},
		{
			name: "separator default",
			node: breadcrumb.Separator(),
		},
		{
			name: "separator custom",
			node: breadcrumb.Separator(g.Text(">")),
		},
		{
			name: "page",
			node: breadcrumb.Page(g.Text("Dashboard")),
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
