.PHONY: run build test vet fmt fmt-check lint clean migrate migrate-down migrate-status seed help govulncheck test-backup-restore ci
.PHONY: test-cover test-coverage-check test-race swagger-check smoke-check compose-check
.PHONY: docker-build docker-run docker-stop docker-logs
.PHONY: compose-up compose-down compose-restart compose-logs compose-migrate compose-seed
.PHONY: compose-observability compose-observability-down
.PHONY: bench loadtest proto

# 默认目标
.DEFAULT_GOAL := help

# 变量
BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/jimu-server
CLI_BIN := $(BIN_DIR)/jimu-cli
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
	@echo "  make backup               备份数据库（mariadb-dump/mysqldump）"
	@echo "  make restore              从备份恢复数据库（需 BACKUP_FILE=...，FORCE=1 跳过确认）"
	@echo "  make test-backup-restore  备份/恢复往返测试（需运行中 mariadb 容器）"
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
	@echo "  make compose-observability      启动监控栈（OpenObserve：日志/指标/追踪/告警）"
	@echo "  make compose-observability-down 停止监控栈"
	@echo ""
	@echo "工具:"
	@echo "  make clean                清理构建产物"
	@echo "  make swagger              生成 API 文档"
	@echo "  make proto                重新生成 gRPC 代码"
	@echo "  make bench                运行性能基准测试"
	@echo "  make loadtest             本地 HTTP 压测（需 hey）"
	@echo "  make ci                   本地 CI 检查（无外部依赖：fmt/vet/lint/test/coverage/race/swagger/smoke/build/govulncheck）"
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

## backup: 备份数据库（需 mysqldump，输出到 ./backups，环境变量见 scripts/backup.sh）
backup:
	@./scripts/backup.sh

## restore: 从备份恢复数据库（需 mysql，用法: make restore BACKUP_FILE=./backups/xxx.sql.gz）
restore:
	@./scripts/restore.sh $(BACKUP_FILE)

## test-backup-restore: 备份/恢复往返测试（通过 docker exec 调用容器内 mariadb，用法: make test-backup-restore [CONTAINER=jimu-test-mysql]）
test-backup-restore:
	@./scripts/test_backup_restore.sh $(CONTAINER)

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

## compose-observability: 启动监控栈（OpenObserve：日志/指标/追踪/告警/仪表盘）
compose-observability:
	$(DOCKER_COMPOSE) --profile observability up -d openobserve
	@echo "OpenObserve started:"
	@echo "  UI/API:  http://localhost:5080 (admin@jimu.local / admin)"
	@echo "  OTLP:    localhost:5081"
	@echo "启用应用推送：OTEL_ENABLED=true make compose-up（或对 server 容器设 OTEL_ENABLED=true）"

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

## proto: 从 proto/ 重新生成 gRPC 代码（需 protoc + protoc-gen-go + protoc-gen-go-grpc）
proto:
	@command -v protoc >/dev/null 2>&1 || { echo "protoc 未安装: brew install protobuf"; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go 未安装: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "protoc-gen-go-grpc 未安装: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }
	protoc --go_out=. --go_opt=module=jimu \
	       --go-grpc_out=. --go-grpc_opt=module=jimu \
	       proto/jimu/v1/*.proto
	@echo "gRPC 代码已重新生成"

## cli: 编译 CLI
cli: build-cli

## all: 格式化 -> 静态检查 -> 测试 -> 编译
all: fmt vet test build

## bench: 运行性能基准测试
bench:
	go test -bench=. -benchmem -run='^$$' ./internal/shared/id/... ./internal/modules/auth/application/... ./internal/platform/notification/... ./internal/platform/http/... ./internal/platform/queue/...

## bench-ci: 性能回归门禁（绝对阈值模式，CI 用）
bench-ci:
	@./scripts/bench_ci.sh --absolute

## loadtest: 本地 HTTP 压测（需 hey，默认打健康检查）
loadtest:
	@./scripts/loadtest.sh

## govulncheck: 依赖漏洞扫描（go run 免安装，与 CI 命令一致）
govulncheck:
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## test-cover: 运行测试并生成覆盖率（与 CI Test job 一致）
test-cover:
	go test ./... -coverprofile=coverage.out

## test-coverage-check: 校验覆盖率阈值（默认 70%，与 CI Test job 一致）
test-coverage-check:
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	THRESHOLD=70; \
	echo "📈 Total coverage: $${COVERAGE}%"; \
	if [ $$(awk "BEGIN{print ($${COVERAGE}<$${THRESHOLD})}") -eq 1 ]; then \
		echo "❌ Coverage $${COVERAGE}% is below threshold $${THRESHOLD}%"; \
		go tool cover -func=coverage.out; exit 1; \
	fi; \
	echo "✅ Coverage check passed ($${COVERAGE}% >= $${THRESHOLD}%)"

## test-race: 竞争检测测试（与 CI Test job 一致）
test-race:
	go test -race ./...

## swagger-check: 校验 OpenAPI 文档为最新（与 CI Test job 一致）
swagger-check:
	$(SWAG) init -g $(SERVER_CMD) -o docs/openapi >/dev/null
	@git diff --exit-code docs/openapi || { \
		echo "❌ docs/openapi 不是最新，请运行 make swagger"; exit 1; \
	}
	@echo "✅ OpenAPI 文档为最新"

## smoke-check: 校验 smoke 脚本语法（与 CI Test job 一致）
smoke-check:
	@bash -n scripts/test_runtime_security.sh
	@bash -n scripts/smoke_api_contract.sh
	@bash -n scripts/test_backup_restore.sh
	@echo "✅ Smoke 脚本语法正确"

## compose-check: 隔离 Compose 运行时与 API 契约验证（不会读取或修改本地 .env、secrets、数据卷）
compose-check:
	@./scripts/test_runtime_security.sh
	@./scripts/smoke_api_contract.sh

## ci: 本地 CI 检查（无外部依赖部分，完整 CI 见 .github/workflows/ci.yml）
ci: fmt-check vet lint test-cover test-coverage-check test-race swagger-check smoke-check build govulncheck
	@echo "✅ All local CI checks passed"

## release-check: 发布前检查（Go 门禁 + govulncheck + 隔离 Compose/API smoke）
release-check: fmt-check vet test govulncheck compose-check
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
