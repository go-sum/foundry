package kv

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/go-sum/foundry/pkg/componentry/interactive/pagination"
	"github.com/go-sum/foundry/pkg/componentry/patterns/pager"
	"github.com/go-sum/foundry/pkg/componentry/ui/core"
	"github.com/go-sum/foundry/pkg/componentry/ui/data"
	"github.com/go-sum/foundry/pkg/web/render"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func indexContent(basePath string, result KeyListResult, pg pager.Pager) g.Node {
	return h.Div(
		h.Class("space-y-6 py-6"),
		h.Div(
			h.Class("flex flex-col gap-1"),
			h.H1(h.Class("text-3xl font-bold tracking-tight"), g.Text("KV Store")),
			h.P(h.Class("text-muted-foreground"), g.Textf("%d keys", result.Total)),
		),
		kvContentRegion(basePath, result, pg),
	)
}

// kvContentRegion renders the key list table wrapped in #kv-content.
// This is the HTMX swap target for both pagination and navigation from key detail back to the list.
func kvContentRegion(basePath string, result KeyListResult, pg pager.Pager) g.Node {
	var paginationNode g.Node
	if pg.HasPages() {
		paginationNode = paginationControls(basePath, pg)
	}
	return h.Div(
		h.ID("kv-content"),
		keyListView(basePath, result.Keys),
		paginationNode,
	)
}

func keyListView(basePath string, keys []KeyEntry) g.Node {
	if len(keys) == 0 {
		return h.Div(
			h.Class("rounded-lg border bg-card p-8 text-center"),
			h.P(h.Class("text-muted-foreground"), g.Text("No keys found.")),
		)
	}

	rows := make([]g.Node, len(keys))
	for i, k := range keys {
		keyURL := basePath + "/keys/" + url.PathEscape(k.Name)
		rows[i] = data.Table.Row(data.RowProps{},
			data.Table.Cell(
				h.A(
					h.Class("font-mono font-medium text-primary hover:underline"),
					h.Href(keyURL),
					render.HXGet(keyURL),
					render.HXTarget("#kv-content"),
					render.HXSwap("outerHTML"),
					render.HXPushURL(keyURL),
					g.Text(k.Name),
				),
			),
		)
	}

	return data.Table.Root(
		data.Table.Header(
			data.Table.Row(data.RowProps{}, data.Table.Head(g.Text("Key"))),
		),
		data.Table.Body(data.BodyProps{}, rows...),
	)
}

// keyDetailContent renders the key detail view wrapped in #kv-content.
// Serves as the outerHTML swap target when navigating from the key list.
func keyDetailContent(basePath string, detail KeyDetail) g.Node {
	return h.Div(
		h.ID("kv-content"),
		h.Div(
			h.Class("space-y-6 py-6"),
			h.Div(
				h.Class("flex flex-col gap-2"),
				h.A(
					h.Class("text-sm text-muted-foreground hover:underline w-fit"),
					h.Href(basePath),
					render.HXGet(basePath),
					render.HXTarget("#kv-content"),
					render.HXSwap("outerHTML"),
					render.HXPushURL(basePath),
					g.Text("← All Keys"),
				),
				h.Div(
					h.Class("flex flex-wrap items-center gap-3"),
					h.H1(h.Class("text-2xl font-bold font-mono break-all"), g.Text(detail.Key)),
					valueTypeBadge(detail.ValueType),
					core.Badge(core.BadgeProps{
						Variant:  core.BadgeOutline,
						Children: []g.Node{g.Text(formatSize(detail.Size))},
					}),
				),
			),
			valueRegion(basePath, detail),
		),
	)
}

func valueTypeBadge(valueType string) g.Node {
	switch valueType {
	case "json":
		return core.Badge(core.BadgeProps{
			Variant:  core.BadgeDefault,
			Children: []g.Node{g.Text("JSON")},
		})
	case "binary":
		return core.Badge(core.BadgeProps{
			Variant:  core.BadgeDestructive,
			Children: []g.Node{g.Text("binary")},
		})
	default:
		return core.Badge(core.BadgeProps{
			Variant:  core.BadgeSecondary,
			Children: []g.Node{g.Text("text")},
		})
	}
}

// valueRegion renders the key's value display. Its #value-region id makes it
// the outerHTML swap target for the HTMX refresh button (kv.key.value endpoint).
func valueRegion(basePath string, detail KeyDetail) g.Node {
	valueURL := basePath + "/keys/" + url.PathEscape(detail.Key) + "/value"
	return h.Div(
		h.ID("value-region"),
		h.Class("rounded-lg border"),
		h.Div(
			h.Class("flex items-center justify-between border-b px-4 py-3"),
			h.Span(h.Class("font-semibold text-sm"), g.Text("Value")),
			h.A(
				h.Class("text-xs text-muted-foreground hover:underline cursor-pointer"),
				render.HXGet(valueURL),
				render.HXTarget("#value-region"),
				render.HXSwap("outerHTML"),
				g.Text("↻ Refresh"),
			),
		),
		h.Pre(
			h.Class("overflow-x-auto p-4 text-xs font-mono whitespace-pre-wrap break-all"),
			g.Text(detail.Value),
		),
	)
}

func paginationControls(basePath string, pg pager.Pager) g.Node {
	pageURL := func(page int) string {
		return fmt.Sprintf("%s?page=%d&per_page=%d", basePath, page, pg.PerPage)
	}

	var prevExtra []g.Node
	if !pg.IsFirst() {
		prevExtra = []g.Node{
			render.HXGet(pageURL(pg.PrevPage())),
			render.HXTarget("#kv-content"),
			render.HXSwap("outerHTML"),
		}
	}

	var nextExtra []g.Node
	if !pg.IsLast() {
		nextExtra = []g.Node{
			render.HXGet(pageURL(pg.NextPage())),
			render.HXTarget("#kv-content"),
			render.HXSwap("outerHTML"),
		}
	}

	items := []g.Node{
		pagination.Item(pagination.Previous(nil, pageURL(pg.PrevPage()), pg.IsFirst(), prevExtra...)),
	}

	for _, p := range pg.PageRange(2) {
		if p == -1 {
			items = append(items, pagination.Item(pagination.Ellipsis()))
		} else {
			u := pageURL(p)
			items = append(items, pagination.Item(
				pagination.Link(u, p == pg.Page,
					render.HXGet(u),
					render.HXTarget("#kv-content"),
					render.HXSwap("outerHTML"),
					g.Text(strconv.Itoa(p)),
				),
			))
		}
	}

	items = append(items, pagination.Item(pagination.Next(nil, pageURL(pg.NextPage()), pg.IsLast(), nextExtra...)))

	return h.Div(
		h.Class("border-t p-3"),
		pagination.Root(pagination.Content(items...)),
	)
}

func formatSize(bytes int) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
