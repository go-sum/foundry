package redisstore

import (
	_ "embed"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/session_read.lua
var sessionReadLua string

//go:embed lua/session_save.lua
var sessionSaveLua string

var (
	sessionReadScript = redis.NewScript(sessionReadLua)
	sessionSaveScript = redis.NewScript(sessionSaveLua)
)

// SessionRead loads a session record and refreshes its idle TTL without
// extending the absolute expiry.
func (s *RedisStore) SessionRead(ctx context.Context, key string, now time.Time) ([]byte, int64, bool, error) {
	result, err := sessionReadScript.Run(ctx, s.client, []string{key}, now.UnixMilli()).Result()
	if err != nil {
		return nil, 0, false, fmt.Errorf("kv: session read %q: %w", key, err)
	}

	items, ok := result.([]any)
	if !ok || len(items) < 1 {
		return nil, 0, false, fmt.Errorf("kv: session read %q: unexpected result %T", key, result)
	}

	found, err := redisInt64(items[0])
	if err != nil {
		return nil, 0, false, fmt.Errorf("kv: session read %q: %w", key, err)
	}
	if found == 0 {
		return nil, 0, false, nil
	}
	if len(items) < 3 {
		return nil, 0, false, fmt.Errorf("kv: session read %q: incomplete result", key)
	}

	data, err := redisBytes(items[1])
	if err != nil {
		return nil, 0, false, fmt.Errorf("kv: session read %q: %w", key, err)
	}
	version, err := redisInt64(items[2])
	if err != nil {
		return nil, 0, false, fmt.Errorf("kv: session read %q: %w", key, err)
	}
	return data, version, true, nil
}

// SessionSave atomically enforces optimistic concurrency and sets the record TTL
// to the earlier of the absolute expiry and idle timeout.
func (s *RedisStore) SessionSave(ctx context.Context, key string, data []byte, absolute time.Time, idleTTL time.Duration, version int64, now time.Time) (int64, bool, bool, error) {
	result, err := sessionSaveScript.Run(
		ctx,
		s.client,
		[]string{key},
		data,
		unixMillis(absolute),
		durationMillis(idleTTL),
		version,
		unixMillis(now),
	).Result()
	if err != nil {
		return 0, false, false, fmt.Errorf("kv: session save %q: %w", key, err)
	}

	items, ok := result.([]any)
	if !ok || len(items) < 1 {
		return 0, false, false, fmt.Errorf("kv: session save %q: unexpected result %T", key, result)
	}

	saved, err := redisInt64(items[0])
	if err != nil {
		return 0, false, false, fmt.Errorf("kv: session save %q: %w", key, err)
	}
	if saved == 0 {
		return 0, true, false, nil
	}
	if saved == 2 {
		return 0, false, true, nil
	}
	if len(items) < 2 {
		return 0, false, false, fmt.Errorf("kv: session save %q: missing version result", key)
	}

	nextVersion, err := redisInt64(items[1])
	if err != nil {
		return 0, false, false, fmt.Errorf("kv: session save %q: %w", key, err)
	}
	return nextVersion, false, false, nil
}

func redisInt64(v any) (int64, error) {
	switch x := v.(type) {
	case int64:
		return x, nil
	case string:
		return strconv.ParseInt(x, 10, 64)
	case []byte:
		return strconv.ParseInt(string(x), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected integer type %T", v)
	}
}

func redisBytes(v any) ([]byte, error) {
	switch x := v.(type) {
	case string:
		return []byte(x), nil
	case []byte:
		return x, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected byte payload type %T", v)
	}
}

func unixMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func durationMillis(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	ms := d.Milliseconds()
	if ms == 0 {
		// Clamp sub-millisecond TTLs to 1ms so they still expire instead of persisting forever.
		return 1
	}
	return ms
}
