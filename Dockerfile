# Build stage
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Install swag CLI
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate Swagger docs in cmd/api/docs
RUN swag init -g ./cmd/api/main.go -o ./cmd/api/docs


# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o main ./cmd/api

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/main .

# Copy Swagger docs
COPY --from=builder /app/cmd/api/docs ./cmd/api/docs

# Copy .env.example as default
COPY --from=builder /app/.env .env

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:3000/health || exit 1

CMD ["./main"]
