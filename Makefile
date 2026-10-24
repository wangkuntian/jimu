.PHONY: help build build-server build-cli clean dev
.PHONY: image ensure-image init prepare-secrets prepare-observability-secrets
.PHONY: up up-observability down restart status logs adminer
.PHONY: migrate migrate-down migrate-status seed
.PHONY: backup restore build-backup-image compose-db-backup compose-db-restore test-backup-restore
.PHONY: test test-coverage test-cover test-coverage-check test-coverage-check-local test-race check-makefile
.PHONY: vet fmt fmt-check lint check-log-usage check-capabilities check-templates test-scaffold-matrix
.PHONY: profiles-check compose-report compose-report-check check-skills skills-install
.PHONY: swagger swagger-check smoke-check compose-check govulncheck check ci release-check
.PHONY: proto bench bench-ci loadtest cli all hooks

.DEFAULT_GOAL := help
.NOTPARALLEL: up up-observability

BIN_DIR := bin
# PROFILE 是编译形态；build、image、migrate、seed 与 dev 必须使用同一个值。
PROFILE ?= full
SERVER_PKG := ./cmd/server
SERVER_BIN = $(BIN_DIR)/jimu-server$(if $(filter-out full,$(PROFILE)),-$(PROFILE),)
CLI_BIN = $(BIN_DIR)/jimu-cli$(if $(filter-out full,$(PROFILE)),-$(PROFILE),)
SERVER_CMD := cmd/server/main.go
CLI_PKG := ./cmd/cli

ENV ?= dev
VERSION ?= dev
APP_VERSION ?= dev
LDFLAGS := -X main.version=$(APP_VERSION)

COMPOSE = docker compose --env-file /dev/null -f docker-compose.yml
COMPOSE_ALL = $(COMPOSE) -f compose.observability.yml
IMAGE_REPO ?= jimu
IMAGE_TAG = $(IMAGE_REPO):$(VERSION)
LATEST_TAG = $(IMAGE_REPO):latest
IMAGE_STAMP_DIR ?= .overlay/images
JIMU_SECRETS_DIR ?= $(CURDIR)/secrets
COMPOSE_ENV = APP_ENV=$(ENV) JIMU_SECRETS_DIR=$(JIMU_SECRETS_DIR) JIMU_IMAGE=$(IMAGE_TAG)

SWAG := go run -mod=mod github.com/swaggo/swag/cmd/swag
LINT_VERSION ?= v2.15.0
GOVULNCHECK_VERSION ?= v1.8.0

##@ 入门

help: ## 显示帮助
	@awk 'BEGIN { FS = ":.*## "; printf "\nJimu Backend Framework\n\nUsage:\n  make <target>\n" } /^##@/ { printf "\n%s\n", substr($$0, 5) } /^[a-zA-Z0-9_.-]+:.*## / { printf "  make %-24s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ 本地开发

build: build-server build-cli ## 编译本机 server 和 CLI（不构建镜像）

build-server: ## 编译本机服务端（PROFILE=<name>）
	@mkdir -p $(BIN_DIR)
	overlay="$$(go run ./tools/profileoverlay "$(PROFILE)")" && go build -ldflags "$(LDFLAGS)" -overlay="$$overlay" -o $(SERVER_BIN) $(SERVER_PKG)

build-cli: ## 编译本机 CLI（PROFILE=<name>）
	@mkdir -p $(BIN_DIR)
	overlay="$$(go run ./tools/profileoverlay "$(PROFILE)")" && go build -ldflags "$(LDFLAGS)" -overlay="$$overlay" -o $(CLI_BIN) $(CLI_PKG)

dev: ## 启动依赖、迁移/种子并以 air 热重载（PROFILE=<name>）
	ENV=$(ENV) PROFILE=$(PROFILE) ./scripts/dev.sh

clean: ## 清理构建产物
	rm -rf $(BIN_DIR)
	rm -rf .overlay
	rm -f coverage.out coverage.html

##@ Compose 生命周期

init: ## 初始化核心 Secret（旧 .env 仅用于显式迁移）
	@legacy_env="$(LEGACY_ENV_FILE)"; \
	if [ -z "$$legacy_env" ] && [ -f "$(CURDIR)/.env" ]; then legacy_env="$(CURDIR)/.env"; fi; \
	if [ -n "$$legacy_env" ]; then \
		LEGACY_ENV_FILE="$$legacy_env" SECRETS_DIR=$(JIMU_SECRETS_DIR) \
		ALLOW_NEW_SECURITY_KEYS="$(ALLOW_NEW_SECURITY_KEYS)" FORCE="$(FORCE)" \
		./scripts/init-secrets.sh; \
	else \
		SECRETS_DIR=$(JIMU_SECRETS_DIR) \
		ALLOW_NEW_SECURITY_KEYS="$(ALLOW_NEW_SECURITY_KEYS)" FORCE="$(FORCE)" \
		./scripts/init-secrets.sh; \
	fi

prepare-secrets:
	@SECRETS_DIR=$(JIMU_SECRETS_DIR) \
		ALLOW_NEW_SECURITY_KEYS="$(ALLOW_NEW_SECURITY_KEYS)" FORCE="$(FORCE)" \
		./scripts/init-secrets.sh --no-legacy

prepare-observability-secrets:
	@SECRETS_DIR=$(JIMU_SECRETS_DIR) \
		ALLOW_NEW_SECURITY_KEYS="$(ALLOW_NEW_SECURITY_KEYS)" FORCE="$(FORCE)" \
		./scripts/init-secrets.sh --observability --no-legacy

image: ## 无条件构建版本镜像并更新 latest
	@PROFILE="$(PROFILE)" VERSION="$(VERSION)" IMAGE_REPO="$(IMAGE_REPO)" \
		IMAGE_STAMP_DIR="$(IMAGE_STAMP_DIR)" FORCE=1 ./scripts/ensure-image.sh

ensure-image:
	@PROFILE="$(PROFILE)" VERSION="$(VERSION)" IMAGE_REPO="$(IMAGE_REPO)" \
		IMAGE_STAMP_DIR="$(IMAGE_STAMP_DIR)" FORCE=0 ./scripts/ensure-image.sh

up: prepare-secrets ensure-image ## 初始化 Secret、按需构建镜像并启动核心栈
	$(COMPOSE_ENV) $(COMPOSE) up -d --no-build

up-observability: prepare-observability-secrets ensure-image ## 启动观测栈并初始化 dashboard（配置 webhook 时同步告警）
	$(COMPOSE_ENV) $(COMPOSE_ALL) up -d --no-build
	JIMU_SECRETS_DIR=$(JIMU_SECRETS_DIR) ./scripts/observability.sh start

down: ## 停止全部 Compose 服务（保留数据卷）
	$(COMPOSE_ENV) $(COMPOSE_ALL) down

restart: ## 重启已有 Compose 容器（不重建镜像）
	$(COMPOSE_ENV) $(COMPOSE_ALL) restart

status: ## 查看 Compose 服务状态
	$(COMPOSE_ENV) $(COMPOSE_ALL) ps

logs: ## 跟踪服务端日志
	$(COMPOSE_ENV) $(COMPOSE_ALL) logs -f server

adminer: ## 显式启动 Adminer 开发工具
	$(COMPOSE_ENV) $(COMPOSE) --profile dev up -d adminer

migrate: ensure-image ## 按需构建所选 PROFILE 镜像并在当前 Compose 栈执行迁移
	$(COMPOSE_ENV) $(COMPOSE) run --rm jimu-init ./jimu migrate up

seed: ensure-image ## 按需构建所选 PROFILE 镜像并在当前 Compose 栈插入初始数据
	$(COMPOSE_ENV) $(COMPOSE) run --rm jimu-init ./jimu seed

##@ 数据库与备份

migrate-down: ensure-image ## 按需构建所选 PROFILE 镜像并在当前 Compose 栈回滚最后一次迁移
	$(COMPOSE_ENV) $(COMPOSE) run --rm jimu-init ./jimu migrate down

migrate-status: ensure-image ## 按需构建所选 PROFILE 镜像并在当前 Compose 栈查看迁移状态
	$(COMPOSE_ENV) $(COMPOSE) run --rm jimu-init ./jimu migrate status

backup: ## 主机侧备份数据库
	@./scripts/backup.sh

restore: ## 主机侧恢复数据库（BACKUP_FILE=<path>）
	@./scripts/restore.sh $(BACKUP_FILE)

build-backup-image: ## 构建备份任务镜像 jimu-backup
	docker build -f deploy/backup/Dockerfile -t jimu-backup:latest --build-arg PG_CLIENT_VERSION=$(or $(PG_CLIENT_VERSION),17) .

compose-db-backup: ## 在数据库容器内备份到 ./backups
	$(COMPOSE_ENV) $(COMPOSE) exec -T mariadb bash /opt/jimu/scripts/backup.sh /backups

compose-db-restore: ## 在数据库容器内恢复（FILE=/backups/<name>.sql.gz）
	@test -n "$(FILE)" || { echo "❌ 用法: make compose-db-restore FILE=/backups/jimu_YYYYmmdd_HHMMSS.sql.gz"; exit 1; }
	$(COMPOSE_ENV) $(COMPOSE) exec -T -e FORCE=1 mariadb bash /opt/jimu/scripts/restore.sh "$(FILE)"

test-backup-restore: ## 备份/恢复往返测试（CONTAINER=<name>）
	@./scripts/test_backup_restore.sh $(CONTAINER)

##@ 质量门禁

test: ## 运行测试
	go test ./... -v

test-coverage: ## 运行测试并生成覆盖率报告
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-cover: ## 运行测试并生成 coverage.out
	go test ./... -coverprofile=coverage.out

test-coverage-check: ## 校验全仓覆盖率阈值与逐包下限
	@COVERAGE=$$(go tool cover -func=coverage.out | grep -E '^total:' | awk '{print $$3}' | sed 's/%//'); \
	THRESHOLD=70; \
	printf '%s' "$${COVERAGE}" | grep -Eq '^[0-9]+([.][0-9]+)?$$' || { echo "❌ 无法解析总覆盖率：go tool cover -func 失败，或 coverage.out 陈旧/不可读（COVERAGE='$${COVERAGE}'）"; exit 1; }; \
	echo "📈 Total coverage: $${COVERAGE}%"; \
	if [ $$(awk "BEGIN{print ($${COVERAGE}<$${THRESHOLD})}") -eq 1 ]; then \
		echo "❌ Coverage $${COVERAGE}% is below threshold $${THRESHOLD}%"; \
		go tool cover -func=coverage.out; exit 1; \
	fi; \
	echo "✅ Coverage check passed ($${COVERAGE}% >= $${THRESHOLD}%)"
	@bash scripts/check_coverage_floors.sh coverage.out

test-coverage-check-local: ## 本地覆盖率检查（SKIP_COVERAGE_FLOORS=1）
	SKIP_COVERAGE_FLOORS=1 $(MAKE) --no-print-directory test-coverage-check

test-race: ## 运行竞争检测测试
	go test -race ./...

vet: ## 静态分析
	go vet ./...

fmt: ## 格式化代码
	go fmt ./...

fmt-check: ## 检查代码格式（不修改）
	@echo "Checking gofmt..."
	@if gofmt -l . | grep -q .; then \
		echo "以下文件需要格式化:"; \
		gofmt -l .; \
		exit 1; \
	else \
		echo "所有文件格式正确"; \
	fi

lint: ## 静态检查（版本与 CI 一致）
	@if command -v golangci-lint >/dev/null 2>&1 && golangci-lint version 2>/dev/null | grep -qw "$(LINT_VERSION:v%=%)"; then \
		golangci-lint run ./...; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		echo "本地 golangci-lint 版本与 CI（$(LINT_VERSION)）不一致，改用 go run 固定版本"; \
		go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(LINT_VERSION) run ./...; \
	else \
		echo "golangci-lint 未安装，使用 go vet 替代"; \
		go vet ./...; \
	fi

check-log-usage: ## 检查结构化日志调用
	@go run ./tools/logcheck "./internal/..." "./cmd/..." "./tools/..."

check-capabilities: ## 校验能力声明、迁移、驱动与资产归属
	@go run ./tools/checkcapabilities

check-templates: ## 生成最小项目并执行构建与能力门禁
	JIMU_HEAVY_MATRIX=1 go test ./tools/generator/ -run TestTemplatesDrift -count=1 -timeout 30m

test-scaffold-matrix: ## 运行重型脚手架生成/构建/运行矩阵
	JIMU_HEAVY_MATRIX=1 go test ./tools/generator/ -count=1 -timeout 60m -v

profiles-check: ## 构建全部形态并检查依赖闭包
	@bash scripts/check_profiles.sh

compose-report: ## 生成形态编译面报告
	@go run ./tools/composereport

compose-report-check: ## 校验形态编译面报告无漂移
	@go run ./tools/composereport -check

check-skills: ## 校验 Agent skill 契约
	@./scripts/check_skills.sh

skills-install: ## 安装 skills/ 到 Agent 发现路径
	@./scripts/install_skills.sh

swagger: ## 生成 OpenAPI 文档
	@assets=$$(go run ./tools/profileassets "$(PROFILE)") || exit 1; \
	printf '%s\n' "$$assets" | grep -qx 'docs/openapi' || { echo "SKIP swagger：形态 $(PROFILE) 未编入 apidocs"; exit 0; }; \
	echo "$(SWAG) init -g $(SERVER_CMD) -o docs/openapi"; \
	$(SWAG) init -g $(SERVER_CMD) -o docs/openapi

swagger-check: ## 校验 OpenAPI 文档为最新
	@assets=$$(go run ./tools/profileassets "$(PROFILE)") || exit 1; \
	printf '%s\n' "$$assets" | grep -qx 'docs/openapi' || { echo "SKIP swagger-check：形态 $(PROFILE) 未编入 apidocs"; exit 0; }; \
	echo "$(SWAG) init -g $(SERVER_CMD) -o docs/openapi"; \
	$(SWAG) init -g $(SERVER_CMD) -o docs/openapi >/dev/null || exit 1; \
	git diff --exit-code docs/openapi || { echo "❌ docs/openapi 不是最新，请运行 make swagger"; exit 1; }; \
	echo "✅ OpenAPI 文档为最新"

smoke-check: ## 校验自动化脚本语法与部署 manifest 契约
	@bash -n scripts/dev.sh
	@bash -n scripts/image-fingerprint.sh
	@bash -n scripts/ensure-image.sh
	@bash -n scripts/test_shards.sh
	@bash -n scripts/check_ci_gate.sh
	@bash -n scripts/test_runtime_security.sh
	@bash -n scripts/smoke_api_contract.sh
	@bash -n scripts/install_db_clients.sh
	@bash -n scripts/db_common.sh
	@bash -n scripts/backup.sh
	@bash -n scripts/restore.sh
	@bash -n scripts/test_backup_restore.sh
	@bash -n scripts/test_deployment_manifests.sh
	@./scripts/test_deployment_manifests.sh
	@echo "✅ Smoke 脚本语法与部署 manifest 契约正确"

compose-check: ## 运行隔离 Compose 与 API 契约验证
	@./scripts/test_runtime_security.sh
	@./scripts/smoke_api_contract.sh

govulncheck: ## 扫描可达依赖漏洞
	@go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

check-makefile: ## 校验 Makefile 生命周期、镜像缓存与 Secret 接线
	@./scripts/test_makefile_contract.sh

check: fmt-check vet lint check-log-usage check-capabilities profiles-check compose-report-check check-makefile test ## 运行开发质量门禁
	@echo "✅ All checks passed"

ci: fmt-check vet lint check-log-usage check-capabilities profiles-check compose-report-check check-makefile test-cover test-coverage-check-local test-race swagger-check smoke-check build govulncheck test-scaffold-matrix ## 运行完整本地 CI
	@echo "✅ All local CI checks passed"

release-check: fmt-check vet check-log-usage check-capabilities profiles-check compose-report-check check-makefile test govulncheck compose-check test-scaffold-matrix ## 运行发布前检查
	@echo "All checks passed"

##@ 工具

proto: ## 重新生成 gRPC 代码
	@command -v protoc >/dev/null 2>&1 || { echo "protoc 未安装: brew install protobuf"; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go 未安装: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "protoc-gen-go-grpc 未安装: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }
	protoc --go_out=. --go_opt=module=jimu \
	       --go-grpc_out=. --go-grpc_opt=module=jimu \
	       proto/jimu/v1/*.proto
	@echo "gRPC 代码已重新生成"

bench: ## 运行性能基准测试
	go test -bench=. -benchmem -run='^$$' ./internal/shared/id/... ./internal/capabilities/auth/application/... ./internal/capabilities/notification/... ./internal/kernel/http/... ./internal/capabilities/queue/...

bench-ci: ## 运行性能回归门禁
	@./scripts/bench_ci.sh --absolute

loadtest: ## 本地 HTTP 压测
	@./scripts/loadtest.sh

cli: build-cli ## 编译 CLI

all: fmt vet test build ## 格式化、静态检查、测试并编译

hooks: ## 启用仓库 git hooks
	git config core.hooksPath githooks
	@echo "git hooks enabled (hooksPath=githooks: commit-msg + pre-commit wrapper)"
