package router

import (
	"context"
	"net/url"

	"github.com/go-sum/web"
)

func testContext(method, path string) *web.Context {
	u, _ := url.Parse("http://example.com" + path)
	req := web.NewRequest(method, u)
	return web.NewContext(context.Background(), req)
}
