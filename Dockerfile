FROM golang:1.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build the application (adjust path if needed)
RUN CGO_ENABLED=0 GOOS=linux go build -o /frappuccino ./cmd

# Final stage
FROM alpine:latest
WORKDIR /

# Copy the binary from builder
COPY --from=builder /frappuccino /frappuccino

EXPOSE 8080
ENTRYPOINT ["/frappuccino"]