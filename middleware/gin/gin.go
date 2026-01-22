// Package gin предоставляет Gin middleware для ограничения запросов
package gin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter"
	"github.com/BuzzLyutic/ratelimiter-go-v2/middleware"
)

type Config struct {
	// Limiter используемый ограничитель запросов
	Limiter limiter.Limiter

	// KeyFunc достает ключ из запроса
	// По умолчанию: IP-адрес клиента
	KeyFunc func(c *gin.Context) string

	// ErrorHandler обрабатывает ошибки ограничителя запросов
	// По умолчанию: возвращает 429 с телом JSON.
	ErrorHandler func(c *gin.Context, result limiter.Result)

	// SkipFunc определяет нужно ли пропустить ограничение запроса
	SkipFunc func(c *gin.Context) bool
}

// DefaultKeyFunc возвращает клиентский IP используя Gin ClientIP().
func DefaultKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

// DefaultErrorHandler возвращает ответ 429.
func DefaultErrorHandler(c *gin.Context, result limiter.Result) {
	c.Header(middleware.HeaderRetryAfter, strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
	c.AbortWithStatusJSON(429, gin.H{
		"error":       "rate limit exceeded",
		"retry_after": int64(result.RetryAfter.Seconds()),
	})
}

// RateLimit возвращает Gin middleware для ограничения запросов
func RateLimit(cfg Config) gin.HandlerFunc {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = DefaultKeyFunc
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = DefaultErrorHandler
	}

	return func(c *gin.Context) {
		if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
			c.Next()
			return
		}

		key := cfg.KeyFunc(c)

		result := cfg.Limiter.Allow(key)

		c.Header(middleware.HeaderRateLimitLimit, strconv.FormatInt(result.Limit, 10))
		c.Header(middleware.HeaderRateLimitRemaining, strconv.FormatInt(result.Remaining, 10))
		c.Header(middleware.HeaderRateLimitReset, strconv.FormatInt(result.ResetAt.Unix(), 10))

		if !result.Allowed {
			cfg.ErrorHandler(c, result)
			return
		}

		c.Next()
	}
}

// RateLimitByKey возвращает middleware использующий кастомный ключ.
func RateLimitByKey(l limiter.Limiter, keyFunc func(c *gin.Context) string) gin.HandlerFunc {
	return RateLimit(Config{
		Limiter: l,
		KeyFunc: keyFunc,
	})
}

// RateLimitByIP возвращает middleware, ограничивающий по IP.
func RateLimitByIP(l limiter.Limiter) gin.HandlerFunc {
	return RateLimit(Config{
		Limiter: l,
		KeyFunc: DefaultKeyFunc,
	})
}
