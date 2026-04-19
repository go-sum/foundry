package form_test

import (
	"strings"
	"testing"

	"github.com/go-sum/componentry/form"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestSwitch(t *testing.T) {
	tests := []struct {
		name        string
		props       form.SwitchProps
		wantAttrs   []string
		absentAttrs []string
	}{
		{
			name:        "unchecked",
			props:       form.SwitchProps{ID: "notifications", Name: "notifications"},
			wantAttrs:   []string{`type="checkbox"`, `role="switch"`, `id="notifications"`, `name="notifications"`},
			absentAttrs: []string{" checked", " disabled"},
		},
		{
			name:      "checked",
			props:     form.SwitchProps{ID: "sw", Name: "sw", Checked: true},
			wantAttrs: []string{"checked", `role="switch"`},
		},
		{
			name:      "disabled",
			props:     form.SwitchProps{ID: "sw2", Name: "sw2", Disabled: true},
			wantAttrs: []string{"disabled", `role="switch"`},
		},
		{
			name:      "with value",
			props:     form.SwitchProps{ID: "sw3", Name: "sw3", Value: "on"},
			wantAttrs: []string{`value="on"`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, form.Switch(tc.props))
			// sr-only peer hidden input
			if !strings.Contains(got, "sr-only peer") {
				t.Errorf("%s: expected sr-only peer on input, got:\n%s", tc.name, got)
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

func TestSwitch_visualStructure(t *testing.T) {
	got := testutil.RenderNode(t, form.Switch(form.SwitchProps{ID: "sw"}))
	// outer span
	if !strings.HasPrefix(got, "<span") {
		t.Errorf("Switch: expected outer <span>, got:\n%s", got)
	}
	// track span
	if !strings.Contains(got, "peer-checked:bg-primary") {
		t.Errorf("Switch: expected track with peer-checked:bg-primary, got:\n%s", got)
	}
	// thumb span
	if !strings.Contains(got, "peer-checked:translate-x-4") {
		t.Errorf("Switch: expected thumb with peer-checked:translate-x-4, got:\n%s", got)
	}
}
