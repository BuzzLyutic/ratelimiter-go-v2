package middleware

import (
	"net/http"
)

// Stdlib возвращает middleware для net/http.
func Stdlib(cfg Config) func(http.Handler) http.Handler {
	applyDefaults(&cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.SkipFunc != nil && cfg.SkipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			key := cfg.KeyFunc(r)

			result := cfg.Limiter.Allow(key)

			SetRateLimitHeaders(w, result)

			if !result.Allowed {
				cfg.ErrorHandler(w, r, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
