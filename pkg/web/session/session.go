// Package session provides HTTP session middleware backed by a pluggable Store.
//
// Two built-in stores are provided:
//   - MemoryStore: server-side sessions keyed by a random 256-bit ID.
//   - CookieStore: client-side sessions encoded directly in the cookie via a Codec.
//
// The Session type is safe for concurrent use from multiple goroutines.
package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/go-sum/web"
)

var contextKey = &struct{}{}

// sessionPayload is the JSON structure persisted in the backing store.
type sessionPayload struct {
	V map[string]json.RawMessage `json:"v,omitempty"` // session values
	F map[string]json.RawMessage `json:"f,omitempty"` // flash values for next request
}

// Session is request-scoped session state. All methods are safe for concurrent use.
type Session struct {
	mu           sync.RWMutex
	token        string                     // opaque store token (session ID or encoded blob)
	oldToken     string                     // previous token; deleted from store on Regenerate
	values       map[string]json.RawMessage // persistent key/value data
	currentFlash map[string]json.RawMessage // flash from previous request; available this request
	nextFlash    map[string]json.RawMessage // flash set this request; available next request
	version      int64                      // optimistic concurrency version from store
	fresh        bool                       // true when session was just created
	dirty        bool                       // true when session needs saving
	destroyed    bool                       // true when Destroy was called
	regenerated  bool                       // true when Regenerate was called
}

// ID returns the opaque session token (store key or encoded blob).
func (s *Session) ID() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

// IsNew reports whether the session was created during the current request.
func (s *Session) IsNew() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fresh
}

// Has reports whether key is present in the session.
func (s *Session) Has(key string) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.values[key]
	return ok
}

// Set marshals value and stores it under key. Calling Set marks the session dirty.
func (s *Session) Set(key string, value any) error {
	if s == nil {
		return fmt.Errorf("web/session: Set called on nil session")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("web/session: Set %q: %w", key, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = data
	s.dirty = true
	s.destroyed = false
	return nil
}

// Unset removes key from the session.
func (s *Session) Unset(key string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	s.dirty = true
}

// Keys returns the sorted session value keys.
func (s *Session) Keys() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(s.values))
	for k := range s.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Flash stores value under key for retrieval in the next request via FlashPop.
func (s *Session) Flash(key string, value any) error {
	if s == nil {
		return fmt.Errorf("web/session: Flash called on nil session")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("web/session: Flash %q: %w", key, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.nextFlash == nil {
		s.nextFlash = make(map[string]json.RawMessage)
	}
	s.nextFlash[key] = data
	s.dirty = true
	return nil
}

// Destroy clears all session data and schedules the cookie for deletion.
// The response will include a Clear-Site-Data header.
func (s *Session) Destroy() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = make(map[string]json.RawMessage)
	s.currentFlash = nil
	s.nextFlash = nil
	s.destroyed = true
	s.dirty = true
}

// Regenerate assigns a new session token while preserving session data.
// The old token is deleted from the store, preventing session fixation.
func (s *Session) Regenerate() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.oldToken = s.token
	s.token = ""
	s.regenerated = true
	s.dirty = true
}

// Get decodes a typed value from the session.
// Returns the zero value and false if key is absent.
func Get[T any](s *Session, key string) (T, bool, error) {
	var zero T
	if s == nil {
		return zero, false, nil
	}
	s.mu.RLock()
	raw, ok := s.values[key]
	s.mu.RUnlock()
	if !ok {
		return zero, false, nil
	}
	if err := json.Unmarshal(raw, &zero); err != nil {
		return zero, false, fmt.Errorf("web/session: Get %q: %w", key, err)
	}
	return zero, true, nil
}

// FlashPop retrieves and removes a flash value set by the previous request.
// Returns the zero value and false if key is absent. Unconsumed flash values
// are dropped at the end of the request (they are not re-saved).
func FlashPop[T any](s *Session, key string) (T, bool, error) {
	var zero T
	if s == nil {
		return zero, false, nil
	}
	s.mu.Lock()
	raw, ok := s.currentFlash[key]
	if ok {
		delete(s.currentFlash, key)
		s.dirty = true
	}
	s.mu.Unlock()
	if !ok {
		return zero, false, nil
	}
	if err := json.Unmarshal(raw, &zero); err != nil {
		return zero, false, fmt.Errorf("web/session: FlashPop %q: %w", key, err)
	}
	return zero, true, nil
}

// FromContext returns the session injected by Middleware.
func FromContext(c *web.Context) (*Session, bool) {
	return web.Get[*Session](c, contextKey)
}

func newSession() *Session {
	return &Session{
		values:       make(map[string]json.RawMessage),
		currentFlash: make(map[string]json.RawMessage),
		fresh:        true,
	}
}

func sessionFromData(data []byte, token string, version int64) *Session {
	var p sessionPayload
	if err := json.Unmarshal(data, &p); err != nil {
		// Corrupted data → start fresh.
		return newSession()
	}
	s := &Session{token: token, version: version}
	if p.V != nil {
		s.values = p.V
	} else {
		s.values = make(map[string]json.RawMessage)
	}
	if p.F != nil {
		s.currentFlash = p.F
	} else {
		s.currentFlash = make(map[string]json.RawMessage)
	}
	return s
}

// marshalPayload serializes values + nextFlash for storage.
// currentFlash is intentionally excluded — flash is one-shot.
func (s *Session) marshalPayload() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(sessionPayload{V: s.values, F: s.nextFlash})
}
