package page

import (
	"testing"

	"github.com/go-sum/componentry/interactive/theme"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web/render"
)

func TestHomePage(t *testing.T) {
	req := view.Request{}
	got := render.RenderNode(t, HomePage(req, "/hello/World"))

	themeScript := render.RenderNode(t, theme.InitScript())
	selectorScript := render.RenderNode(t, theme.ThemeScript())

	const btnClass = "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] cursor-pointer text-foreground underline-offset-4 hover:underline h-9 px-4 py-2"

	want := `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Home</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		`<meta name="csrf-token" content="">` +
		`<meta name="htmx-config" content="{&#34;includeIndicatorStyles&#34;:false,&#34;antiForgery&#34;:{&#34;headerName&#34;:&#34;X-CSRF-Token&#34;,&#34;parameterName&#34;:&#34;_csrf&#34;,&#34;token&#34;:&#34;&#34;}}">` +
		`<link rel="stylesheet" href="/css/app.css">` +
		themeScript + `</head>` +
		`<body><script src="/js/htmx.min.js" defer nonce=""></script>` +
		`<div id="flash" hx-swap-oob="true" aria-live="polite"></div>` +
		`<div class="min-h-screen bg-background">` +
		`<div class="max-w-2xl mx-auto py-16 px-4">` +
		`<h1 class="text-2xl font-bold text-foreground mb-4">Welcome to Foundry</h1>` +
		`<p class="text-muted-foreground mb-8">A Go web application built on W3C Web API primitives.</p>` +
		`<a class="` + btnClass + `" href="/hello/World">Say hello to World</a>` +
		`</div></div>` +
		selectorScript + `</body></html>`

	if got != want {
		t.Errorf("HomePage output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHomeContent(t *testing.T) {
	req := view.Request{}
	got := render.RenderNode(t, HomeContent(req, "/hello/World"))

	const btnClass = "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] cursor-pointer text-foreground underline-offset-4 hover:underline h-9 px-4 py-2"

	want := `<div class="max-w-2xl mx-auto py-16 px-4">` +
		`<h1 class="text-2xl font-bold text-foreground mb-4">Welcome to Foundry</h1>` +
		`<p class="text-muted-foreground mb-8">A Go web application built on W3C Web API primitives.</p>` +
		`<a class="` + btnClass + `" href="/hello/World">Say hello to World</a>` +
		`</div>`

	if got != want {
		t.Errorf("HomeContent output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}
