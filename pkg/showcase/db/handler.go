package db

import (
	"github.com/go-sum/showcase"
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
	tables, err := listTables(c.Context(), h.cfg.Pool, h.cfg.Schema)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	return h.cfg.Page(c, "Database Tables", indexContent(h.cfg.BasePath, tables))
}

func (h *handler) Table(c *web.Context) (web.Response, error) {
	tableName := c.Param("table")

	valid, err := validateTable(c.Context(), h.cfg.Pool, h.cfg.Schema, tableName)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	if !valid {
		return web.Response{}, web.ErrNotFound("table not found")
	}

	columns, err := tableColumns(c.Context(), h.cfg.Pool, h.cfg.Schema, tableName)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	indexes, err := tableIndexes(c.Context(), h.cfg.Pool, h.cfg.Schema, tableName)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}

	pg := showcase.ParsePager(c, h.cfg.PerPage, h.cfg.MaxPerPage)
	td, err := queryTableData(c.Context(), h.cfg.Pool, h.cfg.Schema, tableName, pg.Limit(), pg.Offset())
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	pg.SetTotal(td.Total)

	content := tableDetailContent(h.cfg.BasePath, tableName, columns, indexes, td, pg)
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(content)
	}
	return h.cfg.Page(c, "Table: "+tableName, content)
}

func (h *handler) TableData(c *web.Context) (web.Response, error) {
	tableName := c.Param("table")

	valid, err := validateTable(c.Context(), h.cfg.Pool, h.cfg.Schema, tableName)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	if !valid {
		return web.Response{}, web.ErrNotFound("table not found")
	}

	pg := showcase.ParsePager(c, h.cfg.PerPage, h.cfg.MaxPerPage)
	td, err := queryTableData(c.Context(), h.cfg.Pool, h.cfg.Schema, tableName, pg.Limit(), pg.Offset())
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	pg.SetTotal(td.Total)

	return render.Fragment(dataRegion(h.cfg.BasePath, tableName, td, pg))
}
