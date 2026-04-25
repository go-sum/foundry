package showcase

import (
	"strconv"

	"github.com/go-sum/componentry/patterns/pager"
	"github.com/go-sum/web"
	g "maragu.dev/gomponents"
)

// PageFunc renders a full HTML page within the host application's layout.
// Each showcase subpackage accepts a PageFunc in its Config so that the
// package stays decoupled from starter/internal/view.
type PageFunc func(c *web.Context, title string, content g.Node) (web.Response, error)

// ParsePager reads page and per_page query params from c and returns a Pager
// configured with defaultPerPage and maxPerPage.
func ParsePager(c *web.Context, defaultPerPage, maxPerPage int) pager.Pager {
	q := c.URL().Query()
	page := 1
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		page = p
	}
	perPage := defaultPerPage
	if pp, err := strconv.Atoi(q.Get("per_page")); err == nil && pp > 0 {
		perPage = pp
	}
	if maxPerPage > 0 && perPage > maxPerPage {
		perPage = maxPerPage
	}
	return pager.Pager{Page: page, PerPage: perPage}
}
