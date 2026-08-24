# Build stage
FROM golang:1.27-alpine AS builder

# Install git (required by some dependencies for version resolution)
RUN apk add --no-cache git

WORKDIR /app

# 国内加速：使用七牛云 Go 模块代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=off

# Copy module files
COPY go.mod go.sum ./

# Copy source and build
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
COPY migrations/ ./migrations/
COPY conf/ ./conf/
COPY configs/ ./configs/
COPY docs/openapi/ ./docs/openapi/
RUN mkdir -p logs && chown -R jimu:jimu /app

USER jimu

EXPOSE 8080 9090

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:9090/livez || exit 1

CMD ["./server"]
