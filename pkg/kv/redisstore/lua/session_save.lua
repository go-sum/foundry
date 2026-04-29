local key = KEYS[1]
local data = ARGV[1]
local absolute = tonumber(ARGV[2]) or 0
local idle = tonumber(ARGV[3]) or 0
local expected = tonumber(ARGV[4]) or 0
local now = tonumber(ARGV[5]) or 0

local current = redis.call("HGET", key, "v")
if not current then
	if expected ~= 0 then
		return {0}
	end
else
	local currentVersion = tonumber(current) or 0
	if currentVersion ~= expected then
		return {0}
	end
end

local ttl = 0
if idle > 0 then
	ttl = idle
end
if absolute > 0 then
	local remaining = absolute - now
	if remaining <= 0 then
		return {2}
	end
	if ttl == 0 or remaining < ttl then
		ttl = remaining
	end
end

local nextVersion = expected + 1
redis.call("HSET", key,
	"d", data,
	"v", nextVersion,
	"a", absolute,
	"i", idle
)

if ttl > 0 then
	redis.call("PEXPIRE", key, ttl)
else
	redis.call("PERSIST", key)
end

return {1, tostring(nextVersion)}
