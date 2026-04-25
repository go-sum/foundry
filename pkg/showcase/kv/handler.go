package kv

import (
	"strconv"

	"github.com/go-sum/componentry/patterns/pager"
	"github.com/go-sum/web"
	"github.com/go-sum/web/htmx"
	"github.com/go-sum/web/render"
)

type handler struct {
	cfg Config
}

func newHandler(cfg Config) *handler {
	return &handler{cfg: cfg}
}

func (h *handler) Index(c *web.Context) (web.Response, error) {
	pg := parsePager(c, h.cfg.PerPage, h.cfg.MaxPerPage)
	result, err := listKeys(c.Context(), h.cfg.Store, "*", pg.Page, pg.PerPage)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	pg.SetTotal(result.Total)

	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(kvContentRegion(h.cfg.BasePath, result, pg))
	}
	return h.cfg.Page(c, "KV Store", indexContent(h.cfg.BasePath, result, pg))
}

func (h *handler) Key(c *web.Context) (web.Response, error) {
	key := c.Param("key")

	detail, err := getKeyDetail(c.Context(), h.cfg.Store, key)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	if !detail.Exists {
		return web.Response{}, web.ErrNotFound("key not found")
	}

	content := keyDetailContent(h.cfg.BasePath, detail)
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(content)
	}
	return h.cfg.Page(c, "Key: "+key, content)
}

func (h *handler) KeyValue(c *web.Context) (web.Response, error) {
	key := c.Param("key")

	detail, err := getKeyDetail(c.Context(), h.cfg.Store, key)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	if !detail.Exists {
		return web.Response{}, web.ErrNotFound("key not found")
	}

	return render.Fragment(valueRegion(h.cfg.BasePath, detail))
}

func parsePager(c *web.Context, defaultPerPage, maxPerPage int) pager.Pager {
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
