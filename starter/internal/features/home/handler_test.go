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

func TestHandlerShow(t *testing.T) {
	h := NewHandler(func() []router.Route { return nil })

	u, _ := url.Parse("/")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	vr := view.NewRequest(c, nil)
	want := render.RenderNode(t, page.HomePage(vr))
	if string(body) != want {
		t.Fatalf("body mismatch")
	}

	req.Headers.Set("HX-Request", "true")
	c = web.NewContext(context.Background(), req)
	resp, err = h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	want = render.RenderNode(t, page.HomeContent(vr))
	if string(body) != want {
		t.Fatalf("partial body mismatch")
	}
}
