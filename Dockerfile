FROM golang:1.23

WORKDIR /app

COPY . .

RUN go build -o frappuccino ./cmd/main.go

EXPOSE 8080

CMD ["./main"]