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
local curr_key = key .. ':' .. current_window
local prev_key = key .. ':' .. previous_window

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
    -- Увеличить текущее окно
    redis.call('INCRBYFLOAT', curr_key, requested)
    redis.call('EXPIRE', curr_key, window * 2)
    
    allowed = 1
    remaining = limit - estimated - requested
else
    -- Расчитать время повтора
    retry_after = window - elapsed
end

if remaining < 0 then remaining = 0 end

return {allowed, math.floor(remaining), tostring(retry_after)}
