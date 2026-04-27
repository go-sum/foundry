package layout

import (
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/interactive/theme"
	"github.com/go-sum/foundry/pkg/web/render"

	g "maragu.dev/gomponents"
)

func TestPage(t *testing.T) {
	props := Props{
		Title:     "Test Page",
		Nonce:     "abc123",
		CSRFToken: "tok-csrf",
		Flash:     []string{"saved"},
		Children:  []g.Node{g.Text("content")},
	}
	got := render.RenderNode(t, Page(props))

	themeScript := render.RenderNode(t, theme.InitScript())

	want := `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Test Page</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		`<meta name="csrf-token" content="tok-csrf">` +
		`<meta name="htmx-config" content="{&#34;includeIndicatorStyles&#34;:false,&#34;antiForgery&#34;:{&#34;headerName&#34;:&#34;X-CSRF-Token&#34;,&#34;parameterName&#34;:&#34;_csrf&#34;,&#34;token&#34;:&#34;tok-csrf&#34;}}">` +
		`<link rel="stylesheet" href="/css/app.css">` +
		themeScript + `</head>` +
		`<body class="bg-background text-foreground min-h-screen flex flex-col"><script src="/js/htmx.min.js" defer nonce="abc123"></script>` +
		`<div id="flash" class="container mx-auto px-4 pt-4 grid gap-2" hx-swap-oob="true" aria-live="polite"><div class="relative w-full rounded-lg border px-4 py-3 text-sm grid gap-1.5 items-start backdrop-blur-sm border-primary/30 bg-primary/20 text-primary [&amp;_[data-alert-description]]:text-muted-foreground" role="alert" aria-live="polite" data-dismissible=""><div class="grid justify-items-start gap-1 text-sm" data-alert-description="">saved</div><button data-dismiss="alert" class="absolute top-3 right-3 opacity-70 hover:opacity-100 transition-opacity outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50" type="button" aria-label="Dismiss">×</button></div></div>` +
		`<main class="container mx-auto px-4 py-6 flex-1">content</main>` +
		`<script src="/js/componentry.min.js" defer></script></body></html>`

	if got != want {
		t.Errorf("Page output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}
