// Gin API пример с ограничением запросов
package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
	ginmw "github.com/BuzzLyutic/ratelimiter-go-v2/middleware/gin"
)

func main() {
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Minute,
		Burst:    5,
	})

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(ginmw.RateLimitByIP(limiter))

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to Rate Limited API",
			"tip":     "Check X-RateLimit-* headers in response",
		})
	})

	r.GET("/api/users", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"users": []string{"alice", "bob", "charlie"},
		})
	})

	r.GET("/api/time", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"time": time.Now().Format(time.RFC3339),
		})
	})

	fmt.Println("Gin API starting on http://localhost:8080")
	fmt.Println("Rate limit: 10 req/min, burst: 5")
	fmt.Println("\nTry:")
	fmt.Println("  curl -i http://localhost:8080/")
	fmt.Println("  for i in {1..10}; do curl -s http://localhost:8080/api/time; done")

	r.Run(":8080")
}
