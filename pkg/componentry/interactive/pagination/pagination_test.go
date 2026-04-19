package pagination_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/interactive/pagination"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestPagination(t *testing.T) {
	tests := []struct {
		name string
		node g.Node
	}{
		{
			name: "root",
			node: pagination.Root(),
		},
		{
			name: "content",
			node: pagination.Content(),
		},
		{
			name: "item",
			node: pagination.Item(),
		},
		{
			name: "link active",
			node: pagination.Link("/page/2", true),
		},
		{
			name: "link inactive",
			node: pagination.Link("/page/3", false),
		},
		{
			name: "previous enabled",
			node: pagination.Previous("/page/1", false),
		},
		{
			name: "previous disabled",
			node: pagination.Previous("", true),
		},
		{
			name: "next enabled",
			node: pagination.Next("/page/3", false),
		},
		{
			name: "next disabled",
			node: pagination.Next("", true),
		},
		{
			name: "ellipsis",
			node: pagination.Ellipsis(),
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
