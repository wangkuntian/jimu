.PHONY: run build test vet fmt fmt-check lint clean migrate migrate-down migrate-status seed help govulncheck test-backup-restore ci
.PHONY: test-cover test-coverage-check test-race swagger-check smoke-check compose-check profiles-check compose-report
.PHONY: docker-build docker-run docker-stop docker-logs
.PHONY: compose-up compose-down compose-restart compose-logs compose-migrate compose-seed
.PHONY: bench loadtest proto secrets

# 默认目标
.DEFAULT_GOAL := help

# 变量
BIN_DIR := bin
# 形态（profile）：full/minimal/saas/enterprise/machine；full 为默认（与提交态 active 一致）。
# 切形态不改动任何受版本控制的文件：profileoverlay 生成 .overlay/<profile>/ 并在构建期叠加。
PROFILE ?= full
SERVER_PKG := ./cmd/server
# 用递归展开（= 而非 :=）：下方 include .env 可能在解析期之后才把 PROFILE 改成别的形态，
# 立即展开会让「.env 设 PROFILE=minimal」变成「用 minimal 构建、产物名却仍是 bin/jimu-server」。
SERVER_BIN = $(BIN_DIR)/jimu-server$(if $(filter-out full,$(PROFILE)),-$(PROFILE),)
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
# golangci-lint 版本：与 .github/workflows/ci.yml 的 GOLANGCI_LINT_VERSION 一致，
# 使本地 `make lint` 与 CI 的 lint 结论含义相同；本地二进制版本不同时改用 go run 固定版本；
# 仅在 golangci-lint 完全缺失时才退化为 go vet（不具备版本一致性）
LINT_VERSION ?= v2.7.2

# 根据 APP_ENV 自动生成 --profile 参数：dev 环境启动 adminer
COMPOSE_DEV_PROFILE = $(if $(filter dev,$(APP_ENV)),dev)
# observability（OpenObserve 监控栈）默认开启（与 docker-compose 的 ${OTEL_ENABLED:-true} 一致）；
# 显式 OTEL_ENABLED=false 关闭（不启动监控栈、server 不推送）
COMPOSE_OBS_PROFILE = $(if $(filter false,$(OTEL_ENABLED)),,observability)
# 组合 flag（如 --profile dev --profile observability）
COMPOSE_PROFILE_FLAG = $(if $(strip $(COMPOSE_DEV_PROFILE) $(COMPOSE_OBS_PROFILE)),$(addprefix --profile ,$(COMPOSE_DEV_PROFILE) $(COMPOSE_OBS_PROFILE)))

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
	@echo "  make build-server         编译服务端（PROFILE=<name> 可选形态，默认 full，产物 bin/jimu-server[-<name>]）"
	@echo "  make test                 运行测试"
	@echo "  make vet                  静态分析"
	@echo "  make fmt                  格式化代码"
	@echo "  make lint                 静态检查"
	@echo "  make check-log-usage      检查日志调用均为 *w 系列（防 k/v 粘连）"
	@echo "  make check-capabilities   校验能力自描述（Owns）与驱动可用集/选中集一致"
	@echo "  make profiles-check       构建 5 个形态（overlay 叠加 cmd/server）+ golden 依赖闭包门禁"
	@echo "                            （JIMU_PROFILES_SMOKE=1 时额外启动并检查 /readyz）"
	@echo "  make compose-report       生成各形态（overlay 叠加 cmd/server）的编译面报告 docs/profiles/compose-report.md"
	@echo ""
	@echo "数据库:"
	@echo "  make migrate              本地执行迁移"
	@echo "  make migrate-down         本地回滚迁移"
	@echo "  make migrate-status       查看迁移状态"
	@echo "  make seed                 本地插入初始数据"
	@echo "  make build-backup-image   构建备份任务镜像 jimu-backup（K8s/Helm CronJob 用）"
	@echo "  make compose-db-backup    在数据库容器内备份（脚本已挂载，产物落 ./backups）"
	@echo "  make compose-db-restore   在数据库容器内恢复（需 FILE=/backups/xxx.sql.gz，破坏性）"
	@echo "  make backup               主机侧备份（需本机 mariadb-dump/mysqldump 客户端）"
	@echo "  make restore              主机侧恢复（需 BACKUP_FILE=...，FORCE=1 跳过确认）"
	@echo "  make test-backup-restore  备份/恢复往返测试（需运行中 mariadb 容器）"
	@echo ""
	@echo "Docker 容器（单容器，需外部 DB + Redis）:"
	@echo "  make docker-build         构建镜像（PROFILE=<name> 可选形态，默认 full）"
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
	@echo "  可选服务（环境变量开启）：OTEL_ENABLED=true 启动监控栈 + server 遥测推送（OpenObserve+采集+dashboard）"
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
# 用 $$(...) 而不是 $(shell ...)：形态名非法时 profileoverlay 带清晰错误非零退出，make 随之
# 失败，而不是留下一个空的 -overlay=。先把路径赋给变量再构建：赋值语句的退出码就是 $$(...)
# 的退出码，&& 因而能截断构建；若直接写成 `go build -overlay=$$(...)`，失败的命令替换只留下
# 空的 -overlay=，go build 会**静默按提交态（full）构建**并成功退出。
build-server:
	@mkdir -p $(BIN_DIR)
	overlay="$$(go run ./tools/profileoverlay "$(PROFILE)")" && go build -ldflags "$(LDFLAGS)" -overlay="$$overlay" -o $(SERVER_BIN) $(SERVER_PKG)

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

## build-backup-image: 构建数据库备份任务镜像（内置 mariadb-dump + pg_dump + 仓库脚本；PG 客户端版本可配）
build-backup-image:
	docker build -f deploy/backup/Dockerfile -t jimu-backup:latest --build-arg PG_CLIENT_VERSION=$(or $(PG_CLIENT_VERSION),17) .

## compose-db-backup: 在数据库容器内执行备份（脚本已挂载，输出到宿主机 ./backups）
compose-db-backup:
	$(DOCKER_COMPOSE) $(COMPOSE_PROFILE_FLAG) exec -T mariadb bash /opt/jimu/scripts/backup.sh /backups

## compose-db-restore: 在数据库容器内执行恢复（破坏性；用法: make compose-db-restore FILE=/backups/jimu_xxx.sql.gz）
compose-db-restore:
	@test -n "$(FILE)" || { echo "❌ 用法: make compose-db-restore FILE=/backups/jimu_YYYYmmdd_HHMMSS.sql.gz"; exit 1; }
	$(DOCKER_COMPOSE) $(COMPOSE_PROFILE_FLAG) exec -T -e FORCE=1 mariadb bash /opt/jimu/scripts/restore.sh "$(FILE)"

## test-backup-restore: 备份/恢复往返测试（通过 docker exec 调用容器内 mariadb，用法: make test-backup-restore [CONTAINER=jimu-test-mysql]）
test-backup-restore:
	@./scripts/test_backup_restore.sh $(CONTAINER)

# ========== Docker 单容器 ==========

## docker-build: 构建 Docker 镜像（PROFILE=<name> 可选形态，默认 full）
docker-build:
	docker build --build-arg PROFILE=$(PROFILE) -t "$(DOCKER_IMAGE)" .

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
# 统一入口：make compose-up / compose-down。
# 可选服务通过环境变量开启：
#   dev           : APP_ENV=dev（adminer）
#   observability : OTEL_ENABLED=true（统一开关：启动监控栈 + server 遥测推送）
#                   （opens OpenObserve + MySQL/Redis 采集 + 默认 dashboard）

## compose-up: 启动服务（compose-up 开启 observability 时自动初始化 dashboard）
compose-up:
	$(DOCKER_COMPOSE) $(COMPOSE_PROFILE_FLAG) up -d
	@if [ "$(COMPOSE_OBS_PROFILE)" = "observability" ]; then bash scripts/observability.sh start; fi

## compose-down: 停止并删除所有 compose 服务容器（保留数据卷）
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

## compose-seed: Compose 环境插入初始数据（需 .env 提供 ADMIN_PASSWORD）
compose-seed:
	@test -n "$(ADMIN_PASSWORD)" || { echo "❌ 缺少 ADMIN_PASSWORD：请在 .env 中设置管理员初始密码"; exit 1; }
	$(DOCKER_COMPOSE) $(COMPOSE_PROFILE_FLAG) run --rm -e ADMIN_PASSWORD="$(ADMIN_PASSWORD)" server ./jimu seed

## secrets: 从 .env 生成 Docker Secrets 文件（./secrets/*.txt；compose 各服务经 _FILE 挂载）
##         依赖变量：DB_ROOT_PASSWORD / DB_PASSWORD / JWT_SECRET / ZO_OBSERVE_ROOT_USER_*
##         已存在的文件默认不覆盖（MAKE_SECRETS_FORCE=1 强制重新生成）
secrets:
	@bash scripts/gen-secrets.sh

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

## lint: 静态检查（需要 golangci-lint；版本与 CI 一致，见 LINT_VERSION）
lint:
	@if command -v golangci-lint >/dev/null 2>&1 && golangci-lint version 2>/dev/null | grep -qw "$(LINT_VERSION:v%=%)"; then \
		golangci-lint run ./...; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		echo "本地 golangci-lint 版本与 CI（$(LINT_VERSION)）不一致，改用 go run 固定版本"; \
		go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(LINT_VERSION) run ./...; \
	else \
		echo "golangci-lint 未安装，使用 go vet 替代"; \
		go vet ./...; \
	fi

## check-log-usage: 检查日志调用符合结构化规范（logcheck 静态分析：防粘连 R1 /
## 禁动态 key R2 / 字段词汇表 R3 / 禁嵌套对象 R4；规则与 AGENTS.md 日志调用规范同步）
check-log-usage:
	@go run ./tools/logcheck "./internal/..." "./cmd/..." "./tools/..."

## check-capabilities: 校验能力自描述（Owns）与迁移归属一致，并静态门禁驱动选择
##                      （可用集目录存在、核心零驱动、形态选中集==import 闭包、驱动归属）；
##                      当前未接入 make ci/release-check，收口见 P2.8。
check-capabilities:
	@go run ./tools/checkcapabilities

## profiles-check: 构建 5 个形态（overlay 叠加 cmd/server）+ golden 依赖闭包门禁；
##                  构建或门禁失败即非零退出。
##                  设置 JIMU_PROFILES_SMOKE=1 后额外以 APP_ENV=dev 启动各形态并轮询
##                  /readyz（需 DB+Redis）；未设置时逐形态打印 SKIP，不静默跳过。
profiles-check:
	@bash scripts/check_profiles.sh

## compose-report: 生成各形态（profile，overlay 叠加 cmd/server）的编译面报告
##                 （docs/profiles/compose-report.md，入库）。
##                 指标：二进制大小 / 路由数 / 迁移数 / 表数 / 本仓闭包代码量与文件数 /
##                 重型依赖（aws-sdk-go-v2 / kafka-go / amqp091-go / excelize）/
##                 go.mod 直接依赖数（各形态相同，见报告的「层②边界」）；不连库、不启动监听。
compose-report:
	@go run ./tools/composereport

## clean: 清理构建产物（含按形态隔离的 overlay 产物 .overlay/）
clean:
	rm -rf $(BIN_DIR)
	rm -rf .overlay
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
	go test -bench=. -benchmem -run='^$$' ./internal/shared/id/... ./internal/capabilities/auth/application/... ./internal/capabilities/notification/... ./internal/kernel/http/... ./internal/capabilities/queue/...

## bench-ci: 性能回归门禁（绝对阈值模式，CI 用）
bench-ci:
	@./scripts/bench_ci.sh --absolute

## loadtest: 本地 HTTP 压测（需 hey，默认打健康检查）
loadtest:
	@./scripts/loadtest.sh

## govulncheck: 依赖漏洞扫描（go run 免安装；豁免清单见 scripts/govulncheck.sh）
govulncheck:
	@bash scripts/govulncheck.sh

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
	@bash -n scripts/govulncheck.sh
	@bash -n scripts/install_db_clients.sh
	@bash -n scripts/db_common.sh
	@bash -n scripts/backup.sh
	@bash -n scripts/restore.sh
	@bash -n scripts/test_backup_restore.sh
	@echo "✅ Smoke 脚本语法正确"

## compose-check: 隔离 Compose 运行时与 API 契约验证（不会读取或修改本地 .env、secrets、数据卷）
compose-check:
	@./scripts/test_runtime_security.sh
	@./scripts/smoke_api_contract.sh

## ci: 本地 CI 检查（无外部依赖部分，完整 CI 见 .github/workflows/ci.yml）
ci: fmt-check vet lint check-log-usage test-cover test-coverage-check test-race swagger-check smoke-check build govulncheck
	@echo "✅ All local CI checks passed"

## release-check: 发布前检查（Go 门禁 + govulncheck + 隔离 Compose/API smoke）
release-check: fmt-check vet check-log-usage test govulncheck compose-check
	@echo "All checks passed"

## hooks: 启用 git 钩子（core.hooksPath=githooks：commit-msg 全英文检查 + pre-commit 框架包装；框架检查需 pip install pre-commit）
hooks:
	git config core.hooksPath githooks
	@echo "git hooks enabled (hooksPath=githooks: commit-msg + pre-commit wrapper)"

## dev: 热重载开发模式（需 air: go install github.com/air-verse/air@latest）
dev:
	@if command -v air >/dev/null 2>&1; then \
		APP_ENV=dev air; \
	else \
		echo "air 未安装，运行: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi
