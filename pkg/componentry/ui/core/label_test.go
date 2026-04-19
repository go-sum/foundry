package core_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/ui/core"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestLabel(t *testing.T) {
	tests := []struct {
		name  string
		props core.LabelProps
		text  string
	}{
		{
			name:  "basic label",
			props: core.LabelProps{For: "email"},
			text:  "Email",
		},
		{
			name:  "required label",
			props: core.LabelProps{For: "name", Required: true},
			text:  "Name",
		},
		{
			name:  "error label",
			props: core.LabelProps{For: "email", Error: "Email is required"},
			text:  "Email",
		},
		{
			name:  "required with error",
			props: core.LabelProps{For: "email", Required: true, Error: "Invalid email"},
			text:  "Email",
		},
		{
			name:  "no for attr",
			props: core.LabelProps{},
			text:  "Standalone",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, core.Label(tc.props, g.Text(tc.text)))
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}
