package redis

import (
	"context"
	"sync"
	"time"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
)

// FallbackStore оборачивает Redis хранилище с failover в in-memory
type FallbackStore struct {
	redis     *Store
	memory    *tokenbucket.Limiter
	mu        sync.RWMutex
	useMemory bool
	checkInterval time.Duration
	lastCheck time.Time
}

// FallbackConfig содержит конфиг резервного хранилища
type FallbackConfig struct {
	Redis         Config
	CheckInterval time.Duration // Частота проверки доступности Redis
	MemoryConfig  tokenbucket.Config
}

// NewFallbackStore создает хранилище, откатывающееся в in-memory при недоступности Redis
func NewFallbackStore(cfg FallbackConfig) (*FallbackStore, error) {
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 5 * time.Second
	}

	fs := &FallbackStore{
		memory:        tokenbucket.New(cfg.MemoryConfig),
		checkInterval: cfg.CheckInterval,
		useMemory:     true,
	}

	redis, err := New(cfg.Redis)
	if err != nil {
		return fs, nil
	}

	fs.redis = redis
	fs.useMemory = false

	// Start health checker
	go fs.healthChecker()

	return fs, nil
}

// healthChecker периодически проверяет доступность Redis
func (fs *FallbackStore) healthChecker() {
	ticker := time.NewTicker(fs.checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		fs.checkRedis()
	}
}

// checkRedis проверяет доступность Redis и обновляет состояние
func (fs *FallbackStore) checkRedis() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.redis == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := fs.redis.Ping(ctx)
	if err != nil {
		if !fs.useMemory {
			fs.useMemory = true
		}
	} else {
		if fs.useMemory {
			fs.useMemory = false
		}
	}
	fs.lastCheck = time.Now()
}

// TokenBucket
func (fs *FallbackStore) TokenBucket(ctx context.Context, key string, rate float64, capacity int64, tokens int64) (*TokenBucketResult, error) {
	fs.mu.RLock()
	useMemory := fs.useMemory
	fs.mu.RUnlock()

	if useMemory || fs.redis == nil {
		result := fs.memory.AllowN(key, tokens)
		return &TokenBucketResult{
			Allowed:    result.Allowed,
			Remaining:  result.Remaining,
			RetryAfter: result.RetryAfter,
		}, nil
	}

	result, err := fs.redis.TokenBucket(ctx, key, rate, capacity, tokens)
	if err != nil {
		fs.mu.Lock()
		fs.useMemory = true
		fs.mu.Unlock()

		memResult := fs.memory.AllowN(key, tokens)
		return &TokenBucketResult{
			Allowed:    memResult.Allowed,
			Remaining:  memResult.Remaining,
			RetryAfter: memResult.RetryAfter,
		}, nil
	}

	return result, nil
}

func (fs *FallbackStore) IsUsingMemory() bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.useMemory
}

func (fs *FallbackStore) Close() error {
	if fs.redis != nil {
		return fs.redis.Close()
	}
	return nil
}
