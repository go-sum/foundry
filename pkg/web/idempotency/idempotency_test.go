package idempotency_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/web"
	"github.com/go-sum/web/idempotency"
)

func makeContext(method string) *web.Context {
	u, _ := url.Parse("/test")
	req := web.NewRequest(method, u)
	return web.NewContext(context.Background(), req)
}

func makeContextWithHeader(method, headerKey, headerVal string) *web.Context {
	c := makeContext(method)
	c.Headers.Set(headerKey, headerVal)
	return c
}

func TestMiddleware_CacheMissExecutesHandler(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mw := idempotency.Middleware(store, time.Minute)

	called := 0
	handler := mw(func(c *web.Context) (web.Response, error) {
		called++
		return web.Text(http.StatusCreated, "created"), nil
	})

	c := makeContextWithHeader(http.MethodPost, "Idempotency-Key", "key-1")
	resp, err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusCreated)
	}
	if called != 1 {
		t.Fatalf("handler called %d times, want 1", called)
	}
}

func TestMiddleware_CacheHitReplaysResponse(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mw := idempotency.Middleware(store, time.Minute)

	called := 0
	handler := mw(func(c *web.Context) (web.Response, error) {
		called++
		return web.Text(http.StatusCreated, "created"), nil
	})

	c := makeContextWithHeader(http.MethodPost, "Idempotency-Key", "key-2")
	_, _ = handler(c)

	// Second request with same key — should replay from cache.
	c2 := makeContextWithHeader(http.MethodPost, "Idempotency-Key", "key-2")
	resp, err := handler(c2)
	if err != nil {
		t.Fatalf("unexpected error on cache hit: %v", err)
	}
	if resp.Status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusCreated)
	}
	if called != 1 {
		t.Fatalf("handler called %d times, want 1 (cache hit should not call handler)", called)
	}

	// Verify body is replayed.
	if resp.Body != nil {
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "created") {
			t.Fatalf("cached body = %q, want to contain 'created'", body)
		}
	}
}

func TestMiddleware_NonPostDeletePassesThrough(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mw := idempotency.Middleware(store, time.Minute)

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPatch} {
		called := 0
		handler := mw(func(c *web.Context) (web.Response, error) {
			called++
			return web.Text(http.StatusOK, "ok"), nil
		})

		c := makeContextWithHeader(method, "Idempotency-Key", "key-3")
		_, err := handler(c)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", method, err)
		}
		if called != 1 {
			t.Fatalf("%s: handler called %d times, want 1", method, called)
		}
	}
}

func TestMiddleware_MissingIdempotencyKeyPassesThrough(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mw := idempotency.Middleware(store, time.Minute)

	called := 0
	handler := mw(func(c *web.Context) (web.Response, error) {
		called++
		return web.Text(http.StatusCreated, "created"), nil
	})

	// POST without Idempotency-Key header.
	c := makeContext(http.MethodPost)
	_, err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Fatalf("handler called %d times, want 1", called)
	}
}

func TestMiddleware_StoreFailurePassesThrough(t *testing.T) {
	store := &failingStore{err: errors.New("store unavailable")}
	mw := idempotency.Middleware(store, time.Minute)

	called := 0
	handler := mw(func(c *web.Context) (web.Response, error) {
		called++
		return web.Text(http.StatusCreated, "created"), nil
	})

	c := makeContextWithHeader(http.MethodPost, "Idempotency-Key", "key-4")
	resp, err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusCreated)
	}
	if called != 1 {
		t.Fatalf("handler called %d times, want 1 (store failure should pass through)", called)
	}
}

// failingStore always returns an error from Get.
type failingStore struct {
	err error
}

func (s *failingStore) Get(_ context.Context, _ string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s *failingStore) Put(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return s.err
}
