.PHONY: run build test vet fmt fmt-check lint clean migrate swagger cli docker docker-run docker-stop docker-logs docker-up docker-down docker-restart docker-compose-logs help

# 默认目标
.DEFAULT_GOAL := help

# 变量
BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/server
CLI_BIN := $(BIN_DIR)/jimu
SERVER_CMD := cmd/server/main.go
CLI_CMD := cmd/cli/main.go
DOCKER_COMPOSE ?= docker compose
DOCKER_IMAGE := jimu:latest
DOCKER_CONTAINER := jimu-server
SWAG := go run -mod=mod github.com/swaggo/swag/cmd/swag
ENV ?= dev

# 加载 .env 文件（如果存在）
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
	@echo "  make run              编译并运行二进制"
	@echo "  make run-go           直接 go run 运行（不编译）"
	@echo "  make build            编译二进制到 bin/"
	@echo ""
	@echo "Docker 容器（单容器，需外部 DB + Redis）:"
	@echo "  make docker           构建镜像"
	@echo "  make docker-run       运行容器（前台）"
	@echo "  make docker-stop      停止并删除容器"
	@echo "  make docker-logs      查看容器日志"
	@echo ""
	@echo "Docker Compose（一键启动全部服务）:"
	@echo "  make docker-up        启动所有服务"
	@echo "  make docker-down      停止所有服务"
	@echo "  make docker-restart   重启所有服务"
	@echo "  make docker-compose-logs  查看应用日志"
	@echo ""
	@echo "数据库迁移:"
	@echo "  make migrate          本地执行迁移"
	@echo "  make migrate-docker   容器内执行迁移"
	@echo "  make migrate-compose  Compose 环境执行迁移"
	@echo "  make seed             本地插入初始数据"
	@echo "  make seed-compose     Compose 环境插入初始数据"

# ========== 本地运行 ==========

## run: 编译并运行二进制
run: build-server
	./$(SERVER_BIN)

## run-go: 直接 go run 运行（不编译）
run-go:
	JIMU_ENV=$(ENV) go run $(SERVER_CMD)

## build: 编译服务端和 CLI
build: build-server build-cli

## build-server: 编译 HTTP 服务
build-server:
	@mkdir -p $(BIN_DIR)
	go build -o $(SERVER_BIN) $(SERVER_CMD)

## build-cli: 编译 CLI 工具
build-cli:
	@mkdir -p $(BIN_DIR)
	go build -o $(CLI_BIN) $(CLI_CMD)

# ========== Docker 镜像 ==========

## docker: 构建 Docker 镜像
docker:
	docker build -t $(DOCKER_IMAGE) .

## build-docker: 构建镜像（docker 别名）
build-docker: docker

## push-docker: 推送镜像到仓库（需先 docker tag）
push-docker:
	docker push $(DOCKER_IMAGE)

# ========== Docker 单容器 ==========

## container-run: 运行容器（前台，需外部 DB + Redis）
container-run:
	docker run --rm -it \
		--name $(DOCKER_CONTAINER) \
		-p 8080:8080 \
		-e JIMU_ENV=prod \
		-e JIMU__DB__HOST=host.docker.internal \
		-e JIMU__REDIS__ADDR=host.docker.internal:6379 \
		$(DOCKER_IMAGE)

## container-stop: 停止并删除容器
container-stop:
	docker stop $(DOCKER_CONTAINER) 2>/dev/null || true
	docker rm $(DOCKER_CONTAINER) 2>/dev/null || true

## container-logs: 查看容器日志
container-logs:
	docker logs -f $(DOCKER_CONTAINER)

## docker-run: 运行单容器（container-run 别名）
docker-run: container-run

## docker-stop: 停止单容器（container-stop 别名）
docker-stop: container-stop

## docker-logs: 查看单容器日志（container-logs 别名）
docker-logs: container-logs

# ========== Docker Compose ==========

## compose-up: 启动所有服务（依赖 + 应用）
compose-up:
	$(DOCKER_COMPOSE) up -d

## compose-down: 停止所有服务
compose-down:
	$(DOCKER_COMPOSE) down

## compose-restart: 重启所有服务
compose-restart:
	$(DOCKER_COMPOSE) restart

## compose-logs: 查看应用日志
compose-logs:
	$(DOCKER_COMPOSE) logs -f server

## docker-up: 启动 Compose 环境（compose-up 别名）
docker-up: compose-up

## docker-down: 停止 Compose 环境（compose-down 别名）
docker-down: compose-down

## docker-restart: 重启 Compose 环境（compose-restart 别名）
docker-restart: compose-restart

## docker-compose-logs: 查看 Compose 应用日志（compose-logs 别名）
docker-compose-logs: compose-logs

# ========== 数据库迁移 ==========

## migrate: 本地执行迁移
migrate:
	JIMU_ENV=$(ENV) go run $(CLI_CMD) migrate up

## migrate-down: 本地回滚最后一次迁移
migrate-down:
	JIMU_ENV=$(ENV) go run $(CLI_CMD) migrate down

## migrate-status: 本地查看迁移状态
migrate-status:
	JIMU_ENV=$(ENV) go run $(CLI_CMD) migrate status

## migrate-docker: 容器内执行迁移
migrate-docker:
	docker run --rm \
		--network host \
		-e JIMU_ENV=prod \
		-e JIMU__DB__HOST=host.docker.internal \
		-e JIMU__REDIS__ADDR=host.docker.internal:6379 \
		$(DOCKER_IMAGE) ./jimu migrate up

## migrate-compose: Compose 环境执行迁移
migrate-compose:
	$(DOCKER_COMPOSE) run --rm server ./jimu migrate up

## migrate-compose-down: Compose 环境回滚迁移
migrate-compose-down:
	$(DOCKER_COMPOSE) run --rm server ./jimu migrate down

## migrate-compose-status: Compose 环境查看迁移状态
migrate-compose-status:
	$(DOCKER_COMPOSE) run --rm server ./jimu migrate status

# ========== 数据初始化 ==========

## seed: 本地插入初始数据
seed:
	JIMU_ENV=$(ENV) go run $(CLI_CMD) seed

## seed-docker: 容器内插入初始数据
seed-docker:
	docker run --rm \
		--network host \
		-e JIMU_ENV=prod \
		-e JIMU__DB__HOST=host.docker.internal \
		-e JIMU__REDIS__ADDR=host.docker.internal:6379 \
		$(DOCKER_IMAGE) ./jimu seed

## seed-compose: Compose 环境插入初始数据
seed-compose:
	$(DOCKER_COMPOSE) run --rm server ./jimu seed

# ========== 工具 ==========

## test: 运行所有测试
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

## cli: 编译 CLI（build-cli 别名）
cli: build-cli

## all: 格式化 -> 静态检查 -> 测试 -> 编译
all: fmt vet test build

## release-check: 发布前检查（fmt-check + vet + test）
release-check: fmt-check vet test
	@echo "All checks passed"
