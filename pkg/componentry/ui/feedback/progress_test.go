package feedback_test

import (
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/ui/feedback"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestProgress_default(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{
		Value: 50,
		Max:   100,
	}))
	if !strings.Contains(got, "<progress") {
		t.Errorf("Progress default: expected <progress> element, got:\n%s", got)
	}
	if !strings.Contains(got, `value="50"`) {
		t.Errorf("Progress default: expected value=50, got:\n%s", got)
	}
	if !strings.Contains(got, `max="100"`) {
		t.Errorf("Progress default: expected max=100, got:\n%s", got)
	}
}

func TestProgress_withLabel(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{
		Value: 30,
		Label: "Upload Progress",
	}))
	if !strings.Contains(got, "Upload Progress") {
		t.Errorf("Progress withLabel: expected label text, got:\n%s", got)
	}
}

func TestProgress_showValue(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{
		Value:     75,
		Max:       100,
		ShowValue: true,
	}))
	if !strings.Contains(got, "75%") {
		t.Errorf("Progress showValue: expected '75%%' in output, got:\n%s", got)
	}
}

func TestProgress_successVariant(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{
		Variant: feedback.ProgressSuccess,
		Value:   80,
	}))
	if !strings.Contains(got, "progress-success") {
		t.Errorf("Progress success: expected progress-success class, got:\n%s", got)
	}
}

func TestProgress_dangerVariant(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{
		Variant: feedback.ProgressDanger,
		Value:   20,
	}))
	if !strings.Contains(got, "progress-danger") {
		t.Errorf("Progress danger: expected progress-danger class, got:\n%s", got)
	}
}

func TestProgress_warningVariant(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{
		Variant: feedback.ProgressWarning,
		Value:   60,
	}))
	if !strings.Contains(got, "progress-warning") {
		t.Errorf("Progress warning: expected progress-warning class, got:\n%s", got)
	}
}

func TestProgress_smSize(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{
		Size:  feedback.ProgressSm,
		Value: 50,
	}))
	if !strings.Contains(got, "h-1") {
		t.Errorf("Progress sm: expected h-1 class, got:\n%s", got)
	}
}

func TestProgress_lgSize(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{
		Size:  feedback.ProgressLg,
		Value: 50,
	}))
	if !strings.Contains(got, "h-4") {
		t.Errorf("Progress lg: expected h-4 class, got:\n%s", got)
	}
}

func TestProgress_defaultSize(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{Value: 50}))
	if !strings.Contains(got, "h-2.5") {
		t.Errorf("Progress default size: expected h-2.5 class, got:\n%s", got)
	}
}

func TestProgress_withID(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{
		ID:    "upload",
		Value: 50,
	}))
	if !strings.Contains(got, `id="upload"`) {
		t.Errorf("Progress withID: expected id=upload, got:\n%s", got)
	}
}

func TestProgress_defaultMax(t *testing.T) {
	// max=0 defaults to 100
	got := testutil.RenderNode(t, feedback.Progress(feedback.ProgressProps{Value: 50, Max: 0}))
	if !strings.Contains(got, `max="100"`) {
		t.Errorf("Progress defaultMax: expected max=100, got:\n%s", got)
	}
}
