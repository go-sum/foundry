package dialog_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/interactive/dialog"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestDialog(t *testing.T) {
	tests := []struct {
		name string
		node g.Node
	}{
		{
			name: "content empty",
			node: dialog.Content("confirm-dialog"),
		},
		{
			name: "content with children",
			node: dialog.Content("dlg", g.Text("dialog content")),
		},
		{
			name: "trigger",
			node: dialog.Trigger("confirm-dialog", g.Text("Open")),
		},
		{
			name: "close",
			node: dialog.Close(g.Text("Cancel")),
		},
		{
			name: "title",
			node: dialog.Title("confirm-dialog", g.Text("Are you sure?")),
		},
		{
			name: "description",
			node: dialog.Description("confirm-dialog", g.Text("This action cannot be undone.")),
		},
		{
			name: "header",
			node: dialog.Header(g.Text("header")),
		},
		{
			name: "footer",
			node: dialog.Footer(g.Text("footer")),
		},
		{
			name: "root",
			node: dialog.Root(g.Text("content")),
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
