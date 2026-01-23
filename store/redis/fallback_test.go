package redis

import (
	"context"
	"testing"
	"time"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
)

func TestFallbackStore_UsesRedis(t *testing.T) {
	cfg := getTestConfig()
	
	store, err := NewFallbackStore(FallbackConfig{
		Redis: cfg,
		MemoryConfig: tokenbucket.Config{
			Rate:     100,
			Interval: time.Minute,
			Burst:    10,
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	if store.redis != nil && store.IsUsingMemory() {
		t.Error("should use Redis when available")
	}

	ctx := context.Background()
	result, err := store.TokenBucket(ctx, "test:fallback:1", 10, 10, 1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !result.Allowed {
		t.Error("first request should be allowed")
	}
}

func TestFallbackStore_FallsBackToMemory(t *testing.T) {
	store, err := NewFallbackStore(FallbackConfig{
		Redis: Config{
			Addr:        "localhost:9999", // Неправильный порт
			DialTimeout: 100 * time.Millisecond,
		},
		MemoryConfig: tokenbucket.Config{
			Rate:     100,
			Interval: time.Minute,
			Burst:    10,
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	if !store.IsUsingMemory() {
		t.Error("should use memory when Redis unavailable")
	}

	// Запрос все еще должен отработать
	ctx := context.Background()
	result, err := store.TokenBucket(ctx, "test:fallback:2", 10, 10, 1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !result.Allowed {
		t.Error("first request should be allowed")
	}
}

func TestFallbackStore_RateLimiting(t *testing.T) {
	store, err := NewFallbackStore(FallbackConfig{
		Redis: Config{
			Addr:        "localhost:9999", 
			DialTimeout: 100 * time.Millisecond,
		},
		MemoryConfig: tokenbucket.Config{
			Rate:     10,
			Interval: time.Second,
			Burst:    5,
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		result, err := store.TokenBucket(ctx, "test:fallback:3", 10, 5, 1)
		if err != nil {
			t.Fatalf("TokenBucket failed: %v", err)
		}
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	result, err := store.TokenBucket(ctx, "test:fallback:3", 10, 5, 1)
	if err != nil {
		t.Fatalf("TokenBucket failed: %v", err)
	}
	if result.Allowed {
		t.Error("6th request should be denied")
	}
}
