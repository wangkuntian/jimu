# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o jimu cmd/cli/main.go

# Runtime stage
FROM alpine:3.24.1

RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -S jimu && adduser -S jimu -G jimu

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/jimu .
COPY configs/ ./configs/
COPY migrations/ ./migrations/
COPY conf/ ./conf/
RUN mkdir -p logs && chown -R jimu:jimu /app

USER jimu

EXPOSE 8080 9090

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:9090/livez || exit 1

CMD ["./server"]
