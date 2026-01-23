// Package tokenbucket реализует алгоритм ограничения token bucket
package tokenbucket

import (
	"sync"
	"time"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter"
)

// Config хранит конфиг для token bucket лимитера
type Config struct {
	Rate int64 // Кол-во токенов, добавляемых в token bucket за определенный интервал
	Interval time.Duration // Временной интервал добавления токенов
	Burst int64 // Максимальное количество токенов в bucket. По умолчанию равно Rate
}

// bucket представляет собой ведро токенов для одного ключа
type bucket struct {
	tokens     float64   // текущее кол-во токенов
	lastRefill time.Time // Когда в последний раз в ведро добавлялись токены
}

// Limiter реализует алгоритм ограничения запросов token bucket
type Limiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     float64       // токены в секунду
	capacity int64         // максимальное кол-во токенов (burst)
	interval time.Duration // для расчета сброса
}

// New создает новый ограничитель запросов token bucket
func New(cfg Config) *Limiter {
	burst := cfg.Burst
	if burst == 0 {
		burst = cfg.Rate
	}

	// Конвертируем rate в токены в секунду для упрощения вычислений
	ratePerSecond := float64(cfg.Rate) / cfg.Interval.Seconds()

	return &Limiter{
		buckets:  make(map[string]*bucket),
		rate:     ratePerSecond,
		capacity: burst,
		interval: cfg.Interval,
	}
}

// Allow проверяет получит ли один запрос от данного ключа доступ
func (l *Limiter) Allow(key string) limiter.Result {
	return l.AllowN(key, 1)
}

// AllowN проверяет получат ли n запросов от данного ключа доступ
func (l *Limiter) AllowN(key string, n int64) limiter.Result {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	b, exists := l.buckets[key]
	if !exists {
		// Новый bucket создается полным
		b = &bucket{
			tokens:     float64(l.capacity),
			lastRefill: now,
		}
		l.buckets[key] = b
	}

	// Добавить новые токены в зависимости от истекшего времени
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * l.rate

	// Токены не должны превышать максимальную емкость
	if b.tokens > float64(l.capacity) {
		b.tokens = float64(l.capacity)
	}

	b.lastRefill = now

	// Проверка есть ли запрашиваемое кол-во токенов
	tokensNeeded := float64(n)

	if b.tokens >= tokensNeeded {
		// Предоставить доступ
		b.tokens -= tokensNeeded
		return limiter.Result{
			Allowed:    true,
			Remaining:  int64(b.tokens),
			Limit:      l.capacity,
			RetryAfter: 0,
			ResetAt:    now.Add(l.interval),
		}
	}

	// Отклонить запрос
	// Расчитать время, когда добавится необходимое кол-во токенов
	tokensShortage := tokensNeeded - b.tokens
	waitSeconds := tokensShortage / l.rate
	retryAfter := time.Duration(waitSeconds * float64(time.Second))

	return limiter.Result{
		Allowed:    false,
		Remaining:  0,
		Limit:      l.capacity,
		RetryAfter: retryAfter,
		ResetAt:    now.Add(retryAfter),
	}
}

// Clean удаляет неиспользуемые бакеты через определенный промежуток времени
// Вызывается периодически для предотвращения утечек памяти
func (l *Limiter) Clean(maxAge time.Duration) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for key, b := range l.buckets {
		if now.Sub(b.lastRefill) > maxAge {
			delete(l.buckets, key)
			cleaned++
		}
	}

	return cleaned
}

// Len возвращает кол-во активных бакетов
func (l *Limiter) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.buckets)
}
