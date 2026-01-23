// Package redis предоставляет Redis-backed хранилище для ограничителей запросов.
package redis

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed scripts/token_bucket.lua
var tokenBucketScript string

//go:embed scripts/sliding_window.lua
var slidingWindowScript string

// Кастомные ошибки
var (
	ErrRedisUnavailable = errors.New("redis is unavailable")
	ErrScriptFailed     = errors.New("lua script execution failed")
)

// Config содержит конфиг соединения Redis
type Config struct {
	Addr string // Addr адрес сервера Redis (default: "localhost:6379")
	Password string // Пароль для аутентификации Redis (default: "")
	DB int // DB Номер базы данных Redis (default: 0)
	PoolSize int // PoolSize максимальное кол-во соединений (default: 10)
	DialTimeout time.Duration // DialTimeout таймаут для установления новых соединений (default: 5s)
	ReadTimeout time.Duration // ReadTimeout таймаут для операций чтения (default: 3s)
	WriteTimeout time.Duration // WriteTimeout таймаут для операций записи (default: 3s)
	KeyPrefix string // KeyPrefix приставка ко всем ключам (default: "ratelimit:")
} 

// DefaultConfig возвращает конфиг со значениями по умолчанию
func DefaultConfig() Config {
	return Config{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		KeyPrefix:    "ratelimit:",
	}
}

// Store это Redis-backed хранилище ограничителей запросов.
type Store struct {
	client          *redis.Client
	keyPrefix       string
	tokenBucketSHA  string
	slidingWindowSHA string
}

// New создает новое хранилище Redis
func New(cfg Config) (*Store, error) {
	if cfg.Addr == "" {
		cfg.Addr = "localhost:6379"
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "ratelimit:"
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 10
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Тест соединения
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	store := &Store{
		client:    client,
		keyPrefix: cfg.KeyPrefix,
	}

	// Загрузка скриптов Lua
	if err := store.loadScripts(ctx); err != nil {
		return nil, fmt.Errorf("failed to load lua scripts: %w", err)
	}

	return store, nil
}

// loadScripts загружает скрипты Lua в Redis и хранит их SHA хэши.
func (s *Store) loadScripts(ctx context.Context) error {
	var err error

	s.tokenBucketSHA, err = s.client.ScriptLoad(ctx, tokenBucketScript).Result()
	if err != nil {
		return fmt.Errorf("failed to load token bucket script: %w", err)
	}

	s.slidingWindowSHA, err = s.client.ScriptLoad(ctx, slidingWindowScript).Result()
	if err != nil {
		return fmt.Errorf("failed to load sliding window script: %w", err)
	}

	return nil
}

// TokenBucketResult содержит результаты операций с token bucket
type TokenBucketResult struct {
	Allowed    bool
	Remaining  int64
	RetryAfter time.Duration
}

// TokenBucket выполняет алгоритм token bucket
func (s *Store) TokenBucket(ctx context.Context, key string, rate float64, capacity int64, tokens int64) (*TokenBucketResult, error) {
	fullKey := s.keyPrefix + "tb:" + key
	now := float64(time.Now().UnixNano()) / 1e9

	result, err := s.client.EvalSha(ctx, s.tokenBucketSHA, []string{fullKey},
		rate,
		capacity,
		now,
		tokens,
	).Slice()

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScriptFailed, err)
	}

	allowed, _ := result[0].(int64)
	remaining, _ := result[1].(int64)
	retryAfterStr, _ := result[2].(string)
	retryAfterSec, _ := strconv.ParseFloat(retryAfterStr, 64)

	return &TokenBucketResult{
		Allowed:    allowed == 1,
		Remaining:  remaining,
		RetryAfter: time.Duration(retryAfterSec * float64(time.Second)),
	}, nil
}

// SlidingWindowResult содержит результаты операций sliding window
type SlidingWindowResult struct {
	Allowed    bool
	Remaining  int64
	RetryAfter time.Duration
}

// SlidingWindow выполняет sliding window counter алгоритм.
func (s *Store) SlidingWindow(ctx context.Context, key string, window time.Duration, limit int64, count int64) (*SlidingWindowResult, error) {
	fullKey := s.keyPrefix + "sw:" + key
	now := float64(time.Now().UnixMilli()) / 1000.0
	windowSec := window.Seconds()

	result, err := s.client.EvalSha(ctx, s.slidingWindowSHA, []string{fullKey},
		now,
		windowSec,
		limit,
		count,
	).Slice()

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScriptFailed, err)
	}

	allowed, _ := result[0].(int64)
	remaining, _ := result[1].(int64)
	retryAfterStr, _ := result[2].(string)
	retryAfterSec, _ := strconv.ParseFloat(retryAfterStr, 64)

	return &SlidingWindowResult{
		Allowed:    allowed == 1,
		Remaining:  remaining,
		RetryAfter: time.Duration(retryAfterSec * float64(time.Second)),
	}, nil
}

// Ping проверяет доступность Redis
func (s *Store) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Close закрывает соединение Redis
func (s *Store) Close() error {
	return s.client.Close()
}

// Client вовзращает клиента Redis
func (s *Store) Client() *redis.Client {
	return s.client
}
