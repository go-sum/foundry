// Package demo provides pure fragment-rendering functions and URL path constants
// for the componentry showcase's live HTMX demo endpoints.
//
// The path constants act as the single source of truth so the showcase page
// and the starter's route registration always stay in sync.
package demo

import (
	"fmt"
	"strings"

	uiform "github.com/go-sum/componentry/form"
	"github.com/go-sum/componentry/interactive/pagination"
	"github.com/go-sum/componentry/ui/core"
	"github.com/go-sum/componentry/ui/data"
	"github.com/go-sum/componentry/ui/feedback"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// URL path constants for the demo endpoints. Import these both from the
// showcase (for hx-get attribute values) and from the starter (for route
// registration) so the URLs never diverge.
const (
	PathSearch   = "/componentry/demo/search"
	PathValidate = "/componentry/demo/validate"
	PathPaginate = "/componentry/demo/paginate"
	PathRegion   = "/componentry/demo/region"
	PathOOBToast = "/componentry/demo/oob-toast"
)

// OOBToast returns a success toast fragment with an hx-swap-oob attribute that
// instructs HTMX to append it to #toast-container out-of-band, independent of
// the primary swap target. The endpoint returns this as the entire response body
// alongside hx-swap="none" on the trigger so the primary target is untouched.
func OOBToast() g.Node {
	return feedback.Toast(feedback.ToastProps{
		Title:       "Saved!",
		Description: "Changes saved in the background.",
		Variant:     feedback.ToastSuccess,
		Dismissible: true,
		Extra: []g.Node{
			g.Attr("hx-swap-oob", "beforeend:#toast-container"),
			g.Attr("data-controller", "toast"),
			g.Attr("data-toast-duration", "5000"),
		},
	})
}

type user struct {
	Name   string
	Role   string
	Status string
}

var users = []user{
	{"Alice Johnson", "Admin", "Active"},
	{"Bob Smith", "Editor", "Inactive"},
	{"Carol White", "Viewer", "Active"},
	{"David Brown", "Editor", "Active"},
	{"Eve Davis", "Admin", "Active"},
	{"Frank Miller", "Viewer", "Inactive"},
	{"Grace Wilson", "Editor", "Active"},
	{"Henry Moore", "Viewer", "Active"},
	{"Iris Taylor", "Admin", "Inactive"},
	{"Jack Anderson", "Editor", "Active"},
}

// SearchResults returns a table fragment filtered by query (case-insensitive
// substring match on name). An empty query returns all rows.
func SearchResults(query string) g.Node {
	q := strings.ToLower(query)
	var rows []g.Node
	for _, u := range users {
		if q == "" || strings.Contains(strings.ToLower(u.Name), q) {
			active := u.Status == "Active"
			variant := core.BadgeDefault
			if !active {
				variant = core.BadgeSecondary
			}
			rows = append(rows, data.Table.Row(data.RowProps{},
				data.Table.Cell(g.Text(u.Name)),
				data.Table.Cell(g.Text(u.Role)),
				data.Table.Cell(core.Badge(core.BadgeProps{Variant: variant, Children: []g.Node{g.Text(u.Status)}})),
			))
		}
	}
	if len(rows) == 0 {
		rows = append(rows, data.Table.Row(data.RowProps{},
			h.Td(h.Class("py-4 text-center text-sm text-muted-foreground"), g.Attr("colspan", "3"), g.Text("No results found.")),
		))
	}
	return data.Table.Root(
		data.Table.Header(
			data.Table.Row(data.RowProps{},
				data.Table.Head(g.Text("Name")),
				data.Table.Head(g.Text("Role")),
				data.Table.Head(g.Text("Status")),
			),
		),
		data.Table.Body(data.BodyProps{ID: "search-results"}, rows...),
	)
}

// ValidationResult returns a form field fragment with inline validation
// feedback. Supported fields: "email" and "username".
func ValidationResult(field, value string) g.Node {
	var errMsg string
	switch field {
	case "email":
		if !strings.Contains(value, "@") || !strings.Contains(value, ".") {
			errMsg = "Please enter a valid email address."
		}
	case "username":
		if len(value) < 3 {
			errMsg = "Username must be at least 3 characters."
		}
	}

	inputType := uiform.TypeText
	if field == "email" {
		inputType = uiform.TypeEmail
	}

	var extra []g.Node
	if errMsg != "" {
		extra = append(extra, g.Attr("aria-invalid", "true"))
	}

	return h.Div(
		h.ID("validate-field"),
		uiform.Input(uiform.InputProps{
			ID:    "validate-" + field,
			Name:  field,
			Type:  inputType,
			Value: value,
			Extra: extra,
		}),
		g.If(errMsg != "", h.P(h.Class("mt-1 text-xs text-destructive"), g.Text(errMsg))),
		g.If(errMsg == "" && value != "", h.P(h.Class("mt-1 text-xs text-green-600 dark:text-green-400"), g.Text("Looks good!"))),
	)
}

type item struct {
	ID    int
	Name  string
	Price string
}

var catalog = func() []item {
	names := []string{
		"Wireless Keyboard", "Ergonomic Mouse", "USB-C Hub", "Standing Desk",
		"Monitor Arm", "Cable Organiser", "Webcam HD", "LED Desk Lamp",
		"Noise-Cancelling Headphones", "Laptop Stand", "Mechanical Keyboard",
		"Trackpad", "Display Port Cable", "HDMI Adapter", "Desk Mat",
		"Keyboard Wrist Rest", "Cable Clips", "Monitor Light Bar",
		"Wireless Charger", "USB Hub", "Power Strip", "Chair Cushion",
		"Blue Light Glasses", "Phone Stand", "Stylus Pen", "Drawing Tablet",
		"Smart Plug", "Desk Fan", "Air Purifier", "White Noise Machine",
	}
	out := make([]item, len(names))
	for i, n := range names {
		out[i] = item{ID: i + 1, Name: n, Price: fmt.Sprintf("$%d.%02d", 19+i*3, (i*7)%100)}
	}
	return out
}()

// PaginatedTable returns a table fragment for the given page. The fragment
// includes both the table rows and pagination controls so HTMX can replace
// the entire region in one swap.
func PaginatedTable(page, perPage int) g.Node {
	total := len(catalog)
	if perPage < 1 {
		perPage = 10
	}
	totalPages := (total + perPage - 1) / perPage
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * perPage
	end := start + perPage
	if end > total {
		end = total
	}
	slice := catalog[start:end]

	var rows []g.Node
	for _, it := range slice {
		rows = append(rows, data.Table.Row(data.RowProps{},
			data.Table.Cell(g.Text(fmt.Sprintf("%d", it.ID))),
			data.Table.Cell(g.Text(it.Name)),
			data.Table.Cell(g.Text(it.Price)),
		))
	}

	prevURL := ""
	if page > 1 {
		prevURL = fmt.Sprintf("%s?page=%d&per_page=%d", PathPaginate, page-1, perPage)
	}
	nextURL := ""
	if page < totalPages {
		nextURL = fmt.Sprintf("%s?page=%d&per_page=%d", PathPaginate, page+1, perPage)
	}

	paginationAttrs := func(url string) []g.Node {
		if url == "" {
			return nil
		}
		return []g.Node{
			g.Attr("hx-get", url),
			g.Attr("hx-target", "#paginate-region"),
			g.Attr("hx-swap", "outerHTML"),
		}
	}

	return h.Div(
		h.ID("paginate-region"),
		data.Table.Root(
			data.Table.Header(
				data.Table.Row(data.RowProps{},
					data.Table.Head(g.Text("#")),
					data.Table.Head(g.Text("Product")),
					data.Table.Head(g.Text("Price")),
				),
			),
			data.Table.Body(data.BodyProps{}, rows...),
		),
		pagination.Root(
			pagination.Content(
				pagination.Item(pagination.Previous(prevURL, page == 1, paginationAttrs(prevURL)...)),
				pagination.Item(pagination.Next(nextURL, page == totalPages, paginationAttrs(nextURL)...)),
			),
		),
		h.P(
			h.Class("mt-2 text-xs text-muted-foreground"),
			g.Text(fmt.Sprintf("Page %d of %d — %d items", page, totalPages, total)),
		),
	)
}

type region struct {
	Value string
	Label string
}

var regions = map[string][]region{
	"se": {
		{Value: "stockholm", Label: "Stockholm"},
		{Value: "gothenburg", Label: "Gothenburg"},
		{Value: "malmo", Label: "Malmö"},
		{Value: "uppsala", Label: "Uppsala"},
	},
	"us": {
		{Value: "ca", Label: "California"},
		{Value: "ny", Label: "New York"},
		{Value: "tx", Label: "Texas"},
		{Value: "wa", Label: "Washington"},
	},
	"de": {
		{Value: "berlin", Label: "Berlin"},
		{Value: "munich", Label: "Munich"},
		{Value: "hamburg", Label: "Hamburg"},
		{Value: "cologne", Label: "Cologne"},
	},
}

// RegionOptions returns a select fragment populated with regions for the given
// country ID (e.g. "se", "us", "de"). Unknown country IDs return a disabled
// placeholder select.
func RegionOptions(countryID string) g.Node {
	opts, ok := regions[strings.ToLower(countryID)]
	if !ok {
		return h.Div(
			h.ID("region-field"),
			feedback.Alert.Root(
				feedback.AlertProps{Variant: feedback.AlertDefault},
				feedback.Alert.Description(g.Text("No regions available for the selected country.")),
			),
		)
	}

	var selectOpts []uiform.Option
	for _, r := range opts {
		selectOpts = append(selectOpts, uiform.Option{Value: r.Value, Label: r.Label})
	}

	return h.Div(
		h.ID("region-field"),
		uiform.Select(uiform.SelectProps{
			ID:      "region",
			Name:    "region",
			Options: selectOpts,
		}),
	)
}
