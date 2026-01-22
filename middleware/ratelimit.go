// Package middleware provides HTTP middleware for rate limiting.
package middleware

import (
	"net/http"
	"strconv"
	//"time"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter"
)

// Заголовки для ограничителя запросов (RFC 6585 compliant).
const (
	HeaderRateLimitLimit     = "X-RateLimit-Limit"
	HeaderRateLimitRemaining = "X-RateLimit-Remaining"
	HeaderRateLimitReset     = "X-RateLimit-Reset"
	HeaderRetryAfter         = "Retry-After"
)

// KeyFunc достает ключ из запроса
type KeyFunc func(r *http.Request) string

// ErrorHandler обрабатывает ошибки, связанные с превышением лимита
type ErrorHandler func(w http.ResponseWriter, r *http.Request, result limiter.Result)

// Config содержит конфиг для middleware
type Config struct {
	// Limiter используемый ограничитель.
	Limiter limiter.Limiter

	// По умолчанию: IP-адрес клиента
	KeyFunc KeyFunc

	// По умолчанию: возвращает 429 с телом JSON.
	ErrorHandler ErrorHandler

	// SkipFunc определяет нужно ли пропустить ограничение запроса
	// возвращает true когда нужно пропустить ограничение для текущего запроса
	SkipFunc func(r *http.Request) bool
}

// DefaultKeyFunc возвращает IP-адрес клиента
func DefaultKeyFunc(r *http.Request) string {
	// Проверить X-Forwarded-For сначала (для прокси)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Проверить X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Откат к RemoteAddr
	return r.RemoteAddr
}

// DefaultErrorHandler возвращает 429 ответ с JSON телом.
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, result limiter.Result) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(HeaderRetryAfter, strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
	w.WriteHeader(http.StatusTooManyRequests)
	
	w.Write([]byte(`{"error":"rate limit exceeded","retry_after":` + 
		strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10) + `}`))
}

// SetRateLimitHeaders добавляет заголовки к запросу.
func SetRateLimitHeaders(w http.ResponseWriter, result limiter.Result) {
	w.Header().Set(HeaderRateLimitLimit, strconv.FormatInt(result.Limit, 10))
	w.Header().Set(HeaderRateLimitRemaining, strconv.FormatInt(result.Remaining, 10))
	w.Header().Set(HeaderRateLimitReset, strconv.FormatInt(result.ResetAt.Unix(), 10))
}

// applyDefaults заполняет конфиг значениями по умолчанию.
func applyDefaults(cfg *Config) {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = DefaultKeyFunc
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = DefaultErrorHandler
	}
}
