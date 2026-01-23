// Package metrics предоставляет Prometheus метрики для ограничителей запросов
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics хранит все метрики
type Metrics struct {
	RequestsTotal *prometheus.CounterVec // RequestsTotal общее кол-во проверок доступа
	RequestsAllowed *prometheus.CounterVec // RequestsAllowed кол-во допущенных запросов
	RequestsLimited *prometheus.CounterVec // RequestsLimited кол-во заблокированных запросов
	CheckDuration *prometheus.HistogramVec // CheckDuration длительность проверки доступа запроса
	KeysActive *prometheus.GaugeVec // KeysActive текущее кол-во активных ключей
}

// Config содержит конфиг метрик.
type Config struct {
	// Namespace пространство имен метрик (префикс)
	// Default: "ratelimiter"
	Namespace string

	// Subsystem подсистема метрик
	// Default: ""
	Subsystem string

	// ConstLabels метки, добавляемые метрикам
	ConstLabels prometheus.Labels
}

// DefaultConfig возвращает конфигурацию метрик по умолчанию
func DefaultConfig() Config {
	return Config{
		Namespace: "ratelimiter",
	}
}

// New создает сущность метрик с дефолтным регистром Prometheus
func New(cfg Config) *Metrics {
	return NewWithRegistry(cfg, prometheus.DefaultRegisterer)
}

// NewWithRegistry создает сущность метрик с кастомным регистром
func NewWithRegistry(cfg Config, reg prometheus.Registerer) *Metrics {
	if cfg.Namespace == "" {
		cfg.Namespace = "ratelimiter"
	}

	factory := promauto.With(reg)

	return &Metrics{
		RequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   cfg.Namespace,
				Subsystem:   cfg.Subsystem,
				Name:        "requests_total",
				Help:        "Total number of rate limit checks.",
				ConstLabels: cfg.ConstLabels,
			},
			[]string{"algorithm", "result"},
		),

		RequestsAllowed: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   cfg.Namespace,
				Subsystem:   cfg.Subsystem,
				Name:        "requests_allowed_total",
				Help:        "Total number of allowed requests.",
				ConstLabels: cfg.ConstLabels,
			},
			[]string{"algorithm"},
		),

		RequestsLimited: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   cfg.Namespace,
				Subsystem:   cfg.Subsystem,
				Name:        "requests_limited_total",
				Help:        "Total number of rate-limited requests.",
				ConstLabels: cfg.ConstLabels,
			},
			[]string{"algorithm"},
		),

		CheckDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   cfg.Namespace,
				Subsystem:   cfg.Subsystem,
				Name:        "check_duration_seconds",
				Help:        "Duration of rate limit checks in seconds.",
				ConstLabels: cfg.ConstLabels,
				// Бакеты оптимизированы для задержек на уровне микросекунд
				Buckets: []float64{
					0.00001, // 10µs
					0.00005, // 50µs
					0.0001,  // 100µs
					0.0005,  // 500µs
					0.001,   // 1ms
					0.005,   // 5ms
					0.01,    // 10ms
					0.05,    // 50ms
					0.1,     // 100ms
				},
			},
			[]string{"algorithm"},
		),

		KeysActive: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   cfg.Namespace,
				Subsystem:   cfg.Subsystem,
				Name:        "keys_active",
				Help:        "Current number of active rate limit keys.",
				ConstLabels: cfg.ConstLabels,
			},
			[]string{"algorithm"},
		),
	}
}

// ObserveAllow записывает допущенные запросы
func (m *Metrics) ObserveAllow(algorithm string, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(algorithm, "allowed").Inc()
	m.RequestsAllowed.WithLabelValues(algorithm).Inc()
	m.CheckDuration.WithLabelValues(algorithm).Observe(duration.Seconds())
}

// ObserveDeny записывает отклоненные запросы
func (m *Metrics) ObserveDeny(algorithm string, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(algorithm, "denied").Inc()
	m.RequestsLimited.WithLabelValues(algorithm).Inc()
	m.CheckDuration.WithLabelValues(algorithm).Observe(duration.Seconds())
}

// SetActiveKeys устанавливает кол-во активных ключей
func (m *Metrics) SetActiveKeys(algorithm string, count int) {
	m.KeysActive.WithLabelValues(algorithm).Set(float64(count))
}
