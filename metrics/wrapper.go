package metrics

import (
	"time"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter"
)

// LimiterWrapper оборачивает Limiter коллекцией метрик
type LimiterWrapper struct {
	limiter   limiter.Limiter
	metrics   *Metrics
	algorithm string
}

// Wrap создает новый инструментальный ограничитель
func Wrap(l limiter.Limiter, m *Metrics, algorithm string) *LimiterWrapper {
	return &LimiterWrapper{
		limiter:   l,
		metrics:   m,
		algorithm: algorithm,
	}
}

// Allow проверяет доступ для запроса и сохраняет метрику
func (w *LimiterWrapper) Allow(key string) limiter.Result {
	return w.AllowN(key, 1)
}

// AllowN проверяет доступ для n запросов и сохраняет метрику
func (w *LimiterWrapper) AllowN(key string, n int64) limiter.Result {
	start := time.Now()
	result := w.limiter.AllowN(key, n)
	duration := time.Since(start)

	if result.Allowed {
		w.metrics.ObserveAllow(w.algorithm, duration)
	} else {
		w.metrics.ObserveDeny(w.algorithm, duration)
	}

	return result
}

// Unwrap возвращает подлежащий ограничитель
func (w *LimiterWrapper) Unwrap() limiter.Limiter {
	return w.limiter
}
