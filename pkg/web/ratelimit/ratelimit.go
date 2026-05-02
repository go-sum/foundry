package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/headers"
)

// storageKeySeparator is a null byte — guaranteed not to appear in profile names or
// IP addresses, so profile+key composites can never collide across namespaces.
const storageKeySeparator = "\x00"

// Decision is the result of a rate-limit check.
type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
	Limit      int
	Remaining  int
	ResetAfter time.Duration
}

// Store executes the atomic token-consumption decision for a specific key and policy.
type Store interface {
	Allow(ctx context.Context, key string, policy Policy) (Decision, error)
}

// Limiter resolves named profiles and delegates token accounting to a store.
type Limiter struct {
	store    Store
	profiles map[RateLimitProfile]Policy
	logger   *slog.Logger
}

var (
	ErrStoreRequired   = errors.New("web/ratelimit: store is required")
	ErrProfileRequired = errors.New("web/ratelimit: profile is required")
	ErrKeyRequired     = errors.New("web/ratelimit: key is required")
	ErrProfileUnknown  = errors.New("web/ratelimit: unknown profile")
	ErrPolicyInvalid   = errors.New("web/ratelimit: invalid policy")
)

// New constructs a Limiter from cfg.
func New(cfg Config) (*Limiter, error) {
	if cfg.Store == nil {
		return nil, ErrStoreRequired
	}
	profiles := make(map[RateLimitProfile]Policy, len(cfg.Profiles))
	for name, policy := range cfg.Profiles {
		if name == "" {
			return nil, ErrProfileRequired
		}
		if err := policy.validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		profiles[name] = policy
	}
	limiter := &Limiter{
		store:    cfg.Store,
		profiles: profiles,
		logger:   cfg.Logger,
	}
	limiter.logStartup()
	return limiter, nil
}

// Allow checks whether key may consume one token from the named profile.
func (l *Limiter) Allow(ctx context.Context, profile RateLimitProfile, key string) (Decision, error) {
	if l == nil || l.store == nil {
		return Decision{}, ErrStoreRequired
	}
	if profile == "" {
		return Decision{}, ErrProfileRequired
	}
	if strings.TrimSpace(key) == "" {
		return Decision{}, ErrKeyRequired
	}
	policy, ok := l.profiles[profile]
	if !ok {
		return Decision{}, fmt.Errorf("%w: %s", ErrProfileUnknown, profile)
	}
	decision, err := l.store.Allow(ctx, storageKey(profile, key), policy)
	if err != nil {
		return Decision{}, err
	}
	l.logHit(ctx, profile, key, policy, decision)
	return decision, nil
}

// Ping probes the backing store when it supports connectivity checks.
func (l *Limiter) Ping(ctx context.Context) error {
	if l == nil || l.store == nil {
		return nil
	}
	if p, ok := l.store.(interface{ Ping(context.Context) error }); ok {
		return p.Ping(ctx)
	}
	return nil
}

// Close releases backing-store resources when supported.
func (l *Limiter) Close() error {
	if l == nil || l.store == nil {
		return nil
	}
	if c, ok := l.store.(interface{ Close() error }); ok {
		return c.Close()
	}
	return nil
}

func storageKey(profile RateLimitProfile, key string) string {
	return string(profile) + storageKeySeparator + key
}

func fullWindow(policy Policy) time.Duration {
	return time.Duration(policy.Capacity) * policy.RefillPer
}

func hitCount(decision Decision) int {
	return max(decision.Limit-decision.Remaining, 0)
}

func keyHash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:6])
}

func (l *Limiter) logStartup() {
	if l == nil || l.logger == nil {
		return
	}
	profiles := make([]RateLimitProfile, 0, len(l.profiles))
	for profile := range l.profiles {
		profiles = append(profiles, profile)
	}
	slices.Sort(profiles)
	for _, profile := range profiles {
		policy := l.profiles[profile]
		l.logger.Debug(
			"rate limit profile started",
			"profile", string(profile),
			"capacity", policy.Capacity,
			"refill_per", policy.RefillPer,
			"window", fullWindow(policy),
		)
	}
}

func (l *Limiter) logHit(ctx context.Context, profile RateLimitProfile, key string, policy Policy, decision Decision) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.DebugContext(
		ctx,
		"rate limit hit",
		"profile", string(profile),
		"key_hash", keyHash(key),
		"allowed", decision.Allowed,
		"hits", hitCount(decision),
		"limit", decision.Limit,
		"remaining", max(decision.Remaining, 0),
		"window", fullWindow(policy),
		"retry_after", decision.RetryAfter,
		"reset_after", decision.ResetAfter,
	)
}

// Middleware returns a web.Middleware that enforces the named rate-limit profile.
func Middleware(cfg MiddlewareConfig) (web.Middleware, error) {
	if cfg.Limiter == nil {
		return nil, ErrStoreRequired
	}
	if cfg.Profile == "" {
		return nil, ErrProfileRequired
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = FixedKey(string(cfg.Profile))
	}

	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}

			key, err := cfg.KeyFunc(c)
			if err != nil {
				return handleMiddlewareError(cfg, c, err, next)
			}

			decision, err := cfg.Limiter.Allow(c.Context(), cfg.Profile, key)
			if err != nil {
				return handleMiddlewareError(cfg, c, err, next)
			}
			if !decision.Allowed {
				return web.Response{}, tooManyRequests(decision)
			}

			resp, err := next(c)
			attachHeaders(&resp, decision)
			return resp, err
		}
	}, nil
}

func handleMiddlewareError(cfg MiddlewareConfig, c *web.Context, err error, next web.Handler) (web.Response, error) {
	wrapped := errors.Join(web.ErrTransient, err)
	if cfg.OnError != nil {
		cfg.OnError(wrapped, c)
	}
	if cfg.FailClosed {
		return web.Response{}, web.ErrUnavailable("service unavailable", wrapped)
	}
	return next(c)
}

func tooManyRequests(decision Decision) error {
	err := web.ErrTooManyRequests(decision.RetryAfter)
	if decision.Limit > 0 {
		err = err.WithHeader("X-RateLimit-Limit", strconv.Itoa(decision.Limit))
	}
	err = err.WithHeader("X-RateLimit-Remaining", strconv.Itoa(max(decision.Remaining, 0)))
	if decision.ResetAfter > 0 {
		err = err.WithHeader("X-RateLimit-Reset", strconv.Itoa(secondsCeil(decision.ResetAfter)))
	}
	return err
}

func attachHeaders(resp *web.Response, decision Decision) {
	if resp == nil {
		return
	}
	if decision.Limit > 0 {
		resp.Headers.Set("X-RateLimit-Limit", strconv.Itoa(decision.Limit))
	}
	resp.Headers.Set("X-RateLimit-Remaining", strconv.Itoa(max(decision.Remaining, 0)))
	if decision.ResetAfter > 0 {
		resp.Headers.Set("X-RateLimit-Reset", strconv.Itoa(secondsCeil(decision.ResetAfter)))
	}
}

func secondsCeil(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	secs := int(d / time.Second)
	if d%time.Second != 0 {
		secs++
	}
	if secs < 1 {
		return 1
	}
	return secs
}

// FixedKey returns a KeyFunc that always uses the same key.
func FixedKey(key string) KeyFunc {
	return func(_ *web.Context) (string, error) {
		return key, nil
	}
}

// BuildKey joins non-empty key parts with ":" for dynamic rate-limit keys.
func BuildKey(parts ...string) string {
	keyParts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		keyParts = append(keyParts, part)
	}
	return strings.Join(keyParts, ":")
}

// RealIP extracts the client IP from the request, stripping the TCP port and
// unmapping IPv4-mapped-IPv6 addresses. It returns ErrKeyRequired when the
// context is nil or the remote address cannot be parsed.
func RealIP(c *web.Context) (string, error) {
	if c == nil {
		return "", ErrKeyRequired
	}
	addr, ok := parseRemoteIP(c.Request.RemoteAddr())
	if !ok {
		return "", fmt.Errorf("%w: unparseable remote address %q", ErrKeyRequired, c.Request.RemoteAddr())
	}
	return addr.String(), nil
}

// RemoteAddrKey uses the request remote address verbatim as the limiter key.
func RemoteAddrKey(c *web.Context) (string, error) {
	return RealIP(c)
}

// RealIPFromTrustedXFF returns a key function that trusts X-Forwarded-For only
// when the immediate peer is in trustedProxies.
func RealIPFromTrustedXFF(trustedProxies ...string) (KeyFunc, error) {
	prefixes := make([]netip.Prefix, 0, len(trustedProxies))
	for _, trustedProxy := range trustedProxies {
		prefix, err := netip.ParsePrefix(trustedProxy)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, prefix)
	}

	return func(c *web.Context) (string, error) {
		if c == nil {
			return "", ErrKeyRequired
		}
		remoteIP, ok := parseRemoteIP(c.Request.RemoteAddr())
		if !ok || !trustedProxy(remoteIP, prefixes) {
			return RealIP(c)
		}

		xff := c.Headers().Get("X-Forwarded-For")
		if xff == "" {
			return remoteIP.String(), nil
		}
		entries := strings.Split(xff, ",")
		for i := len(entries) - 1; i >= 0; i-- {
			candidate := strings.TrimSpace(entries[i])
			if candidate == "" {
				continue
			}
			parsed, err := netip.ParseAddr(candidate)
			if err != nil {
				continue
			}
			parsed = parsed.Unmap()
			if trustedProxy(parsed, prefixes) {
				continue
			}
			return parsed.String(), nil
		}
		return remoteIP.String(), nil
	}, nil
}

// RealIPFromForwarded returns a key function that trusts RFC 7239 Forwarded
// only when the immediate peer is in trustedProxies.
func RealIPFromForwarded(trustedProxies ...string) (KeyFunc, error) {
	prefixes := make([]netip.Prefix, 0, len(trustedProxies))
	for _, trustedProxy := range trustedProxies {
		prefix, err := netip.ParsePrefix(trustedProxy)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, prefix)
	}

	return func(c *web.Context) (string, error) {
		if c == nil {
			return "", ErrKeyRequired
		}
		remoteIP, ok := parseRemoteIP(c.Request.RemoteAddr())
		if !ok || !trustedProxy(remoteIP, prefixes) {
			return RealIP(c)
		}

		fwdHeader := c.Headers().Get("Forwarded")
		if fwdHeader == "" {
			return remoteIP.String(), nil
		}

		elements, err := headers.ParseForwarded(fwdHeader)
		if err != nil || len(elements) == 0 {
			return remoteIP.String(), nil
		}
		for i := len(elements) - 1; i >= 0; i-- {
			forVal := elements[i].For
			if forVal == "" {
				continue
			}
			host, _, splitErr := net.SplitHostPort(forVal)
			if splitErr != nil {
				host = forVal
			}
			host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
			addr, parseErr := netip.ParseAddr(host)
			if parseErr != nil {
				continue
			}
			addr = addr.Unmap()
			if trustedProxy(addr, prefixes) {
				continue
			}
			return addr.String(), nil
		}
		return remoteIP.String(), nil
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
