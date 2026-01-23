// Базовый пример — прямое использование API без HTTP сервера
package main

import (
	"fmt"
	"time"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
)

func main() {
	fmt.Println("=== Basic Rate Limiter Example ===")

	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     5,
		Interval: time.Second,
		Burst:    3,
	})

	for i := 1; i <= 10; i++ {
		result := limiter.Allow("user:123")

		if result.Allowed {
			fmt.Printf("Request %2d: ✅ Allowed (remaining: %d)\n", i, result.Remaining)
		} else {
			fmt.Printf("Request %2d: ❌ Denied (retry after: %v)\n", i, result.RetryAfter.Round(time.Millisecond))
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n--- Waiting for tokens to refill ---")
	time.Sleep(1 * time.Second)

	result := limiter.Allow("user:123")
	fmt.Printf("Request 11: %s (remaining: %d)\n",
		map[bool]string{true: "✅ Allowed", false: "❌ Denied"}[result.Allowed],
		result.Remaining)
}
