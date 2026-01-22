-- Token Bucket Algorithm
-- KEYS[1] = ключ бакета
-- ARGV[1] = токены в секунду
-- ARGV[2] = максимальная емкость токенов
-- ARGV[3] = now (unix временная метка с микросекундами)
-- ARGV[4] = запрошенные токены

local key = KEYS[1]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

-- Получить текущее состояние
local data = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])

-- Инициализировать если новый
if tokens == nil then
    tokens = capacity
    last_refill = now
end

-- Рассчитать кол-во пополняемых токенов
local elapsed = now - last_refill
local tokens_to_add = elapsed * rate
tokens = math.min(capacity, tokens + tokens_to_add)

-- Обновить время последнего пополнения
last_refill = now

-- Проверить доступ для запроса
local allowed = 0
local remaining = tokens
local retry_after = 0

if tokens >= requested then
    tokens = tokens - requested
    allowed = 1
    remaining = tokens
else
    -- Расчитать время ожидания
    local tokens_needed = requested - tokens
    retry_after = tokens_needed / rate
end

-- Сохранить состояние
redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)

-- Установить TTL
local ttl = math.ceil(capacity / rate) * 2
if ttl < 60 then ttl = 60 end
redis.call('EXPIRE', key, ttl)

return {allowed, math.floor(remaining), tostring(retry_after)}
