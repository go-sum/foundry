package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"
)

// MemoryStoreConfig controls the lifecycle of idle keys in MemoryStore.
type MemoryStoreConfig struct {
	ExpiresIn time.Duration
	Now       func() time.Time
}

// MemoryStore is an in-memory token bucket store for tests and single-process runtimes.
type MemoryStore struct {
	mu          sync.Mutex
	visitors    map[string]*visitor
	expiresIn   time.Duration
	lastCleanup time.Time
	now         func() time.Time
}

type visitor struct {
	tokens   float64
	lastSeen time.Time
}

// NewMemoryStore creates a MemoryStore.
func NewMemoryStore(cfg MemoryStoreConfig) *MemoryStore {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	expiresIn := cfg.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3 * time.Minute
	}
	return &MemoryStore{
		visitors:  make(map[string]*visitor),
		expiresIn: expiresIn,
		now:       now,
	}
}

// Allow consumes a token from key according to policy.
func (s *MemoryStore) Allow(_ context.Context, key string, policy Policy) (Decision, error) {
	if err := policy.validate(); err != nil {
		return Decision{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	s.cleanupLocked(now)

	v, exists := s.visitors[key]
	if !exists {
		v = &visitor{
			tokens:   float64(policy.Capacity),
			lastSeen: now,
		}
		s.visitors[key] = v
	}

	refill := float64(now.Sub(v.lastSeen)) / float64(policy.RefillPer)
	v.tokens += refill
	if v.tokens > float64(policy.Capacity) {
		v.tokens = float64(policy.Capacity)
	}
	v.lastSeen = now

	decision := Decision{
		Limit: policy.Capacity,
	}

	if v.tokens < 1 {
		needed := 1 - v.tokens
		wait := time.Duration(math.Ceil(needed * float64(policy.RefillPer)))
		if wait < time.Second {
			wait = time.Second
		}
		decision.Allowed = false
		decision.RetryAfter = wait
		decision.Remaining = 0
		decision.ResetAfter = fullResetAfter(policy, v.tokens)
		return decision, nil
	}

	v.tokens--
	decision.Allowed = true
	decision.Remaining = int(math.Floor(v.tokens))
	decision.ResetAfter = fullResetAfter(policy, v.tokens)
	return decision, nil
}

func fullResetAfter(policy Policy, tokens float64) time.Duration {
	missing := float64(policy.Capacity) - tokens
	if missing <= 0 {
		return 0
	}
	return time.Duration(math.Ceil(missing * float64(policy.RefillPer)))
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
