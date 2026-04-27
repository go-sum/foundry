package form_test

import (
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/form"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestCheckbox(t *testing.T) {
	tests := []struct {
		name        string
		props       form.CheckboxProps
		wantAttrs   []string
		absentAttrs []string
	}{
		{
			name:        "unchecked",
			props:       form.CheckboxProps{ID: "remember", Name: "remember"},
			wantAttrs:   []string{`type="checkbox"`, `id="remember"`, `name="remember"`},
			absentAttrs: []string{" checked", " disabled", " required"},
		},
		{
			name:      "checked",
			props:     form.CheckboxProps{ID: "accept", Name: "accept", Checked: true},
			wantAttrs: []string{"checked"},
		},
		{
			name:      "disabled",
			props:     form.CheckboxProps{ID: "off", Name: "off", Disabled: true},
			wantAttrs: []string{"disabled"},
		},
		{
			name:      "required",
			props:     form.CheckboxProps{ID: "terms", Name: "terms", Required: true},
			wantAttrs: []string{`required`},
		},
		{
			name:      "with value",
			props:     form.CheckboxProps{ID: "opt", Name: "option", Value: "yes"},
			wantAttrs: []string{`value="yes"`},
		},
		{
			name:        "disabled checked",
			props:       form.CheckboxProps{ID: "dc", Name: "dc", Disabled: true, Checked: true},
			wantAttrs:   []string{"checked", "disabled"},
			absentAttrs: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, form.Checkbox(tc.props))
			// Composite structure: peer input sr-only
			if !strings.Contains(got, "sr-only peer") {
				t.Errorf("%s: expected sr-only peer on input, got:\n%s", tc.name, got)
			}
			if !strings.Contains(got, `type="checkbox"`) {
				t.Errorf("%s: expected type=checkbox, got:\n%s", tc.name, got)
			}
			for _, want := range tc.wantAttrs {
				if !strings.Contains(got, want) {
					t.Errorf("%s: expected %q in output, got:\n%s", tc.name, want, got)
				}
			}
			for _, absent := range tc.absentAttrs {
				if strings.Contains(got, absent) {
					t.Errorf("%s: expected %q to be absent, got:\n%s", tc.name, absent, got)
				}
			}
		})
	}
}

func TestCheckbox_visualStructure(t *testing.T) {
	got := testutil.RenderNode(t, form.Checkbox(form.CheckboxProps{ID: "ch"}))
	// outer span
	if !strings.HasPrefix(got, "<span") {
		t.Errorf("Checkbox: expected outer <span>, got:\n%s", got)
	}
	// peer-checked visual box span
	if !strings.Contains(got, "peer-checked:border-primary") {
		t.Errorf("Checkbox: expected peer-checked:border-primary visual span, got:\n%s", got)
	}
	// SVG checkmark
	if !strings.Contains(got, "<svg") {
		t.Errorf("Checkbox: expected SVG checkmark element, got:\n%s", got)
	}
}

func TestRadio(t *testing.T) {
	tests := []struct {
		name        string
		props       form.RadioProps
		wantAttrs   []string
		absentAttrs []string
	}{
		{
			name:        "unchecked",
			props:       form.RadioProps{ID: "opt1", Name: "color", Value: "red"},
			wantAttrs:   []string{`type="radio"`, `id="opt1"`, `name="color"`, `value="red"`},
			absentAttrs: []string{" checked", " disabled"},
		},
		{
			name:      "checked",
			props:     form.RadioProps{ID: "opt2", Name: "color", Value: "blue", Checked: true},
			wantAttrs: []string{"checked"},
		},
		{
			name:      "disabled",
			props:     form.RadioProps{ID: "opt3", Name: "color", Value: "green", Disabled: true},
			wantAttrs: []string{"disabled"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, form.Radio(tc.props))
			if !strings.Contains(got, "sr-only peer") {
				t.Errorf("%s: expected sr-only peer on input, got:\n%s", tc.name, got)
			}
			if !strings.Contains(got, `type="radio"`) {
				t.Errorf("%s: expected type=radio, got:\n%s", tc.name, got)
			}
			for _, want := range tc.wantAttrs {
				if !strings.Contains(got, want) {
					t.Errorf("%s: expected %q in output, got:\n%s", tc.name, want, got)
				}
			}
			for _, absent := range tc.absentAttrs {
				if strings.Contains(got, absent) {
					t.Errorf("%s: expected %q to be absent, got:\n%s", tc.name, absent, got)
				}
			}
		})
	}
}

func TestRadio_visualStructure(t *testing.T) {
	got := testutil.RenderNode(t, form.Radio(form.RadioProps{ID: "r1"}))
	// outer span
	if !strings.HasPrefix(got, "<span") {
		t.Errorf("Radio: expected outer <span>, got:\n%s", got)
	}
	// ring span
	if !strings.Contains(got, "rounded-full") {
		t.Errorf("Radio: expected rounded-full class in ring span, got:\n%s", got)
	}
	// dot span
	if !strings.Contains(got, "peer-checked:bg-primary") {
		t.Errorf("Radio: expected peer-checked:bg-primary dot span, got:\n%s", got)
	}
}
