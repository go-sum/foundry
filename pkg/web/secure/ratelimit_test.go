package secure

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/go-sum/web"
)

func TestMemoryStore_AllowsWithinRate(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		Rate:  10,
		Burst: 5,
	})

	for i := range 5 {
		allowed, _, err := store.Allow("client1")
		if err != nil {
			t.Fatalf("Allow(%d) error: %v", i, err)
		}
		if !allowed {
			t.Fatalf("Allow(%d) = false, want true (within burst)", i)
		}
	}
}

func TestMemoryStore_DeniesExceedingBurst(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		Rate:  10,
		Burst: 3,
	})

	// Exhaust the burst.
	for range 3 {
		if _, _, err := store.Allow("client1"); err != nil {
			t.Fatalf("Allow: %v", err)
		}
	}

	// Next request should be denied.
	allowed, _, err := store.Allow("client1")
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if allowed {
		t.Error("Allow() = true, want false (burst exceeded)")
	}
}

func TestMemoryStore_ReplenishesTokensOverTime(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		Rate:  100, // 100 per second = 1 token per 10ms
		Burst: 1,
	})

	// Use the one token.
	allowed, _, _ := store.Allow("client1")
	if !allowed {
		t.Fatal("first request should be allowed")
	}

	// Should be denied immediately.
	allowed, _, _ = store.Allow("client1")
	if allowed {
		t.Fatal("second request should be denied (no tokens)")
	}

	// Wait for token replenishment.
	time.Sleep(20 * time.Millisecond)

	allowed, _, err := store.Allow("client1")
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if !allowed {
		t.Error("Allow() = false after replenishment, want true")
	}
}

func TestMemoryStore_SeparateIdentifiers(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		Rate:  10,
		Burst: 1,
	})

	// Exhaust client1.
	if _, _, err := store.Allow("client1"); err != nil {
		t.Fatalf("Allow: %v", err)
	}

	// client2 should still be allowed.
	allowed, _, err := store.Allow("client2")
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if !allowed {
		t.Error("client2 should be allowed independently of client1")
	}
}

func TestRateLimit_Returns429WhenDenied(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		Rate:  10,
		Burst: 1,
	})

	mw := RateLimit(RateLimitConfig{
		Store: store,
		IdentifierFunc: func(_ *web.Context) string {
			return "test-client"
		},
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})

	// First request uses the burst token.
	resp, _ := handler(web.NewContext(context.Background(), req))
	if resp.Status != http.StatusOK {
		t.Fatalf("first request: status = %d, want %d", resp.Status, http.StatusOK)
	}

	// Second request should be rate limited.
	_, err := handler(web.NewContext(context.Background(), req))

	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error on rate limit, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusTooManyRequests {
		t.Fatalf("second request: error status = %d, want %d", webErr.Status, http.StatusTooManyRequests)
	}
	if webErr.Title != "Too Many Requests" {
		t.Errorf("error title = %q, want %q", webErr.Title, "Too Many Requests")
	}
}

func TestRateLimit_PassesThroughWhenAllowed(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		Rate:  100,
		Burst: 10,
	})

	mw := RateLimit(RateLimitConfig{
		Store: store,
		IdentifierFunc: func(_ *web.Context) string {
			return "test-client"
		},
	})

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called")
	}
}

func TestRateLimit_SkipperBypassesRateLimiting(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		Rate:  10,
		Burst: 1,
	})

	mw := RateLimit(RateLimitConfig{
		Store: store,
		IdentifierFunc: func(_ *web.Context) string {
			return "test-client"
		},
		Skipper: func(_ *web.Context) bool {
			return true
		},
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})

	// Even after many requests, skipper should bypass rate limiting.
	for i := range 10 {
		resp, _ := handler(web.NewContext(context.Background(), req))
		if resp.Status != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i, resp.Status, http.StatusOK)
		}
	}
}

func TestRateLimit_DefaultIdentifierUsesRemoteAddr(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		Rate:  10,
		Burst: 1,
	})

	mw := RateLimit(RateLimitConfig{
		Store: store,
		// IdentifierFunc not set -- should default to RemoteAddr.
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("192.168.1.1:12345")

	// First request should succeed.
	resp, _ := handler(web.NewContext(context.Background(), req))
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}

	// Second request from same addr should be denied (burst=1).
	_, err := handler(web.NewContext(context.Background(), req))

	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error on rate limit, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", webErr.Status, http.StatusTooManyRequests)
	}

	// Request from different addr should succeed.
	req2 := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req2.SetRemoteAddr("10.0.0.1:54321")
	resp, _ = handler(web.NewContext(context.Background(), req2))
	if resp.Status != http.StatusOK {
		t.Fatalf("different addr status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestRateLimit_PanicsOnNilStore(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil store, got none")
		}
	}()
	RateLimit(RateLimitConfig{Store: nil})
}

func TestRateLimit_StoreError_FailsOpen(t *testing.T) {
	store := &fakeErrorStore{}

	mw := RateLimit(RateLimitConfig{
		Store: store,
		IdentifierFunc: func(_ *web.Context) string {
			return "client"
		},
	})

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d (fail open on store error)", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called on store error (should fail open)")
	}
}

type fakeErrorStore struct{}

func (f *fakeErrorStore) Allow(_ string) (bool, time.Duration, error) {
	return false, 0, io.ErrUnexpectedEOF
}

func TestNewMemoryStore_DefaultBurst(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		Rate: 5.5,
	})
	// Burst should be ceil(5.5) = 6.
	// We should be able to make 6 requests.
	for i := range 6 {
		allowed, _, _ := store.Allow("client")
		if !allowed {
			t.Fatalf("request %d denied, expected 6 requests allowed (burst=ceil(5.5)=6)", i)
		}
	}
	allowed, _, _ := store.Allow("client")
	if allowed {
		t.Error("7th request allowed, expected denial at burst=6")
	}
}

func TestNewMemoryStore_ZeroRate_MinimumBurst(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		Rate: 0,
	})
	// Burst should be at least 1.
	allowed, _, _ := store.Allow("client")
	if !allowed {
		t.Error("first request denied with minimum burst=1")
	}
	allowed, _, _ = store.Allow("client")
	if allowed {
		t.Error("second request allowed, expected denial at burst=1")
	}
}

func TestMemoryStore_RetryAfterSet(t *testing.T) {
	s := NewMemoryStore(MemoryStoreConfig{Rate: 1, Burst: 1})
	// First request: allowed
	ok, _, err := s.Allow("user")
	if err != nil || !ok {
		t.Fatalf("Allow #1: ok=%v err=%v", ok, err)
	}
	// Second request: denied; retryAfter should be > 0
	ok, retryAfter, err := s.Allow("user")
	if err != nil {
		t.Fatalf("Allow #2 err: %v", err)
	}
	if ok {
		t.Fatal("Allow #2: expected denied")
	}
	if retryAfter <= 0 {
		t.Errorf("retryAfter = %v, want > 0", retryAfter)
	}
}

func TestRateLimit_Denied_SetsRetryAfterHeader(t *testing.T) {
	s := NewMemoryStore(MemoryStoreConfig{Rate: 1, Burst: 1})
	mw := RateLimit(RateLimitConfig{Store: s})

	handler := mw(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	// First: allowed
	if _, err := handler(web.NewContext(context.Background(), req)); err != nil {
		t.Fatalf("first request (allowed): %v", err)
	}
	// Second: denied
	_, err := handler(web.NewContext(context.Background(), req))

	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error on rate limit, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", webErr.Status, http.StatusTooManyRequests)
	}
	if webErr.RetryAfter <= 0 {
		t.Error("RetryAfter not set on 429 error")
	}
}

func TestRealIPFromTrustedXFF_UsesHeaderOnlyForTrustedProxy(t *testing.T) {
	fn, err := RealIPFromTrustedXFF("10.0.0.0/8")
	if err != nil {
		t.Fatalf("RealIPFromTrustedXFF() error = %v", err)
	}

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("10.1.2.3:1234")
	req.Headers.Set("X-Forwarded-For", "203.0.113.9, 10.1.2.3")

	got := fn(web.NewContext(context.Background(), req))
	if got != "203.0.113.9" {
		t.Fatalf("got %q, want %q", got, "203.0.113.9")
	}

	req.SetRemoteAddr("192.168.1.10:1234")
	got = fn(web.NewContext(context.Background(), req))
	if got != "192.168.1.10:1234" {
		t.Fatalf("got %q, want %q", got, "192.168.1.10:1234")
	}
}

func TestRealIPFromForwarded_TrustedPeerValidForHeader(t *testing.T) {
	fn, err := RealIPFromForwarded("10.0.0.0/8")
	if err != nil {
		t.Fatalf("RealIPFromForwarded() error = %v", err)
	}

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("10.1.2.3:1234")
	req.Headers.Set("Forwarded", "for=1.2.3.4")

	got := fn(web.NewContext(context.Background(), req))
	if got != "1.2.3.4" {
		t.Errorf("got %q, want %q", got, "1.2.3.4")
	}
}

func TestRealIPFromForwarded_TrustedPeerIPv6ForHeader(t *testing.T) {
	fn, err := RealIPFromForwarded("10.0.0.0/8")
	if err != nil {
		t.Fatalf("RealIPFromForwarded() error = %v", err)
	}

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("10.1.2.3:1234")
	req.Headers.Set("Forwarded", `for="[::1]"`)

	got := fn(web.NewContext(context.Background(), req))
	if got != "::1" {
		t.Errorf("got %q, want %q", got, "::1")
	}
}

func TestRealIPFromForwarded_UntrustedPeerIgnoresHeader(t *testing.T) {
	fn, err := RealIPFromForwarded("10.0.0.0/8")
	if err != nil {
		t.Fatalf("RealIPFromForwarded() error = %v", err)
	}

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("192.168.1.10:4321")
	req.Headers.Set("Forwarded", "for=1.2.3.4")

	got := fn(web.NewContext(context.Background(), req))
	if got != "192.168.1.10:4321" {
		t.Errorf("got %q, want %q", got, "192.168.1.10:4321")
	}
}

func TestRealIPFromForwarded_AbsentHeader_ReturnsPeerIP(t *testing.T) {
	fn, err := RealIPFromForwarded("10.0.0.0/8")
	if err != nil {
		t.Fatalf("RealIPFromForwarded() error = %v", err)
	}

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("10.1.2.3:1234")
	// No Forwarded header set.

	got := fn(web.NewContext(context.Background(), req))
	if got != "10.1.2.3" {
		t.Errorf("got %q, want %q", got, "10.1.2.3")
	}
}

func TestRealIPFromForwarded_MalformedForValue_FallsBackToRemoteAddr(t *testing.T) {
	fn, err := RealIPFromForwarded("10.0.0.0/8")
	if err != nil {
		t.Fatalf("RealIPFromForwarded() error = %v", err)
	}

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("10.1.2.3:1234")
	req.Headers.Set("Forwarded", "for=not-an-ip")

	got := fn(web.NewContext(context.Background(), req))
	if got != "10.1.2.3" {
		t.Errorf("got %q, want %q", got, "10.1.2.3")
	}
}

func TestRealIPFromForwarded_EmptyTrustedList_AlwaysReturnsRemoteAddr(t *testing.T) {
	fn, err := RealIPFromForwarded()
	if err != nil {
		t.Fatalf("RealIPFromForwarded() error = %v", err)
	}

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("10.1.2.3:1234")
	req.Headers.Set("Forwarded", "for=1.2.3.4")

	got := fn(web.NewContext(context.Background(), req))
	if got != "10.1.2.3:1234" {
		t.Errorf("got %q, want %q", got, "10.1.2.3:1234")
	}
}
