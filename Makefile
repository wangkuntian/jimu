.PHONY: run build test vet fmt fmt-check lint clean migrate migrate-down migrate-status seed help
.PHONY: docker-build docker-run docker-stop docker-logs
.PHONY: compose-up compose-down compose-restart compose-logs compose-migrate compose-seed
.PHONY: compose-observability compose-observability-down

# 默认目标
.DEFAULT_GOAL := help

# 变量
BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/server
CLI_BIN := $(BIN_DIR)/jimu
SERVER_CMD := cmd/server/main.go
CLI_CMD := cmd/cli/main.go
VERSION ?= dev
# 注入版本号到两个 main 包
LDFLAGS := -X main.version=$(VERSION)
DOCKER_COMPOSE ?= docker compose
DOCKER_IMAGE = jimu:latest
DOCKER_CONTAINER := jimu-server
SWAG := go run -mod=mod github.com/swaggo/swag/cmd/swag
ENV ?= dev

# 根据 APP_ENV 自动生成 --profile 参数：dev 环境启动 adminer
COMPOSE_PROFILE_FLAG = $(if $(filter dev,$(APP_ENV)),--profile dev)

# 加载 .env 文件（如果存在），导出敏感变量供 docker 命令使用
# 本地 make run/migrate/seed 直接读取 configs/ 中的 YAML，无需环境变量
ifneq (,$(wildcard .env))
    include .env
    export
endif

## help: 显示帮助信息
help:
	@echo "Jimu Backend Framework"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "本地运行:"
	@echo "  make run                  编译并运行服务端"
	@echo "  make build                编译服务端和 CLI"
	@echo "  make test                 运行测试"
	@echo "  make vet                  静态分析"
	@echo "  make fmt                  格式化代码"
	@echo "  make lint                 静态检查"
	@echo ""
	@echo "数据库:"
	@echo "  make migrate              本地执行迁移"
	@echo "  make migrate-down         本地回滚迁移"
	@echo "  make migrate-status       查看迁移状态"
	@echo "  make seed                 本地插入初始数据"
	@echo ""
	@echo "Docker 容器（单容器，需外部 DB + Redis）:"
	@echo "  make docker-build         构建镜像"
	@echo "  make docker-run           运行容器（前台）"
	@echo "  make docker-stop          停止并删除容器"
	@echo "  make docker-logs          查看容器日志"
	@echo ""
	@echo "Docker Compose（一键启动全部服务）:"
	@echo "  make compose-up           启动所有服务"
	@echo "  make compose-down         停止所有服务"
	@echo "  make compose-restart      重启所有服务"
	@echo "  make compose-logs         查看应用日志"
	@echo "  make compose-migrate      Compose 环境执行迁移"
	@echo "  make compose-seed         Compose 环境插入初始数据"
	@echo "  make compose-observability      启动监控栈（Prometheus + Grafana）"
	@echo "  make compose-observability-down 停止监控栈"
	@echo ""
	@echo "工具:"
	@echo "  make clean                清理构建产物"
	@echo "  make swagger              生成 API 文档"
	@echo "  make release-check        发布前检查"

# ========== 本地运行 ==========

## run: 编译并运行服务端
run: build-server
	APP_ENV=$(ENV) ./$(SERVER_BIN)

## build: 编译服务端和 CLI
build: build-server build-cli

# 内部目标（不直接调用）
build-server:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(SERVER_BIN) $(SERVER_CMD)

build-cli:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(CLI_BIN) $(CLI_CMD)

# ========== 数据库 ==========

## migrate: 本地执行迁移
migrate:
	APP_ENV=$(ENV) go run $(CLI_CMD) migrate up

## migrate-down: 本地回滚迁移
migrate-down:
	APP_ENV=$(ENV) go run $(CLI_CMD) migrate down

## migrate-status: 查看迁移状态
migrate-status:
	APP_ENV=$(ENV) go run $(CLI_CMD) migrate status

## seed: 本地插入初始数据
seed:
	APP_ENV=$(ENV) go run $(CLI_CMD) seed

# ========== Docker 单容器 ==========

## docker-build: 构建 Docker 镜像
docker-build:
	docker build -t "$(DOCKER_IMAGE)" .

## docker-run: 运行容器（前台，需外部 DB + Redis）
docker-run:
	docker run --rm -it \
		--name $(DOCKER_CONTAINER) \
		-p 8080:8080 \
		-e JWT_SECRET \
		-e DB_PASSWORD \
		-v $(PWD)/configs:/app/configs \
		"$(DOCKER_IMAGE)"

## docker-stop: 停止并删除容器
docker-stop:
	docker stop $(DOCKER_CONTAINER) 2>/dev/null || true
	docker rm $(DOCKER_CONTAINER) 2>/dev/null || true

## docker-logs: 查看容器日志
docker-logs:
	docker logs -f $(DOCKER_CONTAINER)

# ========== Docker Compose ==========
# 通过 .env 中 COMPOSE_PROFILES 控制启动的 profile（如 dev 启动 adminer）

## compose-up: 启动所有服务（依赖 + 应用）
compose-up:
	$(DOCKER_COMPOSE) $(COMPOSE_PROFILE_FLAG) up -d

## compose-down: 停止所有服务
compose-down:
	$(DOCKER_COMPOSE) $(COMPOSE_PROFILE_FLAG) down

## compose-restart: 重启所有服务
compose-restart:
	$(DOCKER_COMPOSE) $(COMPOSE_PROFILE_FLAG) restart

## compose-logs: 查看应用日志
compose-logs:
	$(DOCKER_COMPOSE) $(COMPOSE_PROFILE_FLAG) logs -f server

## compose-migrate: Compose 环境执行迁移
compose-migrate:
	$(DOCKER_COMPOSE) $(COMPOSE_PROFILE_FLAG) run --rm server ./jimu migrate up

## compose-seed: Compose 环境插入初始数据
compose-seed:
	$(DOCKER_COMPOSE) $(COMPOSE_PROFILE_FLAG) run --rm server ./jimu seed

## compose-observability: 启动监控栈（Prometheus + Grafana）
compose-observability:
	$(DOCKER_COMPOSE) --profile observability up -d

## compose-observability-down: 停止监控栈
compose-observability-down:
	$(DOCKER_COMPOSE) --profile observability down

# ========== 工具 ==========

## test: 运行测试
test:
	go test ./... -v

## test-coverage: 运行测试并生成覆盖率报告
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## vet: 静态分析
vet:
	go vet ./...

## fmt: 格式化代码
fmt:
	go fmt ./...

## fmt-check: 检查代码格式（不修改）
fmt-check:
	@echo "Checking gofmt..."
	@if gofmt -l . | grep -q .; then \
		echo "以下文件需要格式化:"; \
		gofmt -l .; \
		exit 1; \
	else \
		echo "所有文件格式正确"; \
	fi

## lint: 静态检查（需要 golangci-lint）
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint 未安装，使用 go vet 替代"; \
		go vet ./...; \
	fi

## clean: 清理构建产物
clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html

## swagger: 生成 API 文档
swagger:
	$(SWAG) init -g $(SERVER_CMD) -o docs/openapi

## cli: 编译 CLI
cli: build-cli

## all: 格式化 -> 静态检查 -> 测试 -> 编译
all: fmt vet test build

## release-check: 发布前检查（fmt-check + vet + test）
release-check: fmt-check vet test
	@echo "All checks passed"

## hooks: 安装 pre-commit 钩子（需 pip install pre-commit）
hooks:
	pre-commit install
	@echo "pre-commit hooks installed"

## dev: 热重载开发模式（需 air: go install github.com/air-verse/air@latest）
dev:
	@if command -v air >/dev/null 2>&1; then \
		APP_ENV=dev air; \
	else \
		echo "air 未安装，运行: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi
