package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
	"github.com/BuzzLyutic/ratelimiter-go-v2/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter(l *tokenbucket.Limiter) *gin.Engine {
	r := gin.New()
	r.Use(RateLimitByIP(l))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})
	return r
}

func TestRateLimit_Allowed(t *testing.T) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     100,
		Interval: time.Minute,
		Burst:    10,
	})

	router := setupRouter(limiter)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Проверить заголовки
	if w.Header().Get(middleware.HeaderRateLimitLimit) == "" {
		t.Error("missing X-RateLimit-Limit header")
	}
	if w.Header().Get(middleware.HeaderRateLimitRemaining) == "" {
		t.Error("missing X-RateLimit-Remaining header")
	}
	if w.Header().Get(middleware.HeaderRateLimitReset) == "" {
		t.Error("missing X-RateLimit-Reset header")
	}
}

func TestRateLimit_Denied(t *testing.T) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    2, 
	})

	router := setupRouter(limiter)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if i < 2 {
			if w.Code != http.StatusOK {
				t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
			}
		} else {
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("request %d: expected 429, got %d", i+1, w.Code)
			}
			if w.Header().Get(middleware.HeaderRetryAfter) == "" {
				t.Error("missing Retry-After header")
			}
		}
	}
}

func TestRateLimit_DifferentIPs(t *testing.T) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    1,
	})

	router := setupRouter(limiter)

	// Разные IP должны иметь отдельные лимиты
	ips := []string{"192.168.1.1:12345", "192.168.1.2:12345"}

	for _, ip := range ips {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("IP %s: expected 200, got %d", ip, w.Code)
		}
	}
}

func TestRateLimit_Skip(t *testing.T) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    1,
	})

	r := gin.New()
	r.Use(RateLimit(Config{
		Limiter: limiter,
		SkipFunc: func(c *gin.Context) bool {
			// Пропуск для /health конечной точки
			return c.Request.URL.Path == "/health"
		},
	}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// /health не должен подвергаться ограничителям
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("health check %d should not be rate limited", i+1)
		}
	}
}

func TestRateLimit_CustomKeyFunc(t *testing.T) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    1,
	})

	r := gin.New()
	r.Use(RateLimitByKey(limiter, func(c *gin.Context) string {
		// Ограничение по API ключу
		return c.GetHeader("X-API-Key")
	}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "key-123")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if i == 0 && w.Code != http.StatusOK {
			t.Error("first request should be allowed")
		}
		if i == 1 && w.Code != http.StatusTooManyRequests {
			t.Error("second request should be denied (same key)")
		}
	}
}

func BenchmarkRateLimit(b *testing.B) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     1000000,
		Interval: time.Second,
		Burst:    1000000,
	})

	router := setupRouter(limiter)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}
