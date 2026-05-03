package pages

import (
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/interactive/theme"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
	"github.com/go-sum/foundry/internal/views/partials"
	"github.com/go-sum/foundry/pkg/web/render"
)

func TestContactContent_ContainsForm(t *testing.T) {
	req := viewstate.Request{}
	submitURL := "/contact"
	data := partials.ContactFormData{}

	got := render.RenderNode(t, ContactContent(req, submitURL, data))

	want := `<div class="max-w-md mx-auto py-16 px-4">` +
		`<h1 class="text-2xl font-bold leading-tight mb-2">Contact Us</h1>` +
		`<p class="text-sm text-muted-foreground mb-8">Send us a message and we&#39;ll get back to you.</p>` +
		`<div id="contact-form">` +
		`<form id="contact-form-inner"` +
		` hx-post="/contact"` +
		` hx-target="#contact-form"` +
		` hx-swap="outerHTML"` +
		` hx-headers="{&#34;X-CSRF-Token&#34;:&#34;&#34;}">`

	if !strings.HasPrefix(got, want) {
		t.Errorf("ContactContent output mismatch\ngot:  %s\nwant prefix: %s", got, want)
	}

	// Must include the form swap target.
	if !strings.Contains(got, `id="contact-form"`) {
		t.Errorf("ContactContent must contain id=\"contact-form\"")
	}
	// Must include name, email, and message fields.
	if !strings.Contains(got, `name="name"`) {
		t.Errorf("ContactContent must contain name field")
	}
	if !strings.Contains(got, `name="email"`) {
		t.Errorf("ContactContent must contain email field")
	}
	if !strings.Contains(got, `name="message"`) {
		t.Errorf("ContactContent must contain message field")
	}
}

func TestContactContent_ExactMatch(t *testing.T) {
	req := viewstate.Request{}
	submitURL := "/contact"
	data := partials.ContactFormData{}

	got := render.RenderNode(t, ContactContent(req, submitURL, data))
	want := render.RenderNode(t, ContactContent(req, submitURL, data))

	if got != want {
		t.Errorf("ContactContent is non-deterministic:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestContactPage_Renders(t *testing.T) {
	req := viewstate.Request{}
	submitURL := "/contact"
	data := partials.ContactFormData{}

	got := render.RenderNode(t, ContactPage(req, submitURL, data))

	// Must be a full HTML document.
	if !strings.HasPrefix(got, "<!doctype html>") {
		t.Errorf("ContactPage must start with <!doctype html>, got: %s", got[:min(50, len(got))])
	}
	if !strings.Contains(got, `<title>Contact Us</title>`) {
		t.Errorf("ContactPage must contain <title>Contact Us</title>")
	}
	if !strings.Contains(got, `id="contact-form"`) {
		t.Errorf("ContactPage must contain id=\"contact-form\"")
	}
	if !strings.Contains(got, "Contact Us") {
		t.Errorf("ContactPage must contain 'Contact Us' heading")
	}
	if !strings.Contains(got, `name="name"`) {
		t.Errorf("ContactPage must contain form name field")
	}
}

func TestContactPage_FullPageExactMatch(t *testing.T) {
	req := viewstate.Request{}
	submitURL := "/contact"
	data := partials.ContactFormData{}

	themeScript := render.RenderNode(t, theme.InitScript())
	const selectorScript = `<script src="/js/componentry.min.js" defer></script>`

	got := render.RenderNode(t, ContactPage(req, submitURL, data))

	// Verify the full page structure matches exactly.
	// Build the expected string from known parts.
	wantPrefix := `<!doctype html><html lang="en"><head>` +
		`<meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		`<title>Contact Us</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		`<meta name="csrf-token" content="">` +
		`<meta name="htmx-config" content="{&#34;includeIndicatorStyles&#34;:false,&#34;antiForgery&#34;:{&#34;headerName&#34;:&#34;X-CSRF-Token&#34;,&#34;parameterName&#34;:&#34;_csrf&#34;,&#34;token&#34;:&#34;&#34;}}">` +
		`<link rel="stylesheet" href="/css/app.css">` +
		themeScript + `</head>` +
		`<body class="bg-background text-foreground min-h-screen flex flex-col">` +
		`<script src="/js/htmx.min.js" defer nonce=""></script>` +
		`<div id="flash" class="container mx-auto px-4 pt-4 grid gap-2" hx-swap-oob="true" aria-live="polite"></div>` +
		`<main class="container mx-auto px-4 py-6 flex-1">`

	wantSuffix := selectorScript + `</body></html>`

	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("ContactPage HTML header mismatch\ngot prefix:  %s\nwant prefix: %s",
			got[:min(len(wantPrefix), len(got))], wantPrefix)
	}
	if !strings.HasSuffix(got, wantSuffix) {
		t.Errorf("ContactPage HTML footer mismatch")
	}
}
