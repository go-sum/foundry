package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/go-sum/foundry/pkg/web"
)

type memEntry struct {
	data       []byte
	version    int64
	absolute   time.Time
	idleTTL    time.Duration
	lastAccess time.Time
}

func (e memEntry) expired() bool {
	now := time.Now()
	if !e.absolute.IsZero() && now.After(e.absolute) {
		return true
	}
	if e.idleTTL > 0 && now.After(e.lastAccess.Add(e.idleTTL)) {
		return true
	}
	return false
}

// MemoryStore is an in-memory Store kept primarily for tests.
// It exists so tests can exercise large server-side session payloads without
// hitting cookie-size limits in CookieStore or depending on a real KV service.
// In the starter app, the composition root rejects this store unless
// APP_ENV=testing so it cannot be selected accidentally in dev or production.
// A background goroutine sweeps expired entries every minute; call Stop to drain it.
type MemoryStore struct {
	mu       sync.RWMutex
	entries  map[string]memEntry
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewMemoryStore creates a test-oriented in-memory session store and starts its
// background sweep goroutine.
func NewMemoryStore() *MemoryStore {
	m := &MemoryStore{
		entries: make(map[string]memEntry),
		stopCh:  make(chan struct{}),
	}
	web.Go(nil, "session.memory.sweep", m.sweep)
	return m
}

// Stop shuts down the background sweep goroutine.
func (m *MemoryStore) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
}

func (m *MemoryStore) sweep() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.mu.Lock()
			for token, e := range m.entries {
				if e.expired() {
					delete(m.entries, token)
				}
			}
			m.mu.Unlock()
		case <-m.stopCh:
			return
		}
	}
}

// Read implements Store.
func (m *MemoryStore) Read(_ context.Context, token string) ([]byte, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[token]
	if !ok || entry.expired() {
		delete(m.entries, token)
		return nil, 0, ErrSessionNotFound
	}
	entry.lastAccess = time.Now()
	m.entries[token] = entry
	return entry.data, entry.version, nil
}

// Save implements Store.
func (m *MemoryStore) Save(_ context.Context, token string, data []byte, absolute time.Time, idleTTL time.Duration, version int64) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if token == "" {
		var err error
		token, err = randomToken()
		if err != nil {
			return "", err
		}
		m.entries[token] = memEntry{
			data:       data,
			version:    1,
			absolute:   absolute,
			idleTTL:    idleTTL,
			lastAccess: time.Now(),
		}
		return token, nil
	}

	if existing, ok := m.entries[token]; ok && existing.version != version {
		return "", ErrVersionConflict
	}
	m.entries[token] = memEntry{
		data:       data,
		version:    version + 1,
		absolute:   absolute,
		idleTTL:    idleTTL,
		lastAccess: time.Now(),
	}
	return token, nil
}

// Delete implements Store.
func (m *MemoryStore) Delete(_ context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, token)
	return nil
}

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("web/session: generating token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
