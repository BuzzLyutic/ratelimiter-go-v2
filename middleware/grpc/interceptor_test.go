package grpc

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
)

func TestUnaryServerInterceptor_Allowed(t *testing.T) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     100,
		Interval: time.Minute,
		Burst:    10,
	})

	interceptor := UnaryServerInterceptor(Config{
		Limiter: limiter,
	})

	// Мок хэндлера
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	// Мок инфо
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	ctx := context.Background()
	resp, err := interceptor(ctx, nil, info, handler)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp != "ok" {
		t.Errorf("expected 'ok', got %v", resp)
	}
}

func TestUnaryServerInterceptor_Denied(t *testing.T) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    2,
	})

	interceptor := UnaryServerInterceptor(Config{
		Limiter: limiter,
	})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	ctx := context.Background()

	for i := 0; i < 2; i++ {
		_, err := interceptor(ctx, nil, info, handler)
		if err != nil {
			t.Errorf("request %d: expected no error, got %v", i+1, err)
		}
	}

	_, err := interceptor(ctx, nil, info, handler)
	if err == nil {
		t.Error("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.ResourceExhausted {
		t.Errorf("expected ResourceExhausted, got %v", st.Code())
	}
}

func TestUnaryServerInterceptor_Skip(t *testing.T) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    1,
	})

	interceptor := UnaryServerInterceptor(Config{
		Limiter: limiter,
		SkipFunc: func(fullMethod string) bool {
			return fullMethod == "/test.Service/Health"
		},
	})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	ctx := context.Background()

	healthInfo := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Health"}
	for i := 0; i < 10; i++ {
		_, err := interceptor(ctx, nil, healthInfo, handler)
		if err != nil {
			t.Errorf("health check %d should not be rate limited", i+1)
		}
	}
}

func TestUnaryServerInterceptor_CustomKeyFunc(t *testing.T) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    1,
	})

	interceptor := UnaryServerInterceptor(Config{
		Limiter: limiter,
		KeyFunc: func(ctx context.Context) string {
			return "custom-key"
		},
	})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	ctx := context.Background()

	_, err := interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Errorf("first request should be allowed: %v", err)
	}

	_, err = interceptor(ctx, nil, info, handler)
	if err == nil {
		t.Error("second request should be denied")
	}
}

func BenchmarkUnaryServerInterceptor(b *testing.B) {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     1000000,
		Interval: time.Second,
		Burst:    1000000,
	})

	interceptor := UnaryServerInterceptor(Config{
		Limiter: limiter,
	})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			interceptor(ctx, nil, info, handler)
		}
	})
}
