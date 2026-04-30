package page

import (
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/interactive/theme"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
	"github.com/go-sum/foundry/pkg/web/render"
)

func TestHomePage(t *testing.T) {
	req := viewstate.Request{}
	got := render.RenderNode(t, HomePage(req, nil))

	themeScript := render.RenderNode(t, theme.InitScript())
	const selectorScript = `<script src="/js/componentry.min.js" defer></script>`

	want := `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Home</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		`<meta name="csrf-token" content="">` +
		`<meta name="htmx-config" content="{&#34;includeIndicatorStyles&#34;:false,&#34;antiForgery&#34;:{&#34;headerName&#34;:&#34;X-CSRF-Token&#34;,&#34;parameterName&#34;:&#34;_csrf&#34;,&#34;token&#34;:&#34;&#34;}}">` +
		`<link rel="stylesheet" href="/css/app.css">` +
		themeScript + `</head>` +
		`<body class="bg-background text-foreground min-h-screen flex flex-col"><script src="/js/htmx.min.js" defer nonce=""></script>` +
		`<div id="flash" class="container mx-auto px-4 pt-4 grid gap-2" hx-swap-oob="true" aria-live="polite"></div>` +
		`<main class="container mx-auto px-4 py-6 flex-1">` +
		`<div class="mx-auto flex max-w-3xl flex-col items-center justify-center gap-8 py-24 text-center">` +
		`<div class="space-y-4">` +
		`<p class="text-sm font-medium uppercase tracking-[0.2em] text-muted-foreground">Go Web Starter</p>` +
		`<h1 class="text-2xl font-bold">Welcome to Foundry</h1>` +
		`<p class="mx-auto max-w-2xl text-sm text-muted-foreground">A Go web application built on W3C Web API primitives.</p>` +
		`</div>` +
		`</div>` +
		`</main>` +
		selectorScript + `</body></html>`

	if got != want {
		t.Errorf("HomePage output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHomeContent_NoServices(t *testing.T) {
	req := viewstate.Request{}
	got := render.RenderNode(t, HomeContent(req, nil))

	want := `<div class="mx-auto flex max-w-3xl flex-col items-center justify-center gap-8 py-24 text-center">` +
		`<div class="space-y-4">` +
		`<p class="text-sm font-medium uppercase tracking-[0.2em] text-muted-foreground">Go Web Starter</p>` +
		`<h1 class="text-2xl font-bold">Welcome to Foundry</h1>` +
		`<p class="mx-auto max-w-2xl text-sm text-muted-foreground">A Go web application built on W3C Web API primitives.</p>` +
		`</div>` +
		`</div>`

	if got != want {
		t.Errorf("HomeContent output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHomeContent_HealthyService(t *testing.T) {
	req := viewstate.Request{}
	got := render.RenderNode(t, HomeContent(req, []ServiceStatus{{Name: "Database", Healthy: true}}))

	want := `<div class="mx-auto flex max-w-3xl flex-col items-center justify-center gap-8 py-24 text-center">` +
		`<div class="space-y-4">` +
		`<p class="text-sm font-medium uppercase tracking-[0.2em] text-muted-foreground">Go Web Starter</p>` +
		`<h1 class="text-2xl font-bold">Welcome to Foundry</h1>` +
		`<p class="mx-auto max-w-2xl text-sm text-muted-foreground">A Go web application built on W3C Web API primitives.</p>` +
		`</div>` +
		`<div class="flex flex-wrap gap-3 justify-center">` +
		`<div class="flex items-center gap-2 rounded-lg border bg-card px-4 py-3 text-sm shadow-sm">` +
		`<span class="size-2 rounded-full bg-green-500"></span>` +
		`<span class="font-medium text-foreground">Database</span>` +
		`<span class="text-muted-foreground">Healthy</span>` +
		`</div>` +
		`</div>` +
		`</div>`

	if got != want {
		t.Errorf("HomeContent healthy output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHomeContent_UnhealthyService(t *testing.T) {
	req := viewstate.Request{}
	got := render.RenderNode(t, HomeContent(req, []ServiceStatus{{Name: "Database", Healthy: false}}))

	want := `<div class="mx-auto flex max-w-3xl flex-col items-center justify-center gap-8 py-24 text-center">` +
		`<div class="space-y-4">` +
		`<p class="text-sm font-medium uppercase tracking-[0.2em] text-muted-foreground">Go Web Starter</p>` +
		`<h1 class="text-2xl font-bold">Welcome to Foundry</h1>` +
		`<p class="mx-auto max-w-2xl text-sm text-muted-foreground">A Go web application built on W3C Web API primitives.</p>` +
		`</div>` +
		`<div class="flex flex-wrap gap-3 justify-center">` +
		`<div class="flex items-center gap-2 rounded-lg border bg-card px-4 py-3 text-sm shadow-sm">` +
		`<span class="size-2 rounded-full bg-red-500"></span>` +
		`<span class="font-medium text-foreground">Database</span>` +
		`<span class="text-muted-foreground">Unavailable</span>` +
		`</div>` +
		`</div>` +
		`</div>`

	if got != want {
		t.Errorf("HomeContent unhealthy output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}
