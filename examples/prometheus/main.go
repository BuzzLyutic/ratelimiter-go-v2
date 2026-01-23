package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/BuzzLyutic/ratelimiter-go-v2/limiter/tokenbucket"
	"github.com/BuzzLyutic/ratelimiter-go-v2/metrics"
	ginmw "github.com/BuzzLyutic/ratelimiter-go-v2/middleware/gin"
)

func main() {
	// 1. Создать метрики
	m := metrics.New(metrics.DefaultConfig())

	// 2. Создать ограничитель
	limiter := tokenbucket.New(tokenbucket.Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    5,
	})

	// 3. Обернуть метриками
	wrappedLimiter := metrics.Wrap(limiter, m, "tokenbucket")

	// 4. Создать Gin роутер
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 5. Добавить middleware
	r.Use(ginmw.RateLimitByIP(wrappedLimiter))

	// 6. Конечная точка метрик Prometheus
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 7. Протестить конечные точки
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	// 8. Периодически обновлять активные ключи
	go func() {
		for range time.Tick(10 * time.Second) {
			m.SetActiveKeys("tokenbucket", limiter.Len())
		}
	}()

	fmt.Println("Server starting on :8080")
	fmt.Println("Metrics: http://localhost:8080/metrics")
	fmt.Println("Test:    http://localhost:8080/api/test")
	log.Fatal(http.ListenAndServe(":8080", r))
}
