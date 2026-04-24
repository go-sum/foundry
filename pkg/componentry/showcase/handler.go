package showcase

import (
	"strconv"

	"github.com/go-sum/componentry/showcase/demo"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
)

type handler struct{ cfg Config }

func newHandler(cfg Config) *handler { return &handler{cfg: cfg} }

// Show renders the component showcase full page.
func (h *handler) Show(c *web.Context) (web.Response, error) {
	return h.cfg.Page(c, "Component Showcase", Showcase())
}

// Search returns a filtered user table fragment for the live search demo.
func (h *handler) Search(c *web.Context) (web.Response, error) {
	return render.Fragment(demo.SearchResults(c.URL().Query().Get("q")))
}

// Validate returns an inline validation fragment for the given field and value.
// Reads ?field=email plus the field value from a same-named param (e.g. ?field=email&email=user@example.com).
func (h *handler) Validate(c *web.Context) (web.Response, error) {
	q := c.URL().Query()
	field := q.Get("field")
	value := q.Get(field)
	if value == "" {
		value = q.Get("value")
	}
	return render.Fragment(demo.ValidationResult(field, value))
}

// Paginate returns a paginated product table fragment.
func (h *handler) Paginate(c *web.Context) (web.Response, error) {
	q := c.URL().Query()
	page, _ := strconv.Atoi(q.Get("page"))        // defaults to 0 on invalid input
	perPage, _ := strconv.Atoi(q.Get("per_page")) // defaults to 0 on invalid input
	return render.Fragment(demo.PaginatedTable(page, perPage))
}

// OOBToast returns an out-of-band toast fragment. The trigger must use
// hx-swap="none" so HTMX processes only the OOB element in the response.
func (h *handler) OOBToast(c *web.Context) (web.Response, error) {
	return render.Fragment(demo.OOBToast())
}

// Region returns a select fragment of regions for the given country ID.
// Accepts the country as a path param (/region/{id}) or query param (?country=).
func (h *handler) Region(c *web.Context) (web.Response, error) {
	id := c.Param("id")
	if id == "" {
		id = c.URL().Query().Get("country")
	}
	return render.Fragment(demo.RegionOptions(id))
}
