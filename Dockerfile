FROM golang:1.21-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git ca-certificates tzdata build-base

WORKDIR /app

# Копируем go.mod и go.sum для кеширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-extldflags "-static"' \
    -o main cmd/main-app/main.go

# Финальный образ
FROM alpine:latest

# Устанавливаем необходимые пакеты (включая wget для healthcheck)
RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /root/

# Копируем собранное приложение
COPY --from=builder /app/main .

# Копируем статические файлы (только если папка существует)
COPY --from=builder /app/static ./static/ 

# Открываем порт
EXPOSE 8080

# Healthcheck
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Запуск приложения
CMD ["./main"]
