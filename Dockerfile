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
# 形态（profile）：full（默认）/minimal/saas/enterprise/machine；见 Makefile 与 docs/plans 的 P2.5b。
ARG PROFILE=full
# 捕获工具打印的 overlay JSON 路径（busybox ash 支持 $(...)）；赋值语句的退出码就是命令替换的
# 退出码，形态名非法时构建失败 —— 若直接写进 go build 的 -overlay=，失败只会留下空的 overlay，
# 镜像会**静默按 full 构建**。产物按形态隔离在 .overlay/<profile>/。
RUN OVERLAY="$(go run ./tools/profileoverlay "${PROFILE}")" && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -overlay="$OVERLAY" -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o jimu cmd/cli/main.go

# Runtime stage
FROM alpine:3.24.1

# 升级基础镜像包到最新（修复基线 CVE，如 OpenSSL），再安装所需包
RUN apk --no-cache upgrade && \
    apk --no-cache add ca-certificates tzdata curl && \
    addgroup -S jimu && adduser -S jimu -G jimu

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/jimu .
COPY conf/ ./conf/
COPY configs/ ./configs/
COPY docs/openapi/ ./docs/openapi/
RUN mkdir -p logs && chown -R jimu:jimu /app

USER jimu

EXPOSE 8080 9090

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:9090/livez || exit 1

CMD ["./server"]
