package home

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/page"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
	"github.com/go-sum/web/router"
)

func testRouter(t *testing.T) *router.Router {
	t.Helper()
	rt := router.New()
	router.Register(rt,
		router.GET("/hello/{name}", "hello.show", nil),
	)
	return rt
}

func TestHandlerShow(t *testing.T) {
	h := NewHandler(testRouter(t))

	u, _ := url.Parse("/")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	vr := view.NewRequest(c)
	want := render.RenderNode(t, page.HomePage(vr, "/hello/World"))
	if string(body) != want {
		t.Fatalf("body mismatch")
	}

	req.Headers.Set("HX-Request", "true")
	c = web.NewContext(context.Background(), req)
	resp, err = h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	want = render.RenderNode(t, page.HomeContent(vr, "/hello/World"))
	if string(body) != want {
		t.Fatalf("partial body mismatch")
	}
}
