package core_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/ui/core"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestBadge(t *testing.T) {
	tests := []struct {
		name  string
		props core.BadgeProps
	}{
		{
			name:  "default variant",
			props: core.BadgeProps{Children: []g.Node{g.Text("New")}},
		},
		{
			name:  "secondary variant",
			props: core.BadgeProps{Variant: core.BadgeSecondary, Children: []g.Node{g.Text("Draft")}},
		},
		{
			name:  "destructive variant",
			props: core.BadgeProps{Variant: core.BadgeDestructive, Children: []g.Node{g.Text("Error")}},
		},
		{
			name:  "outline variant",
			props: core.BadgeProps{Variant: core.BadgeOutline, Children: []g.Node{g.Text("Beta")}},
		},
		{
			name:  "with id",
			props: core.BadgeProps{ID: "my-badge", Children: []g.Node{g.Text("Tagged")}},
		},
		{
			name:  "with icon child",
			props: core.BadgeProps{Children: []g.Node{g.Text("★"), g.Text(" Featured")}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, core.Badge(tc.props))
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}
