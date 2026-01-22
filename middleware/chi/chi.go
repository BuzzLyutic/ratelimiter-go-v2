// Package chi предоставляет Chi middleware для ограничения запросов
package chi

import (
	"net/http"
	"strconv"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter"
	"github.com/BuzzLyutic/ratelimiter-go-v2/middleware"
)

type Config struct {
	// Limiter используемый ограничитель запросов
	Limiter limiter.Limiter

	// KeyFunc достает ключ из запроса
	// По умолчанию: IP-адрес клиента
	KeyFunc middleware.KeyFunc

	// ErrorHandler обрабатывает ошибки ограничителя запросов
	// По умолчанию: возвращает 429 с телом JSON
	ErrorHandler middleware.ErrorHandler

	// SkipFunc определяет нужно ли пропустить ограничение запроса
	SkipFunc func(r *http.Request) bool
}

// RateLimit возвращает Chi middleware для ограничения запросов
func RateLimit(cfg Config) func(http.Handler) http.Handler {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = middleware.DefaultKeyFunc
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = middleware.DefaultErrorHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.SkipFunc != nil && cfg.SkipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			key := cfg.KeyFunc(r)

			result := cfg.Limiter.Allow(key)

			w.Header().Set(middleware.HeaderRateLimitLimit, strconv.FormatInt(result.Limit, 10))
			w.Header().Set(middleware.HeaderRateLimitRemaining, strconv.FormatInt(result.Remaining, 10))
			w.Header().Set(middleware.HeaderRateLimitReset, strconv.FormatInt(result.ResetAt.Unix(), 10))

			if !result.Allowed {
				cfg.ErrorHandler(w, r, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitByIP возвращает middleware, ограничивающий по IP.
func RateLimitByIP(l limiter.Limiter) func(http.Handler) http.Handler {
	return RateLimit(Config{
		Limiter: l,
	})
}

// RateLimitByKey возвращает middleware использующий кастомный ключ.
func RateLimitByKey(l limiter.Limiter, keyFunc middleware.KeyFunc) func(http.Handler) http.Handler {
	return RateLimit(Config{
		Limiter: l,
		KeyFunc: keyFunc,
	})
}
