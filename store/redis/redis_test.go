package redis

import (
	"context"
	"os"
	"testing"
	"time"
)

func getTestConfig() Config {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	cfg := DefaultConfig()
	cfg.Addr = addr
	cfg.KeyPrefix = "test:ratelimit:"
	cfg.PoolSize = 100
	return cfg
}

func setupTestStore(t *testing.T) *Store {
	t.Helper()

	cfg := getTestConfig()
	store, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		keys, _ := store.client.Keys(ctx, "test:ratelimit:*").Result()
		if len(keys) > 0 {
			store.client.Del(ctx, keys...)
		}
		store.Close()
	})

	return store
}

func TestNew(t *testing.T) {
	cfg := getTestConfig()
	store, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	if store.tokenBucketSHA == "" {
		t.Error("token bucket script not loaded")
	}
	if store.slidingWindowSHA == "" {
		t.Error("sliding window script not loaded")
	}
}

func TestPing(t *testing.T) {
	store := setupTestStore(t)

	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestTokenBucket_Basic(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	rate := 10.0
	capacity := int64(5)

	for i := 0; i < 5; i++ {
		result, err := store.TokenBucket(ctx, "user:1", rate, capacity, 1)
		if err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	result, err := store.TokenBucket(ctx, "user:1", rate, capacity, 1)
	if err != nil {
		t.Fatalf("request 6 failed: %v", err)
	}
	if result.Allowed {
		t.Error("request 6 should be denied")
	}
	if result.RetryAfter <= 0 {
		t.Error("RetryAfter should be positive")
	}
}

func TestTokenBucket_DifferentKeys(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	rate := 10.0
	capacity := int64(1)

	result1, err := store.TokenBucket(ctx, "user:1", rate, capacity, 1)
	if err != nil {
		t.Fatalf("TokenBucket user:1 failed: %v", err)
	}
	result2, err := store.TokenBucket(ctx, "user:2", rate, capacity, 1)
	if err != nil {
		t.Fatalf("TokenBucket user:2 failed: %v", err)
	}

	if !result1.Allowed || !result2.Allowed {
		t.Error("different keys should have separate buckets")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	rate := 10.0
	capacity := int64(1)

	_, err := store.TokenBucket(ctx, "user:refill", rate, capacity, 1)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	result, err := store.TokenBucket(ctx, "user:refill", rate, capacity, 1)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if result.Allowed {
		t.Error("should be denied after using token")
	}

	time.Sleep(150 * time.Millisecond)

	result, err = store.TokenBucket(ctx, "user:refill", rate, capacity, 1)
	if err != nil {
		t.Fatalf("third request failed: %v", err)
	}
	if !result.Allowed {
		t.Error("should be allowed after refill")
	}
}

func TestSlidingWindow_Basic(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	window := time.Second
	limit := int64(5)

	for i := 0; i < 5; i++ {
		result, err := store.SlidingWindow(ctx, "user:sw:1", window, limit, 1)
		if err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	result, err := store.SlidingWindow(ctx, "user:sw:1", window, limit, 1)
	if err != nil {
		t.Fatalf("request 6 failed: %v", err)
	}
	if result.Allowed {
		t.Error("request 6 should be denied")
	}
}

func TestSlidingWindow_WindowExpiry(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	window := 200 * time.Millisecond
	limit := int64(2)

	_, err := store.SlidingWindow(ctx, "user:sw:expiry3", window, limit, 1)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	_, err = store.SlidingWindow(ctx, "user:sw:expiry3", window, limit, 1)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	result, err := store.SlidingWindow(ctx, "user:sw:expiry3", window, limit, 1)
	if err != nil {
		t.Fatalf("third request failed: %v", err)
	}
	if result.Allowed {
		t.Error("should be denied when limit reached")
	}

	time.Sleep(500 * time.Millisecond)

	result, err = store.SlidingWindow(ctx, "user:sw:expiry3", window, limit, 1)
	if err != nil {
		t.Fatalf("fourth request failed: %v", err)
	}
	if !result.Allowed {
		t.Error("should be allowed after 2 windows expire")
	}
}

// Бенчмарки

func BenchmarkTokenBucket(b *testing.B) {
	cfg := getTestConfig()
	store, err := New(cfg)
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	rate := 1000000.0
	capacity := int64(1000000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.TokenBucket(ctx, "bench:user:1", rate, capacity, 1)
		}
	})
}

func BenchmarkTokenBucket_MultipleKeys(b *testing.B) {
	cfg := getTestConfig()
	store, err := New(cfg)
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	rate := 1000000.0
	capacity := int64(1000000)

	keys := make([]string, 100)
	for i := range keys {
		keys[i] = "bench:user:" + string(rune('0'+i%10)) + string(rune('0'+i/10))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = store.TokenBucket(ctx, keys[i%len(keys)], rate, capacity, 1)
			i++
		}
	})
}

func BenchmarkSlidingWindow(b *testing.B) {
	cfg := getTestConfig()
	store, err := New(cfg)
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	window := time.Hour
	limit := int64(1000000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.SlidingWindow(ctx, "bench:sw:1", window, limit, 1)
		}
	})
}
