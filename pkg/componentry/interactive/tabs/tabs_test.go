package tabs_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/interactive/tabs"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestTabs(t *testing.T) {
	tests := []struct {
		name string
		node g.Node
	}{
		{
			name: "root",
			node: tabs.Root("my-tabs", "tab1"),
		},
		{
			name: "root with children",
			node: tabs.Root("tabs", "t1", g.Text("content")),
		},
		{
			name: "list empty",
			node: tabs.List(),
		},
		{
			name: "trigger active",
			node: tabs.Trigger("my-tabs", "tab1", true, g.Text("Tab 1")),
		},
		{
			name: "trigger inactive",
			node: tabs.Trigger("my-tabs", "tab2", false, g.Text("Tab 2")),
		},
		{
			name: "trigger aria controls",
			node: tabs.Trigger("tabs", "detail", false),
		},
		{
			name: "content default",
			node: tabs.Content("my-tabs", "tab1", true, g.Text("Panel 1")),
		},
		{
			name: "content hidden",
			node: tabs.Content("my-tabs", "tab2", false, g.Text("Panel 2")),
		},
		{
			name: "content aria labelledby",
			node: tabs.Content("tabs", "detail", false),
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
