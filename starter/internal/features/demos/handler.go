package demos

import (
	"strconv"

	"github.com/go-sum/componentry/showcase"
	"github.com/go-sum/componentry/showcase/demo"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
	"github.com/go-sum/web/router"
)

// Handler serves the component showcase page and live demo fragment endpoints.
type Handler struct {
	rt      *router.Router
	reqOpts []view.RequestOption
}

// NewHandler creates a new demos Handler.
func NewHandler(rt *router.Router, opts ...view.RequestOption) *Handler {
	return &Handler{rt: rt, reqOpts: opts}
}

// Show renders the component showcase page.
func (h *Handler) Show(c *web.Context) (web.Response, error) {
	vr := view.NewRequest(c, h.reqOpts...)
	return view.Render(vr, vr.Page("Component Showcase", showcase.Showcase()), nil)
}

// Search returns a filtered user table fragment for the live search demo.
func (h *Handler) Search(c *web.Context) (web.Response, error) {
	return render.Fragment(demo.SearchResults(c.URL().Query().Get("q")))
}

// Validate returns an inline validation fragment for the given field and value.
// Reads ?field=email plus the field value from a same-named param (e.g. ?field=email&email=user@example.com).
func (h *Handler) Validate(c *web.Context) (web.Response, error) {
	q := c.URL().Query()
	field := q.Get("field")
	value := q.Get(field)
	if value == "" {
		value = q.Get("value")
	}
	return render.Fragment(demo.ValidationResult(field, value))
}

// Paginate returns a paginated product table fragment.
func (h *Handler) Paginate(c *web.Context) (web.Response, error) {
	q := c.URL().Query()
	page, _ := strconv.Atoi(q.Get("page"))         // defaults to 0 on invalid input
	perPage, _ := strconv.Atoi(q.Get("per_page")) // defaults to 0 on invalid input
	return render.Fragment(demo.PaginatedTable(page, perPage))
}

// OOBToast returns an out-of-band toast fragment. The trigger must use
// hx-swap="none" so HTMX processes only the OOB element in the response.
func (h *Handler) OOBToast(c *web.Context) (web.Response, error) {
	return render.Fragment(demo.OOBToast())
}

// Region returns a select fragment of regions for the given country ID.
// Accepts the country as a path param (/region/{id}) or query param (?country=).
func (h *Handler) Region(c *web.Context) (web.Response, error) {
	id := c.Param("id")
	if id == "" {
		id = c.URL().Query().Get("country")
	}
	return render.Fragment(demo.RegionOptions(id))
}
