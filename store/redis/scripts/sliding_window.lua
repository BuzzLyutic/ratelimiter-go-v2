-- Sliding Window Counter Algorithm
-- KEYS[1] = ключ счетчика
-- ARGV[1] = now (unix временная метка)
-- ARGV[2] = размер окна (сек)
-- ARGV[3] = лимит
-- ARGV[4] = запрос

local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

-- Рассчитать метки предыдущего и текущего окон
local current_window = math.floor(now / window) * window
local previous_window = current_window - window

-- Ключи для предыдущего и текущего окна
local curr_key = key .. ':' .. string.format("%.3f", current_window)
local prev_key = key .. ':' .. string.format("%.3f", previous_window)

-- Получить счетчики
local curr_count = tonumber(redis.call('GET', curr_key)) or 0
local prev_count = tonumber(redis.call('GET', prev_key)) or 0

-- Расчитать вес предыдущего окна
local elapsed = now - current_window
local weight = 1 - (elapsed / window)
if weight < 0 then weight = 0 end

-- Расчитать текущий счетчик
local estimated = curr_count + prev_count * weight

-- Проверить доступ
local allowed = 0
local remaining = limit - estimated
local retry_after = 0

if estimated + requested <= limit then

    redis.call('INCRBYFLOAT', curr_key, requested)
    
    local ttl = math.ceil(window * 2)
    if ttl < 1 then ttl = 1 end
    redis.call('EXPIRE', curr_key, ttl)
    
    allowed = 1
    remaining = limit - estimated - requested
else
    retry_after = window - elapsed
    if retry_after < 0 then retry_after = 0 end
end

if remaining < 0 then remaining = 0 end

return {allowed, math.floor(remaining), tostring(retry_after)}
