package session

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"
)

// KVStoreConfig controls how server-side sessions are keyed in the backing KV
// store.
type KVStoreConfig struct {
	Prefix string
}

// KVBackend is the minimal session-specific contract required by KVStore.
// It is consumer-owned by pkg/web/session rather than producer-owned by the
// generic KV module.
type KVBackend interface {
	SessionRead(ctx context.Context, key string, now time.Time) (data []byte, version int64, found bool, err error)
	SessionSave(ctx context.Context, key string, data []byte, absolute time.Time, idleTTL time.Duration, version int64, now time.Time) (nextVersion int64, conflict bool, expired bool, err error)
	Delete(ctx context.Context, keys ...string) error
}

// KVStore persists session records in a shared KV backend. It exists for
// production deployments that need server-side session state without using
// client-side cookie payloads.
type KVStore struct {
	backend KVBackend
	prefix  string
}

// tokenEncodedLen is the base64.RawURLEncoding length of a 32-byte random token
// (⌈32×8/6⌉ = 43 characters).
const tokenEncodedLen = 43

// DefaultKVStoreConfig returns the production default key namespace for KV
// backed sessions.
func DefaultKVStoreConfig() KVStoreConfig {
	return KVStoreConfig{Prefix: "session:"}
}

// NewKVStore builds a session store over a session-capable KV backend.
func NewKVStore(backend KVBackend, cfgs ...KVStoreConfig) *KVStore {
	if backend == nil {
		panic("web/session: KVStore backend must not be nil")
	}
	cfg := DefaultKVStoreConfig()
	if len(cfgs) > 0 && cfgs[0].Prefix != "" {
		cfg = cfgs[0]
	}
	return &KVStore{
		backend: backend,
		prefix:  cfg.Prefix,
	}
}

// Read implements Store.
func (s *KVStore) Read(ctx context.Context, token string) ([]byte, int64, error) {
	if !validKVSessionToken(token) {
		return nil, 0, ErrSessionNotFound
	}
	data, version, found, err := s.backend.SessionRead(ctx, s.key(token), time.Now())
	if err != nil {
		return nil, 0, fmt.Errorf("web/session: kv store read: %w", err)
	}
	if !found {
		return nil, 0, ErrSessionNotFound
	}
	return data, version, nil
}

// Save implements Store.
func (s *KVStore) Save(ctx context.Context, token string, data []byte, absolute time.Time, idleTTL time.Duration, version int64) (string, error) {
	now := time.Now()

	if token == "" {
		for range 4 {
			candidate, err := randomToken()
			if err != nil {
				return "", err
			}
			_, conflict, expired, err := s.backend.SessionSave(ctx, s.key(candidate), data, absolute, idleTTL, 0, now)
			if conflict {
				continue
			}
			if expired {
				return "", ErrSessionNotFound
			}
			if err != nil {
				return "", fmt.Errorf("web/session: kv store save: %w", err)
			}
			return candidate, nil
		}
		return "", fmt.Errorf("web/session: kv store save: unable to allocate unique token")
	}

	_, conflict, expired, err := s.backend.SessionSave(ctx, s.key(token), data, absolute, idleTTL, version, now)
	if conflict {
		return "", ErrVersionConflict
	}
	if expired {
		return "", ErrSessionNotFound
	}
	if err != nil {
		return "", fmt.Errorf("web/session: kv store save: %w", err)
	}
	return token, nil
}

// Delete implements Store.
func (s *KVStore) Delete(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	if err := s.backend.Delete(ctx, s.key(token)); err != nil {
		return fmt.Errorf("web/session: kv store delete: %w", err)
	}
	return nil
}

func (s *KVStore) key(token string) string {
	return s.prefix + token
}

func validKVSessionToken(token string) bool {
	if len(token) != tokenEncodedLen {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil
}
