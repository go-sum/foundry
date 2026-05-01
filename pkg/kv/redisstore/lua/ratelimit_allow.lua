local key = KEYS[1]
local now_ms = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local refill_ms = tonumber(ARGV[3])
local ttl_ms = tonumber(ARGV[4])

local state = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(state[1])
local ts = tonumber(state[2])

if tokens == nil then
  tokens = capacity
  ts = now_ms
end

if ts == nil then
  ts = now_ms
end

if now_ms > ts then
  local elapsed = now_ms - ts
  tokens = math.min(capacity, tokens + (elapsed / refill_ms))
  ts = now_ms
end

local allowed = 0
local retry_after_ms = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
else
  retry_after_ms = math.ceil((1 - tokens) * refill_ms)
end

local remaining = math.floor(tokens)
if remaining < 0 then
  remaining = 0
end

local reset_after_ms = math.ceil((capacity - tokens) * refill_ms)
if reset_after_ms < 0 then
  reset_after_ms = 0
end

redis.call("HSET", key, "tokens", tokens, "ts", ts)
redis.call("PEXPIRE", key, ttl_ms)

return {allowed, retry_after_ms, remaining, reset_after_ms}
