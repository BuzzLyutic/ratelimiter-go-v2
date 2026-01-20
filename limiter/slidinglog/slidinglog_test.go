package slidinglog

import (
	"fmt"
	"testing"
	"time"
)

func TestNew(t *testing.T) { // Проверяем создание нового ограничителя запросов
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

func TestAllow_Basic(t *testing.T) { // Проверить базовую работоспособность ограничителя
	l := New(Config{
		Limit:  5,
		Window: time.Minute,
	})

	// Первые 5 запросов должны получить доступ
	for i := 0; i < 5; i++ {
		result := l.Allow("user:1")
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
		if result.Remaining != int64(4-i) {
			t.Errorf("request %d: expected remaining %d, got %d", i+1, 4-i, result.Remaining)
		}
	}

	// 6-й запрос должен быть отклонен
	result := l.Allow("user:1")
	if result.Allowed {
		t.Error("6th request should be denied")
	}
	if result.Remaining != 0 {
		t.Errorf("expected remaining 0, got %d", result.Remaining)
	}
}

func TestAllow_DifferentKeys(t *testing.T) { // Проверяем, что для разных ключей создаются разные окна
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

func TestAllow_WindowExpiry(t *testing.T) { // Проверяем скольжение окна
	l := New(Config{
		Limit:  2,
		Window: 100 * time.Millisecond,
	})

	// Используем оба слота
	l.Allow("user:1")
	l.Allow("user:1")

	// Должен быть отклонен
	result := l.Allow("user:1")
	if result.Allowed {
		t.Error("should be denied when limit reached")
	}

	// Ожидаем устарения окна
	time.Sleep(150 * time.Millisecond)

	// Теперь запрос должен получить доступ
	result = l.Allow("user:1")
	if !result.Allowed {
		t.Error("should be allowed after window expires")
	}
}

func TestAllowN(t *testing.T) { // Проверяем корректность работы для n запросов сразу
	l := New(Config{
		Limit:  10,
		Window: time.Minute,
	})

	// Запрос требует 5 токенов сразу
	result := l.AllowN("user:1", 5)
	if !result.Allowed {
		t.Error("AllowN(5) should be allowed")
	}
	if result.Remaining != 5 {
		t.Errorf("expected 5 remaining, got %d", result.Remaining)
	}

	// Запрашиваем еще 6
	result = l.AllowN("user:1", 6)
	if result.Allowed {
		t.Error("AllowN(6) should be denied")
	}

	// Запрашиваем еще 5
	result = l.AllowN("user:1", 5)
	if !result.Allowed {
		t.Error("AllowN(5) should be allowed")
	}
}

func TestAllow_RetryAfter(t *testing.T) { // Проверяем корректность времени повтора
	l := New(Config{
		Limit:  1,
		Window: time.Second,
	})

	// Используем токен
	l.Allow("user:1")

	// Проверяем информацию о повторе
	result := l.Allow("user:1")
	if result.Allowed {
		t.Error("should be denied")
	}

	// RetryAfter должен быть примерно 1 сек.
	if result.RetryAfter < 500*time.Millisecond || result.RetryAfter > 1100*time.Millisecond {
		t.Errorf("RetryAfter should be ~1s, got %v", result.RetryAfter)
	}
}

func TestClean(t *testing.T) { // Проверка очистки неиспользуемых окон
	l := New(Config{
		Limit:  10,
		Window: time.Second,
	})

	l.Allow("user:1")
	l.Allow("user:2")
	l.Allow("user:3")

	if l.Len() != 3 {
		t.Errorf("expected 3 windows, got %d", l.Len())
	}

	// Ждем и очищаем
	time.Sleep(50 * time.Millisecond)
	cleaned := l.Clean(10 * time.Millisecond)

	if cleaned != 3 {
		t.Errorf("expected to clean 3, cleaned %d", cleaned)
	}

	if l.Len() != 0 {
		t.Errorf("expected 0 windows, got %d", l.Len())
	}
}

func TestAllow_Concurrent(t *testing.T) { // Проверяем параллельную обработку
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
		Limit:  int64(b.N) * 100,
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
