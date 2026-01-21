// Package slidinglog реализует алгоритм ограничения запросов методом скользящего окна (далее ограничитель-окно)
package slidinglog

import (
	"sync"
	"time"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter"
)

// Config содержит конфигурацию ограничителя-окна
type Config struct {
	Limit int64 // Максимальное кол-во запросов допустимое в рамках окна
	Window time.Duration // Временное окно ограничения запросов
}

// window хранит временные метки запросов для одного ключа
type window struct {
	timestamps []time.Time
}

// Limiter реализует алгоритм ограничения запросов методом скользящего окна
type Limiter struct {
	mu      sync.RWMutex
	windows map[string]*window
	limit   int64
	window  time.Duration
}

// New создает новый ограничитель
func New(cfg Config) *Limiter {
	return &Limiter{
		windows: make(map[string]*window),
		limit:   cfg.Limit,
		window:  cfg.Window,
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
	windowStart := now.Add(-l.window)

	w, exists := l.windows[key]
	if !exists {
		w = &window{
			timestamps: make([]time.Time, 0, l.limit),
		}
		l.windows[key] = w
	}

	// Удалить старые метки (вне окна)
	w.timestamps = l.removeExpired(w.timestamps, windowStart)

	// Посчитать текущие запросы в окне
	currentCount := int64(len(w.timestamps))

	// Проверить, можно ли добавить n новых меток
	if currentCount+n <= l.limit {
		// Добавить n меток
		for i := int64(0); i < n; i++ {
			w.timestamps = append(w.timestamps, now)
		}

		return limiter.Result{
			Allowed:    true,
			Remaining:  l.limit - currentCount - n,
			Limit:      l.limit,
			RetryAfter: 0,
			ResetAt:    now.Add(l.window),
		}
	}

	// Расчет повторной попытки (когда устареет старейший запрос)
	var retryAfter time.Duration
	if len(w.timestamps) > 0 {
		oldestExpiry := w.timestamps[0].Add(l.window)
		retryAfter = oldestExpiry.Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
	}

	return limiter.Result{
		Allowed:    false,
		Remaining:  0,
		Limit:      l.limit,
		RetryAfter: retryAfter,
		ResetAt:    now.Add(retryAfter),
	}
}

// removeExpired удаляет метки вне временного окна
func (l *Limiter) removeExpired(timestamps []time.Time, windowStart time.Time) []time.Time {
	// Найти первый валидный индекс
	validIdx := 0
	for i, ts := range timestamps {
		if ts.After(windowStart) {
			validIdx = i
			break
		}
		validIdx = i + 1
	}

	if validIdx >= len(timestamps) {
		return timestamps[:0] // Все устарели, переиспользуем массив
	}

	// Сдвигаем валидные временные метки в начало
	return timestamps[validIdx:]
}

// Clean удаляет неиспользуемые окна после определенного промежутка времени
func (l *Limiter) Clean(maxAge time.Duration) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for key, w := range l.windows {
		// Если окно пустое или все метки устарели
		if len(w.timestamps) == 0 {
			delete(l.windows, key)
			cleaned++
			continue
		}

		// Проверить является ли новейшая метка в окне старше maxAge
		newest := w.timestamps[len(w.timestamps)-1]
		if now.Sub(newest) > maxAge {
			delete(l.windows, key)
			cleaned++
		}
	}

	return cleaned
}

// Len вовзращает кол-во активных окон
func (l *Limiter) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.windows)
}
