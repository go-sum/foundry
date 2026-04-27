package form_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/form"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestField(t *testing.T) {
	tests := []struct {
		name  string
		props form.FieldProps
	}{
		{
			name: "basic field",
			props: form.FieldProps{
				ID:      "email",
				Label:   "Email",
				Control: g.Text("<input>"),
			},
		},
		{
			name: "with description",
			props: form.FieldProps{
				ID:          "email",
				Label:       "Email",
				Description: "We'll never share your email.",
				Control:     g.Text("<input>"),
			},
		},
		{
			name: "with hint",
			props: form.FieldProps{
				ID:      "password",
				Label:   "Password",
				Hint:    "At least 8 characters.",
				Control: g.Text("<input>"),
			},
		},
		{
			name: "with errors",
			props: form.FieldProps{
				ID:      "email",
				Label:   "Email",
				Errors:  []string{"Email is required", "Must be a valid email"},
				Control: g.Text("<input>"),
			},
		},
		{
			name: "with all assistive text",
			props: form.FieldProps{
				ID:          "email",
				Label:       "Email",
				Description: "Your primary email.",
				Hint:        "Use your work email.",
				Errors:      []string{"Invalid format"},
				Control:     g.Text("<input>"),
			},
		},
		{
			name: "required field",
			props: form.FieldProps{
				ID:       "name",
				Label:    "Name",
				Required: true,
				Control:  g.Text("<input>"),
			},
		},
		{
			name:  "zero value",
			props: form.FieldProps{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, form.Field(tc.props))
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}

func TestFieldControlAttrs(t *testing.T) {
	tests := []struct {
		name        string
		controlID   string
		description string
		hint        string
		errors      []string
		wantAttrs   []string
		wantAbsent  []string
	}{
		{
			name:      "no assistive text",
			controlID: "email",
			wantAttrs: nil,
		},
		{
			name:        "description only",
			controlID:   "email",
			description: "desc",
			wantAttrs:   []string{`aria-describedby="email-description"`},
			wantAbsent:  []string{"aria-errormessage", "aria-invalid"},
		},
		{
			name:      "hint only",
			controlID: "email",
			hint:      "hint",
			wantAttrs: []string{`aria-describedby="email-hint"`},
			wantAbsent: []string{"aria-errormessage", "aria-invalid"},
		},
		{
			name:      "errors only",
			controlID: "email",
			errors:    []string{"Required"},
			wantAttrs: []string{
				`aria-describedby="email-error"`,
				`aria-errormessage="email-error"`,
				`aria-invalid="true"`,
			},
		},
		{
			name:        "description and hint",
			controlID:   "email",
			description: "desc",
			hint:        "hint",
			wantAttrs:   []string{`aria-describedby="email-description email-hint"`},
			wantAbsent:  []string{"aria-errormessage", "aria-invalid"},
		},
		{
			name:        "description and errors",
			controlID:   "email",
			description: "desc",
			errors:      []string{"Required"},
			wantAttrs: []string{
				`aria-describedby="email-description email-error"`,
				`aria-errormessage="email-error"`,
				`aria-invalid="true"`,
			},
		},
		{
			name:      "hint and errors",
			controlID: "email",
			hint:      "hint",
			errors:    []string{"Required"},
			wantAttrs: []string{
				`aria-describedby="email-hint email-error"`,
				`aria-errormessage="email-error"`,
				`aria-invalid="true"`,
			},
		},
		{
			name:        "all present",
			controlID:   "email",
			description: "desc",
			hint:        "hint",
			errors:      []string{"Required"},
			wantAttrs: []string{
				`aria-describedby="email-description email-hint email-error"`,
				`aria-errormessage="email-error"`,
				`aria-invalid="true"`,
			},
		},
		{
			name:        "empty controlID",
			controlID:   "",
			description: "desc",
			wantAttrs:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := form.FieldControlAttrs(tc.controlID, tc.description, tc.hint, tc.errors)

			if tc.wantAttrs == nil {
				if len(attrs) != 0 {
					t.Errorf("expected nil attrs, got %d nodes", len(attrs))
				}
				return
			}

			// Render the attrs into a wrapper div so we can inspect them
			rendered := testutil.RenderNode(t, g.El("div", attrs...))
			for _, want := range tc.wantAttrs {
				if !containsStr(rendered, want) {
					t.Errorf("expected %q in rendered output:\n%s", want, rendered)
				}
			}
			for _, absent := range tc.wantAbsent {
				if containsStr(rendered, absent) {
					t.Errorf("expected %q to be absent from rendered output:\n%s", absent, rendered)
				}
			}
		})
	}
}

func containsStr(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
