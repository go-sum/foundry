package form_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/form"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestTextarea(t *testing.T) {
	tests := []struct {
		name  string
		props form.TextareaProps
	}{
		{
			name:  "basic textarea",
			props: form.TextareaProps{ID: "message", Name: "message"},
		},
		{
			name:  "with placeholder",
			props: form.TextareaProps{ID: "message", Name: "message", Placeholder: "Type your message…"},
		},
		{
			name:  "with value",
			props: form.TextareaProps{ID: "message", Name: "message", Value: "Hello world"},
		},
		{
			name:  "with rows",
			props: form.TextareaProps{ID: "message", Name: "message", Rows: 6},
		},
		{
			name:  "with error",
			props: form.TextareaProps{ID: "message", Name: "message", HasError: true},
		},
		{
			name:  "disabled",
			props: form.TextareaProps{ID: "message", Name: "message", Disabled: true},
		},
		{
			name:  "required",
			props: form.TextareaProps{ID: "message", Name: "message", Required: true},
		},
		{
			name:  "with extra attrs",
			props: form.TextareaProps{ID: "message", Name: "message", Extra: []g.Node{g.Attr("data-autoresize", "true")}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, form.Textarea(tc.props))
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}
