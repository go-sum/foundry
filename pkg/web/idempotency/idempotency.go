package idempotency

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-sum/foundry/pkg/web"
)

// Store persists idempotency records. Implementations must be durable across
// process restarts. See §11.2: the TTL must exceed the maximum retry window.
type Store interface {
	// Get returns (payload, true, nil) if a completed result exists for key.
	// Returns (nil, false, nil) if no record exists.
	Get(ctx context.Context, key string) (payload []byte, ok bool, err error)

	// Put stores payload under key with the given TTL.
	Put(ctx context.Context, key string, payload []byte, ttl time.Duration) error
}

// cachedResponse is the serialized form stored in the Store.
type cachedResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

// Middleware returns a web.Middleware that deduplicates non-idempotent
// requests (POST, DELETE) using the Idempotency-Key request header.
// On a cache hit the stored response is replayed verbatim. On a miss the
// handler runs and the response is stored before returning.
//
// PUT and PATCH are idempotent by HTTP semantics and are passed through.
// Requests without an Idempotency-Key header are passed through.
func Middleware(store Store, ttl time.Duration) web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			// Only apply to non-idempotent methods.
			method := c.Method()
			if method != http.MethodPost && method != http.MethodDelete {
				return next(c)
			}

			key := c.Headers().Get("Idempotency-Key")
			if key == "" {
				return next(c)
			}

			// Check cache.
			payload, ok, err := store.Get(c.Context(), key)
			if err != nil {
				// Store failure — pass through (fail open).
				return next(c)
			}
			if ok {
				var cached cachedResponse
				if jerr := json.Unmarshal(payload, &cached); jerr == nil {
					resp := web.Response{
						Status: cached.Status,
						Body:   io.NopCloser(bytes.NewReader(cached.Body)),
					}
					for k, v := range cached.Headers {
						resp.Headers.Set(k, v)
					}
					return resp, nil
				}
			}

			// Execute handler.
			resp, herr := next(c)
			if herr != nil {
				return resp, herr
			}

			// Store response.
			var body []byte
			if resp.Body != nil {
				buf := new(bytes.Buffer)
				_, _ = buf.ReadFrom(resp.Body)
				body = buf.Bytes()
				resp.Body = io.NopCloser(bytes.NewReader(body))
			}
			hdrs := make(map[string]string)
			// Capture response headers.
			resp.Headers.ForEach(func(name string, values []string) {
				if len(values) > 0 {
					hdrs[name] = values[0]
				}
			})
			cached := cachedResponse{Status: resp.Status, Headers: hdrs, Body: body}
			if data, merr := json.Marshal(cached); merr == nil {
				_ = store.Put(c.Context(), key, data, ttl)
			}
			return resp, nil
		}
	}
}

// MemoryStore is an in-memory idempotency store.
//
// WARNING: Not suitable for production — records are lost on process restart,
// allowing double-execution after failover. See §11.2.
type MemoryStore struct {
	mu      sync.Mutex
	records map[string]memRecord
}

type memRecord struct {
	payload   []byte
	expiresAt time.Time
}

// NewMemoryStore creates an in-memory Store for use in tests.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: make(map[string]memRecord)}
}

func (s *MemoryStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[key]
	if !ok || time.Now().After(r.expiresAt) {
		return nil, false, nil
	}
	return r.payload, true, nil
}

func (s *MemoryStore) Put(_ context.Context, key string, payload []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[key] = memRecord{payload: payload, expiresAt: time.Now().Add(ttl)}
	return nil
}

// ErrInFlight is returned by a Store implementation when a concurrent
// request for the same key is already in progress. Middleware implementations
// may use this to block until the in-flight request completes (not implemented
// by MemoryStore).
var ErrInFlight = errors.New("idempotency: request in flight")
