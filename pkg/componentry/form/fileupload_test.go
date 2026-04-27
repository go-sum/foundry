package form_test

import (
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/form"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestFileUpload_default(t *testing.T) {
	got := testutil.RenderNode(t, form.FileUpload(form.FileUploadProps{ID: "file", Name: "attachment"}))
	if !strings.Contains(got, `data-file-upload`) {
		t.Errorf("FileUpload default: expected data-file-upload attr, got:\n%s", got)
	}
	// Default prompt rendered
	if !strings.Contains(got, "Drag") {
		t.Errorf("FileUpload default: expected default prompt, got:\n%s", got)
	}
	if !strings.Contains(got, `type="file"`) {
		t.Errorf("FileUpload default: expected hidden file input, got:\n%s", got)
	}
}

func TestFileUpload_customPrompt(t *testing.T) {
	got := testutil.RenderNode(t, form.FileUpload(form.FileUploadProps{
		ID:     "file",
		Name:   "doc",
		Prompt: "Upload your resume",
	}))
	if !strings.Contains(got, "Upload your resume") {
		t.Errorf("FileUpload customPrompt: expected custom prompt text, got:\n%s", got)
	}
	if strings.Contains(got, "Drag") {
		t.Errorf("FileUpload customPrompt: expected no default prompt when custom given, got:\n%s", got)
	}
}

func TestFileUpload_accept(t *testing.T) {
	got := testutil.RenderNode(t, form.FileUpload(form.FileUploadProps{
		ID:     "img",
		Name:   "image",
		Accept: "image/png,image/jpeg",
	}))
	if !strings.Contains(got, `accept="image/png,image/jpeg"`) {
		t.Errorf("FileUpload accept: expected accept attr, got:\n%s", got)
	}
}

func TestFileUpload_multiple(t *testing.T) {
	got := testutil.RenderNode(t, form.FileUpload(form.FileUploadProps{
		ID:       "files",
		Name:     "files",
		Multiple: true,
	}))
	if !strings.Contains(got, "multiple") {
		t.Errorf("FileUpload multiple: expected multiple attr, got:\n%s", got)
	}
}

func TestFileUpload_disabled(t *testing.T) {
	got := testutil.RenderNode(t, form.FileUpload(form.FileUploadProps{
		ID:       "dis",
		Name:     "dis",
		Disabled: true,
	}))
	if !strings.Contains(got, "disabled") {
		t.Errorf("FileUpload disabled: expected disabled attr, got:\n%s", got)
	}
}

func TestFileUpload_error(t *testing.T) {
	got := testutil.RenderNode(t, form.FileUpload(form.FileUploadProps{
		ID:       "err",
		Name:     "err",
		HasError: true,
	}))
	if !strings.Contains(got, "border-destructive") {
		t.Errorf("FileUpload error: expected border-destructive class, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-invalid="true"`) {
		t.Errorf("FileUpload error: expected aria-invalid=true on input, got:\n%s", got)
	}
}

func TestFileUpload_isLabel(t *testing.T) {
	got := testutil.RenderNode(t, form.FileUpload(form.FileUploadProps{ID: "f", Name: "f"}))
	if !strings.HasPrefix(got, "<label") {
		t.Errorf("FileUpload: expected <label> root element, got:\n%s", got)
	}
}

func TestFileUpload_ariaLiveSpan(t *testing.T) {
	got := testutil.RenderNode(t, form.FileUpload(form.FileUploadProps{ID: "f", Name: "f"}))
	if !strings.Contains(got, `aria-live="polite"`) {
		t.Errorf("FileUpload: expected aria-live=polite file name span, got:\n%s", got)
	}
	if !strings.Contains(got, `data-file-name`) {
		t.Errorf("FileUpload: expected data-file-name span, got:\n%s", got)
	}
}
