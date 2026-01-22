// Package grpc предоставляет gRPC интерцепторы для ограничения запросов.
package grpc

import (
	"context"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter"
)

// KeyFunc достает ключ из контекста
type KeyFunc func(ctx context.Context) string

// Config содержит конфигурацию интерцептора
type Config struct {
	// Limiter используемый ограничитель запросов
	Limiter limiter.Limiter

	// KeyFunc достает ключ из запроса
	// По умолчанию: IP-адрес клиента
	KeyFunc KeyFunc

	// SkipFunc определяет нужно ли пропустить ограничение запроса
	// Получает полное имя метода ("/package.Service/Method").
	SkipFunc func(fullMethod string) bool
}

// DefaultKeyFunc возвращает клиентский IP из информации пира
func DefaultKeyFunc(ctx context.Context) string {
	// Пытаемся получить информацию пира
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
			return xff[0]
		}
	}

	return "unknown"
}

// UnaryServerInterceptor возвращает перехватчик для ограничения запроса.
func UnaryServerInterceptor(cfg Config) grpc.UnaryServerInterceptor {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = DefaultKeyFunc
	}

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if cfg.SkipFunc != nil && cfg.SkipFunc(info.FullMethod) {
			return handler(ctx, req)
		}

		key := cfg.KeyFunc(ctx)

		result := cfg.Limiter.Allow(key)

		if !result.Allowed {
			return nil, status.Errorf(
				codes.ResourceExhausted,
				"rate limit exceeded, retry after %v",
				result.RetryAfter,
			)
		}

		header := metadata.Pairs(
			"x-ratelimit-limit", formatInt64(result.Limit),
			"x-ratelimit-remaining", formatInt64(result.Remaining),
		)
		grpc.SetHeader(ctx, header)

		return handler(ctx, req)
	}
}

func StreamServerInterceptor(cfg Config) grpc.StreamServerInterceptor {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = DefaultKeyFunc
	}

	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		if cfg.SkipFunc != nil && cfg.SkipFunc(info.FullMethod) {
			return handler(srv, ss)
		}

		key := cfg.KeyFunc(ctx)

		result := cfg.Limiter.Allow(key)

		if !result.Allowed {
			return status.Errorf(
				codes.ResourceExhausted,
				"rate limit exceeded, retry after %v",
				result.RetryAfter,
			)
		}

		header := metadata.Pairs(
			"x-ratelimit-limit", formatInt64(result.Limit),
			"x-ratelimit-remaining", formatInt64(result.Remaining),
		)
		grpc.SetHeader(ctx, header)

		return handler(srv, ss)
	}
}

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}
