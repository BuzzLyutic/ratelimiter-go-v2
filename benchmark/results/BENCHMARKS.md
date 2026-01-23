# Результаты бенчмарков

## Среда выполнения
- **CPU**: AMD Ryzen 5 5500U
- **OS**: Windows 11 / Ubuntu 22.04
- **Go**: 1.24
- **Date**: 21.01.2026

## In-Memory алгоритмы

| Алгоритм        | Один ключ | Параллельно   | Много ключей | Память  | Пропускная способность     |
|------------------|------------|------------|---------------|---------|----------------|
| Token Bucket     | 138.9 ns   | -          | 164.9 ns      | 0 B/op  | ~7.2M ops/sec  |
| Sliding Log      | 59.9 ns    | 732.7 ns   | 244.3 ns      | 24 B/op | ~16.7M ops/sec |
| Sliding Counter  | 83.3 ns    | 196.1 ns   | 213.7 ns      | 0 B/op  | ~12M ops/sec   |

## Redis Store

| Operation | Latency | Memory | Throughput |
|-----------|---------|--------|------------|
| Token Bucket | 234 µs | 615 B/op | ~4.3K ops/sec |
| Sliding Window | 230 µs | 604 B/op | ~4.3K ops/sec |

## HTTP Middleware

| Framework | Latency | Memory | Allocations |
|-----------|---------|--------|-------------|
| Gin | 3488 ns | 6775 B/op | 35 allocs |
| Chi | 3077 ns | 6674 B/op | 30 allocs |

## gRPC Interceptor

| Type | Latency | Memory | Allocations |
|------|---------|--------|-------------|
| Unary | 731.9 ns | 640 B/op | 10 allocs |

## Prometheus Metrics Overhead

| Scenario | Latency | Overhead |
|----------|---------|----------|
| Without metrics | 165.7 ns | - |
| With metrics | 459.7 ns | +294 ns (~2x) |

## Анализ

### Token Bucket
- Наиболее стабильная производительность во всех сценариях
- Нулевое распределение ресурсов
- Лучший выбор для ограничения скорости общего назначения

### Sliding Log  
- Самый быстрый для однопоточного доступа
- Производительность снижается в условиях конкурентности (мьютекс + рост слайса)
- Объем памяти растет при увеличении числа запросов O(n)
- Лучше всего подходит для точного подсчета при низком трафике

### Sliding Counter
- Наилучшая производительность при параллельных запросах (5.1M ops/sec в условиях конкурентности)
- Постоянная память O(1)
- Незначительный компромисс между точностью и производительностью
- Лучшее решение для систем с высокой пропускной способностью

## Ключевые моменты

1. **Token Bucket** — лучший баланс скорости и памяти для общего использования
2. **Sliding Counter** — лучшая производительность при параллельных запросах
3. **Sliding Log** — самый быстрый для single-key, но память O(n)
4. **Redis** добавляет ~230µs latency (сеть + Lua script)
5. **Prometheus** добавляет ~300ns overhead — приемлемо для production

## Рекомендации

| Use Case                    | Рекомендуемый алгоритм |
|-----------------------------|----------------------|
| API Gateway                 | Token Bucket         |
| User quotas                 | Sliding Counter      |
| Precise billing/audit       | Sliding Log          |
| High-concurrency service    | Sliding Counter      |


| Use Case | Рекомендация | Объяснение |
|----------|-------------|-----|
| Single server API | Token Bucket (memory) | Быстрый, простой, допускающий всплески |
| Microservices cluster | Redis + Sliding Counter | Распределенный, согласованный |
| High-precision billing | Sliding Log | Точный подсчет |
| Maximum throughput | Sliding Counter (memory) | Наилучшая параллельная производительность |