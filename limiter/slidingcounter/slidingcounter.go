// Package slidingcounter реализует алгоритм счетчика скользящего окна для ограничения запросов
package slidingcounter

import (
	"sync"
	"time"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter"
)

// Config содержит конфиг ограничителя
type Config struct {
	Limit int64 // Максимальное кол-во запросов, допустимых в пределе окна
	Window time.Duration // Временное окно ограничения запросов
}

// counter содержит счетчики для текущего и предыдущего окна
type counter struct {
	currCount   int64     // счетчик в текущем окне
	prevCount   int64     // счетчик в предыдущем окне
	currStart   time.Time // начало текущего окна
}

// Limiter реализует алгоритм счетчика скользящего окна.
type Limiter struct {
	mu       sync.RWMutex
	counters map[string]*counter
	limit    int64
	window   time.Duration
}

// New создает новый ограничитель
func New(cfg Config) *Limiter {
	return &Limiter{
		counters: make(map[string]*counter),
		limit:    cfg.Limit,
		window:   cfg.Window,
	}
}

// Allow проверяет доступ для одного запроса данного ключа
func (l *Limiter) Allow(key string) limiter.Result {
	return l.AllowN(key, 1)
}

// AllowN проверяет доступ для N запросов данного ключа
func (l *Limiter) AllowN(key string, n int64) limiter.Result {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	c, exists := l.counters[key]
	if !exists {
		c = &counter{
			currStart: now.Truncate(l.window),
		}
		l.counters[key] = c
	}

	// Проверяем необходимость сдвига окна
	l.slideWindow(c, now)

	// Вычислить взвешенный счетчик
	weight := l.calculateWeight(c, now)
	estimatedCount := float64(c.currCount) + float64(c.prevCount)*weight

	// Проверить можно ли выдать доступ n запросам
	if int64(estimatedCount)+n <= l.limit {
		c.currCount += n

		newEstimated := float64(c.currCount) + float64(c.prevCount)*weight
		remaining := l.limit - int64(newEstimated)
		if remaining < 0 {
			remaining = 0
		}

		return limiter.Result{
			Allowed:    true,
			Remaining:  remaining,
			Limit:      l.limit,
			RetryAfter: 0,
			ResetAt:    c.currStart.Add(l.window),
		}
	}

	// Рассчитать время повтора
	retryAfter := l.calculateRetryAfter(c, now, n)

	return limiter.Result{
		Allowed:    false,
		Remaining:  0,
		Limit:      l.limit,
		RetryAfter: retryAfter,
		ResetAt:    now.Add(retryAfter),
	}
}

// slideWindow перемещается на другое окно при необходимости
func (l *Limiter) slideWindow(c *counter, now time.Time) {
	windowStart := now.Truncate(l.window)

	// То же окно
	if windowStart.Equal(c.currStart) {
		return
	}

	// Следующее окно
	if windowStart.Equal(c.currStart.Add(l.window)) {
		c.prevCount = c.currCount
		c.currCount = 0
		c.currStart = windowStart
		return
	}

	// Пропущено более одного окна - сброс
	c.prevCount = 0
	c.currCount = 0
	c.currStart = windowStart
}

// calculateWeight возвращает вес предыдущего окна (0.0 до 1.0).
func (l *Limiter) calculateWeight(c *counter, now time.Time) float64 {
	elapsed := now.Sub(c.currStart)
	progress := float64(elapsed) / float64(l.window)

	// Вес предыдущего окна = сколько от текущего окна еще НЕ истрачено
	weight := 1.0 - progress
	if weight < 0 {
		weight = 0
	}
	if weight > 1 {
		weight = 1
	}

	return weight
}

// calculateRetryAfter рассчитывает когда запрос сможет получить доступ.
func (l *Limiter) calculateRetryAfter(c *counter, now time.Time, n int64) time.Duration {
	// Простейший подход - подождать следующего окна
	nextWindow := c.currStart.Add(l.window)
	retryAfter := nextWindow.Sub(now)

	if retryAfter < 0 {
		retryAfter = 0
	}

	return retryAfter
}

// Clean удаляет неиспользуемые счетчики через определенный промежуток времени
func (l *Limiter) Clean(maxAge time.Duration) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for key, c := range l.counters {
		// Если текущее окно старше maxAge
		if now.Sub(c.currStart) > maxAge+l.window {
			delete(l.counters, key)
			cleaned++
		}
	}

	return cleaned
}

// Len возвращает кол-во активных счетчиков
func (l *Limiter) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.counters)
}
