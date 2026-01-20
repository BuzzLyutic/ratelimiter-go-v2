package tokenbucket

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) { // Тестируем создание ограничителя запросов
	cfg := Config{
		Rate:     100,
		Interval: time.Minute,
		Burst:    10,
	}

	l := New(cfg)

	if l.capacity != 10 {
		t.Errorf("expected capacity 10, got %d", l.capacity)
	}

	expectedRate := 100.0 / 60.0
	if l.rate != expectedRate {
		t.Errorf("expected rate %f, got %f", expectedRate, l.rate)
	}
}

func TestNew_DefaultBurst(t *testing.T) { // Тестируем установку Burst по умолчанию
	cfg := Config{
		Rate:     100,
		Interval: time.Minute,
		// Burst не установлен
	}

	l := New(cfg)

	if l.capacity != 100 {
		t.Errorf("expected capacity 100 (default to Rate), got %d", l.capacity)
	}
}

func TestAllow_Basic(t *testing.T) { // Тестируем базовую работоспособность ограничителя
	l := New(Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    10,
	})

	// Первые 10 запросов должны получить доступ (bucket создается полным)
	for i := 0; i < 10; i++ {
		result := l.Allow("user:1")
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 11-й запрос должен быть отклонен
	result := l.Allow("user:1")
	if result.Allowed {
		t.Error("11th request should be denied")
	}
}

func TestAllow_DifferentKeys(t *testing.T) { // Тестируем создание разных бакетов
	l := New(Config{
		Rate:     1,
		Interval: time.Second,
		Burst:    1,
	})

	// Разным ключам соответствуют разные бакеты
	result1 := l.Allow("user:1")
	result2 := l.Allow("user:2")

	if !result1.Allowed || !result2.Allowed {
		t.Error("different keys should have separate limits")
	}
}

func TestAllow_Refill(t *testing.T) { // Тестируем механику пополнения токенов
	l := New(Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    1,
	})

	// Используем единственный токен
	result := l.Allow("user:1")
	if !result.Allowed {
		t.Error("first request should be allowed")
	}

	// Должен быть отклонен теперь
	result = l.Allow("user:1")
	if result.Allowed {
		t.Error("second request should be denied")
	}

	// Ждем пополнения (10 токенов/сек = 0.1 секунда на токен)
	time.Sleep(150 * time.Millisecond)

	// Должен быть доступен после пополнения
	result = l.Allow("user:1")
	if !result.Allowed {
		t.Error("request after refill should be allowed")
	}
}

func TestAllowN(t *testing.T) { // Проверяем доступ для множества токенов сразу
	l := New(Config{
		Rate:     100,
		Interval: time.Second,
		Burst:    10,
	})

	// Запрос 5-ти токенов за раз
	result := l.AllowN("user:1", 5)
	if !result.Allowed {
		t.Error("AllowN(5) should be allowed")
	}
	if result.Remaining != 5 {
		t.Errorf("expected 5 remaining, got %d", result.Remaining)
	}

	// Запрос еще 6-ти токенов (должен быть отклонен, т.к. осталось только 5 токенов)
	result = l.AllowN("user:1", 6)
	if result.Allowed {
		t.Error("AllowN(6) should be denied (only 5 remaining)")
	}

	// Опять запрос 5-ти токенов (должен пройти успешно)
	result = l.AllowN("user:1", 5)
	if !result.Allowed {
		t.Error("AllowN(5) should be allowed")
	}
}

func TestAllow_RetryAfter(t *testing.T) { // Проверяем корректность времени ожидания повтора
	l := New(Config{
		Rate:     1,
		Interval: time.Second,
		Burst:    1,
	})

	// Используем токен
	l.Allow("user:1")

	// Получаем информацию о повторе
	result := l.Allow("user:1")
	if result.Allowed {
		t.Error("should be denied")
	}

	// RetryAfter должен быть ~1 сек. (1 токен/сек)
	if result.RetryAfter < 500*time.Millisecond || result.RetryAfter > 1500*time.Millisecond {
		t.Errorf("RetryAfter should be ~1s, got %v", result.RetryAfter)
	}
}

func TestClean(t *testing.T) { // Проверяем механизм очистки неиспользуемых бакетов
	l := New(Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    10,
	})

	// Создаем несколько бакетов
	l.Allow("user:1")
	l.Allow("user:2")
	l.Allow("user:3")

	if l.Len() != 3 {
		t.Errorf("expected 3 buckets, got %d", l.Len())
	}

	// Очищаем установив короткий maxAge
	time.Sleep(10 * time.Millisecond)
	cleaned := l.Clean(5 * time.Millisecond)

	if cleaned != 3 {
		t.Errorf("expected to clean 3 buckets, cleaned %d", cleaned)
	}

	if l.Len() != 0 {
		t.Errorf("expected 0 buckets after clean, got %d", l.Len())
	}
}

func TestAllow_Concurrent(t *testing.T) { // Проверяем race condition
	l := New(Config{
		Rate:     1000,
		Interval: time.Second,
		Burst:    100,
	})

	// Запускаем параллельные запросы
	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func() {
			l.Allow("user:1")
			done <- true
		}()
	}

	// Ждем завершения всех горутин
	for i := 0; i < 100; i++ {
		<-done
	}

	// Нет паники = успех
}

// Бенчмарки

func BenchmarkAllow(b *testing.B) {
	l := New(Config{
		Rate:     1000000,
		Interval: time.Second,
		Burst:    1000000,
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
		Rate:     1000000,
		Interval: time.Second,
		Burst:    1000000,
	})

	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = "user:" + string(rune(i))
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
