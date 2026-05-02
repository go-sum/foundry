package redisstore

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/ratelimit_allow.lua
var rateLimitAllowLua string

var rateLimitAllowScript = redis.NewScript(rateLimitAllowLua)

// RateLimitAllow atomically consumes one token from the named bucket.
func (s *RedisStore) RateLimitAllow(ctx context.Context, key string, capacity int, refillPer time.Duration, now time.Time) (bool, time.Duration, int, time.Duration, error) {
	refillMS := durationMillis(refillPer)
	ttl := time.Duration(capacity+1) * refillPer
	if ttl < refillPer {
		ttl = refillPer
	}

	result, err := rateLimitAllowScript.Run(
		ctx,
		s.client,
		[]string{key},
		now.UnixMilli(),
		capacity,
		refillMS,
		durationMillis(ttl),
	).Result()
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("kv: ratelimit allow: %w", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) < 4 {
		return false, 0, 0, 0, fmt.Errorf("kv: ratelimit allow: unexpected result %T", result)
	}

	allowed, err := redisInt64(items[0])
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("kv: ratelimit allow: %w", err)
	}
	retryAfterMS, err := redisInt64(items[1])
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("kv: ratelimit allow: %w", err)
	}
	remaining, err := redisInt64(items[2])
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("kv: ratelimit allow: %w", err)
	}
	resetAfterMS, err := redisInt64(items[3])
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("kv: ratelimit allow: %w", err)
	}

	return allowed == 1,
		time.Duration(retryAfterMS) * time.Millisecond,
		int(remaining),
		time.Duration(resetAfterMS) * time.Millisecond,
		nil
}
