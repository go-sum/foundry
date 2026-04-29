local key = KEYS[1]
local now = tonumber(ARGV[1]) or 0
local fields = redis.call("HMGET", key, "d", "v", "a", "i")
local data = fields[1]
if not data then
	return {0}
end

local version = tonumber(fields[2]) or 0
local absolute = tonumber(fields[3]) or 0
local idle = tonumber(fields[4]) or 0

if absolute > 0 then
	local remaining = absolute - now
	if remaining <= 0 then
		redis.call("DEL", key)
		return {0}
	end
	if idle > 0 then
		if remaining < idle then
			redis.call("PEXPIRE", key, remaining)
		else
			redis.call("PEXPIRE", key, idle)
		end
	else
		redis.call("PEXPIRE", key, remaining)
	end
elseif idle > 0 then
	redis.call("PEXPIRE", key, idle)
end

return {1, data, tostring(version)}
