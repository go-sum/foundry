package tooltip_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/interactive/tooltip"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestTooltip(t *testing.T) {
	tests := []struct {
		name string
		node g.Node
	}{
		{
			name: "trigger attrs",
			node: g.El("button", tooltip.TriggerAttrs("my-tooltip")...),
		},
		{
			name: "trigger attrs empty id",
			node: g.El("button", tooltip.TriggerAttrs("")...),
		},
		{
			name: "root empty",
			node: tooltip.Root(),
		},
		{
			name: "root with children",
			node: tooltip.Root(g.Text("trigger")),
		},
		{
			name: "trigger with children",
			node: tooltip.Trigger(g.Text("hover me")),
		},
		{
			name: "content",
			node: tooltip.Content("my-tooltip", g.Text("Tooltip text")),
		},
		{
			name: "click root",
			node: tooltip.ClickRoot(),
		},
		{
			name: "click content",
			node: tooltip.ClickContent("click-tip", g.Text("Click tooltip")),
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
