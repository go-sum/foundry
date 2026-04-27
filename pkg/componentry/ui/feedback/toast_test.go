package feedback_test

import (
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/ui/feedback"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestToast_default(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{}))
	if !strings.Contains(got, `role="status"`) {
		t.Errorf("Toast default: expected role=status, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-live="polite"`) {
		t.Errorf("Toast default: expected aria-live=polite, got:\n%s", got)
	}
}

func TestToast_error(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{Variant: feedback.ToastError}))
	if !strings.Contains(got, `role="alert"`) {
		t.Errorf("Toast error: expected role=alert, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-live="assertive"`) {
		t.Errorf("Toast error: expected aria-live=assertive, got:\n%s", got)
	}
}

func TestToast_warning(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{Variant: feedback.ToastWarning}))
	if !strings.Contains(got, `role="alert"`) {
		t.Errorf("Toast warning: expected role=alert, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-live="assertive"`) {
		t.Errorf("Toast warning: expected aria-live=assertive, got:\n%s", got)
	}
}

func TestToast_success(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{Variant: feedback.ToastSuccess}))
	if !strings.Contains(got, `role="status"`) {
		t.Errorf("Toast success: expected role=status, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-live="polite"`) {
		t.Errorf("Toast success: expected aria-live=polite, got:\n%s", got)
	}
	if !strings.Contains(got, "border-success") {
		t.Errorf("Toast success: expected success styling, got:\n%s", got)
	}
}

func TestToast_info(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{Variant: feedback.ToastInfo}))
	if !strings.Contains(got, `role="status"`) {
		t.Errorf("Toast info: expected role=status, got:\n%s", got)
	}
	if !strings.Contains(got, "border-primary") {
		t.Errorf("Toast info: expected primary styling, got:\n%s", got)
	}
}

func TestToast_withTitle(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{Title: "Success!"}))
	if !strings.Contains(got, "Success!") {
		t.Errorf("Toast withTitle: expected title 'Success!', got:\n%s", got)
	}
	if !strings.Contains(got, "font-medium") {
		t.Errorf("Toast withTitle: expected font-medium class on title, got:\n%s", got)
	}
}

func TestToast_withDescription(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{Description: "Your changes have been saved."}))
	if !strings.Contains(got, "Your changes have been saved.") {
		t.Errorf("Toast withDescription: expected description text, got:\n%s", got)
	}
}

func TestToast_dismissible(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{Dismissible: true}))
	if !strings.Contains(got, "data-dismissible") {
		t.Errorf("Toast dismissible: expected data-dismissible attr, got:\n%s", got)
	}
	if !strings.Contains(got, "data-dismiss") {
		t.Errorf("Toast dismissible: expected dismiss button, got:\n%s", got)
	}
	if !strings.Contains(got, "<button") {
		t.Errorf("Toast dismissible: expected <button> element, got:\n%s", got)
	}
}

func TestToast_notDismissible(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{}))
	if strings.Contains(got, "data-dismissible") {
		t.Errorf("Toast notDismissible: expected no data-dismissible, got:\n%s", got)
	}
	if strings.Contains(got, "<button") {
		t.Errorf("Toast notDismissible: expected no dismiss button, got:\n%s", got)
	}
}

func TestToast_fixed(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{Position: feedback.PositionTopRight}))
	if !strings.Contains(got, "fixed") {
		t.Errorf("Toast fixed: expected fixed position class, got:\n%s", got)
	}
	if !strings.Contains(got, "top-4") {
		t.Errorf("Toast fixed top-right: expected top-4, got:\n%s", got)
	}
	if !strings.Contains(got, "right-4") {
		t.Errorf("Toast fixed top-right: expected right-4, got:\n%s", got)
	}
}

func TestToast_inline(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{}))
	if strings.Contains(got, "fixed") {
		t.Errorf("Toast inline: expected no fixed class when Position empty, got:\n%s", got)
	}
}

func TestToast_withID(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{ID: "notification-1"}))
	if !strings.Contains(got, `id="notification-1"`) {
		t.Errorf("Toast withID: expected id=notification-1, got:\n%s", got)
	}
}

func TestToast_bottomLeft(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Toast(feedback.ToastProps{Position: feedback.PositionBottomLeft}))
	if !strings.Contains(got, "bottom-4") {
		t.Errorf("Toast bottom-left: expected bottom-4, got:\n%s", got)
	}
	if !strings.Contains(got, "left-4") {
		t.Errorf("Toast bottom-left: expected left-4, got:\n%s", got)
	}
}
