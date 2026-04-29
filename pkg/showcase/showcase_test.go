package showcase

import (
	"context"
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/icons"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
	g "maragu.dev/gomponents"
)

func testPage(*web.Context, string, g.Node) (web.Response, error) {
	return web.Text(200, "ok"), nil
}

type fakeKVStore struct{}

func (fakeKVStore) Ping(context.Context) error                               { return nil }
func (fakeKVStore) Get(context.Context, string) ([]byte, error)              { return nil, nil }
func (fakeKVStore) Set(context.Context, string, []byte, kv.SetOptions) error { return nil }
func (fakeKVStore) Delete(context.Context, ...string) error                  { return nil }
func (fakeKVStore) Exists(context.Context, ...string) (int64, error)         { return 0, nil }
func (fakeKVStore) Close() error                                             { return nil }

func TestRoutes_AlwaysIncludesComponentry(t *testing.T) {
	rt := router.New()
	router.Register(rt, Routes(Config{
		Icons: icons.NewRegistry(),
		Page: func(c *web.Context, title string, content g.Node) (web.Response, error) {
			return testPage(c, title, content)
		},
	})...)

	if _, err := rt.Reverse("demos.showcase", nil); err != nil {
		t.Fatalf("Reverse(demos.showcase) error = %v", err)
	}
	if _, err := rt.Reverse("db.index", nil); err == nil {
		t.Fatal("Reverse(db.index) error = nil, want non-nil without DB")
	}
	if _, err := rt.Reverse("kv.index", nil); err == nil {
		t.Fatal("Reverse(kv.index) error = nil, want non-nil without KV")
	}
	if _, err := rt.Reverse("queue.index", nil); err == nil {
		t.Fatal("Reverse(queue.index) error = nil, want non-nil without DB")
	}
}

func TestRoutes_ConditionallyIncludesKV(t *testing.T) {
	rt := router.New()
	router.Register(rt, Routes(Config{
		Icons: icons.NewRegistry(),
		KV:    fakeKVStore{},
		Page: func(c *web.Context, title string, content g.Node) (web.Response, error) {
			return testPage(c, title, content)
		},
	})...)

	if _, err := rt.Reverse("kv.index", nil); err != nil {
		t.Fatalf("Reverse(kv.index) error = %v", err)
	}
}
