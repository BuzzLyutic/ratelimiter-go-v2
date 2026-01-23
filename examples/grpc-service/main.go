// gRPC service пример с ограничением запросов
package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
	rlgrpc "github.com/BuzzLyutic/ratelimiter-go-v2/middleware/grpc"
)

// Простой echo service (proto не требуется для демо)
type echoServer struct{}

func main() {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     100,
		Interval: time.Minute,
		Burst:    10,
	})

	rateLimitInterceptor := rlgrpc.UnaryServerInterceptor(rlgrpc.Config{
		Limiter: limiter,
		SkipFunc: func(method string) bool {
			return method == "/grpc.health.v1.Health/Check"
		},
	})

	server := grpc.NewServer(
		grpc.UnaryInterceptor(rateLimitInterceptor),
	)

	reflection.Register(server)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	fmt.Println("gRPC server starting on :50051")
	fmt.Println("Rate limit: 100 req/min, burst: 10")
	fmt.Println("\nTo test (requires grpcurl):")
	fmt.Println("  grpcurl -plaintext localhost:50051 list")

	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
