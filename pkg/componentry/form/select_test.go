package form_test

import (
	"strings"
	"testing"

	"github.com/go-sum/componentry/form"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestSelect_empty(t *testing.T) {
	got := testutil.RenderNode(t, form.Select(form.SelectProps{ID: "lang", Name: "lang"}))
	if !strings.HasPrefix(got, "<select") {
		t.Errorf("Select empty: expected <select> element, got:\n%s", got)
	}
	if !strings.Contains(got, `id="lang"`) {
		t.Errorf("Select empty: expected id=lang, got:\n%s", got)
	}
	if !strings.Contains(got, `name="lang"`) {
		t.Errorf("Select empty: expected name=lang, got:\n%s", got)
	}
	if strings.Contains(got, "<option") {
		t.Errorf("Select empty: expected no option elements, got:\n%s", got)
	}
}

func TestSelect_options(t *testing.T) {
	got := testutil.RenderNode(t, form.Select(form.SelectProps{
		ID:   "fruit",
		Name: "fruit",
		Options: []form.Option{
			{Value: "apple", Label: "Apple"},
			{Value: "banana", Label: "Banana"},
		},
	}))
	if !strings.Contains(got, `value="apple"`) {
		t.Errorf("Select options: expected value=apple, got:\n%s", got)
	}
	if !strings.Contains(got, "Apple") {
		t.Errorf("Select options: expected label 'Apple', got:\n%s", got)
	}
	if !strings.Contains(got, `value="banana"`) {
		t.Errorf("Select options: expected value=banana, got:\n%s", got)
	}
	if !strings.Contains(got, "Banana") {
		t.Errorf("Select options: expected label 'Banana', got:\n%s", got)
	}
}

func TestSelect_selected(t *testing.T) {
	got := testutil.RenderNode(t, form.Select(form.SelectProps{
		ID:       "size",
		Name:     "size",
		Selected: "medium",
		Options: []form.Option{
			{Value: "small", Label: "Small"},
			{Value: "medium", Label: "Medium"},
			{Value: "large", Label: "Large"},
		},
	}))
	// The selected option must carry the selected attribute
	if !strings.Contains(got, "selected") {
		t.Errorf("Select selected: expected 'selected' attribute, got:\n%s", got)
	}
	// Count occurrences to ensure only one is selected
	count := strings.Count(got, " selected")
	if count != 1 {
		t.Errorf("Select selected: expected exactly 1 selected attr, got %d in:\n%s", count, got)
	}
}

func TestSelect_selectedValues(t *testing.T) {
	got := testutil.RenderNode(t, form.Select(form.SelectProps{
		ID:             "tags",
		Name:           "tags",
		Multiple:       true,
		SelectedValues: []string{"go", "rust"},
		Options: []form.Option{
			{Value: "go", Label: "Go"},
			{Value: "rust", Label: "Rust"},
			{Value: "python", Label: "Python"},
		},
	}))
	count := strings.Count(got, " selected")
	if count != 2 {
		t.Errorf("Select selectedValues: expected 2 selected attrs, got %d in:\n%s", count, got)
	}
}

func TestSelect_optgroups(t *testing.T) {
	got := testutil.RenderNode(t, form.Select(form.SelectProps{
		ID:   "car",
		Name: "car",
		Groups: []form.OptGroup{
			{
				Label: "Swedish",
				Options: []form.Option{
					{Value: "volvo", Label: "Volvo"},
				},
			},
		},
	}))
	if !strings.Contains(got, "<optgroup") {
		t.Errorf("Select optgroups: expected <optgroup> element, got:\n%s", got)
	}
	if !strings.Contains(got, "Swedish") {
		t.Errorf("Select optgroups: expected group label 'Swedish', got:\n%s", got)
	}
	if !strings.Contains(got, "Volvo") {
		t.Errorf("Select optgroups: expected option 'Volvo', got:\n%s", got)
	}
}

func TestSelect_multiple(t *testing.T) {
	got := testutil.RenderNode(t, form.Select(form.SelectProps{
		ID:       "multi",
		Name:     "multi",
		Multiple: true,
	}))
	if !strings.Contains(got, "multiple") {
		t.Errorf("Select multiple: expected multiple attr, got:\n%s", got)
	}
}

func TestSelect_error(t *testing.T) {
	got := testutil.RenderNode(t, form.Select(form.SelectProps{
		ID:       "broken",
		Name:     "broken",
		HasError: true,
	}))
	if !strings.Contains(got, "border-destructive") {
		t.Errorf("Select error: expected border-destructive class, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-invalid="true"`) {
		t.Errorf("Select error: expected aria-invalid=true, got:\n%s", got)
	}
}

func TestSelect_disabled(t *testing.T) {
	got := testutil.RenderNode(t, form.Select(form.SelectProps{
		ID:       "dis",
		Name:     "dis",
		Disabled: true,
	}))
	if !strings.Contains(got, "disabled") {
		t.Errorf("Select disabled: expected disabled attr, got:\n%s", got)
	}
}
