package core_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/ui/core"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestButton(t *testing.T) {
	tests := []struct {
		name  string
		props core.ButtonProps
	}{
		{
			name:  "default variant",
			props: core.ButtonProps{Label: "Click me"},
		},
		{
			name:  "destructive variant",
			props: core.ButtonProps{Label: "Delete", Variant: core.VariantDestructive},
		},
		{
			name:  "outline variant",
			props: core.ButtonProps{Label: "Cancel", Variant: core.VariantOutline},
		},
		{
			name:  "secondary variant",
			props: core.ButtonProps{Label: "Secondary", Variant: core.VariantSecondary},
		},
		{
			name:  "ghost variant",
			props: core.ButtonProps{Label: "Ghost", Variant: core.VariantGhost},
		},
		{
			name:  "link variant",
			props: core.ButtonProps{Label: "Link", Variant: core.VariantLink},
		},
		{
			name:  "destructive-ghost variant",
			props: core.ButtonProps{Label: "Remove", Variant: core.VariantDestructiveGhost},
		},
		{
			name:  "size sm",
			props: core.ButtonProps{Label: "Small", Size: core.SizeSm},
		},
		{
			name:  "size lg",
			props: core.ButtonProps{Label: "Large", Size: core.SizeLg},
		},
		{
			name:  "disabled button",
			props: core.ButtonProps{Label: "Disabled", Disabled: true},
		},
		{
			name:  "full width",
			props: core.ButtonProps{Label: "Full Width", FullWidth: true},
		},
		{
			name:  "submit type",
			props: core.ButtonProps{Label: "Submit", Type: "submit"},
		},
		{
			name:  "link href",
			props: core.ButtonProps{Label: "Go", Href: "/home"},
		},
		{
			name:  "disabled link",
			props: core.ButtonProps{Label: "Go", Href: "/home", Disabled: true},
		},
		{
			name:  "link with target",
			props: core.ButtonProps{Label: "External", Href: "https://example.com", Target: "_blank"},
		},
		{
			name:  "with id",
			props: core.ButtonProps{ID: "my-btn", Label: "Identified"},
		},
		{
			name:  "with extra attrs",
			props: core.ButtonProps{Label: "Extra", Extra: []g.Node{g.Attr("data-test", "value")}},
		},
		{
			name:  "with children",
			props: core.ButtonProps{Children: []g.Node{g.Text("Icon"), g.Text(" Label")}},
		},
		{
			name:  "zero value",
			props: core.ButtonProps{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, core.Button(tc.props))
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}
