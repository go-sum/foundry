package redisstore_test

import (
	"context"
	"crypto/tls"
	"errors"
	"net/url"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/kv/redisstore"
)

// testConfig returns a RedisStore config from TEST_KV_URL.
// Skips the test if the env var is not set.
// URL format: redis://host:port/db (e.g. redis://kv:6379/1)
func testConfig(t *testing.T) redisstore.Config {
	t.Helper()
	raw := os.Getenv("TEST_KV_URL")
	if raw == "" {
		t.Skip("TEST_KV_URL not set; skipping integration test")
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse TEST_KV_URL: %v", err)
	}

	cfg := redisstore.Config{Addr: u.Host}
	if u.User != nil {
		cfg.Password, _ = u.User.Password()
	}
	if u.Scheme == "rediss" {
		cfg.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if len(u.Path) > 1 {
		cfg.DB, _ = strconv.Atoi(u.Path[1:])
	}
	return cfg
}

// newTestStore creates a connected RedisStore and registers cleanup.
func newTestStore(t *testing.T) *redisstore.RedisStore {
	t.Helper()
	cfg := testConfig(t)

	store := redisstore.New(cfg)

	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Skipf("cannot reach KV server at %s: %v", cfg.Addr, err)
	}

	t.Cleanup(func() { _ = store.Close() })
	return store
}

// uniqueKey returns a test-unique key to avoid collisions between parallel tests.
func uniqueKey(t *testing.T, suffix string) string {
	t.Helper()
	return "test:" + t.Name() + ":" + suffix
}

func TestPing(t *testing.T) {
	store := newTestStore(t)
	if err := store.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestSetGetRoundtrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "hello")
	val := []byte("world")

	t.Cleanup(func() { _ = store.Delete(ctx, key) })

	if err := store.Set(ctx, key, val, kv.SetOptions{}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(val) {
		t.Fatalf("Get = %q, want %q", got, val)
	}
}

func TestGetMissingKeyReturnsErrNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "missing")

	_, err := store.Get(ctx, key)
	if !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("Get missing key: got %v, want kv.ErrNotFound", err)
	}
}

func TestSetWithTTL(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "ttl")

	t.Cleanup(func() { _ = store.Delete(ctx, key) })

	if err := store.Set(ctx, key, []byte("ephemeral"), kv.SetOptions{TTL: 1 * time.Second}); err != nil {
		t.Fatalf("Set with TTL: %v", err)
	}

	// Key should exist immediately.
	if _, err := store.Get(ctx, key); err != nil {
		t.Fatalf("Get immediately after Set: %v", err)
	}

	// Wait for expiry.
	time.Sleep(1500 * time.Millisecond)

	_, err := store.Get(ctx, key)
	if !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("Get after TTL expiry: got %v, want kv.ErrNotFound", err)
	}
}

func TestDeleteRemovesKey(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "del")

	if err := store.Set(ctx, key, []byte("gone"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := store.Get(ctx, key)
	if !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("Get after Delete: got %v, want kv.ErrNotFound", err)
	}
}

func TestDeleteNonExistentKeyIsNoop(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "nope")

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete non-existent: %v", err)
	}
}

func TestDeleteMultipleKeys(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	k1 := uniqueKey(t, "a")
	k2 := uniqueKey(t, "b")

	t.Cleanup(func() { _ = store.Delete(ctx, k1, k2) })

	if err := store.Set(ctx, k1, []byte("1"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", k1, err)
	}
	if err := store.Set(ctx, k2, []byte("2"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", k2, err)
	}

	if err := store.Delete(ctx, k1, k2); err != nil {
		t.Fatalf("Delete multiple: %v", err)
	}
	if _, err := store.Get(ctx, k1); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("k1 still exists after multi-delete")
	}
	if _, err := store.Get(ctx, k2); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("k2 still exists after multi-delete")
	}
}

func TestDeleteNoArgs(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Delete(ctx); err != nil {
		t.Fatalf("Delete(): %v", err)
	}
}

func TestExistsReturnsCounts(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	k1 := uniqueKey(t, "ex1")
	k2 := uniqueKey(t, "ex2")
	k3 := uniqueKey(t, "ex3")

	t.Cleanup(func() { _ = store.Delete(ctx, k1, k2) })

	if err := store.Set(ctx, k1, []byte("a"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", k1, err)
	}
	if err := store.Set(ctx, k2, []byte("b"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", k2, err)
	}

	n, err := store.Exists(ctx, k1, k2, k3)
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if n != 2 {
		t.Fatalf("Exists = %d, want 2", n)
	}
}

func TestExistsNoKeys(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	n, err := store.Exists(ctx)
	if err != nil {
		t.Fatalf("Exists(): %v", err)
	}
	if n != 0 {
		t.Fatalf("Exists() = %d, want 0", n)
	}
}

func TestScanMatchesPrefix(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	prefix := uniqueKey(t, "scan")
	k1 := prefix + ":a"
	k2 := prefix + ":b"
	k3 := prefix + ":c"

	t.Cleanup(func() { _ = store.Delete(ctx, k1, k2, k3) })

	if err := store.Set(ctx, k1, []byte("1"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", k1, err)
	}
	if err := store.Set(ctx, k2, []byte("2"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", k2, err)
	}
	if err := store.Set(ctx, k3, []byte("3"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", k3, err)
	}

	var found []string
	err := store.Scan(ctx, prefix+":*", func(key string) error {
		found = append(found, key)
		return nil
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	sort.Strings(found)
	want := []string{k1, k2, k3}
	sort.Strings(want)

	if len(found) != len(want) {
		t.Fatalf("Scan found %d keys, want %d: %v", len(found), len(want), found)
	}
	for i := range want {
		if found[i] != want[i] {
			t.Fatalf("Scan[%d] = %q, want %q", i, found[i], want[i])
		}
	}
}

func TestScanNoMatches(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	prefix := uniqueKey(t, "nomatch")

	called := false
	err := store.Scan(ctx, prefix+":*", func(key string) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if called {
		t.Fatal("fn should not be called when no keys match")
	}
}

func TestScanStopsOnCallbackError(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	prefix := uniqueKey(t, "stopcb")
	k1 := prefix + ":a"
	k2 := prefix + ":b"

	t.Cleanup(func() { _ = store.Delete(ctx, k1, k2) })

	if err := store.Set(ctx, k1, []byte("1"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", k1, err)
	}
	if err := store.Set(ctx, k2, []byte("2"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", k2, err)
	}

	sentinel := errors.New("stop")
	err := store.Scan(ctx, prefix+":*", func(key string) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Scan should propagate callback error: got %v, want %v", err, sentinel)
	}
}

func TestClosePreventsFurtherOps(t *testing.T) {
	cfg := testConfig(t)

	store := redisstore.New(cfg)

	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := store.Ping(context.Background()); err == nil {
		t.Fatal("Ping after Close should return an error")
	}
}

func TestSetOverwritesExistingKey(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "overwrite")

	t.Cleanup(func() { _ = store.Delete(ctx, key) })

	if err := store.Set(ctx, key, []byte("first"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", key, err)
	}
	if err := store.Set(ctx, key, []byte("second"), kv.SetOptions{}); err != nil {
		t.Fatalf("Set %q: %v", key, err)
	}

	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "second" {
		t.Fatalf("Get = %q, want %q", got, "second")
	}
}

func TestSetEmptyValue(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "empty")

	t.Cleanup(func() { _ = store.Delete(ctx, key) })

	if err := store.Set(ctx, key, []byte(""), kv.SetOptions{}); err != nil {
		t.Fatalf("Set empty: %v", err)
	}
	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Get = %q, want empty", got)
	}
}

func TestSetBinaryValue(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := uniqueKey(t, "binary")
	val := []byte{0x00, 0x01, 0xFF, 0xFE, 0x80}

	t.Cleanup(func() { _ = store.Delete(ctx, key) })

	if err := store.Set(ctx, key, val, kv.SetOptions{}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got) != len(val) {
		t.Fatalf("Get length = %d, want %d", len(got), len(val))
	}
	for i := range val {
		if got[i] != val[i] {
			t.Fatalf("Get[%d] = %02x, want %02x", i, got[i], val[i])
		}
	}
}
