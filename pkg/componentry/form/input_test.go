package form_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/form"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestInput(t *testing.T) {
	tests := []struct {
		name  string
		props form.InputProps
	}{
		{
			name:  "text type default",
			props: form.InputProps{ID: "name", Name: "name"},
		},
		{
			name:  "email type",
			props: form.InputProps{ID: "email", Name: "email", Type: form.TypeEmail},
		},
		{
			name:  "password type",
			props: form.InputProps{ID: "password", Name: "password", Type: form.TypePassword},
		},
		{
			name:  "with placeholder",
			props: form.InputProps{ID: "email", Name: "email", Type: form.TypeEmail, Placeholder: "you@example.com"},
		},
		{
			name:  "with value",
			props: form.InputProps{ID: "name", Name: "name", Value: "Jane Doe"},
		},
		{
			name:  "with error",
			props: form.InputProps{ID: "email", Name: "email", HasError: true},
		},
		{
			name:  "disabled",
			props: form.InputProps{ID: "name", Name: "name", Disabled: true},
		},
		{
			name:  "required",
			props: form.InputProps{ID: "name", Name: "name", Required: true},
		},
		{
			name:  "readonly",
			props: form.InputProps{ID: "name", Name: "name", Value: "Fixed", Readonly: true},
		},
		{
			name:  "with extra attrs",
			props: form.InputProps{ID: "name", Name: "name", Extra: []g.Node{g.Attr("data-lpignore", "true")}},
		},
		{
			name:  "number type",
			props: form.InputProps{ID: "qty", Name: "qty", Type: form.TypeNumber},
		},
		{
			name:  "search type",
			props: form.InputProps{ID: "q", Name: "q", Type: form.TypeSearch},
		},
		{
			name:  "zero value",
			props: form.InputProps{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, form.Input(tc.props))
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}
