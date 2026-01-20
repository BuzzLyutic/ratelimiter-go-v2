package store

import (
	"context"
	"time"
)

type Storage interface {
	//Операции для Token bucket
	TokenBucketTake(ctx context.Context, key string, rate float64, capacity int64, now time.Time) (allowed bool, remaining int64, retryAfter time.Duration, err error)

	//Операции для Sliding window
	SlidingWindowAdd(ctx context.Context, key string, now time.Time, window time.Duration) error
	SlidingWindowCount(ctx context.Context, key string, now time.Time, window time.Duration) (int64, error)

	//Операции для Sliding counter
	SlidingCounterIncr(ctx context.Context, key string, now time.Time, window time.Duration) (int64, error)

	// Закрытие соединения с хранилищем
	Close() error
}
