# RateLimiter


[![CI](https://github.com/BuzzLyutic/ratelimiter-go-v2/workflows/CI/badge.svg)](https://github.com/BuzzLyutic/ratelimiter-go-v2/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/BuzzLyutic/ratelimiter-go-v2)](https://goreportcard.com/report/github.com/BuzzLyutic/ratelimiter-go-v2)
[![GoDoc](https://pkg.go.dev/badge/github.com/BuzzLyutic/ratelimiter-go-v2)](https://pkg.go.dev/github.com/BuzzLyutic/ratelimiter-go-v2)
[![Coverage](https://codecov.io/gh/BuzzLyutic/ratelimiter-go-v2/branch/main/graph/badge.svg)](https://codecov.io/gh/BuzzLyutic/ratelimiter-go-v2)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Высокопроизводительный распределённый rate limiter для Go с поддержкой нескольких алгоритмов и observability.

---

## ✨ Особенности

- **3 алгоритма**: Token Bucket, Sliding Window Log, Sliding Window Counter
- **Высокая производительность**: 500K+ ops/sec (in-memory), 50K+ ops/sec (Redis)
- **Распределенность**: Redis backend с Lua скриптами для атомарных операций
- **Middleware**: Gin, Chi, gRPC интерцепторы
- **Наблюдаемость**: Prometheus метрики из коробки
- **Graceful Degradation**: Откат к in-memory когда Redis недоступен
- **Zero Dependencies**: Основная библиотека использует только stdlib
- **Production Ready**: Rate limit заголовки, настраиваемая Key extraction

---

## 📁 Структура проекта
```text

ratelimiter/
├── limiter/           # Rate limiting algorithms
│   ├── limiter.go     # Common interface
│   ├── tokenbucket/   # Token bucket implementation
│   ├── slidinglog/    # Sliding window log
│   └── slidingcounter/# Sliding window counter
├── store/             # Storage backends
│   ├── memory/        # In-memory store
│   └── redis/         # Redis store with Lua scripts
├── middleware/        # HTTP/gRPC middleware
│   ├── gin/           # Gin middleware
│   ├── chi/           # Chi middleware
│   └── grpc/          # gRPC interceptors
├── metrics/           # Prometheus instrumentation
├── examples/          # Usage examples
│   ├── basic/         # Simple CLI example
│   ├── demo/          # Interactive demo
│   ├── gin-api/       # Gin HTTP API
│   ├── chi-api/       # Chi HTTP API
│   └── prometheus/    # Full observability example
└── docs/              # Documentation
```
---

## 📊 Производительность

| Backend | Пропускная способность | Задержка (P99) | Память |
|---------|------------|---------------|--------|
| Token Bucket (memory) | ~7.2M ops/sec | <1 µs | 0 B/op |
| Sliding Counter (memory) | ~12M ops/sec | <1 µs | 0 B/op |
| Redis | ~4-5K ops/sec | ~230 µs | 615 B/op |
| With Prometheus | ~2.6M ops/sec | ~460 ns overhead | 0 B/op |

> [benchmark/results/BENCHMARKS.md](benchmark/results/BENCHMARKS.md) для просмотра деталей.

---

## 🛠️ Технологии и инструменты

| Категория | Технологии |
|-----------|------------|
| **Язык** | Go 1.22+ |
| **Хранилище** | In-memory, Redis 7+ |
| **HTTP фреймворки** | Gin, Chi, net/http |
| **RPC** | gRPC |
| **Метрики** | Prometheus |
| **Визуализация** | Grafana |
| **Контейнеризация** | Docker, Docker Compose |
| **CI/CD** | GitHub Actions |
| **Тестирование** | Go testing, race detector, benchmarks |
| **Линтинг** | golangci-lint |

---

## 📦 Установка

```bash
go get github.com/BuzzLyutic/ratelimiter-go-v2
```
---

## 🚀 Быстрый старт

### Базовое использование
```Go

package main

import (
    "fmt"
    "time"
    
    "github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
)

func main() {
    // Создаём лимитер: 100 запросов в минуту, burst 10
    limiter := tokenbucket.New(tokenbucket.Config{
        Rate:     100,
        Interval: time.Minute,
        Burst:    10,
    })

    // Проверяем разрешён ли запрос
    result := limiter.Allow("user:123")
    
    if result.Allowed {
        fmt.Printf("Разрешено! Осталось: %d\n", result.Remaining)
    } else {
        fmt.Printf("Заблокировано. Повторить через: %v\n", result.RetryAfter)
    }
}
```

---

## 🧪 Тестирование
### Запуск тестов
```Bash

# Все тесты
make test

# С race detector
make test-race

# С покрытием
make test-cover
# Откроет coverage.html в браузере

# Только определённый пакет
go test -v ./limiter/tokenbucket/...
```
### Запуск бенчмарков
```Bash

# Все бенчмарки
make bench

# Сравнение алгоритмов
make bench-compare

# Конкретный бенчмарк
go test -bench=BenchmarkAllow -benchmem ./limiter/tokenbucket/...
```

### Структура тестов
```text

./limiter/tokenbucket/
├── tokenbucket.go
└── tokenbucket_test.go    # Unit тесты + бенчмарки

./store/redis/
├── redis.go
├── redis_test.go          # Integration тесты (требуют Redis)
└── fallback_test.go       # Тесты graceful degradation

./middleware/gin/
├── gin.go
└── gin_test.go            # HTTP тесты с httptest
```

### CI Pipeline
GitHub Actions автоматически запускает:

- Lint — golangci-lint
- Test — тесты с race detector + coverage
- Build — проверка компиляции
- Benchmark — бенчмарки (только на main)

---

## 📈 Prometheus метрики

| Метрика | Тип | Описание | 
|---------|------------|---------------|
| **ratelimiter_requests_total** | Counter | Всего запросов по алгоритму и результату |
| **ratelimiter_requests_allowed_total** | Counter | Разрешённые запросы |
| **ratelimiter_requests_limited_total** | Counter | Заблокированные запросы |
| **ratelimiter_check_duration_seconds** | Histogram | Латентность проверки |
| **ratelimiter_keys_active** | Gauge | Количество активных ключей |

---

## 🎮 Попробуй сам

### Интерактивная демонстрация:
```bash
# Run interactive CLI demo
make demo

# Or directly:
go run ./examples/demo/main.go
```

### Пример работы:

```text

[Token Bucket (10/s, burst 5)] > burst 10
  Burst 10: ✅ 5 allowed, ❌ 5 denied

[Token Bucket (10/s, burst 5)] > wait 1
  ⏳ Waiting 1 seconds...
  ✅ Done

[Token Bucket (10/s, burst 5)] > r 3
  ✅ Request 1: Allowed (remaining: 9)
  ✅ Request 2: Allowed (remaining: 8)
  ✅ Request 3: Allowed (remaining: 7)

[Token Bucket (10/s, burst 5)] > algo sc
✅ Switched to: Sliding Counter (10/s)
```

### HTTP API Примеры
```Bash

# старт Gin
make example-gin

# В другом терминале - протестируйте ограничение запросов
for i in {1..10}; do 
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/api/users
done
# Output: 200 200 200 200 200 429 429 429 429 429
```

### Full Stack с мониторингом
```Bash

# Start всех сервисов (Redis, Prometheus, Grafana, API)
make docker-up

# Генерация загрузки
for i in {1..100}; do curl -s http://localhost:8080/api/test; done

# Открыть дашборд Grafana
open http://localhost:3000  # admin/admin

# Остановить все
make docker-down
```
---

## 🏗️ Архитектура
Смотри [docs/architecture.md](docs/architecture.md) для детализированной документации архитектуры.

---

## 🔧 Конфигурация
### Rate Limit заголовки
Все middleware автоматически устанавливают заголовки:

| Заголовок | Описание |
|-----------|------------|
| **X-RateLimit-Limit** | Максимум запросов |
| **X-RateLimit-Remaining** | Осталось запросов |
| **X-RateLimit-Reset** | Unix timestamp сброса |
| **Retry-After** | Секунд до повтора (только при 429) |

### Кастомный Key extraction
```Go

// По User ID
r.Use(ginmw.RateLimitByKey(limiter, func(c *gin.Context) string {
    return c.GetHeader("X-User-ID")
}))

// По API ключу
r.Use(ginmw.RateLimitByKey(limiter, func(c *gin.Context) string {
    return c.GetHeader("X-API-Key")
}))

// По endpoint + IP
r.Use(ginmw.RateLimitByKey(limiter, func(c *gin.Context) string {
    return c.ClientIP() + ":" + c.FullPath()
}))
```
### Пропуск определенных конечных точек
```Go

r.Use(ginmw.RateLimit(ginmw.Config{
    Limiter: limiter,
    SkipFunc: func(c *gin.Context) bool {
        // Don't rate limit health checks
        return c.Request.URL.Path == "/health"
    },
}))
```

---

## 🐳 Docker
```Bash

# Start full stack
docker-compose up -d

# Services:
# - API:        http://localhost:8080
# - Redis:      localhost:6379
# - Prometheus: http://localhost:9090
# - Grafana:    http://localhost:3000

# View logs
docker-compose logs -f

# Stop
docker-compose down
```


## 📄 License
MIT License - see LICENSE for details.
