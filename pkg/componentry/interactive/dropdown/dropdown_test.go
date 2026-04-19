package dropdown_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/interactive/dropdown"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestDropdown(t *testing.T) {
	tests := []struct {
		name string
		node g.Node
	}{
		{
			name: "root empty props",
			node: dropdown.Root(dropdown.Props{}),
		},
		{
			name: "root with id",
			node: dropdown.Root(dropdown.Props{ID: "user-menu"}),
		},
		{
			name: "trigger with children",
			node: dropdown.Trigger(dropdown.TriggerProps{}, g.Text("Options")),
		},
		{
			name: "trigger no children",
			node: dropdown.Trigger(dropdown.TriggerProps{}),
		},
		{
			name: "item link",
			node: dropdown.Item(dropdown.ItemProps{Label: "Settings", Href: "/settings"}),
		},
		{
			name: "item button",
			node: dropdown.Item(dropdown.ItemProps{Label: "Delete"}),
		},
		{
			name: "item disabled link",
			node: dropdown.Item(dropdown.ItemProps{Label: "Archive", Href: "/archive", Disabled: true}),
		},
		{
			name: "item disabled button",
			node: dropdown.Item(dropdown.ItemProps{Label: "Save", Disabled: true}),
		},
		{
			name: "separator",
			node: dropdown.Separator(),
		},
		{
			name: "label",
			node: dropdown.Label("My Account"),
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
