// Interactive demo — test rate limiter from command line
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/slidingcounter"
	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/slidinglog"
	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          🚦 Rate Limiter Интерактивная демонстрация        ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Commands:                                                 ║")
	fmt.Println("║    r [n]     - Make n requests (default: 1)                ║")
	fmt.Println("║    burst n   - Make n requests instantly                   ║")
	fmt.Println("║    wait n    - Wait n seconds                              ║")
	fmt.Println("║    status    - Show current status                         ║")
	fmt.Println("║    algo X    - Switch algorithm (tb/sl/sc)                 ║")
	fmt.Println("║    reset     - Reset limiter                               ║")
	fmt.Println("║    quit      - Exit                                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Default: Token Bucket
	var currentAlgo string
	var allow func(key string) (bool, int64, time.Duration)

	setAlgorithm := func(name string) {
		switch name {
		case "tb":
			limiter := tokenbucket.New(tokenbucket.Config{
				Rate:     10,
				Interval: time.Second,
				Burst:    5,
			})
			allow = func(key string) (bool, int64, time.Duration) {
				r := limiter.Allow(key)
				return r.Allowed, r.Remaining, r.RetryAfter
			}
			currentAlgo = "Token Bucket (10/s, burst 5)"

		case "sl":
			limiter := slidinglog.New(slidinglog.Config{
				Limit:  10,
				Window: time.Second,
			})
			allow = func(key string) (bool, int64, time.Duration) {
				r := limiter.Allow(key)
				return r.Allowed, r.Remaining, r.RetryAfter
			}
			currentAlgo = "Sliding Log (10/s)"

		case "sc":
			limiter := slidingcounter.New(slidingcounter.Config{
				Limit:  10,
				Window: time.Second,
			})
			allow = func(key string) (bool, int64, time.Duration) {
				r := limiter.Allow(key)
				return r.Allowed, r.Remaining, r.RetryAfter
			}
			currentAlgo = "Sliding Counter (10/s)"

		default:
			fmt.Println("Unknown algorithm. Use: tb, sl, sc")
			return
		}
		fmt.Printf("✅ Switched to: %s\n", currentAlgo)
	}

	// Start with Token Bucket
	setAlgorithm("tb")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n[%s] > ", currentAlgo)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		parts := strings.Fields(input)

		if len(parts) == 0 {
			continue
		}

		cmd := parts[0]

		switch cmd {
		case "r", "request":
			n := 1
			if len(parts) > 1 {
				n, _ = strconv.Atoi(parts[1])
			}
			for i := 0; i < n; i++ {
				allowed, remaining, retry := allow("user:demo")
				if allowed {
					fmt.Printf("  ✅ Request %d: Allowed (remaining: %d)\n", i+1, remaining)
				} else {
					fmt.Printf("  ❌ Request %d: Denied (retry: %v)\n", i+1, retry.Round(time.Millisecond))
				}
				if i < n-1 {
					time.Sleep(50 * time.Millisecond)
				}
			}

		case "burst":
			n := 10
			if len(parts) > 1 {
				n, _ = strconv.Atoi(parts[1])
			}
			allowed, denied := 0, 0
			for i := 0; i < n; i++ {
				ok, _, _ := allow("user:demo")
				if ok {
					allowed++
				} else {
					denied++
				}
			}
			fmt.Printf("  Burst %d: ✅ %d allowed, ❌ %d denied\n", n, allowed, denied)

		case "wait":
			secs := 1
			if len(parts) > 1 {
				secs, _ = strconv.Atoi(parts[1])
			}
			fmt.Printf("  ⏳ Waiting %d seconds...\n", secs)
			time.Sleep(time.Duration(secs) * time.Second)
			fmt.Println("  ✅ Done")

		case "status":
			allowed, remaining, retry := allow("user:demo")
			fmt.Printf("  Algorithm: %s\n", currentAlgo)
			fmt.Printf("  Would be allowed: %v\n", allowed)
			fmt.Printf("  Remaining: %d\n", remaining)
			if !allowed {
				fmt.Printf("  Retry after: %v\n", retry)
			}

		case "algo":
			if len(parts) > 1 {
				setAlgorithm(parts[1])
			} else {
				fmt.Println("  Usage: algo [tb|sl|sc]")
			}

		case "reset":
			setAlgorithm(strings.Split(currentAlgo, " ")[0])
			fmt.Println("  ✅ Limiter reset")

		case "quit", "exit", "q":
			fmt.Println("👋 Bye!")
			return

		case "help", "?":
			fmt.Println("  r [n]    - Make n requests")
			fmt.Println("  burst n  - Make n requests instantly")
			fmt.Println("  wait n   - Wait n seconds")
			fmt.Println("  algo X   - Switch: tb=TokenBucket, sl=SlidingLog, sc=SlidingCounter")
			fmt.Println("  quit     - Exit")

		default:
			fmt.Printf("  Unknown command: %s (type 'help')\n", cmd)
		}
	}
}
