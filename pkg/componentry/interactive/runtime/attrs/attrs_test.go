package attrs_test

import (
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/interactive/runtime/attrs"
)

func TestController(t *testing.T) {
	tests := []struct {
		names []string
		want  string
	}{
		{names: []string{"tabs"}, want: "tabs"},
		{names: []string{"tabs", "hotkeys"}, want: "tabs hotkeys"},
		{names: []string{"a", "b", "c"}, want: "a b c"},
	}
	for _, tt := range tests {
		a := attrs.Controller(tt.names...)
		if got := a["data-controller"]; got != tt.want {
			t.Errorf("Controller(%v)[data-controller] = %q, want %q", tt.names, got, tt.want)
		}
		if len(a.Nodes()) != 1 {
			t.Errorf("Controller(%v).Nodes() len = %d, want 1", tt.names, len(a.Nodes()))
		}
	}
}

func TestAction(t *testing.T) {
	tests := []struct {
		event, ctrl, method string
		want                string
	}{
		{"click", "dialog", "open", "click->dialog#open"},
		{"click", "theme", "cycle", "click->theme#cycle"},
		{"submit", "form", "validate", "submit->form#validate"},
	}
	for _, tt := range tests {
		a := attrs.Action(tt.event, tt.ctrl, tt.method)
		if got := a["data-action"]; got != tt.want {
			t.Errorf("Action(%q,%q,%q)[data-action] = %q, want %q", tt.event, tt.ctrl, tt.method, got, tt.want)
		}
		if len(a.Nodes()) != 1 {
			t.Errorf("Action(%q,%q,%q).Nodes() len = %d, want 1", tt.event, tt.ctrl, tt.method, len(a.Nodes()))
		}
	}
}

func TestTarget(t *testing.T) {
	tests := []struct {
		ctrl, name string
		wantKey    string
		wantVal    string
	}{
		{"tabs", "panel", "data-tabs-target", "panel"},
		{"modal", "backdrop", "data-modal-target", "backdrop"},
		{"dropdown", "menu", "data-dropdown-target", "menu"},
	}
	for _, tt := range tests {
		a := attrs.Target(tt.ctrl, tt.name)
		if got := a[tt.wantKey]; got != tt.wantVal {
			t.Errorf("Target(%q,%q)[%q] = %q, want %q", tt.ctrl, tt.name, tt.wantKey, got, tt.wantVal)
		}
		if len(a.Nodes()) != 1 {
			t.Errorf("Target(%q,%q).Nodes() len = %d, want 1", tt.ctrl, tt.name, len(a.Nodes()))
		}
	}
}

func TestCompose_concatenates_controller_and_action(t *testing.T) {
	result := attrs.Compose(
		attrs.Controller("tabs"),
		attrs.Controller("hotkeys"),
	)
	if result["data-controller"] != "tabs hotkeys" {
		t.Errorf("Compose controller = %q, want %q", result["data-controller"], "tabs hotkeys")
	}
}

func TestCompose_concatenates_actions(t *testing.T) {
	result := attrs.Compose(
		attrs.Action("click", "tabs", "select"),
		attrs.Action("keydown", "tabs", "navigate"),
	)
	want := "click->tabs#select keydown->tabs#navigate"
	if result["data-action"] != want {
		t.Errorf("Compose action = %q, want %q", result["data-action"], want)
	}
}

func TestCompose_last_write_wins_for_other_keys(t *testing.T) {
	result := attrs.Compose(
		attrs.Attrs{"id": "first"},
		attrs.Attrs{"id": "second"},
	)
	if result["id"] != "second" {
		t.Errorf("Compose id = %q, want %q", result["id"], "second")
	}
}

func TestCompose_mixed(t *testing.T) {
	result := attrs.Compose(
		attrs.Controller("dialog"),
		attrs.Action("click", "dialog", "open"),
		attrs.Attrs{"aria-haspopup": "dialog"},
	)
	if result["data-controller"] != "dialog" {
		t.Errorf("data-controller = %q", result["data-controller"])
	}
	if result["data-action"] != "click->dialog#open" {
		t.Errorf("data-action = %q", result["data-action"])
	}
	if result["aria-haspopup"] != "dialog" {
		t.Errorf("aria-haspopup = %q", result["aria-haspopup"])
	}
}

