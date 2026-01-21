package slidingcounter

import (
	"fmt"
	"testing"
	"time"
)

func TestNew(t *testing.T) { // Создание нового ограничителя
	cfg := Config{
		Limit:  100,
		Window: time.Minute,
	}

	l := New(cfg)

	if l.limit != 100 {
		t.Errorf("expected limit 100, got %d", l.limit)
	}

	if l.window != time.Minute {
		t.Errorf("expected window 1m, got %v", l.window)
	}
}

func TestAllow_Basic(t *testing.T) { // Базовая работоспособность ограничителя
	l := New(Config{
		Limit:  5,
		Window: time.Minute,
	})

	for i := 0; i < 5; i++ {
		result := l.Allow("user:1")
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	result := l.Allow("user:1")
	if result.Allowed {
		t.Error("6th request should be denied")
	}
}

func TestAllow_DifferentKeys(t *testing.T) { // Создаются ли отдельные ограничители для разных ключей
	l := New(Config{
		Limit:  1,
		Window: time.Minute,
	})

	result1 := l.Allow("user:1")
	result2 := l.Allow("user:2")

	if !result1.Allowed || !result2.Allowed {
		t.Error("different keys should have separate limits")
	}
}

func TestAllow_WindowSlide(t *testing.T) { // Корректность сдвига окна
	l := New(Config{
		Limit:  2,
		Window: 100 * time.Millisecond,
	})

	// Используем оба слота
	l.Allow("user:1")
	l.Allow("user:1")

	result := l.Allow("user:1")
	if result.Allowed {
		t.Error("should be denied when limit reached")
	}

	// Ждем скольжения окна
	time.Sleep(150 * time.Millisecond)

	result = l.Allow("user:1")
	if !result.Allowed {
		t.Error("should be allowed after window slides")
	}
}

func TestAllowN(t *testing.T) { // Корректность обработки сразу n запросов
	l := New(Config{
		Limit:  10,
		Window: time.Minute,
	})

	result := l.AllowN("user:1", 5)
	if !result.Allowed {
		t.Error("AllowN(5) should be allowed")
	}

	result = l.AllowN("user:1", 6)
	if result.Allowed {
		t.Error("AllowN(6) should be denied")
	}

	result = l.AllowN("user:1", 5)
	if !result.Allowed {
		t.Error("AllowN(5) should be allowed")
	}
}

func TestAllow_RetryAfter(t *testing.T) { // Тест повтора после неудачи
	l := New(Config{
		Limit:  1,
		Window: time.Second,
	})

	l.Allow("user:1")

	result := l.Allow("user:1")
	if result.Allowed {
		t.Error("should be denied")
	}

	if result.RetryAfter <= 0 || result.RetryAfter > time.Second {
		t.Errorf("RetryAfter should be between 0 and 1s, got %v", result.RetryAfter)
	}
}

func TestClean(t *testing.T) { // Проверка очистки неиспользуемых счетчиков
	l := New(Config{
		Limit:  10,
		Window: 50 * time.Millisecond,
	})

	l.Allow("user:1")
	l.Allow("user:2")
	l.Allow("user:3")

	if l.Len() != 3 {
		t.Errorf("expected 3 counters, got %d", l.Len())
	}

	// Подождать и очистить
	time.Sleep(100 * time.Millisecond)
	cleaned := l.Clean(10 * time.Millisecond)

	if cleaned != 3 {
		t.Errorf("expected to clean 3, cleaned %d", cleaned)
	}

	if l.Len() != 0 {
		t.Errorf("expected 0 counters, got %d", l.Len())
	}
}

func TestAllow_Concurrent(t *testing.T) { // Проверка параллельности
	l := New(Config{
		Limit:  1000,
		Window: time.Minute,
	})

	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func() {
			l.Allow("user:1")
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestCalculateWeight(t *testing.T) { // Проверка расчета весов окон
	l := New(Config{
		Limit:  100,
		Window: time.Second,
	})

	// Создать счетчик в начале окна
	now := time.Now()
	c := &counter{
		currStart: now.Truncate(time.Second),
	}

	// В самом начале вес должен быть примерно 1
	weight := l.calculateWeight(c, c.currStart)
	if weight < 0.99 {
		t.Errorf("weight at start should be ~1, got %f", weight)
	}

	// Ближе к концу вес стремится к 0
	weight = l.calculateWeight(c, c.currStart.Add(time.Second-time.Millisecond))
	if weight > 0.01 {
		t.Errorf("weight at end should be ~0, got %f", weight)
	}
}

// Бенчмарки

func BenchmarkAllow(b *testing.B) {
	l := New(Config{
		Limit:  int64(b.N) + 1000,
		Window: time.Hour,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Allow("user:1")
	}
}

func BenchmarkAllow_Parallel(b *testing.B) {
	l := New(Config{
		Limit:  1000000,
		Window: time.Hour,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Allow("user:1")
		}
	})
}

func BenchmarkAllow_MultipleKeys(b *testing.B) {
	l := New(Config{
		Limit:  100,
		Window: time.Second,
	})

	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = fmt.Sprintf("user:%d", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			l.Allow(keys[i%len(keys)])
			i++
		}
	})
}
