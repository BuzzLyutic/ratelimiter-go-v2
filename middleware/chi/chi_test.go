package chi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
	"github.com/BuzzLyutic/ratelimiter-go-v2/middleware"
)

func setupRouter(l *tokenbucket.Limiter) *chi.Mux {
	r := chi.NewRouter()
	r.Use(RateLimitByIP(l))
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"ok"}`))
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

		if i < 2 && w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
		if i >= 2 && w.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: expected 429, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimit_Skip(t *testing.T) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    1,
	})

	r := chi.NewRouter()
	r.Use(RateLimit(Config{
		Limiter: limiter,
		SkipFunc: func(r *http.Request) bool {
			return r.URL.Path == "/health"
		},
	}))
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("health check %d should not be rate limited", i+1)
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
