.PHONY: run build test vet fmt fmt-check lint clean migrate swagger cli docker docker-up docker-down help

# 默认目标
.DEFAULT_GOAL := help

# 变量
BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/server
CLI_BIN := $(BIN_DIR)/jimu
SERVER_CMD := cmd/server/main.go
CLI_CMD := cmd/cli/main.go
DOCKER_COMPOSE := docker-compose
ENV ?= dev

## help: 显示帮助信息
help:
	@echo "Jimu Backend Framework"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@awk '/^## /{c=substr($$0,4);getline;gsub(/^[a-z-]+: */,c" ");print "  "$$0}' $(MAKEFILE_LIST) | sort

## run: 运行 HTTP 服务
run:
	JIMU_ENV=$(ENV) go run $(SERVER_CMD)

## run-cli: 运行 CLI 工具
run-cli:
	JIMU_ENV=$(ENV) go run $(CLI_CMD)

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

## migrate: 运行数据库迁移
migrate:
	go run $(CLI_CMD) migrate up

## migrate-down: 回滚最后一次迁移
migrate-down:
	go run $(CLI_CMD) migrate down

## migrate-status: 查看迁移状态
migrate-status:
	go run $(CLI_CMD) migrate status

## seed: 插入初始数据
seed:
	go run $(CLI_CMD) seed

## swagger: 生成 API 文档
swagger:
	swag init -g $(SERVER_CMD) -o docs/openapi

## cli: 编译 CLI（build-cli 别名）
cli: build-cli

## docker: 构建 Docker 镜像
docker:
	docker build -t jimu:latest .

## docker-up: 启动所有容器（依赖 + 应用）
docker-up:
	$(DOCKER_COMPOSE) up -d

## docker-down: 停止所有容器
docker-down:
	$(DOCKER_COMPOSE) down

## docker-logs: 查看应用日志
docker-logs:
	$(DOCKER_COMPOSE) logs -f server

## all: 格式化 -> 静态检查 -> 测试 -> 编译
all: fmt vet test build

## release-check: 发布前检查（fmt-check + vet + test）
release-check: fmt-check vet test
	@echo "All checks passed"
