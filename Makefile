.PHONY: all build test test-race test-cover lint bench clean help

# Go параметры
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Покрытие
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Цвета оформления
GREEN=\033[0;32m
NC=\033[0m # Нет цвета

all: lint test build

## build: Сборка проекта
build:
	@echo "$(GREEN)Building...$(NC)"
	$(GOBUILD) -v ./...

## test: Запуск тестов
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) -v ./...

## test-race: Запуск тестов с детектором race condition
test-race:
	@echo "$(GREEN)Running tests with race detector...$(NC)"
	$(GOTEST) -race -v ./...

## test-cover: Запуск тестов с покрытием
test-cover:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	$(GOCMD) tool cover -func=$(COVERAGE_FILE) | tail -n 1

## lint: Запуск линтера
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## bench: Запуск бенчмарков
bench:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem ./... | tee benchmark/results/latest.txt

## bench-memory: Бенчмарк для хранения только в памяти
bench-memory:
	@echo "$(GREEN)Running memory benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem ./store/memory/... ./limiter/...

## bench-redis: Бенчмарк для хранения только в Redis
bench-redis:
	@echo "$(GREEN)Running redis benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem ./store/redis/...

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

## docker-up: Запуск сервисов docker-compose
docker-up:
	docker-compose up -d

## docker-down: Остановка сервисов docker-compose
docker-down:
	docker-compose down

## help: Вывод доступных команд
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

# Default target
.DEFAULT_GOAL := help