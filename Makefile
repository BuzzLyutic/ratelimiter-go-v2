.PHONY: all build test test-race test-cover lint bench clean help
.PHONY: docker-up docker-down demo example-basic example-gin example-chi

# Go параметры
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Покрытие
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Цвета
GREEN=\033[0;32m
NC=\033[0m

all: lint test build

## build: Сборка всех примеров
build:
	@echo "$(GREEN)Building...$(NC)"
	$(GOBUILD) -v ./...

## test: Запуск тестов
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) -v ./...

## test-race: Запуск тестов с race detector
test-race:
	@echo "$(GREEN)Running tests with race detector...$(NC)"
	$(GOTEST) -race -v ./...

## test-cover: Запуск тестов с покрытием
test-cover:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	$(GOCMD) tool cover -func=$(COVERAGE_FILE) | tail -n 1
	@echo "$(GREEN)Open $(COVERAGE_HTML) in browser$(NC)"

## lint: Запуск линтера
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

## bench: Запуск всех бенчмарков
bench:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem ./... | tee benchmark/results/latest.txt

## bench-compare: Запуск бенчмарков и сравнение алгоритмов
bench-compare:
	@echo "$(GREEN)Comparing algorithms...$(NC)"
	$(GOTEST) -bench=. -benchmem ./limiter/... 

## fmt: Форматирование кода
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOFMT) -s -w .

## tidy: Tidy go.mod
tidy:
	@echo "$(GREEN)Tidying modules...$(NC)"
	$(GOMOD) tidy

## clean: Очистка файлов сборки
clean:
	@echo "$(GREEN)Cleaning...$(NC)"
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	rm -rf bin/ dist/
	$(GOCMD) clean -testcache

## docker-up: Запуск всех сервисов (Redis, Prometheus, Grafana)
docker-up:
	docker-compose up -d
	@echo ""
	@echo "$(GREEN)Services started:$(NC)"
	@echo "  API:        http://localhost:8080"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana:    http://localhost:3000 (admin/admin)"

## docker-down: Остановка всех сервисов
docker-down:
	docker-compose down

## docker-logs: Показать логи
docker-logs:
	docker-compose logs -f

## demo: Запустить интерактивную демонстрацию
demo:
	@echo "$(GREEN)Starting interactive demo...$(NC)"
	$(GOCMD) run ./examples/demo/main.go

example-basic:
	$(GOCMD) run ./examples/basic/main.go

example-gin:
	$(GOCMD) run ./examples/gin-api/main.go

example-chi:
	$(GOCMD) run ./examples/chi-api/main.go

example-prometheus:
	$(GOCMD) run ./examples/prometheus/main.go

## help: Показать доступные команды
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help
