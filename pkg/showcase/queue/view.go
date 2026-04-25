package queue

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/go-sum/componentry/interactive/pagination"
	"github.com/go-sum/componentry/patterns/pager"
	"github.com/go-sum/componentry/ui/core"
	"github.com/go-sum/componentry/ui/data"
	"github.com/go-sum/web/render"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func indexContent(basePath string, queues []QueueSummary) g.Node {
	return h.Div(
		h.Class("space-y-6 py-6"),
		h.Div(
			h.Class("flex flex-col gap-1"),
			h.H1(h.Class("text-3xl font-bold tracking-tight"), g.Text("Queues")),
			h.P(h.Class("text-muted-foreground"), g.Textf("%d queues", len(queues))),
		),
		h.Div(h.ID("queue-content"), queueListView(basePath, queues)),
	)
}

func queueListView(basePath string, queues []QueueSummary) g.Node {
	if len(queues) == 0 {
		return h.Div(
			h.Class("rounded-lg border bg-card p-8 text-center"),
			h.P(h.Class("text-muted-foreground"), g.Text("No queues found.")),
		)
	}

	rows := make([]g.Node, len(queues))
	for i, q := range queues {
		queueURL := basePath + "/" + url.PathEscape(q.Name)
		rows[i] = data.Table.Row(data.RowProps{},
			data.Table.Cell(
				h.A(
					h.Class("font-mono font-medium text-primary hover:underline"),
					h.Href(queueURL),
					render.HXGet(queueURL),
					render.HXTarget("#queue-content"),
					render.HXSwap("outerHTML"),
					render.HXPushURL(queueURL),
					g.Text(q.Name),
				),
			),
			data.Table.Cell(g.Textf("%d", q.Pending)),
			data.Table.Cell(g.Textf("%d", q.Running)),
			data.Table.Cell(g.Textf("%d", q.Completed)),
			data.Table.Cell(g.Textf("%d", q.Failed)),
			data.Table.Cell(g.Textf("%d", q.Dead)),
			data.Table.Cell(g.Textf("%d", q.Total)),
		)
	}

	return data.Table.Root(
		data.Table.Header(
			data.Table.Row(data.RowProps{},
				data.Table.Head(g.Text("Queue")),
				data.Table.Head(g.Text("Pending")),
				data.Table.Head(g.Text("Running")),
				data.Table.Head(g.Text("Completed")),
				data.Table.Head(g.Text("Failed")),
				data.Table.Head(g.Text("Dead")),
				data.Table.Head(g.Text("Total")),
			),
		),
		data.Table.Body(data.BodyProps{}, rows...),
	)
}

func detailContent(basePath, queueName string, counts StatusCounts, activeStatus string, jobs []JobRow, pg pager.Pager) g.Node {
	return h.Div(
		h.ID("queue-content"),
		h.Div(
			h.Class("space-y-6 py-6"),
			h.Div(
				h.Class("flex flex-col gap-2"),
				h.A(
					h.Class("text-sm text-muted-foreground hover:underline w-fit"),
					h.Href(basePath),
					render.HXGet(basePath),
					render.HXTarget("#queue-content"),
					render.HXSwap("outerHTML"),
					render.HXPushURL(basePath),
					g.Text("← All Queues"),
				),
				h.Div(
					h.Class("flex flex-wrap items-center gap-3"),
					h.H1(h.Class("text-2xl font-bold font-mono"), g.Text(queueName)),
					core.Badge(core.BadgeProps{
						Variant:  core.BadgeOutline,
						Children: []g.Node{g.Textf("%d jobs", counts.Total)},
					}),
				),
			),
			statusFilterTabs(basePath, queueName, counts, activeStatus),
			jobsRegion(basePath, queueName, activeStatus, jobs, pg),
		),
	)
}

func statusFilterTabs(basePath, queueName string, counts StatusCounts, activeStatus string) g.Node {
	jobsBase := basePath + "/" + url.PathEscape(queueName) + "/jobs"

	makeTab := func(label string, count int, tabStatus string) g.Node {
		var tabURL string
		if tabStatus == "" {
			tabURL = jobsBase
		} else {
			tabURL = jobsBase + "?status=" + tabStatus
		}
		isActive := activeStatus == tabStatus
		var variant core.BadgeVariant
		if isActive {
			variant = core.BadgeDefault
		} else {
			variant = core.BadgeOutline
		}
		return h.A(
			h.Href(tabURL),
			render.HXGet(tabURL),
			render.HXTarget("#queue-jobs-region"),
			render.HXSwap("outerHTML"),
			core.Badge(core.BadgeProps{
				Variant:  variant,
				Children: []g.Node{g.Textf("%s (%d)", label, count)},
			}),
		)
	}

	return h.Div(
		h.Class("flex flex-wrap gap-2"),
		makeTab("All", counts.Total, ""),
		makeTab("Pending", counts.Pending, "pending"),
		makeTab("Running", counts.Running, "running"),
		makeTab("Completed", counts.Completed, "completed"),
		makeTab("Failed", counts.Failed, "failed"),
		makeTab("Dead", counts.Dead, "dead"),
	)
}

func jobsRegion(basePath, queueName, activeStatus string, jobs []JobRow, pg pager.Pager) g.Node {
	var paginationNode g.Node
	if pg.HasPages() {
		paginationNode = paginationControls(basePath, queueName, activeStatus, pg)
	}
	return h.Div(
		h.ID("queue-jobs-region"),
		h.Class("rounded-lg border"),
		h.Div(
			h.Class("flex items-center justify-between border-b px-4 py-3"),
			h.Div(
				h.Class("flex flex-col gap-0.5"),
				h.Span(h.Class("font-semibold text-sm"), g.Text("Jobs")),
				h.Span(
					h.Class("text-xs text-muted-foreground"),
					g.Textf("Page %d of %d — %d total", pg.Page, pg.TotalPages, pg.TotalItems),
				),
			),
		),
		jobsTable(jobs),
		paginationNode,
	)
}

func jobsTable(jobs []JobRow) g.Node {
	if len(jobs) == 0 {
		return h.Div(
			h.Class("p-8 text-center"),
			h.P(h.Class("text-muted-foreground"), g.Text("No jobs found.")),
		)
	}

	rows := make([]g.Node, len(jobs))
	for i, job := range jobs {
		rows[i] = data.Table.Row(data.RowProps{},
			data.Table.Cell(
				h.Span(h.Class("font-mono text-xs"), h.Title(job.ID), g.Text(truncateID(job.ID))),
			),
			data.Table.Cell(statusBadge(job.Status, job.LastError)),
			data.Table.Cell(g.Text(priorityLabel(job.Priority))),
			data.Table.Cell(g.Textf("%d/%d", job.Attempts, job.MaxAttempts)),
			data.Table.Cell(g.Text(job.RunAt.Format(time.RFC3339))),
			data.Table.Cell(g.Text(job.CreatedAt.Format(time.RFC3339))),
		)
	}

	return data.Table.Root(
		data.Table.Header(
			data.Table.Row(data.RowProps{},
				data.Table.Head(g.Text("ID")),
				data.Table.Head(g.Text("Status")),
				data.Table.Head(g.Text("Priority")),
				data.Table.Head(g.Text("Attempts")),
				data.Table.Head(g.Text("Run At")),
				data.Table.Head(g.Text("Created")),
			),
		),
		data.Table.Body(data.BodyProps{}, rows...),
	)
}

func paginationControls(basePath, queueName, activeStatus string, pg pager.Pager) g.Node {
	pageURL := func(page int) string {
		u := fmt.Sprintf("%s/%s/jobs?page=%d&per_page=%d", basePath, url.PathEscape(queueName), page, pg.PerPage)
		if activeStatus != "" {
			u += "&status=" + activeStatus
		}
		return u
	}

	var prevExtra []g.Node
	if !pg.IsFirst() {
		prevExtra = []g.Node{
			render.HXGet(pageURL(pg.PrevPage())),
			render.HXTarget("#queue-jobs-region"),
			render.HXSwap("outerHTML"),
		}
	}

	var nextExtra []g.Node
	if !pg.IsLast() {
		nextExtra = []g.Node{
			render.HXGet(pageURL(pg.NextPage())),
			render.HXTarget("#queue-jobs-region"),
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
					render.HXTarget("#queue-jobs-region"),
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

func statusBadge(status, lastError string) g.Node {
	var variant core.BadgeVariant
	switch status {
	case "pending":
		variant = core.BadgeSecondary
	case "running":
		variant = core.BadgeDefault
	case "completed":
		variant = core.BadgeOutline
	case "failed", "dead":
		variant = core.BadgeDestructive
	default:
		variant = core.BadgeSecondary
	}
	props := core.BadgeProps{Variant: variant, Children: []g.Node{g.Text(status)}}
	if lastError != "" && (status == "failed" || status == "dead") {
		return h.Span(h.Title(lastError), core.Badge(props))
	}
	return core.Badge(props)
}

func priorityLabel(priority int) string {
	switch priority {
	case 0:
		return "critical"
	case 10:
		return "high"
	case 20:
		return "default"
	case 30:
		return "low"
	default:
		return strconv.Itoa(priority)
	}
}

func truncateID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8] + "…"
}
