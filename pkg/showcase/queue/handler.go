package queue

import (
	"github.com/go-sum/foundry/pkg/showcase/base"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/htmx"
	"github.com/go-sum/foundry/pkg/web/render"
)

type handler struct {
	cfg Config
}

func newHandler(cfg Config) *handler {
	return &handler{cfg: cfg}
}

func (h *handler) Index(c *web.Context) (web.Response, error) {
	queues, err := listQueues(c.Context(), h.cfg.Pool)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	return h.cfg.Page(c, "Queues", indexContent(h.cfg.BasePath, queues))
}

func (h *handler) Detail(c *web.Context) (web.Response, error) {
	queueName := c.Param("queue")
	status := c.URL().Query().Get("status")
	pg := base.ParsePager(c, h.cfg.PerPage, h.cfg.MaxPerPage)
	jobs, total, err := listJobs(c.Context(), h.cfg.Pool, queueName, status, pg.Limit(), pg.Offset())
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	pg.SetTotal(total)
	counts, err := queueStatusCounts(c.Context(), h.cfg.Pool, queueName)
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	content := detailContent(h.cfg.BasePath, queueName, counts, status, jobs, pg)
	if htmx.IsHTMX(c) && !htmx.IsBoosted(c) {
		return render.Fragment(content)
	}
	return h.cfg.Page(c, "Queue: "+queueName, content)
}

func (h *handler) Jobs(c *web.Context) (web.Response, error) {
	queueName := c.Param("queue")
	status := c.URL().Query().Get("status")
	pg := base.ParsePager(c, h.cfg.PerPage, h.cfg.MaxPerPage)
	jobs, total, err := listJobs(c.Context(), h.cfg.Pool, queueName, status, pg.Limit(), pg.Offset())
	if err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	pg.SetTotal(total)
	return render.Fragment(jobsRegion(h.cfg.BasePath, queueName, status, jobs, pg))
}
