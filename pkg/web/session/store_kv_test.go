package session

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeKVEntry struct {
	data      []byte
	version   int64
	absolute  time.Time
	idleTTL   time.Duration
	expiresAt time.Time
}

type fakeKVStore struct {
	entries        map[string]fakeKVEntry
	readErr        error
	saveErr        error
	readCalls      int
	alwaysConflict bool
}

func newFakeKVStore() *fakeKVStore {
	return &fakeKVStore{entries: make(map[string]fakeKVEntry)}
}

func (b *fakeKVStore) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(b.entries, key)
	}
	return nil
}

func (b *fakeKVStore) SessionRead(_ context.Context, key string, now time.Time) ([]byte, int64, bool, error) {
	b.readCalls++
	if b.readErr != nil {
		return nil, 0, false, b.readErr
	}
	entry, ok := b.entries[key]
	if !ok {
		return nil, 0, false, nil
	}
	if !entry.expiresAt.IsZero() && !now.Before(entry.expiresAt) {
		delete(b.entries, key)
		return nil, 0, false, nil
	}
	entry.expiresAt = sessionExpiry(now, entry.absolute, entry.idleTTL)
	b.entries[key] = entry
	return entry.data, entry.version, true, nil
}

func (b *fakeKVStore) SessionSave(_ context.Context, key string, data []byte, absolute time.Time, idleTTL time.Duration, version int64, now time.Time) (int64, bool, bool, error) {
	if b.alwaysConflict {
		return 0, true, false, nil
	}
	if b.saveErr != nil {
		return 0, false, false, b.saveErr
	}
	if !absolute.IsZero() && !now.IsZero() && absolute.Before(now) {
		return 0, false, true, nil
	}
	entry, ok := b.entries[key]
	if !ok {
		if version != 0 {
			return 0, true, false, nil
		}
		b.entries[key] = fakeKVEntry{
			data:      append([]byte(nil), data...),
			version:   1,
			absolute:  absolute,
			idleTTL:   idleTTL,
			expiresAt: sessionExpiry(now, absolute, idleTTL),
		}
		return 1, false, false, nil
	}
	if entry.version != version {
		return 0, true, false, nil
	}
	entry.data = append([]byte(nil), data...)
	entry.version++
	entry.absolute = absolute
	entry.idleTTL = idleTTL
	entry.expiresAt = sessionExpiry(now, absolute, idleTTL)
	b.entries[key] = entry
	return entry.version, false, false, nil
}

func sessionExpiry(now, absolute time.Time, idleTTL time.Duration) time.Time {
	switch {
	case idleTTL > 0 && !absolute.IsZero():
		idleExpiry := now.Add(idleTTL)
		if idleExpiry.Before(absolute) {
			return idleExpiry
		}
		return absolute
	case idleTTL > 0:
		return now.Add(idleTTL)
	default:
		return absolute
	}
}

func TestKVStore_ReadWrite(t *testing.T) {
	kvs := newFakeKVStore()
	store := NewKVStore(kvs)

	ctx := context.Background()
	data := []byte(`{"user":"alice"}`)
	absolute := time.Now().Add(time.Hour)

	token, err := store.Save(ctx, "", data, absolute, 0, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if token == "" {
		t.Fatal("Save returned empty token")
	}
	if _, ok := kvs.entries["session:"+token]; !ok {
		t.Fatalf("expected namespaced key %q in kv store", "session:"+token)
	}

	got, version, err := store.Read(ctx, token)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("Read data = %q, want %q", got, data)
	}
	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}
}

func TestKVStore_CustomPrefix(t *testing.T) {
	kvs := newFakeKVStore()
	store := NewKVStore(kvs, KVStoreConfig{Prefix: "app-a:session:"})

	ctx := context.Background()
	token, err := store.Save(ctx, "", []byte(`{"user":"alice"}`), time.Now().Add(time.Hour), 0, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, ok := kvs.entries["app-a:session:"+token]; !ok {
		t.Fatalf("expected custom-prefixed key %q in kv store", "app-a:session:"+token)
	}
}

func TestKVStore_Update(t *testing.T) {
	kvs := newFakeKVStore()
	store := NewKVStore(kvs)

	ctx := context.Background()
	absolute := time.Now().Add(time.Hour)

	token, err := store.Save(ctx, "", []byte(`"v1"`), absolute, 0, 0)
	if err != nil {
		t.Fatalf("Save initial: %v", err)
	}
	_, version, err := store.Read(ctx, token)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if _, err := store.Save(ctx, token, []byte(`"v2"`), absolute, 0, version); err != nil {
		t.Fatalf("Save update: %v", err)
	}

	got, _, err := store.Read(ctx, token)
	if err != nil {
		t.Fatalf("Read updated: %v", err)
	}
	if string(got) != `"v2"` {
		t.Fatalf("updated data = %q, want %q", got, `"v2"`)
	}
}

func TestKVStore_VersionConflict(t *testing.T) {
	kvs := newFakeKVStore()
	store := NewKVStore(kvs)

	ctx := context.Background()
	absolute := time.Now().Add(time.Hour)

	token, err := store.Save(ctx, "", []byte(`"x"`), absolute, 0, 0)
	if err != nil {
		t.Fatalf("Save initial: %v", err)
	}
	_, version, err := store.Read(ctx, token)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if _, err := store.Save(ctx, token, []byte(`"a"`), absolute, 0, version); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if _, err := store.Save(ctx, token, []byte(`"b"`), absolute, 0, version); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("second Save error = %v, want ErrVersionConflict", err)
	}
}

func TestKVStore_Delete(t *testing.T) {
	kvs := newFakeKVStore()
	store := NewKVStore(kvs)

	ctx := context.Background()
	absolute := time.Now().Add(time.Hour)
	token, err := store.Save(ctx, "", []byte(`"x"`), absolute, 0, 0)
	if err != nil {
		t.Fatalf("Save initial: %v", err)
	}
	if err := store.Delete(ctx, token); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, _, err := store.Read(ctx, token); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Read after Delete = %v, want ErrSessionNotFound", err)
	}
}

func TestKVStore_AbsoluteExpiry(t *testing.T) {
	kvs := newFakeKVStore()
	store := NewKVStore(kvs)

	// Saving with an already-expired absolute deadline returns ErrSessionNotFound.
	_, err := store.Save(context.Background(), "", []byte(`"x"`), time.Now().Add(-time.Second), 0, 0)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Save expired = %v, want ErrSessionNotFound", err)
	}
}

func TestKVStore_IdleExpiry(t *testing.T) {
	kvs := newFakeKVStore()
	store := NewKVStore(kvs)
	token, err := store.Save(context.Background(), "", []byte(`"x"`), time.Now().Add(time.Hour), time.Millisecond, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, _, err := store.Read(context.Background(), token); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Read after idle expiry = %v, want ErrSessionNotFound", err)
	}
}

func TestKVStore_NotFound(t *testing.T) {
	store := NewKVStore(newFakeKVStore())
	if _, _, err := store.Read(context.Background(), "missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Read missing = %v, want ErrSessionNotFound", err)
	}
}

func TestKVStore_InvalidTokenRejectedBeforeKVRead(t *testing.T) {
	kvs := newFakeKVStore()
	store := NewKVStore(kvs)

	cases := []string{
		strings.Repeat("A", 42), // too short
		strings.Repeat("A", 44), // too long
		strings.Repeat("!", 43), // right length but invalid base64 characters
	}
	for _, token := range cases {
		if _, _, err := store.Read(context.Background(), token); !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("Read(%q) = %v, want ErrSessionNotFound", token, err)
		}
	}
	if kvs.readCalls != 0 {
		t.Fatalf("kv store readCalls = %d, want 0 (kv store must not be reached for invalid tokens)", kvs.readCalls)
	}
}

func TestKVStore_ReadError(t *testing.T) {
	store := NewKVStore(&fakeKVStore{readErr: context.DeadlineExceeded, entries: make(map[string]fakeKVEntry)})
	token, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken: %v", err)
	}
	if _, _, err := store.Read(context.Background(), token); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Read error = %v, want context.DeadlineExceeded", err)
	}
}

func TestKVStore_SaveError(t *testing.T) {
	want := errors.New("kv store unavailable")
	store := NewKVStore(&fakeKVStore{saveErr: want, entries: make(map[string]fakeKVEntry)})

	_, err := store.Save(context.Background(), "", []byte(`"x"`), time.Now().Add(time.Minute), time.Minute, 0)
	if err == nil {
		t.Fatal("Save error = nil, want non-nil")
	}
	if !errors.Is(err, want) {
		t.Fatalf("errors.Is(err, want) = false; err = %v", err)
	}
}

func TestKVStore_TokenCollisionExhaustion(t *testing.T) {
	kvs := &fakeKVStore{entries: make(map[string]fakeKVEntry), alwaysConflict: true}
	store := NewKVStore(kvs)

	_, err := store.Save(context.Background(), "", []byte(`"x"`), time.Now().Add(time.Hour), 0, 0)
	if err == nil {
		t.Fatal("Save error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unable to allocate unique token") {
		t.Fatalf("error = %v, want 'unable to allocate unique token'", err)
	}
}
