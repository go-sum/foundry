package head

import (
	"testing"

	"github.com/go-sum/componentry/testutil"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func TestHead(t *testing.T) {
	t.Run("full props produces charset viewport title and all meta in correct order", func(t *testing.T) {
		got := testutil.RenderNode(t, Head(Props{
			Meta: MetaProps{
				Title:       "My Page",
				Description: "A description",
				Favicon:     "/favicon.ico",
				Keywords:    []string{"go", "web"},
				Canonical:   "https://example.com/page",
				Robots:      "noindex",
				OG: &OpenGraph{
					Title:       "OG Title",
					Type:        "article",
					Description: "OG desc",
					Image:       "https://example.com/img.png",
					URL:         "https://example.com/page",
				},
			},
			Stylesheets: []Stylesheet{
				{Href: "/app.css"},
				{Href: "https://cdn.example.com/lib.css", Integrity: "sha384-abc123"},
			},
			Scripts: []Script{
				{Src: "/app.js", Defer: true},
				{Src: "/htmx.js", Async: true},
			},
			Extra: []g.Node{h.Meta(h.Name("theme-color"), h.Content("#ffffff"))},
		}))
		want := `<head>` +
			`<meta charset="UTF-8">` +
			`<meta name="viewport" content="width=device-width, initial-scale=1.0">` +
			`<title>My Page</title>` +
			`<meta name="description" content="A description">` +
			`<link rel="icon" href="/favicon.ico">` +
			`<link rel="canonical" href="https://example.com/page">` +
			`<meta name="robots" content="noindex">` +
			`<meta name="keywords" content="go,web">` +
			`<meta property="og:title" content="OG Title">` +
			`<meta property="og:type" content="article">` +
			`<meta property="og:description" content="OG desc">` +
			`<meta property="og:image" content="https://example.com/img.png">` +
			`<meta property="og:url" content="https://example.com/page">` +
			`<link rel="stylesheet" href="/app.css">` +
			`<link rel="stylesheet" href="https://cdn.example.com/lib.css" integrity="sha384-abc123" crossorigin="anonymous">` +
			`<script src="/app.js" defer></script>` +
			`<script src="/htmx.js" async></script>` +
			`<meta name="theme-color" content="#ffffff">` +
			`</head>`
		if got != want {
			t.Errorf("Head full props:\n got:  %s\n want: %s", got, want)
		}
	})

	t.Run("empty props still produces charset and viewport", func(t *testing.T) {
		got := testutil.RenderNode(t, Head(Props{}))
		want := `<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>`
		if got != want {
			t.Errorf("Head empty props:\n got:  %s\n want: %s", got, want)
		}
	})
}

func TestMetatags(t *testing.T) {
	t.Run("all fields set renders every tag", func(t *testing.T) {
		got := testutil.RenderNode(t, Metatags(MetaProps{
			Title:       "Full Title",
			Description: "Full description",
			Favicon:     "/favicon.ico",
			Keywords:    []string{"foo", "bar"},
			Canonical:   "https://example.com/full",
			Robots:      "noindex,nofollow",
			OG: &OpenGraph{
				Title:       "OG Full",
				Type:        "article",
				Description: "OG description",
				Image:       "https://example.com/og.png",
				URL:         "https://example.com/full",
			},
		}))
		want := `<title>Full Title</title>` +
			`<meta name="description" content="Full description">` +
			`<link rel="icon" href="/favicon.ico">` +
			`<link rel="canonical" href="https://example.com/full">` +
			`<meta name="robots" content="noindex,nofollow">` +
			`<meta name="keywords" content="foo,bar">` +
			`<meta property="og:title" content="OG Full">` +
			`<meta property="og:type" content="article">` +
			`<meta property="og:description" content="OG description">` +
			`<meta property="og:image" content="https://example.com/og.png">` +
			`<meta property="og:url" content="https://example.com/full">`
		if got != want {
			t.Errorf("Metatags all fields:\n got:  %s\n want: %s", got, want)
		}
	})

	t.Run("empty fields renders no tags", func(t *testing.T) {
		got := testutil.RenderNode(t, Metatags(MetaProps{}))
		if got != "" {
			t.Errorf("Metatags empty: expected empty string, got: %q", got)
		}
	})

	t.Run("OG tags only emitted when OG non-nil with Title set", func(t *testing.T) {
		t.Run("OG nil emits no OG tags", func(t *testing.T) {
			got := testutil.RenderNode(t, Metatags(MetaProps{Title: "Title"}))
			want := `<title>Title</title>`
			if got != want {
				t.Errorf("Metatags OG nil:\n got:  %s\n want: %s", got, want)
			}
		})

		t.Run("OG non-nil but empty Title emits no OG tags", func(t *testing.T) {
			got := testutil.RenderNode(t, Metatags(MetaProps{
				Title: "Title",
				OG:    &OpenGraph{Description: "desc"},
			}))
			want := `<title>Title</title>`
			if got != want {
				t.Errorf("Metatags OG no title:\n got:  %s\n want: %s", got, want)
			}
		})

		t.Run("OG non-nil with Title emits OG tags", func(t *testing.T) {
			got := testutil.RenderNode(t, Metatags(MetaProps{
				OG: &OpenGraph{Title: "OG Title"},
			}))
			want := `<meta property="og:title" content="OG Title"><meta property="og:type" content="website">`
			if got != want {
				t.Errorf("Metatags OG with title:\n got:  %s\n want: %s", got, want)
			}
		})

		t.Run("OG type defaults to website when empty", func(t *testing.T) {
			got := testutil.RenderNode(t, Metatags(MetaProps{
				OG: &OpenGraph{Title: "OG Title"},
			}))
			want := `<meta property="og:title" content="OG Title"><meta property="og:type" content="website">`
			if got != want {
				t.Errorf("Metatags OG default type:\n got:  %s\n want: %s", got, want)
			}
		})
	})
}

func TestCSS(t *testing.T) {
	t.Run("with integrity emits crossorigin and integrity attrs", func(t *testing.T) {
		got := testutil.RenderNode(t, CSS(
			Stylesheet{Href: "https://cdn.example.com/lib.css", Integrity: "sha384-abc123"},
		))
		want := `<link rel="stylesheet" href="https://cdn.example.com/lib.css" integrity="sha384-abc123" crossorigin="anonymous">`
		if got != want {
			t.Errorf("CSS integrity:\n got:  %s\n want: %s", got, want)
		}
	})

	t.Run("without integrity emits no crossorigin or integrity", func(t *testing.T) {
		got := testutil.RenderNode(t, CSS(
			Stylesheet{Href: "/app.css"},
		))
		want := `<link rel="stylesheet" href="/app.css">`
		if got != want {
			t.Errorf("CSS no integrity:\n got:  %s\n want: %s", got, want)
		}
	})

	t.Run("skips empty Href entries", func(t *testing.T) {
		got := testutil.RenderNode(t, CSS(
			Stylesheet{Href: "/app.css"},
			Stylesheet{Href: ""},
			Stylesheet{Href: "/other.css"},
		))
		want := `<link rel="stylesheet" href="/app.css"><link rel="stylesheet" href="/other.css">`
		if got != want {
			t.Errorf("CSS skip empty href:\n got:  %s\n want: %s", got, want)
		}
	})

	t.Run("no stylesheets renders empty", func(t *testing.T) {
		got := testutil.RenderNode(t, CSS())
		if got != "" {
			t.Errorf("CSS no stylesheets: expected empty string, got: %q", got)
		}
	})
}

func TestJS(t *testing.T) {
	t.Run("with Defer true emits defer attribute", func(t *testing.T) {
		got := testutil.RenderNode(t, JS(
			Script{Src: "/app.js", Defer: true},
		))
		want := `<script src="/app.js" defer></script>`
		if got != want {
			t.Errorf("JS defer:\n got:  %s\n want: %s", got, want)
		}
	})

	t.Run("with Async true emits async attribute", func(t *testing.T) {
		got := testutil.RenderNode(t, JS(
			Script{Src: "/analytics.js", Async: true},
		))
		want := `<script src="/analytics.js" async></script>`
		if got != want {
			t.Errorf("JS async:\n got:  %s\n want: %s", got, want)
		}
	})

	t.Run("skips empty Src entries", func(t *testing.T) {
		got := testutil.RenderNode(t, JS(
			Script{Src: "/app.js", Defer: true},
			Script{Src: ""},
			Script{Src: "/htmx.js"},
		))
		want := `<script src="/app.js" defer></script><script src="/htmx.js"></script>`
		if got != want {
			t.Errorf("JS skip empty src:\n got:  %s\n want: %s", got, want)
		}
	})

	t.Run("no scripts renders empty", func(t *testing.T) {
		got := testutil.RenderNode(t, JS())
		if got != "" {
			t.Errorf("JS no scripts: expected empty string, got: %q", got)
		}
	})
}
