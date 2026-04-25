package db

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-sum/componentry/interactive/accordion"
	"github.com/go-sum/componentry/interactive/pagination"
	"github.com/go-sum/componentry/patterns/pager"
	"github.com/go-sum/componentry/ui/core"
	"github.com/go-sum/componentry/ui/data"
	"github.com/go-sum/web/render"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

const maxValueLen = 100

// indexContent builds the full content for the table listing page.
// The table list lives inside #db-content; HTMX table links target it with outerHTML swap,
// replacing the list with the table detail view without a full page reload.
func indexContent(basePath string, tables []TableInfo) g.Node {
	return h.Div(
		h.Class("space-y-6 py-6"),
		h.Div(
			h.Class("flex flex-col gap-1"),
			h.H1(h.Class("text-3xl font-bold tracking-tight"), g.Text("Database Tables")),
			h.P(h.Class("text-muted-foreground"), g.Textf("%d tables", len(tables))),
		),
		h.Div(h.ID("db-content"), tableListView(basePath, tables)),
	)
}

func tableListView(basePath string, tables []TableInfo) g.Node {
	if len(tables) == 0 {
		return h.Div(
			h.Class("rounded-lg border bg-card p-8 text-center"),
			h.P(h.Class("text-muted-foreground"), g.Text("No tables found in this schema.")),
		)
	}

	rows := make([]g.Node, len(tables))
	for i, t := range tables {
		tableURL := basePath + "/tables/" + t.Name
		rows[i] = data.Table.Row(data.RowProps{},
			data.Table.Cell(
				h.A(
					h.Class("font-mono font-medium text-primary hover:underline"),
					h.Href(tableURL),
					render.HXGet(tableURL),
					render.HXTarget("#db-content"),
					render.HXSwap("outerHTML"),
					render.HXPushURL(tableURL),
					g.Text(t.Name),
				),
			),
		)
	}

	return data.Table.Root(
		data.Table.Header(
			data.Table.Row(data.RowProps{}, data.Table.Head(g.Text("Table"))),
		),
		data.Table.Body(data.BodyProps{}, rows...),
	)
}

// tableDetailContent builds the table detail view (schema info accordion + data table).
// Wrapped in #db-content to serve as the outerHTML swap target from the index page,
// and to provide a stable root div on direct URL loads.
func tableDetailContent(basePath, tableName string, columns []ColumnInfo, indexes []IndexInfo, td TableData, pg pager.Pager) g.Node {
	return h.Div(
		h.ID("db-content"),
		h.Div(
			h.Class("space-y-6 py-6"),
			h.Div(
				h.Class("flex flex-col gap-2"),
				h.A(
					h.Class("text-sm text-muted-foreground hover:underline w-fit"),
					h.Href(basePath),
					g.Text("← All Tables"),
				),
				h.Div(
					h.Class("flex flex-wrap items-center gap-3"),
					h.H1(h.Class("text-2xl font-bold font-mono"), g.Text(tableName)),
					core.Badge(core.BadgeProps{
						Variant:  core.BadgeSecondary,
						Children: []g.Node{g.Textf("%d columns", len(columns))},
					}),
					core.Badge(core.BadgeProps{
						Variant:  core.BadgeOutline,
						Children: []g.Node{g.Textf("%d indexes", len(indexes))},
					}),
					core.Badge(core.BadgeProps{
						Variant:  core.BadgeOutline,
						Children: []g.Node{g.Textf("%d rows", td.Total)},
					}),
				),
			),
			infoAccordion(columns, indexes),
			dataRegion(basePath, tableName, td, pg),
		),
	)
}

func infoAccordion(columns []ColumnInfo, indexes []IndexInfo) g.Node {
	return accordion.Root(accordion.RootProps{},
		accordion.Item(
			accordion.Trigger(nil,
				h.Span(
					h.Class("flex items-center gap-2"),
					g.Text("Schema"),
					core.Badge(core.BadgeProps{
						Variant:  core.BadgeSecondary,
						Children: []g.Node{g.Textf("%d columns", len(columns))},
					}),
				),
			),
			accordion.Content(schemaTable(columns)),
		),
		accordion.Item(
			accordion.Trigger(nil,
				h.Span(
					h.Class("flex items-center gap-2"),
					g.Text("Indexes"),
					core.Badge(core.BadgeProps{
						Variant:  core.BadgeSecondary,
						Children: []g.Node{g.Textf("%d", len(indexes))},
					}),
				),
			),
			accordion.Content(indexesTable(indexes)),
		),
	)
}

func schemaTable(columns []ColumnInfo) g.Node {
	rows := make([]g.Node, len(columns))
	for i, col := range columns {
		var badges []g.Node
		if col.IsPrimaryKey {
			badges = append(badges, core.Badge(core.BadgeProps{
				Variant:  core.BadgeDestructive,
				Children: []g.Node{g.Text("PK")},
			}))
		} else if !col.IsNullable {
			badges = append(badges, core.Badge(core.BadgeProps{
				Variant:  core.BadgeSecondary,
				Children: []g.Node{g.Text("NOT NULL")},
			}))
		}

		var defaultCell g.Node
		if col.DefaultValue != "" {
			defaultCell = data.Table.Cell(h.Code(h.Class("text-xs font-mono"), g.Text(col.DefaultValue)))
		} else {
			defaultCell = data.Table.Cell(h.Span(h.Class("text-muted-foreground"), g.Text("—")))
		}

		rows[i] = data.Table.Row(data.RowProps{},
			data.Table.Cell(h.Span(h.Class("font-mono font-medium"), g.Text(col.Name))),
			data.Table.Cell(core.Badge(core.BadgeProps{
				Variant:  core.BadgeOutline,
				Children: []g.Node{g.Text(col.DataType)},
			})),
			data.Table.Cell(g.Group(badges)),
			defaultCell,
		)
	}

	return data.Table.Root(
		data.Table.Header(
			data.Table.Row(data.RowProps{},
				data.Table.Head(g.Text("Column")),
				data.Table.Head(g.Text("Type")),
				data.Table.Head(g.Text("Constraints")),
				data.Table.Head(g.Text("Default")),
			),
		),
		data.Table.Body(data.BodyProps{}, rows...),
	)
}

func indexesTable(indexes []IndexInfo) g.Node {
	if len(indexes) == 0 {
		return h.P(h.Class("text-sm text-muted-foreground py-2"), g.Text("No indexes."))
	}

	rows := make([]g.Node, len(indexes))
	for i, idx := range indexes {
		var typeBadge g.Node
		switch {
		case idx.IsPrimary:
			typeBadge = core.Badge(core.BadgeProps{
				Variant:  core.BadgeDestructive,
				Children: []g.Node{g.Text("PRIMARY")},
			})
		case idx.IsUnique:
			typeBadge = core.Badge(core.BadgeProps{
				Variant:  core.BadgeDefault,
				Children: []g.Node{g.Text("UNIQUE")},
			})
		default:
			typeBadge = core.Badge(core.BadgeProps{
				Variant:  core.BadgeSecondary,
				Children: []g.Node{g.Text("INDEX")},
			})
		}

		rows[i] = data.Table.Row(data.RowProps{},
			data.Table.Cell(h.Span(h.Class("font-mono text-sm"), g.Text(idx.Name))),
			data.Table.Cell(typeBadge),
			data.Table.Cell(h.Span(h.Class("font-mono text-sm text-muted-foreground"), g.Text(idx.Columns))),
		)
	}

	return data.Table.Root(
		data.Table.Header(
			data.Table.Row(data.RowProps{},
				data.Table.Head(g.Text("Index")),
				data.Table.Head(g.Text("Type")),
				data.Table.Head(g.Text("Columns")),
			),
		),
		data.Table.Body(data.BodyProps{}, rows...),
	)
}

// dataRegion renders the paginated data table. Its #data-region id makes it
// the outerHTML swap target for HTMX pagination links (db.table.data endpoint).
func dataRegion(basePath, tableName string, td TableData, pg pager.Pager) g.Node {
	var tableContent g.Node
	if len(td.Rows) == 0 {
		tableContent = h.Div(
			h.Class("p-8 text-center"),
			h.P(h.Class("text-muted-foreground"), g.Text("No data in this table.")),
		)
	} else {
		headerCells := make([]g.Node, len(td.Columns))
		for i, col := range td.Columns {
			headerCells[i] = data.Table.Head(h.Span(h.Class("font-mono"), g.Text(col)))
		}

		bodyRows := make([]g.Node, len(td.Rows))
		for i, row := range td.Rows {
			cells := make([]g.Node, len(row))
			for j, val := range row {
				if val == nil {
					cells[j] = data.Table.Cell(
						h.Span(h.Class("font-mono text-xs text-muted-foreground italic"), g.Text("null")),
					)
				} else {
					cells[j] = data.Table.Cell(
						h.Span(h.Class("font-mono text-xs"), g.Text(formatValue(val))),
					)
				}
			}
			bodyRows[i] = data.Table.Row(data.RowProps{}, cells...)
		}

		tableContent = data.Table.Root(
			data.Table.Header(
				data.Table.Row(data.RowProps{}, headerCells...),
			),
			data.Table.Body(data.BodyProps{}, bodyRows...),
		)
	}

	var paginationNode g.Node
	if pg.HasPages() {
		paginationNode = paginationControls(basePath, tableName, pg)
	}

	return h.Div(
		h.ID("data-region"),
		h.Class("rounded-lg border"),
		h.Div(
			h.Class("flex items-center justify-between border-b px-4 py-3"),
			h.Div(
				h.Class("flex flex-col gap-0.5"),
				h.Span(h.Class("font-semibold text-sm"), g.Text("Data")),
				h.Span(
					h.Class("text-xs text-muted-foreground"),
					g.Textf("Page %d of %d — %d total rows", pg.Page, pg.TotalPages, pg.TotalItems),
				),
			),
		),
		tableContent,
		paginationNode,
	)
}

func paginationControls(basePath, tableName string, pg pager.Pager) g.Node {
	dataURL := func(page int) string {
		return fmt.Sprintf("%s/tables/%s/data?page=%d&per_page=%d", basePath, tableName, page, pg.PerPage)
	}

	var prevExtra []g.Node
	if !pg.IsFirst() {
		prevExtra = []g.Node{
			render.HXGet(dataURL(pg.PrevPage())),
			render.HXTarget("#data-region"),
			render.HXSwap("outerHTML"),
		}
	}

	var nextExtra []g.Node
	if !pg.IsLast() {
		nextExtra = []g.Node{
			render.HXGet(dataURL(pg.NextPage())),
			render.HXTarget("#data-region"),
			render.HXSwap("outerHTML"),
		}
	}

	items := []g.Node{
		pagination.Item(pagination.Previous(nil, dataURL(pg.PrevPage()), pg.IsFirst(), prevExtra...)),
	}

	for _, p := range pg.PageRange(2) {
		if p == -1 {
			items = append(items, pagination.Item(pagination.Ellipsis()))
		} else {
			url := dataURL(p)
			items = append(items, pagination.Item(
				pagination.Link(url, p == pg.Page,
					render.HXGet(url),
					render.HXTarget("#data-region"),
					render.HXSwap("outerHTML"),
					g.Text(strconv.Itoa(p)),
				),
			))
		}
	}

	items = append(items, pagination.Item(pagination.Next(nil, dataURL(pg.NextPage()), pg.IsLast(), nextExtra...)))

	return h.Div(
		h.Class("border-t p-3"),
		pagination.Root(pagination.Content(items...)),
	)
}

func formatValue(v any) string {
	switch val := v.(type) {
	case []byte:
		s := fmt.Sprintf("%x", val)
		if len(s) > maxValueLen {
			return s[:maxValueLen] + "…"
		}
		return s
	case time.Time:
		return val.Format(time.RFC3339)
	case string:
		if len(val) > maxValueLen {
			return val[:maxValueLen] + "…"
		}
		return val
	default:
		s := fmt.Sprintf("%v", val)
		if len(s) > maxValueLen {
			return s[:maxValueLen] + "…"
		}
		return s
	}
}
