package contactpartial

import (
	"strings"
	"testing"

	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web/render"
)

const (
	_inputClass    = "flex w-full rounded-md border border-input bg-transparent text-base shadow-xs transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50 md:text-sm h-9 min-w-0 px-3 py-1"
	_textareaClass = "flex w-full rounded-md border border-input bg-transparent text-base shadow-xs transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50 md:text-sm min-h-[60px] px-3 py-2"
	_submitClass   = "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] cursor-pointer bg-primary text-primary-foreground shadow-xs hover:bg-primary/90 h-9 px-4 py-2"
)

func TestContactForm_FormState(t *testing.T) {
	req := view.Request{}
	submitURL := "/contact"
	data := FormData{}

	got := render.RenderNode(t, ContactForm(req, submitURL, data))

	want := `<div id="contact-form">` +
		`<form id="contact-form-inner"` +
		` hx-post="/contact"` +
		` hx-target="#contact-form"` +
		` hx-swap="outerHTML"` +
		` hx-headers="{&#34;X-CSRF-Token&#34;: &#34;&#34;}">` +
		`<div class="grid gap-4">` +
		`<div class="grid gap-2">` +
		`<label class="text-sm font-medium leading-none inline-block" for="name">Name<span class="ml-0.5 text-destructive" aria-hidden="true">*</span></label>` +
		`<input class="` + _inputClass + `" type="text" id="name" name="name" required></div>` +
		`<div class="grid gap-2">` +
		`<label class="text-sm font-medium leading-none inline-block" for="email">Email<span class="ml-0.5 text-destructive" aria-hidden="true">*</span></label>` +
		`<input class="` + _inputClass + `" type="email" id="email" name="email" required></div>` +
		`<div class="grid gap-2">` +
		`<label class="text-sm font-medium leading-none inline-block" for="message">Message<span class="ml-0.5 text-destructive" aria-hidden="true">*</span></label>` +
		`<textarea class="` + _textareaClass + `" id="message" name="message" rows="5" required></textarea></div>` +
		`<div>` +
		`<button class="` + _submitClass + `" type="submit">Send message</button>` +
		`</div></div></form></div>`

	if got != want {
		t.Errorf("ContactForm form state mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestContactForm_FormState_WithSubmitURL(t *testing.T) {
	req := view.Request{}
	submitURL := "/api/contact/submit"
	data := FormData{}

	got := render.RenderNode(t, ContactForm(req, submitURL, data))

	if !strings.Contains(got, `hx-post="/api/contact/submit"`) {
		t.Errorf("ContactForm must use the provided submitURL in hx-post, got: %s", got)
	}
}

func TestContactForm_SuccessState(t *testing.T) {
	req := view.Request{}
	submitURL := "/contact"
	data := FormData{Sent: true}

	got := render.RenderNode(t, ContactForm(req, submitURL, data))

	// Must show the success alert and not the form.
	if !strings.Contains(got, "Message sent!") {
		t.Errorf("success state must contain 'Message sent!', got: %s", got)
	}
	if strings.Contains(got, `id="contact-form-inner"`) {
		t.Errorf("success state must not contain the form element")
	}
	if !strings.Contains(got, `id="contact-form"`) {
		t.Errorf("success state must contain the outer swap-target div")
	}
	// Check the description text (apostrophe must be HTML-encoded).
	if !strings.Contains(got, "Thanks for reaching out. We&#39;ll be in touch soon.") {
		t.Errorf("success state must contain encoded description text")
	}
}

func TestContactForm_SuccessState_ExactMatch(t *testing.T) {
	req := view.Request{}
	submitURL := "/contact"
	data := FormData{Sent: true}

	got := render.RenderNode(t, ContactForm(req, submitURL, data))

	want := `<div id="contact-form">` +
		`<div class="relative w-full rounded-lg border px-4 py-3 text-sm grid gap-1.5 items-start backdrop-blur-sm border-primary/30 bg-primary/20 text-primary [&amp;_[data-alert-description]]:text-muted-foreground"` +
		` role="alert" aria-live="polite">` +
		`<h5 class="line-clamp-1 min-h-4 font-medium tracking-tight">Message sent!</h5>` +
		`<div class="grid justify-items-start gap-1 text-sm" data-alert-description="">` +
		`Thanks for reaching out. We&#39;ll be in touch soon.</div>` +
		`</div></div>`

	if got != want {
		t.Errorf("ContactForm success state mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestContactForm_ValidationErrors(t *testing.T) {
	req := view.Request{}
	submitURL := "/contact"
	data := FormData{
		Name: "Alice",
		Errors: map[string][]string{
			"name": {"is required"},
		},
	}

	got := render.RenderNode(t, ContactForm(req, submitURL, data))

	// Must contain the error message for the name field.
	if !strings.Contains(got, `id="name-error"`) {
		t.Errorf("validation error must include name-error container, got: %s", got)
	}
	if !strings.Contains(got, `<p class="text-xs text-destructive">is required</p>`) {
		t.Errorf("validation error must include error message text, got: %s", got)
	}
	// Name input must have aria-invalid.
	if !strings.Contains(got, `aria-invalid="true"`) {
		t.Errorf("invalid field input must have aria-invalid=true")
	}
	// Must preserve the submitted value.
	if !strings.Contains(got, `value="Alice"`) {
		t.Errorf("form must preserve submitted name value 'Alice'")
	}
	// Must still render the form (not success state).
	if !strings.Contains(got, `id="contact-form-inner"`) {
		t.Errorf("validation error state must include the form element")
	}
}

func TestContactForm_ValidationErrors_MultipleFields(t *testing.T) {
	req := view.Request{}
	submitURL := "/contact"
	data := FormData{
		Errors: map[string][]string{
			"name":    {"is required"},
			"email":   {"must be a valid email address"},
			"message": {"is required"},
		},
	}

	got := render.RenderNode(t, ContactForm(req, submitURL, data))

	if !strings.Contains(got, `id="name-error"`) {
		t.Errorf("expected name-error container")
	}
	if !strings.Contains(got, `id="email-error"`) {
		t.Errorf("expected email-error container")
	}
	if !strings.Contains(got, `id="message-error"`) {
		t.Errorf("expected message-error container")
	}
}

func TestContactForm_PreservesFormValues(t *testing.T) {
	req := view.Request{}
	submitURL := "/contact"
	data := FormData{
		Name:    "Bob",
		Email:   "bob@example.com",
		Message: "My message",
		Errors:  map[string][]string{"name": {"is required"}},
	}

	got := render.RenderNode(t, ContactForm(req, submitURL, data))

	if !strings.Contains(got, `value="bob@example.com"`) {
		t.Errorf("form must preserve email value")
	}
	// Message textarea renders value as text content.
	if !strings.Contains(got, "My message") {
		t.Errorf("form must preserve message value")
	}
}

func TestContactForm_CSRFToken(t *testing.T) {
	req := view.Request{CSRFToken: "test-csrf-token-abc"}
	submitURL := "/contact"
	data := FormData{}

	got := render.RenderNode(t, ContactForm(req, submitURL, data))

	if !strings.Contains(got, "test-csrf-token-abc") {
		t.Errorf("ContactForm must include the CSRF token from the view request")
	}
}

func TestContactForm_HTMLEntities(t *testing.T) {
	req := view.Request{}
	submitURL := "/contact"
	data := FormData{
		Name: "O'Brien & Associates",
		Errors: map[string][]string{
			"name": {"is required"},
		},
	}

	got := render.RenderNode(t, ContactForm(req, submitURL, data))

	// Apostrophe and ampersand must be HTML-encoded in attribute values.
	if !strings.Contains(got, `value="O&#39;Brien &amp; Associates"`) {
		t.Errorf("HTML entities must be encoded in form field values, got: %s", got)
	}
}
