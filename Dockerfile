# Используем официальный образ Golang
FROM golang:1.22 AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем go.mod и go.sum и устанавливаем зависимости (это ускоряет сборку)
COPY go.mod go.sum ./
RUN go mod download

# Копируем все исходники в контейнер
COPY . .

# Собираем бинарный файл
RUN go build -o frappuccino ./cmd/main.go

# Используем минимальный образ для финального контейнера
FROM debian:bookworm-slim

WORKDIR /app

# Копируем бинарник из builder-контейнера
COPY --from=builder /app/frappuccino /app/frappuccino

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["/app/frappuccino"]
