package secure

import (
	"cmp"
	"errors"
	"math"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	validator "github.com/go-playground/validator/v10"
	"github.com/go-sum/web"
	"github.com/go-sum/web/headers"
)

// RateLimitConfig configures rate limiting middleware.
type RateLimitConfig struct {
	// Store implements the rate limit check. Required.
	Store RateLimitStore `validate:"required"`

	// IdentifierFunc extracts the client identifier from the request.
	// Defaults to RemoteAddr.
	IdentifierFunc func(c *web.Context) string

	// FailClosed returns 503 when the backing store errors instead of allowing
	// the request through.
	FailClosed bool

	// OnError is called when the store returns an error.
	OnError func(err error, c *web.Context)

	// Skipper returns true to skip rate limiting for the request.
	Skipper func(c *web.Context) bool
}

// RateLimitStore checks whether a request from the given identifier is allowed.
// Allow returns ok=true if the request is within limits.
// If ok=false, retryAfter is the suggested wait duration (may be 0 if unknown).
type RateLimitStore interface {
	Allow(identifier string) (ok bool, retryAfter time.Duration, err error)
}

// RateLimit returns middleware that enforces per-identifier rate limits.
// Denied requests receive a 429 Too Many Requests response.
func RateLimit(cfg RateLimitConfig) web.Middleware {
	if err := validator.New().Struct(&cfg); err != nil {
		panic(err)
	}
	if cfg.IdentifierFunc == nil {
		cfg.IdentifierFunc = func(c *web.Context) string {
			return c.Request.RemoteAddr()
		}
	}

	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}

			id := cfg.IdentifierFunc(c)
			allowed, retryAfter, err := cfg.Store.Allow(id)
			if err != nil {
				wrapped := errors.Join(web.ErrTransient, err)
				if cfg.OnError != nil {
					cfg.OnError(wrapped, c)
				}
				if cfg.FailClosed {
					return web.Response{}, web.ErrUnavailable("service unavailable", nil)
				}
				return next(c)
			}
			if !allowed {
				return web.Response{}, web.ErrTooManyRequests(retryAfter)
			}

			return next(c)
		}
	}
}

// MemoryStore is an in-memory token bucket rate limiter.
type MemoryStore struct {
	mu          sync.Mutex
	visitors    map[string]*visitor
	rate        float64
	burst       int
	expiresIn   time.Duration
	lastCleanup time.Time
}

type visitor struct {
	tokens   float64
	lastSeen time.Time
}

// MemoryStoreConfig configures the in-memory rate limit store.
type MemoryStoreConfig struct {
	// Rate is the number of requests allowed per second.
	Rate float64

	// Burst is the maximum burst size. Defaults to ceil(Rate), minimum 1.
	Burst int

	// ExpiresIn is how long to keep a visitor entry after last seen.
	// Defaults to 3 minutes.
	ExpiresIn time.Duration
}

// NewMemoryStore creates a new in-memory rate limit store.
func NewMemoryStore(cfg MemoryStoreConfig) *MemoryStore {
	if cfg.Burst <= 0 {
		cfg.Burst = int(math.Ceil(cfg.Rate))
		if cfg.Burst < 1 {
			cfg.Burst = 1
		}
	}
	cfg.ExpiresIn = cmp.Or(cfg.ExpiresIn, 3*time.Minute)
	return &MemoryStore{
		visitors:  make(map[string]*visitor),
		rate:      cfg.Rate,
		burst:     cfg.Burst,
		expiresIn: cfg.ExpiresIn,
	}
}

// Allow checks whether the identifier is within the rate limit.
func (s *MemoryStore) Allow(identifier string) (ok bool, retryAfter time.Duration, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.cleanupLocked(now)

	v, exists := s.visitors[identifier]
	if !exists {
		v = &visitor{tokens: float64(s.burst), lastSeen: now}
		s.visitors[identifier] = v
	}

	elapsed := now.Sub(v.lastSeen).Seconds()
	v.tokens += elapsed * s.rate
	if v.tokens > float64(s.burst) {
		v.tokens = float64(s.burst)
	}
	v.lastSeen = now

	if v.tokens < 1 {
		// Compute seconds until next token is available.
		needed := 1 - v.tokens
		waitSec := needed / s.rate
		return false, time.Duration(waitSec*float64(time.Second)) + time.Second, nil
	}
	v.tokens--
	return true, 0, nil
}

func (s *MemoryStore) cleanupLocked(now time.Time) {
	if now.Sub(s.lastCleanup) < 30*time.Second {
		return
	}
	s.lastCleanup = now
	for id, v := range s.visitors {
		if now.Sub(v.lastSeen) > s.expiresIn {
			delete(s.visitors, id)
		}
	}
}

// RealIPFromTrustedXFF returns an identifier function that only trusts the
// X-Forwarded-For header when the direct peer is within one of the trusted
// proxy prefixes. When the peer is not trusted, RemoteAddr is returned.
func RealIPFromTrustedXFF(trustedProxies ...string) (func(c *web.Context) string, error) {
	prefixes := make([]netip.Prefix, 0, len(trustedProxies))
	for _, trustedProxy := range trustedProxies {
		prefix, err := netip.ParsePrefix(trustedProxy)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, prefix)
	}

	return func(c *web.Context) string {
		remoteIP, ok := parseRemoteIP(c.Request.RemoteAddr())
		if !ok || !trustedProxy(remoteIP, prefixes) {
			return c.Request.RemoteAddr()
		}

		xff := c.Headers.Get("X-Forwarded-For")
		if xff == "" {
			return remoteIP.String()
		}
		candidate := strings.TrimSpace(strings.Split(xff, ",")[0])
		if parsed, err := netip.ParseAddr(candidate); err == nil {
			return parsed.String()
		}
		return remoteIP.String()
	}, nil
}

// RealIPFromForwarded returns an identifier function that reads the client IP
// from the RFC 7239 Forwarded header "for" parameter, only when the direct
// peer is within one of the trusted proxy prefixes. Falls back to RemoteAddr
// when the peer is untrusted or the header is absent/unparseable.
func RealIPFromForwarded(trustedProxies ...string) (func(c *web.Context) string, error) {
	prefixes := make([]netip.Prefix, 0, len(trustedProxies))
	for _, trustedProxy := range trustedProxies {
		prefix, err := netip.ParsePrefix(trustedProxy)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, prefix)
	}

	return func(c *web.Context) string {
		remoteIP, ok := parseRemoteIP(c.Request.RemoteAddr())
		if !ok || !trustedProxy(remoteIP, prefixes) {
			return c.Request.RemoteAddr()
		}

		fwdHeader := c.Headers.Get("Forwarded")
		if fwdHeader == "" {
			return remoteIP.String()
		}

		elements, err := headers.ParseForwarded(fwdHeader)
		if err != nil || len(elements) == 0 || elements[0].For == "" {
			return remoteIP.String()
		}

		forVal := elements[0].For
		// Strip port if present: handle both host:port and [ipv6]:port forms.
		host, _, err := net.SplitHostPort(forVal)
		if err != nil {
			// No port component — use as-is.
			host = forVal
		}
		// Strip surrounding brackets from IPv6 literals.
		host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")

		addr, err := netip.ParseAddr(host)
		if err != nil {
			return remoteIP.String()
		}
		return addr.Unmap().String()
	}, nil
}

func parseRemoteIP(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func trustedProxy(addr netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
