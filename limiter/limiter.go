package limiter

import "time"

type Result struct {
	Allowed bool // Индикатор доступа для запроса
	Remaining int64 // Оставшееся количество запросов в текущем окне
	Limit int64 // Максимальное число допустимых запросов
	RetryAfter time.Duration // Время, которое должен ждать клиент до повторной попытки
	ResetAt time.Time // Время сброса ограничителя
}


type Limiter interface {
	Allow(key string) Result // Проверка доступа для запроса по ключу
	AllowN(key string, n int64) Result // Проверка доступа для N запросов по ключу
}
