package session

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestMemoryStore_ReadWrite(t *testing.T) {
	store := NewMemoryStore()
	defer store.Stop()

	ctx := context.Background()
	data := []byte(`{"v":{"k":"\"v\""}}`)
	absolute := time.Now().Add(time.Hour)

	token, err := store.Save(ctx, "", data, absolute, 0, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if token == "" {
		t.Fatal("Save returned empty token")
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

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()
	defer store.Stop()

	ctx := context.Background()
	abs := time.Now().Add(time.Hour)

	token, _ := store.Save(ctx, "", []byte(`"v1"`), abs, 0, 0)
	_, ver, _ := store.Read(ctx, token)

	_, err := store.Save(ctx, token, []byte(`"v2"`), abs, 0, ver)
	if err != nil {
		t.Fatalf("Save update: %v", err)
	}

	got, _, _ := store.Read(ctx, token)
	if string(got) != `"v2"` {
		t.Fatalf("updated data = %q, want v2", got)
	}
}

func TestMemoryStore_VersionConflict(t *testing.T) {
	store := NewMemoryStore()
	defer store.Stop()

	ctx := context.Background()
	abs := time.Now().Add(time.Hour)

	token, _ := store.Save(ctx, "", []byte(`"x"`), abs, 0, 0)
	_, ver, _ := store.Read(ctx, token)

	// First writer succeeds.
	_, err1 := store.Save(ctx, token, []byte(`"a"`), abs, 0, ver)
	// Second writer with old version fails.
	_, err2 := store.Save(ctx, token, []byte(`"b"`), abs, 0, ver)

	if err1 != nil {
		t.Fatalf("first Save: %v", err1)
	}
	if !errors.Is(err2, ErrVersionConflict) {
		t.Fatalf("second Save error = %v, want ErrVersionConflict", err2)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	defer store.Stop()

	ctx := context.Background()
	abs := time.Now().Add(time.Hour)

	token, _ := store.Save(ctx, "", []byte(`"x"`), abs, 0, 0)
	if err := store.Delete(ctx, token); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, _, err := store.Read(ctx, token)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Read after Delete = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_AbsoluteExpiry(t *testing.T) {
	store := NewMemoryStore()
	defer store.Stop()

	ctx := context.Background()
	// Already expired.
	past := time.Now().Add(-time.Second)

	token, _ := store.Save(ctx, "", []byte(`"x"`), past, 0, 0)
	_, _, err := store.Read(ctx, token)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Read expired = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_IdleExpiry(t *testing.T) {
	store := NewMemoryStore()
	defer store.Stop()

	ctx := context.Background()
	abs := time.Now().Add(time.Hour)
	idle := time.Millisecond // tiny idle TTL

	token, _ := store.Save(ctx, "", []byte(`"x"`), abs, idle, 0)
	time.Sleep(5 * time.Millisecond)

	_, _, err := store.Read(ctx, token)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Read after idle expiry = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_NotFound(t *testing.T) {
	store := NewMemoryStore()
	defer store.Stop()

	_, _, err := store.Read(context.Background(), "nonexistent")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Read missing = %v, want ErrSessionNotFound", err)
	}
}

func TestP0_05_Session_ConcurrentSet_RaceClean(t *testing.T) {
	s := newSession()
	done := make(chan struct{})
	for range 10 {
		go func() {
			_ = s.Set("k", "v")
			_ = s.Has("k")
			_ = s.Keys()
			done <- struct{}{}
		}()
	}
	for range 10 {
		<-done
	}
}

// ---------------------------------------------------------------------------
// G1 — ErrSessionNotFound sentinel is identifiable via errors.Is
// ---------------------------------------------------------------------------

// TestMemoryStore_G1_ErrSessionNotFoundIsIdentifiable verifies that the error
// returned by MemoryStore.Read for an unknown token satisfies errors.Is with
// ErrSessionNotFound. This confirms the sentinel can be used for branching via
// errors.Is rather than direct ==, supporting wrapped error chains.
func TestMemoryStore_G1_ErrSessionNotFoundIsIdentifiable(t *testing.T) {
	store := NewMemoryStore()
	defer store.Stop()

	_, _, err := store.Read(context.Background(), "no-such-token")
	if err == nil {
		t.Fatal("Read returned nil error for unknown token")
	}
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("errors.Is(err, ErrSessionNotFound) = false; got %v", err)
	}
}

// TestMemoryStore_G1_ErrSessionNotFoundWrapped verifies that wrapping
// ErrSessionNotFound with fmt.Errorf still satisfies errors.Is, demonstrating
// that callers can safely use errors.Is for branching on wrapped errors.
func TestMemoryStore_G1_ErrSessionNotFoundWrapped(t *testing.T) {
	wrapped := fmt.Errorf("middleware: %w", ErrSessionNotFound)
	if !errors.Is(wrapped, ErrSessionNotFound) {
		t.Fatal("errors.Is(wrapped, ErrSessionNotFound) = false; sentinel must survive wrapping")
	}
}
