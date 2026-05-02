package redisstore_test

import (
	"context"
	"testing"
	"time"
)

func TestSessionStore_RoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "session")

	now := time.Now()
	version, conflict, expired, err := store.SessionSave(ctx, key, []byte(`{"user":"alice"}`), now.Add(time.Hour), 5*time.Minute, 0, now)
	if err != nil {
		t.Fatalf("SessionSave: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Delete(ctx, key); err != nil {
			t.Logf("cleanup: failed to delete key %q: %v", key, err)
		}
	})
	if conflict {
		t.Fatal("SessionSave conflict = true, want false")
	}
	if expired {
		t.Fatal("SessionSave expired = true, want false")
	}
	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}

	data, gotVersion, found, err := store.SessionRead(ctx, key, time.Now())
	if err != nil {
		t.Fatalf("SessionRead: %v", err)
	}
	if !found {
		t.Fatal("SessionRead found = false, want true")
	}
	if string(data) != `{"user":"alice"}` {
		t.Fatalf("data = %q, want alice payload", data)
	}
	if gotVersion != 1 {
		t.Fatalf("gotVersion = %d, want 1", gotVersion)
	}
}

func TestSessionStore_VersionConflict(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "conflict")

	now := time.Now()
	if _, conflict, _, err := store.SessionSave(ctx, key, []byte(`"v1"`), now.Add(time.Hour), time.Minute, 0, now); err != nil {
		t.Fatalf("SessionSave initial: %v", err)
	} else if conflict {
		t.Fatal("SessionSave initial conflict = true, want false")
	}
	t.Cleanup(func() {
		if err := store.Delete(ctx, key); err != nil {
			t.Logf("cleanup: failed to delete key %q: %v", key, err)
		}
	})
	if _, conflict, _, err := store.SessionSave(ctx, key, []byte(`"v2"`), now.Add(time.Hour), time.Minute, 0, now); err != nil {
		t.Fatalf("SessionSave conflict error = %v, want nil", err)
	} else if !conflict {
		t.Fatal("SessionSave conflict = false, want true")
	}
}

func TestSessionStore_IdleExpiry(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "idle")

	now := time.Now()
	if _, conflict, _, err := store.SessionSave(ctx, key, []byte(`"x"`), now.Add(time.Hour), 100*time.Millisecond, 0, now); err != nil {
		t.Fatalf("SessionSave: %v", err)
	} else if conflict {
		t.Fatal("SessionSave conflict = true, want false")
	}
	time.Sleep(300 * time.Millisecond)

	if _, _, found, err := store.SessionRead(ctx, key, time.Now()); err != nil {
		t.Fatalf("SessionRead after idle expiry error = %v, want nil", err)
	} else if found {
		t.Fatal("SessionRead after idle expiry found = true, want false")
	}
}

func TestSessionStore_AbsoluteExpiry(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "absolute")

	now := time.Now()
	if _, conflict, _, err := store.SessionSave(ctx, key, []byte(`"x"`), now.Add(100*time.Millisecond), time.Hour, 0, now); err != nil {
		t.Fatalf("SessionSave: %v", err)
	} else if conflict {
		t.Fatal("SessionSave conflict = true, want false")
	}
	time.Sleep(300 * time.Millisecond)

	if _, _, found, err := store.SessionRead(ctx, key, time.Now()); err != nil {
		t.Fatalf("SessionRead after absolute expiry error = %v, want nil", err)
	} else if found {
		t.Fatal("SessionRead after absolute expiry found = true, want false")
	}
}

func TestSessionStore_SaveAfterAbsoluteExpiry(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "expiry-save")

	past := time.Now().Add(-time.Second)
	_, conflict, expired, err := store.SessionSave(ctx, key, []byte(`"x"`), past, 0, 0, time.Now())
	if err != nil {
		t.Fatalf("SessionSave error = %v, want nil", err)
	}
	if conflict {
		t.Fatal("SessionSave conflict = true, want false")
	}
	if !expired {
		t.Fatal("SessionSave expired = false, want true (absolute deadline already passed)")
	}

	t.Cleanup(func() {
		if err := store.Delete(ctx, key); err != nil {
			t.Logf("cleanup: failed to delete key %q: %v", key, err)
		}
	})
}
