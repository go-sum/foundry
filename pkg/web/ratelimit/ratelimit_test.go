package ratelimit

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web"
)

type errorStore struct {
	err error
}

type fakeKVStore struct {
	allowed    bool
	retryAfter time.Duration
	remaining  int
	resetAfter time.Duration
	err        error
}

const testProfile = "routes.auth"

func newTestLimiter(t *testing.T, capacity int) *Limiter {
	t.Helper()
	limiter, err := New(Config{
		Store: NewMemoryStore(MemoryStoreConfig{}),
		Profiles: map[string]Policy{
			testProfile: {
				Capacity:  capacity,
				RefillPer: time.Minute,
			},
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return limiter
}

func newDebugLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestMemoryStore_AllowsWithinCapacity(t *testing.T) {
	limiter := newTestLimiter(t, 3)
	for i := 0; i < 3; i++ {
		decision, err := limiter.Allow(context.Background(), testProfile, "client1")
		if err != nil {
			t.Fatalf("Allow(%d) error = %v", i, err)
		}
		if !decision.Allowed {
			t.Fatalf("Allow(%d) denied unexpectedly", i)
		}
	}
}

func TestMemoryStore_DeniesExceedingCapacity(t *testing.T) {
	limiter := newTestLimiter(t, 1)

	first, err := limiter.Allow(context.Background(), testProfile, "client1")
	if err != nil {
		t.Fatalf("first Allow() error = %v", err)
	}
	if !first.Allowed {
		t.Fatal("first Allow() denied unexpectedly")
	}

	decision, err := limiter.Allow(context.Background(), testProfile, "client1")
	if err != nil {
		t.Fatalf("second Allow() error = %v", err)
	}
	if decision.Allowed {
		t.Fatal("second Allow() = allowed, want denied")
	}
	if decision.RetryAfter <= 0 {
		t.Fatalf("RetryAfter = %v, want > 0", decision.RetryAfter)
	}
}

func TestMiddleware_Returns429WhenDenied(t *testing.T) {
	limiter := newTestLimiter(t, 1)
	mw, err := Middleware(MiddlewareConfig{
		Limiter: limiter,
		Profile: testProfile,
		KeyFunc: FixedKey("test-client"),
	})
	if err != nil {
		t.Fatalf("Middleware() error = %v", err)
	}
	handler := mw(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, err := handler(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("first request error = %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", resp.Status, http.StatusOK)
	}

	_, err = handler(web.NewContext(context.Background(), req))
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", webErr.Status, http.StatusTooManyRequests)
	}
}

func TestMiddleware_AttachesHeadersWhenAllowed(t *testing.T) {
	limiter := newTestLimiter(t, 2)
	mw, err := Middleware(MiddlewareConfig{
		Limiter: limiter,
		Profile: testProfile,
		KeyFunc: FixedKey("test-client"),
	})
	if err != nil {
		t.Fatalf("Middleware() error = %v", err)
	}
	handler := mw(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, err := handler(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if got := resp.Headers.Get("X-RateLimit-Limit"); got != "2" {
		t.Fatalf("X-RateLimit-Limit = %q, want %q", got, "2")
	}
	if got := resp.Headers.Get("X-RateLimit-Remaining"); got != "1" {
		t.Fatalf("X-RateLimit-Remaining = %q, want %q", got, "1")
	}
}

func TestMiddleware_DefaultKeyUsesProfile(t *testing.T) {
	limiter := newTestLimiter(t, 1)
	mw, err := Middleware(MiddlewareConfig{
		Limiter: limiter,
		Profile: testProfile,
	})
	if err != nil {
		t.Fatalf("Middleware() error = %v", err)
	}
	handler := mw(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	firstReq := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	firstReq.SetRemoteAddr("203.0.113.10:1234")
	firstResp, err := handler(web.NewContext(context.Background(), firstReq))
	if err != nil {
		t.Fatalf("first request error = %v", err)
	}
	if firstResp.Status != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", firstResp.Status, http.StatusOK)
	}

	secondReq := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	secondReq.SetRemoteAddr("198.51.100.44:1234")
	_, err = handler(web.NewContext(context.Background(), secondReq))
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", webErr.Status, http.StatusTooManyRequests)
	}
}

func TestMiddleware_FailOpenOnStoreError(t *testing.T) {
	limiter, err := New(Config{
		Store: errorStore{err: errors.New("store down")},
		Profiles: map[string]Policy{
			testProfile: {Capacity: 1, RefillPer: time.Minute},
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	mw, err := Middleware(MiddlewareConfig{
		Limiter: limiter,
		Profile: testProfile,
		KeyFunc: FixedKey("test-client"),
	})
	if err != nil {
		t.Fatalf("Middleware() error = %v", err)
	}
	called := false
	handler := mw(func(_ *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, err := handler(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Fatal("next handler was not called")
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
	got, err := fn(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("fn() error = %v", err)
	}
	if got != "203.0.113.9" {
		t.Fatalf("trusted peer key = %q, want %q", got, "203.0.113.9")
	}

	req = web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("192.168.1.10:1234")
	req.Headers.Set("X-Forwarded-For", "203.0.113.9, 10.1.2.3")
	got, err = fn(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("fn() error = %v", err)
	}
	if got != "192.168.1.10" {
		t.Fatalf("untrusted peer key = %q, want %q", got, "192.168.1.10")
	}
}

func TestNewStoreFromConfig_KV(t *testing.T) {
	store, err := NewStoreFromConfig(StoreConfig{
		Type:    StoreTypeKV,
		KVStore: fakeKVStore{},
	})
	if err != nil {
		t.Fatalf("NewStoreFromConfig() error = %v", err)
	}
	if store == nil {
		t.Fatal("NewStoreFromConfig() store = nil, want non-nil")
	}
}

func TestKVStore_Allow(t *testing.T) {
	store := NewKVStore(fakeKVStore{
		allowed:    true,
		retryAfter: 2 * time.Second,
		remaining:  4,
		resetAfter: 5 * time.Second,
	}, KVStoreConfig{})
	decision, err := store.Allow(context.Background(), "routes.all\x00global", Policy{
		Capacity:  5,
		RefillPer: time.Second,
	})
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if !decision.Allowed {
		t.Fatal("Allowed = false, want true")
	}
	if decision.Limit != 5 {
		t.Fatalf("Limit = %d, want %d", decision.Limit, 5)
	}
	if decision.Remaining != 4 {
		t.Fatalf("Remaining = %d, want %d", decision.Remaining, 4)
	}
	if decision.ResetAfter != 5*time.Second {
		t.Fatalf("ResetAfter = %v, want %v", decision.ResetAfter, 5*time.Second)
	}
}

func TestBuildKey(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "single", parts: []string{"email@example.com"}, want: "email@example.com"},
		{name: "multiple", parts: []string{"contact", "email@example.com"}, want: "contact:email@example.com"},
		{name: "skips empty", parts: []string{" contact ", "", " user@example.com "}, want: "contact:user@example.com"},
		{name: "all empty", parts: []string{"", "   "}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildKey(tt.parts...); got != tt.want {
				t.Fatalf("BuildKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLimiter_DebugLogsStartupAndHits(t *testing.T) {
	var buf bytes.Buffer
	limiter, err := New(Config{
		Store:  NewMemoryStore(MemoryStoreConfig{}),
		Logger: newDebugLogger(&buf),
		Profiles: map[string]Policy{
			testProfile: {
				Capacity:  2,
				RefillPer: time.Minute,
			},
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = limiter.Allow(context.Background(), testProfile, "alice@example.com")
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}

	logged := buf.String()
	assertLogLine(t, logged, "rate limit profile started",
		"profile="+testProfile,
		"capacity=2",
		"refill_per=1m0s",
		"window=2m0s",
	)
	assertLogLine(t, logged, "rate limit hit",
		"profile="+testProfile,
		"allowed=true",
		"hits=1",
		"limit=2",
		"remaining=1",
		"window=2m0s",
	)
	if strings.Contains(logged, "alice@example.com") {
		t.Fatalf("raw key leaked in log %q", logged)
	}
}

func TestMiddleware_NilLimiter_ReturnsError(t *testing.T) {
	_, err := Middleware(MiddlewareConfig{
		Profile: testProfile,
	})
	if !errors.Is(err, ErrStoreRequired) {
		t.Fatalf("error = %v, want %v", err, ErrStoreRequired)
	}
}

func TestMiddleware_EmptyProfile_ReturnsError(t *testing.T) {
	limiter := newTestLimiter(t, 1)
	_, err := Middleware(MiddlewareConfig{
		Limiter: limiter,
		Profile: "",
	})
	if !errors.Is(err, ErrProfileRequired) {
		t.Fatalf("error = %v, want %v", err, ErrProfileRequired)
	}
}

func TestMiddleware_FailClosed_PreservesErrorCause(t *testing.T) {
	sentinel := errors.New("store down")
	limiter, err := New(Config{
		Store: errorStore{err: sentinel},
		Profiles: map[string]Policy{
			testProfile: {Capacity: 1, RefillPer: time.Minute},
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	mw, err := Middleware(MiddlewareConfig{
		Limiter:    limiter,
		Profile:    testProfile,
		KeyFunc:    FixedKey("test-client"),
		FailClosed: true,
	})
	if err != nil {
		t.Fatalf("Middleware() error = %v", err)
	}
	handler := mw(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	_, handlerErr := handler(web.NewContext(context.Background(), req))

	var webErr *web.Error
	if !errors.As(handlerErr, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", handlerErr, handlerErr)
	}
	if webErr.Status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", webErr.Status, http.StatusServiceUnavailable)
	}
	if !errors.Is(handlerErr, web.ErrTransient) {
		t.Fatalf("expected errors.Is(err, web.ErrTransient), err = %v", handlerErr)
	}
	if !errors.Is(handlerErr, sentinel) {
		t.Fatalf("expected errors.Is(err, sentinel), err = %v", handlerErr)
	}
}

func TestRealIPFromForwarded_InvalidCIDR(t *testing.T) {
	_, err := RealIPFromForwarded("not-a-cidr")
	if err == nil {
		t.Fatal("RealIPFromForwarded() expected error for bad CIDR, got nil")
	}
}

func TestRealIPFromTrustedXFF_InvalidCIDR(t *testing.T) {
	_, err := RealIPFromTrustedXFF("not-a-cidr")
	if err == nil {
		t.Fatal("RealIPFromTrustedXFF() expected error for bad CIDR, got nil")
	}
}

func TestRealIP_StripsPort(t *testing.T) {
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("203.0.113.10:1234")
	got, err := RealIP(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("RealIP() error = %v", err)
	}
	if got != "203.0.113.10" {
		t.Fatalf("RealIP() = %q, want %q", got, "203.0.113.10")
	}
}

func TestRealIP_BareIP(t *testing.T) {
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("203.0.113.10")
	got, err := RealIP(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("RealIP() error = %v", err)
	}
	if got != "203.0.113.10" {
		t.Fatalf("RealIP() = %q, want %q", got, "203.0.113.10")
	}
}

func TestRealIP_UnmapsIPv6(t *testing.T) {
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("::ffff:127.0.0.1")
	got, err := RealIP(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("RealIP() error = %v", err)
	}
	if got != "127.0.0.1" {
		t.Fatalf("RealIP() = %q, want %q", got, "127.0.0.1")
	}
}

func TestRealIP_NilContext(t *testing.T) {
	_, err := RealIP(nil)
	if !errors.Is(err, ErrKeyRequired) {
		t.Fatalf("RealIP(nil) error = %v, want ErrKeyRequired", err)
	}
}

func TestRemoteAddrKey_StripsPort(t *testing.T) {
	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetRemoteAddr("198.51.100.5:9876")
	got, err := RemoteAddrKey(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("RemoteAddrKey() error = %v", err)
	}
	if got != "198.51.100.5" {
		t.Fatalf("RemoteAddrKey() = %q, want %q", got, "198.51.100.5")
	}
}

func TestRealIPFromForwarded(t *testing.T) {
	fn, err := RealIPFromForwarded("10.0.0.0/8")
	if err != nil {
		t.Fatalf("RealIPFromForwarded() error = %v", err)
	}

	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		want       string
	}{
		{
			name:       "trusted peer valid for",
			remoteAddr: "10.1.2.3:1234",
			forwarded:  "for=203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "untrusted peer ignores header",
			remoteAddr: "192.168.1.10:1234",
			forwarded:  "for=203.0.113.9",
			want:       "192.168.1.10",
		},
		{
			name:       "missing forwarded header",
			remoteAddr: "10.1.2.3:1234",
			forwarded:  "",
			want:       "10.1.2.3",
		},
		{
			name:       "unparseable for value",
			remoteAddr: "10.1.2.3:1234",
			forwarded:  "for=not-an-ip",
			want:       "10.1.2.3",
		},
		{
			name:       "ipv6 bracket notation",
			remoteAddr: "10.1.2.3:1234",
			forwarded:  `for="[::1]"`,
			want:       "::1",
		},
		{
			name:       "right-to-left skips trusted",
			remoteAddr: "10.1.2.3:1234",
			forwarded:  "for=6.6.6.6, for=203.0.113.9, for=10.0.0.5",
			want:       "203.0.113.9",
		},
		{
			name:       "all trusted falls back",
			remoteAddr: "10.1.2.3:1234",
			forwarded:  "for=10.0.0.1, for=10.0.0.2",
			want:       "10.1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
			req.SetRemoteAddr(tt.remoteAddr)
			if tt.forwarded != "" {
				req.Headers.Set("Forwarded", tt.forwarded)
			}
			got, err := fn(web.NewContext(context.Background(), req))
			if err != nil {
				t.Fatalf("fn() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("fn() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRealIPFromTrustedXFF_RightToLeft(t *testing.T) {
	tests := []struct {
		name         string
		trustedCIDRs []string
		remoteAddr   string
		xff          string
		want         string
	}{
		{
			name:         "spoofed leftmost ignored",
			trustedCIDRs: []string{"10.0.0.0/8"},
			remoteAddr:   "10.1.2.3:1234",
			xff:          "6.6.6.6, 203.0.113.9, 10.0.0.5",
			want:         "203.0.113.9",
		},
		{
			name:         "multi-hop skips trusted",
			trustedCIDRs: []string{"10.0.0.0/8", "172.16.0.0/12"},
			remoteAddr:   "10.1.2.3:1234",
			xff:          "1.2.3.4, 172.16.0.1, 10.0.0.5",
			want:         "1.2.3.4",
		},
		{
			name:         "all entries trusted",
			trustedCIDRs: []string{"10.0.0.0/8"},
			remoteAddr:   "10.1.2.3:1234",
			xff:          "10.0.0.1, 10.0.0.2",
			want:         "10.1.2.3",
		},
		{
			name:         "empty xff falls back",
			trustedCIDRs: []string{"10.0.0.0/8"},
			remoteAddr:   "10.1.2.3:1234",
			xff:          "",
			want:         "10.1.2.3",
		},
		{
			name:         "unparseable entry skipped",
			trustedCIDRs: []string{"10.0.0.0/8"},
			remoteAddr:   "10.1.2.3:1234",
			xff:          "garbage, 203.0.113.9",
			want:         "203.0.113.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := RealIPFromTrustedXFF(tt.trustedCIDRs...)
			if err != nil {
				t.Fatalf("RealIPFromTrustedXFF() error = %v", err)
			}
			req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
			req.SetRemoteAddr(tt.remoteAddr)
			if tt.xff != "" {
				req.Headers.Set("X-Forwarded-For", tt.xff)
			}
			got, err := fn(web.NewContext(context.Background(), req))
			if err != nil {
				t.Fatalf("fn() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("fn() = %q, want %q", got, tt.want)
			}
		})
	}
}

type clockCapturingStore struct {
	capturedNow time.Time
}

func (b *clockCapturingStore) RateLimitAllow(_ context.Context, _ string, _ int, _ time.Duration, now time.Time) (bool, time.Duration, int, time.Duration, error) {
	b.capturedNow = now
	return true, 0, 4, time.Second, nil
}

func TestKVStore_UsesInjectedClock(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	storage := &clockCapturingStore{}
	store := NewKVStore(storage, KVStoreConfig{
		Prefix: "test:",
		Now:    func() time.Time { return fixed },
	})

	_, _ = store.Allow(context.Background(), "key", Policy{Capacity: 5, RefillPer: time.Second})

	if !storage.capturedNow.Equal(fixed) {
		t.Fatalf("storage received now = %v, want %v", storage.capturedNow, fixed)
	}
}

func (s errorStore) Allow(_ context.Context, _ string, _ Policy) (Decision, error) {
	return Decision{}, s.err
}

func (b fakeKVStore) RateLimitAllow(_ context.Context, _ string, _ int, _ time.Duration, _ time.Time) (bool, time.Duration, int, time.Duration, error) {
	return b.allowed, b.retryAfter, b.remaining, b.resetAfter, b.err
}

func TestNewLimiter_MemoryStore(t *testing.T) {
	limiter, err := NewLimiter(
		StoreConfig{Type: StoreTypeMemory},
		map[string]Policy{"test": {Capacity: 5, RefillPer: time.Second}},
		nil,
	)
	if err != nil {
		t.Fatalf("NewLimiter() error = %v", err)
	}
	decision, err := limiter.Allow(context.Background(), "test", "key")
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if !decision.Allowed {
		t.Fatal("Allow() = false, want true")
	}
}

func TestNewLimiter_InvalidStoreConfig(t *testing.T) {
	_, err := NewLimiter(
		StoreConfig{Type: "unsupported"},
		map[string]Policy{"test": {Capacity: 1, RefillPer: time.Second}},
		nil,
	)
	if err == nil {
		t.Fatal("NewLimiter() error = nil, want non-nil for unsupported store type")
	}
}

func TestKVStore_ErrorOmitsRawKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "email and IP composite", key: "alice@example.com:192.168.1.1"},
		{name: "email only", key: "alice@example.com"},
		{name: "IP only", key: "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewKVStore(fakeKVStore{
				err: errors.New("connection refused"),
			}, KVStoreConfig{})

			compositeKey := storageKey("routes.auth", tt.key)
			_, err := store.Allow(context.Background(), compositeKey, Policy{
				Capacity:  5,
				RefillPer: time.Second,
			})
			if err == nil {
				t.Fatal("Allow() error = nil, want non-nil")
			}
			errStr := err.Error()
			if strings.Contains(errStr, tt.key) {
				t.Fatalf("error string contains raw key %q:\n  %s", tt.key, errStr)
			}
			wantHash := keyHash(compositeKey)
			if !strings.Contains(errStr, wantHash) {
				t.Fatalf("error string missing hashed key %q:\n  %s", wantHash, errStr)
			}
		})
	}
}

func assertLogLine(t *testing.T, output, msg string, kvPairs ...string) {
	t.Helper()
	lines := strings.Split(output, "\n")
	msgNeedle := `msg="` + msg + `"`
	var found string
	for _, line := range lines {
		if strings.Contains(line, msgNeedle) {
			found = line
			break
		}
	}
	if found == "" {
		t.Fatalf("log line with %s not found in output:\n%s", msgNeedle, output)
	}
	for _, kv := range kvPairs {
		if !strings.Contains(found, kv) {
			t.Errorf("log line missing %q:\n  line: %s", kv, found)
		}
	}
}
