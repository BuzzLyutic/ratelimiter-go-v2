FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем файлы go mod
COPY go.mod go.sum ./
RUN go mod download

# Копируем источник
COPY . .

# Сборка
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./examples/prometheus/main.go

# Runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
