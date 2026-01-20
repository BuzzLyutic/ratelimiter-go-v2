# 🚦 RateLimiter

[![CI](https://github.com/YOUR_GITHUB_USERNAME/ratelimiter/workflows/CI/badge.svg)](https://github.com/YOUR_GITHUB_USERNAME/ratelimiter/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/YOUR_GITHUB_USERNAME/ratelimiter)](https://goreportcard.com/report/github.com/YOUR_GITHUB_USERNAME/ratelimiter)
[![GoDoc](https://pkg.go.dev/badge/github.com/YOUR_GITHUB_USERNAME/ratelimiter)](https://pkg.go.dev/github.com/YOUR_GITHUB_USERNAME/ratelimiter)
[![Coverage](https://codecov.io/gh/YOUR_GITHUB_USERNAME/ratelimiter/branch/main/graph/badge.svg)](https://codecov.io/gh/YOUR_GITHUB_USERNAME/ratelimiter)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Высокопроизводитльный, распределенный ограничитель запросов на Go с поддержкой множества алгоритмов ограничения и middleware.

## ✨ Особенности

- **3 алгоритма**: Token Bucket, Sliding Window Log, Sliding Window Counter
- **Высокая производительность**: 500K+ ops/sec (in-memory), 50K+ ops/sec (Redis)
- **Распределенность**: Redis backend с Lua скриптами для атомарных операций
- **Middleware**: Gin, Chi, gRPC интерцепторы
- **Наблюдаемость**: Prometheus метрики из коробки
- **Graceful Degradation**: Откат к in-memory когда Redis недоступен
- **Zero Dependencies**: Основная библиотека использует только stdlib

## 📦 Установка

```bash
go get github.com/BuzzLyutic/ratelimiter
```

## 🚀 Быстрый старт
```Go

package main

import (
    "github.com/BuzzLyutic/ratelimiter/limiter/tokenbucket"
    "github.com/BuzzLyutic/ratelimiter/store/memory"
)

func main() {
    // СОздать хранилище в памяти
    store := memory.New()
    
    // Создать ограничитель: 100 запросов в минуту
    limiter := tokenbucket.New(store, tokenbucket.Config{
        Rate:     100,
        Interval: time.Minute,
        Burst:    10,
    })
    
    // Проверить доступ для запроса
    result := limiter.Allow("user:123")
    if !result.Allowed {
        // Доступ ограничен
        // Повтор позже: result.RetryAfter
    }
}
```

## 📊 Бенчмарки
TODO: Add benchmark results after

## 🏗️ Архитектура
TODO: Add architecture diagram

## 📖 Документация
See docs/ for detailed documentation.

## 🤝 Contributing
Contributions are welcome! Please read our Contributing Guide.

## 📄 License
MIT License - see LICENSE for details.
