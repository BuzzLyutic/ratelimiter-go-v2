// Chi API пример с ограничением запросов
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/slidingcounter"
	rlmw "github.com/BuzzLyutic/ratelimiter-go-v2/middleware/chi"
)

func main() {
	limiter := slidingcounter.New(slidingcounter.Config{
		Limit:  20,
		Window: time.Minute,
	})

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Use(rlmw.RateLimitByIP(limiter))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Welcome to Chi API with Sliding Window Counter",
		})
	})

	r.Get("/api/data", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":      []int{1, 2, 3, 4, 5},
			"timestamp": time.Now().Unix(),
		})
	})

	fmt.Println("Chi API starting on http://localhost:8081")
	fmt.Println("Rate limit: 20 req/min (sliding window counter)")
	fmt.Println("\nTry:")
	fmt.Println("  curl -i http://localhost:8081/")

	http.ListenAndServe(":8081", r)
}
