package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
)

func newTestMetrics() (*Metrics, *prometheus.Registry) {
	reg := prometheus.NewRegistry()
	m := NewWithRegistry(DefaultConfig(), reg)
	return m, reg
}

func TestMetrics_ObserveAllow(t *testing.T) {
	m, reg := newTestMetrics()

	m.ObserveAllow("tokenbucket", 100*time.Microsecond)
	m.ObserveAllow("tokenbucket", 200*time.Microsecond)

	// Проверить счетчик
	expected := `
		# HELP ratelimiter_requests_allowed_total Total number of allowed requests.
		# TYPE ratelimiter_requests_allowed_total counter
		ratelimiter_requests_allowed_total{algorithm="tokenbucket"} 2
	`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "ratelimiter_requests_allowed_total"); err != nil {
		t.Errorf("unexpected metric: %v", err)
	}
}

func TestMetrics_ObserveDeny(t *testing.T) {
	m, reg := newTestMetrics()

	m.ObserveDeny("slidingwindow", 150*time.Microsecond)

	// Проверить счетчик
	expected := `
		# HELP ratelimiter_requests_limited_total Total number of rate-limited requests.
		# TYPE ratelimiter_requests_limited_total counter
		ratelimiter_requests_limited_total{algorithm="slidingwindow"} 1
	`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "ratelimiter_requests_limited_total"); err != nil {
		t.Errorf("unexpected metric: %v", err)
	}
}

func TestMetrics_RequestsTotal(t *testing.T) {
	m, _ := newTestMetrics()

	m.ObserveAllow("tokenbucket", time.Microsecond)
	m.ObserveAllow("tokenbucket", time.Microsecond)
	m.ObserveDeny("tokenbucket", time.Microsecond)

	count := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("tokenbucket", "allowed"))
	if count != 2 {
		t.Errorf("expected 2 allowed, got %f", count)
	}

	count = testutil.ToFloat64(m.RequestsTotal.WithLabelValues("tokenbucket", "denied"))
	if count != 1 {
		t.Errorf("expected 1 denied, got %f", count)
	}
}

func TestMetrics_SetActiveKeys(t *testing.T) {
	m, _ := newTestMetrics()

	m.SetActiveKeys("tokenbucket", 100)

	count := testutil.ToFloat64(m.KeysActive.WithLabelValues("tokenbucket"))
	if count != 100 {
		t.Errorf("expected 100 active keys, got %f", count)
	}

	m.SetActiveKeys("tokenbucket", 50)
	count = testutil.ToFloat64(m.KeysActive.WithLabelValues("tokenbucket"))
	if count != 50 {
		t.Errorf("expected 50 active keys, got %f", count)
	}
}

func TestLimiterWrapper(t *testing.T) {
	m, _ := newTestMetrics()

	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    2,
	})

	wrapped := Wrap(limiter, m, "tokenbucket")

	for i := 0; i < 2; i++ {
		result := wrapped.Allow("user:1")
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	result := wrapped.Allow("user:1")
	if result.Allowed {
		t.Error("3rd request should be denied")
	}

	allowed := testutil.ToFloat64(m.RequestsAllowed.WithLabelValues("tokenbucket"))
	if allowed != 2 {
		t.Errorf("expected 2 allowed, got %f", allowed)
	}

	denied := testutil.ToFloat64(m.RequestsLimited.WithLabelValues("tokenbucket"))
	if denied != 1 {
		t.Errorf("expected 1 denied, got %f", denied)
	}
}

func TestLimiterWrapper_AllowN(t *testing.T) {
	m, _ := newTestMetrics()

	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     100,
		Interval: time.Second,
		Burst:    10,
	})

	wrapped := Wrap(limiter, m, "tokenbucket")

	result := wrapped.AllowN("user:1", 5)
	if !result.Allowed {
		t.Error("AllowN(5) should be allowed")
	}

	allowed := testutil.ToFloat64(m.RequestsAllowed.WithLabelValues("tokenbucket"))
	if allowed != 1 {
		t.Errorf("expected 1 allowed call, got %f", allowed)
	}
}

func BenchmarkLimiterWrapper(b *testing.B) {
	reg := prometheus.NewRegistry()
	m := NewWithRegistry(DefaultConfig(), reg)

	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     1000000,
		Interval: time.Second,
		Burst:    1000000,
	})

	wrapped := Wrap(limiter, m, "tokenbucket")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wrapped.Allow("user:1")
		}
	})
}

func BenchmarkLimiterWithoutMetrics(b *testing.B) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     1000000,
		Interval: time.Second,
		Burst:    1000000,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow("user:1")
		}
	})
}
