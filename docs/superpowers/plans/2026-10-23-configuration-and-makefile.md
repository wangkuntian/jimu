# 配置收敛与 Makefile 生命周期重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把日常配置收敛到 `configs/app.yaml`，把敏感值收敛到自动生成的 `secrets/`，并用少量稳定的 Makefile 入口替代现有 `.env` 与重复启动命令。

**Architecture:** 应用配置、部署接线、首次凭据和可选组件凭据分层维护。Makefile 只编排外部命令；Compose 通过一次性 `jimu-init` 服务完成迁移和种子；根仓与生成项目的构建文件统一使用 `make image` 构建镜像。

**Tech Stack:** Go 1.27、GNU Make、Docker Compose v5、Bash、Viper、Go 模板生成器

**Spec:** `docs/superpowers/specs/2026-10-23-configuration-and-makefile-design.md`

## Global Constraints

- 默认回复使用中文。
- 不得在未获用户明确指令时执行 `git add`、`git commit` 或 `git push`；本计划的每个任务以验证和变更汇报为终点。
- `configs/app.yaml` 是用户编辑的业务配置源；不得把真实密码写入受版本控制的 YAML。
- 正常启动不得读取项目根 `.env`；所有 Compose 调用必须带 `--env-file /dev/null`。
- `secrets/` 目录权限为 `0700`，secret 文件权限为 `0600`，默认不覆盖已有文件。
- `security.encryption_key` 非空时必须恰好 32 字节；AES-GCM 不接受其他长度。
- `make image` 构建 `$(IMAGE_REPO):$(VERSION)` 并同时标记 `$(IMAGE_REPO):latest`；默认 `VERSION=dev`。
- `PROFILE` 贯穿本机构建、镜像构建、迁移、种子和开发运行；`VERSION` 只控制镜像标签。
- 删除旧 Makefile 入口时不保留兼容别名。
- 生成项目的 Makefile 同步自动帮助、无 `.env`、`build`/`image`/`dev` 口径；不生成 Compose 专属入口。
- 不处理 OAuth、SMTP、SMS、S3、Webhook、CSRF 等可选集成凭据的自动生成。

## Review Focus

- 已有数据库卷但 secret 不完整时，不能生成新 DB 密码或新字段加密密钥，否则会锁死已有数据。
- 项目根存在旧 `.env` 且含冲突值时，`make up` 的最终配置不能被它改变。
- 字段加密密钥为 31、32、33 字节时，必须分别得到失败、成功、失败。
- 干净检出、无 TTY、无 `secrets/` 时，`make up` 必须能初始化并给出可恢复的错误，不得挂起等待输入。
- OpenObserve 已有数据卷但凭据文件丢失时，必须失败并保留数据，不得用新密码覆盖管理员。

---

### Task 1: 统一管理员密码读取并修正加密密钥校验

**Files:**
- Create: `internal/app/admin_password.go`
- Create: `internal/app/admin_password_test.go`
- Modify: `internal/app/seed.go:23`
- Modify: `internal/profiles/profiles.go:33`
- Modify: `internal/config/validate.go:10`
- Modify: `internal/config/config_test.go`
- Modify: `configs/app.yaml:271`

**Interfaces:**
- Consumes: `config.GetEnvOrFile(fileKey, directKey string) string`
- Produces: `app.AdminPassword() string`，优先读取 `ADMIN_PASSWORD_FILE`，其次读取 `ADMIN_PASSWORD`，结果去除首尾空白。

- [ ] **Step 1: 写管理员密码读取失败测试**

在 `internal/app/admin_password_test.go` 写：

```go
package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/app"

	"github.com/stretchr/testify/require"
)

func TestAdminPasswordPrefersFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin_password")
	require.NoError(t, os.WriteFile(path, []byte("from-file\n"), 0o600))
	t.Setenv("ADMIN_PASSWORD_FILE", path)
	t.Setenv("ADMIN_PASSWORD", "from-env")
	require.Equal(t, "from-file", app.AdminPassword())
}

func TestAdminPasswordFallsBackToEnv(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "from-env")
	require.Equal(t, "from-env", app.AdminPassword())
}

func TestAdminPasswordMissing(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "")
	require.Empty(t, app.AdminPassword())
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/app -run 'TestAdminPassword' -count=1`
Expected: FAIL，`undefined: AdminPassword`

- [ ] **Step 3: 实现 `AdminPassword`**

在 `internal/app/admin_password.go` 实现：

```go
package app

import "jimu/internal/config"

// AdminPassword returns the one-time structural seed password.
// File input is preferred so Compose can mount it without exposing the value in argv.
func AdminPassword() string {
	return config.GetEnvOrFile("ADMIN_PASSWORD_FILE", "ADMIN_PASSWORD")
}
```

- [ ] **Step 4: 让 seed 与启动期结构种子共用该函数**

`internal/app/seed.go` 中 `RunSeed` 改为 `adminPassword := AdminPassword()`；`internal/profiles/profiles.go` 中 `StructuralSeed` 改为 `if app.AdminPassword() == ""`。

在 `internal/app/seed_test.go` 增加 `TestRunSeed_UsesAdminPasswordFile`，复用现有 `newMockGormDB` 与 `expectRunSeedQueries`：写入临时 admin secret 文件，设置 `ADMIN_PASSWORD_FILE` 与空 `ADMIN_PASSWORD`，断言 `RunSeed` 成功。

- [ ] **Step 5: 写加密密钥长度失败测试**

在 `internal/config/config_test.go` 增加：

```go
// 同时补充 github.com/stretchr/testify/require 导入。
func TestValidateEncryptionKeyRequiresExactAES256Length(t *testing.T) {
	for _, tc := range []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"31 bytes", strings.Repeat("a", 31), true},
		{"32 bytes", strings.Repeat("a", 32), false},
		{"33 bytes", strings.Repeat("a", 33), true},
		{"dev placeholder", "dev-encryption-key-change-in-production", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validProdConfig()
			cfg.Security.EncryptionKey = tc.key
			err := cfg.Validate("dev")
			if tc.wantErr {
				require.ErrorContains(t, err, "security.encryption_key")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
```

- [ ] **Step 6: 运行测试确认 33 字节用例失败**

Run: `go test ./internal/config -run TestValidateEncryptionKeyRequiresExactAES256Length -count=1`
Expected: FAIL，33 字节和 dev placeholder 当前未被拒绝。

- [ ] **Step 7: 将校验移入公共校验并改为精确长度**

在 `internal/config/validate.go` 的 `validateCommon` 中增加精确长度校验，并删除 `Validate` 中只在 prod 执行的旧校验：

```go
if c.Security.EncryptionKey != "" && len(c.Security.EncryptionKey) != 32 {
	return errors.New("invalid security.encryption_key, must be exactly 32 bytes when set")
}
```

- [ ] **Step 8: 移除无效的开发占位密钥**

把 `configs/app.yaml` 的 `security.encryption_key` 改为空字符串，并更新注释为“非空时必须恰好 32 字节；Compose 通过 `ENCRYPTION_KEY_FILE` 注入”。

- [ ] **Step 9: 运行聚焦验证**

Run: `go test ./internal/app ./internal/profiles ./internal/config -count=1`
Expected: PASS

- [ ] **Step 10: 停止点**

汇报修改文件与测试结果；不要提交。

---

### Task 2: 新增幂等 secret 初始化脚本

**Files:**
- Create: `scripts/init-secrets.sh`
- Create: `scripts/test_init_secrets.sh`
- Delete: `scripts/gen-secrets.sh`

**Interfaces:**
- Consumes: `SECRETS_DIR`（默认仓库根 `secrets/`）、`ADMIN_PASSWORD`（可选非交互输入）、`LEGACY_ENV_FILE`（可选；默认空，只有 Makefile 在首次迁移时显式指向旧 `.env`）
- Produces: 核心 secret 文件，以及 `--observability` 时生成的 OpenObserve 三文件
- CLI: `scripts/init-secrets.sh [--observability] [--allow-new-security-keys]`

- [ ] **Step 1: 写 secret 初始化失败测试**

在 `scripts/test_init_secrets.sh` 建立临时目录并用断言覆盖：

```bash
SECRETS_DIR="$tmp/secrets" "$ROOT/scripts/init-secrets.sh"
for f in db_root_password.txt db_password.txt jwt_secret.txt \
         encryption_key.txt audit_hash_secret.txt admin_password.txt; do
  test -f "$SECRETS_DIR/$f"
  test "$(file_mode "$SECRETS_DIR/$f")" = "600"
done
test "$(wc -c < "$SECRETS_DIR/encryption_key.txt" | tr -d ' ')" = "32"
test "$(wc -c < "$SECRETS_DIR/jwt_secret.txt" | tr -d ' ')" -ge 32
```

同一脚本还要断言：

```bash
sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}
first="$(sha256 "$SECRETS_DIR/jwt_secret.txt")"
SECRETS_DIR="$SECRETS_DIR" "$ROOT/scripts/init-secrets.sh"
test "$first" = "$(sha256 "$SECRETS_DIR/jwt_secret.txt")"
```

`file_mode` 辅助函数先尝试 `stat -c %a`（Linux），失败再使用 `stat -f %Lp`（macOS）。

再增加以下场景断言：

- `</dev/null` 运行不挂起，并生成随机 admin 密码。
- `--observability` 生成 `otel_auth_password.txt`、`openobserve.env`、`zo_auth_token.txt`；解码 token 等于 `base64(admin@jimu.local:password)`。
- 用临时 `PATH` 注入假 `docker volume inspect`，使 MariaDB/OpenObserve 卷表现为已存在：
  - 缺 DB secret 时退出非零。
  - 缺 encryption/audit secret 时退出非零。
  - 加 `--allow-new-security-keys` 后允许生成缺失安全 secret。
  - OpenObserve 卷存在且无旧密码时退出非零。
- `LEGACY_ENV_FILE` 指向临时 `.env` 时，只迁移六个 secret 键；冲突的 `DB_HOST` 值不得出现于任何生成文件。

- [ ] **Step 2: 运行测试确认失败**

Run: `bash scripts/test_init_secrets.sh`
Expected: FAIL，`scripts/init-secrets.sh: No such file or directory`

- [ ] **Step 3: 实现核心 secret 生成**

`scripts/init-secrets.sh` 使用 `umask 077`、原子写入和 `chmod 0600`，生成：

- `db_root_password.txt`：至少 24 字符随机串
- `db_password.txt`：至少 24 字符随机串
- `jwt_secret.txt`：`openssl rand -hex 32`
- `encryption_key.txt`：恰好 32 个可打印 ASCII 字节，无换行
- `audit_hash_secret.txt`：`openssl rand -hex 32`
- `admin_password.txt`：TTY 时隐藏输入；非 TTY 时随机生成

- [ ] **Step 4: 实现旧 `.env` 的逐文件迁移**

只读取以下键，且只在目标文件缺失时导入：

```text
DB_ROOT_PASSWORD
DB_PASSWORD
JWT_SECRET
ADMIN_PASSWORD
ZO_OBSERVE_ROOT_USER_PASSWORD
```

不得导入 `DB_HOST`、`HTTP_HOST_PORT`、`COMPOSE_*` 等普通配置。迁移后提示 `.env` 已停用，但不删除该文件。

`LEGACY_ENV_FILE` 默认必须为空；脚本不得主动读取仓库根 `.env`。`make init` 在旧 `.env` 存在时显式传入该路径。

- [ ] **Step 5: 实现已有数据卷保护**

通过 `${COMPOSE_MARIADB_VOLUME:-jimu-mariadb-data}` 与 `${COMPOSE_OPENOBSERVE_VOLUME:-jimu-openobserve-data}` 调用 `docker volume inspect`：

- MariaDB 卷存在且 DB secret 缺失：exit 1，提示恢复原凭据。
- MariaDB 卷存在且安全 secret 缺失、未传 `--allow-new-security-keys`：exit 1。
- 传 `--allow-new-security-keys` 时允许生成缺失安全 secret，但输出明确警告。
- OpenObserve 卷存在且找不到旧密码：exit 1，不覆盖管理员。

- [ ] **Step 6: 实现 OpenObserve 派生文件**

`--observability` 时生成：

```text
otel_auth_password.txt   # 密码源
openobserve.env          # ZO_ROOT_USER_EMAIL / ZO_ROOT_USER_PASSWORD
zo_auth_token.txt        # base64(email:password)
```

密码必须满足 OpenObserve 的 8-128 字符、大小写、数字和特殊字符要求。已存在自定义 `zo_auth_token.txt` 时不得静默覆盖。

- [ ] **Step 7: 运行脚本测试**

Run: `bash -n scripts/init-secrets.sh scripts/test_init_secrets.sh && bash scripts/test_init_secrets.sh`
Expected: PASS，所有断言通过

- [ ] **Step 8: 删除旧生成脚本**

删除 `scripts/gen-secrets.sh`；后续任务负责更新所有引用。

- [ ] **Step 9: 停止点**

汇报修改文件与测试结果；不要提交。

---

### Task 3: 重构 Compose 启动链与 observability 叠加

**Files:**
- Modify: `docker-compose.yml`
- Create: `compose.observability.yml`
- Create: `scripts/test_compose_config.sh`
- Modify: `scripts/observability.sh`
- Modify: `scripts/test_runtime_security.sh`
- Modify: `scripts/smoke_api_contract.sh`

**Interfaces:**
- Consumes: `JIMU_IMAGE`、`JIMU_SECRETS_DIR`、`APP_ENV`
- Produces: 核心服务 `mariadb`、`redis`、`jimu-init`、`server`；可选叠加服务 `openobserve`、`otel-collector`
- `jimu-init` command: `/bin/sh -c './jimu migrate up && ./jimu seed'`

- [ ] **Step 1: 写 Compose 契约失败测试**

在 `scripts/test_compose_config.sh` 创建临时 secret 文件并断言：

```bash
JIMU_SECRETS_DIR="$SECRETS_DIR" \
  docker compose --env-file /dev/null -f docker-compose.yml config --quiet

JIMU_SECRETS_DIR="$SECRETS_DIR" \
  docker compose --env-file /dev/null \
    -f docker-compose.yml -f compose.observability.yml config --quiet
```

再通过 `docker compose config --format json` 断言：

```python
services = json.load(sys.stdin)["services"]
assert "jimu-init" in services
assert services["server"]["depends_on"]["jimu-init"]["condition"] == "service_completed_successfully"
assert services["server"]["environment"]["ENCRYPTION_KEY_FILE"] == "/run/secrets/encryption_key"
assert services["server"]["environment"]["AUDIT_HASH_SECRET_FILE"] == "/run/secrets/audit_hash_secret"
```

测试还必须把 `docker-compose.yml` 与 `compose.observability.yml` 复制到临时目录，写入 `JIMU_IMAGE=poison:latest` 的 `.env`，再执行 `--env-file /dev/null` 校验；最终配置中的镜像默认值不得是 `poison:latest`。

- [ ] **Step 2: 运行测试确认失败**

Run: `bash scripts/test_compose_config.sh`
Expected: FAIL，缺少 `jimu-init` 与新增 secret 环境变量

- [ ] **Step 3: 改造核心 Compose**

在 `docker-compose.yml`：

- 固定 `DB_HOST=mariadb`、`DB_PORT=3306`、`DB_USER=jimu`、`DB_NAME=jimu`、`REDIS_ADDR=redis:6379`、`REDIS_DB=0`。
- 删除 `COMPOSE_DB_*` 与 `COMPOSE_REDIS_*`。
- 新增 `encryption_key`、`audit_hash_secret`、`admin_password` secret。
- server 与 jimu-init 注入 `ENCRYPTION_KEY_FILE`、`AUDIT_HASH_SECRET_FILE`、`ADMIN_PASSWORD_FILE`。
- 新增 `jimu-init`，设置 `restart: "no"`，只依赖健康 MariaDB/Redis。
- server 改为 `depends_on jimu-init: service_completed_successfully`。
- 为本机 `make dev` 暴露 `127.0.0.1:${JIMU_DB_HOST_PORT:-3306}` 和 `127.0.0.1:${JIMU_REDIS_HOST_PORT:-6379}`。
- 保留 `adminer` 的 `dev` profile，但不让核心 `make up` 自动启用。

- [ ] **Step 4: 移动 observability 服务到叠加文件**

把 `openobserve`、`otel-collector` 及对应 secret 定义移入 `compose.observability.yml`。该文件：

- 给 server 设置 `OTEL_ENABLED=true`、`OTEL_ENDPOINT=openobserve:5081`、`OTEL_AUTH_EMAIL=admin@jimu.local`、`OTEL_AUTH_PASSWORD_FILE=/run/secrets/otel_auth_password`。
- 给 OpenObserve 使用 `env_file` 长语法并设置 `required: false`，文件为 `${JIMU_SECRETS_DIR:-./secrets}/openobserve.env`，保证核心 `down/status` 在未启用 observability 时仍可解析。
- 给 collector 挂载 `zo_auth_token` secret。
- 不定义默认 `Admin@12345`。

- [ ] **Step 5: 更新 observability 管理脚本**

`scripts/observability.sh` 不再 source `.env`，改为：

```bash
ROOT_DIR=...
SECRETS_DIR="${JIMU_SECRETS_DIR:-$ROOT_DIR/secrets}"
ZO_EMAIL="admin@jimu.local"
ZO_PASSWORD="$(cat "$SECRETS_DIR/otel_auth_password.txt")"
COMPOSE=(docker compose --env-file /dev/null -f "$ROOT_DIR/docker-compose.yml" -f "$ROOT_DIR/compose.observability.yml")
```

管理 start/stop/restart/status 时不得影响核心栈。

- [ ] **Step 6: 更新隔离 smoke 脚本**

`scripts/test_runtime_security.sh` 与 `scripts/smoke_api_contract.sh`：

- 删除 `ENV_FILE` 与 `.env` 生成。
- 使用临时 `JIMU_SECRETS_DIR`、独立 Compose project、process 环境变量。
- 为全部核心 secret 写临时文件，包括 `encryption_key.txt`、`audit_hash_secret.txt`、`admin_password.txt`。
- 不再手工 `migrate up` / `seed`，由 `jimu-init` 完成。
- 继续轮询 `/livez`、`/readyz` 和 API 契约。

- [ ] **Step 7: 验证 Compose**

Run: `bash scripts/test_compose_config.sh`
Expected: PASS

Run: `bash -n scripts/observability.sh scripts/test_runtime_security.sh scripts/smoke_api_contract.sh scripts/test_compose_config.sh`
Expected: PASS

- [ ] **Step 8: 停止点**

汇报修改文件与 Compose 校验结果；不要提交。

---

### Task 4: 重写根 Makefile 并增加本地开发脚本

**Files:**
- Modify: `Makefile`
- Create: `scripts/dev.sh`
- Modify: `.air.toml`
- Create: `scripts/test_makefile_contract.sh`

**Interfaces:**
- Consumes: `PROFILE`、`VERSION`、`ENV`
- Produces: `make up`、`make up-observability`、`make down`、`make restart`、`make status`、`make logs`、`make dev`、`make image`
- Image vars: `IMAGE_REPO ?= jimu`、`VERSION ?= dev`、`IMAGE_TAG = $(IMAGE_REPO):$(VERSION)`、`LATEST_TAG = $(IMAGE_REPO):latest`

- [ ] **Step 1: 写 Makefile 契约失败测试**

在 `scripts/test_makefile_contract.sh` 断言：

```bash
help="$(make help)"
grep -q 'make up ' <<<"$help"
grep -q 'make up-observability ' <<<"$help"
grep -q 'make image ' <<<"$help"
grep -q 'make check ' <<<"$help"
! grep -q 'make compose-up ' <<<"$help"
! grep -q 'include .env' Makefile
make -n image VERSION=v9.9.9 | grep -q 'jimu:v9.9.9'
make -n image VERSION=v9.9.9 | grep -q 'jimu:latest'
make -n up VERSION=v9.9.9 | grep -q 'JIMU_IMAGE=jimu:v9.9.9'
make -n up VERSION=v9.9.9 | grep -q -- '--no-build'
```

- [ ] **Step 2: 运行测试确认失败**

Run: `bash scripts/test_makefile_contract.sh`
Expected: FAIL，旧 Makefile 没有 `up`/`image`

- [ ] **Step 3: 重写 Makefile 公共入口**

删除 `.env` 加载和手工帮助，使用 `##@` 分组与 `## target: 描述` 自动生成帮助。定义：

```make
COMPOSE = docker compose --env-file /dev/null -f docker-compose.yml
COMPOSE_ALL = $(COMPOSE) -f compose.observability.yml
IMAGE_REPO ?= jimu
VERSION ?= dev
IMAGE_TAG = $(IMAGE_REPO):$(VERSION)
LATEST_TAG = $(IMAGE_REPO):latest
JIMU_SECRETS_DIR ?= $(CURDIR)/secrets
COMPOSE_ENV = APP_ENV=$(ENV) JIMU_SECRETS_DIR=$(JIMU_SECRETS_DIR) JIMU_IMAGE=$(IMAGE_TAG)
```

`image` 使用 `docker build --build-arg PROFILE=$(PROFILE) -t $(IMAGE_TAG) .`，再 `docker tag $(IMAGE_TAG) $(LATEST_TAG)`。

`up` 依赖 `init image`，执行：

```bash
$(COMPOSE_ENV) $(COMPOSE) up -d --no-build
```

增加 `.NOTPARALLEL: up up-observability`，确保 `init`、`image`、`compose up` 的顺序固定。

- [ ] **Step 4: 增加生命周期与运维入口**

实现：

- `init`：旧 `.env` 存在时执行 `LEGACY_ENV_FILE=$(CURDIR)/.env ./scripts/init-secrets.sh`，否则执行 `./scripts/init-secrets.sh`
- `init-observability`：`./scripts/init-secrets.sh --observability`
- `up-observability`：依赖 `init-observability image`，使用 `$(COMPOSE_ALL)`
- `down` / `restart` / `status` / `logs`：使用 `$(COMPOSE_ALL)`
- `migrate`：`$(COMPOSE_ENV) $(COMPOSE) run --rm jimu-init ./jimu migrate up`
- `seed`：`$(COMPOSE_ENV) $(COMPOSE) run --rm jimu-init ./jimu seed`
- `adminer`：使用 `--profile dev` 显式启动 adminer
- `check`：组合 `fmt-check vet lint check-log-usage check-capabilities profiles-check compose-report-check test`

删除 `compose-*`、`docker-*`、`run` 与公开的 `secrets` target，不留别名。保留现有备份、检查、发布与工具 target。

- [ ] **Step 5: 实现本机热重载脚本**

`scripts/dev.sh`：

1. 运行 `scripts/init-secrets.sh`。
2. 启动 `mariadb` 与 `redis`。
3. 导出 `DB_HOST=127.0.0.1`、`REDIS_ADDR=127.0.0.1:6379` 及各 `*_FILE` 路径。
4. 生成 `PROFILE` overlay。
5. 执行 CLI `migrate up` 与 `seed`。
6. 导出 `JIMU_OVERLAY` 后 `exec air`。

修改 `.air.toml`：

```toml
cmd = "go build -overlay=\"$JIMU_OVERLAY\" -o ./tmp/server ./cmd/server"
```

- [ ] **Step 6: 运行 Makefile 与脚本验证**

Run: `bash scripts/test_makefile_contract.sh`
Expected: PASS

Run: `bash -n scripts/dev.sh && make -n dev >/dev/null`
Expected: PASS

- [ ] **Step 7: 停止点**

汇报修改文件与验证结果；不要提交。

---

### Task 5: 同步生成项目的 Makefile 模板

**Files:**
- Modify: `tools/generator/render/templates/project/Makefile.tmpl`
- Modify: `tools/generator/buildfiles_test.go`
- Modify: `tools/generator/support/rewrite.go`

**Interfaces:**
- Consumes: 现有单形态 `build` 与 `Dockerfile`
- Produces: 生成项目的 `make help`、`make build`、`make image`、`make dev`；不包含 Compose 专属 target

- [ ] **Step 1: 写模板失败测试**

在 `tools/generator/buildfiles_test.go` 增加：

```go
func TestGeneratedMakefileUsesSharedLifecycleContract(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	out, err := RenderMakefile(set)
	require.NoError(t, err)
	src := string(out)
	assert.NotContains(t, src, "include .env")
	assert.NotContains(t, src, "\nrun:")
	assert.Contains(t, src, "\ndev:")
	assert.Contains(t, src, "\nimage:")
	assert.Contains(t, src, "IMAGE_REPO ?= jimu")
	assert.Contains(t, src, "IMAGE_TAG = $(IMAGE_REPO):$(VERSION)")
	assert.Contains(t, src, "docker tag $(IMAGE_TAG) $(LATEST_TAG)")
	assert.Contains(t, src, "## help:")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./tools/generator -run TestGeneratedMakefileUsesSharedLifecycleContract -count=1`
Expected: FAIL，模板仍有 `.env` 且没有 `image`

- [ ] **Step 3: 更新生成项目模板**

为新项目模板：

- 删除 `.env` include。
- 自动生成帮助。
- 把 `run` 改为 `dev`；生成项目没有 Compose，`dev` 直接 `APP_ENV=$(ENV) go run ./cmd/server`。
- 增加 `image`，构建 `$(IMAGE_REPO):$(VERSION)` 并标记 `$(IMAGE_REPO):latest`。
- 不增加 `up`、`down`、`up-observability`。

- [ ] **Step 4: 更新重写说明**

把 `tools/generator/support/rewrite.go` 中关于旧 `DOCKER_IMAGE = jimu:latest` 的注释同步为新的 `IMAGE_REPO ?= jimu` 语义；不要改变框架镜像命名策略。

- [ ] **Step 5: 运行生成器聚焦测试**

Run: `go test ./tools/generator -run 'TestGeneratedMakefile|TestBuildFilesRenderForEveryCapabilitySelection' -count=1`
Expected: PASS

Run: `go test ./tools/generator/support -count=1`
Expected: PASS

- [ ] **Step 6: 停止点**

汇报修改文件与测试结果；不要提交。

---

### Task 6: 清理文档、CI 与旧配置入口并完成端到端验证

**Files:**
- Delete: `.env.example`
- Modify: `README.md`
- Modify: `configs/app.prod.yaml`
- Modify: `docs/CONTRIBUTING.md`
- Modify: `docs/design/configuration-and-data-lifecycle.md`
- Modify: `docs/releases/v0.4.0.md`
- Modify: `.github/workflows/ci.yml`
- Modify: references in `.github/workflows/ci.yml` to `make secrets`
- Modify: other references to deleted targets in scripts/docs

**Interfaces:**
- Consumes: Task 2 的初始化脚本、Task 3 的 Compose 契约、Task 4 的 Makefile 入口
- Produces: 仓库文档与 CI 使用同一套新入口；仓库中不再有 `.env.example`

- [ ] **Step 1: 更新 CI 失败门禁**

在 `.github/workflows/ci.yml`：

- path filter 删除 `\.env\.example$`。
- `docker` job 的 Compose 校验改为 `docker compose --env-file /dev/null config --quiet`。
- “Check secrets files exist”改为执行 `bash scripts/init-secrets.sh` 或删除；不得再提示 `make secrets`。

- [ ] **Step 2: 更新 README 快速开始**

将快速开始改为：

```bash
make init       # 仅首次需要，可检查生成的 admin 密码
make up         # 构建 jimu:dev 并启动核心栈
```

说明业务配置只看 `configs/app.yaml`，`secrets/` 自动生成且不可手动编辑；旧 `.env` 不再读取。

- [ ] **Step 3: 更新命令表与可观测性文档**

README 命令表删除 `run`、`docker-build`、`docker-run`、`compose-*`，增加 `up`、`up-observability`、`down`、`status`、`logs`、`dev`、`image`、`adminer`。OpenObserve 文档改为 `make up-observability`，删除 `OTEL_ENABLED` 与默认密码 Admin@12345 说明。

- [ ] **Step 4: 更新贡献指南与设计文档**

`docs/CONTRIBUTING.md` 的开发环境改为 `make dev` 或 `make up`；删除 `COMPOSE_ENV=.env.example`。

`docs/design/configuration-and-data-lifecycle.md` 增加四层配置边界、secret 生成/迁移规则和 `make up` 的 `jimu-init` 流程。

把 `configs/app.prod.yaml` 收缩为只覆盖生产差异，保留生产模式、CORS/安全头、DB/Redis 池参数、登录限流、passkey、日志、OTEL 和错误上报等实际不同项；删除与 `configs/app.yaml` 完全相同的键。

- [ ] **Step 5: 更新 release note**

在 `docs/releases/v0.4.0.md` 的“变更/移除/说明”中记录：

- 移除 `.env.example`。
- 旧 Makefile target 直接替换。
- `secrets/` 自动生成和旧 `.env` 一次性迁移。
- `make image` 同时生成版本标签与 `latest`。
- AES 密钥长度修正。

- [ ] **Step 6: 清理旧引用**

Run:

```bash
rg -n '\.env\.example|compose-up|compose-down|compose-restart|compose-logs|compose-migrate|compose-seed|docker-build|docker-run|docker-stop|docker-logs|make secrets' \
  README.md docs/CONTRIBUTING.md docs/design Makefile scripts .github
```

Expected: 只允许历史 release note、迁移说明和 design/plan 文档中出现旧名称；运行入口不得残留。

- [ ] **Step 7: 格式化与配置验证**

Run:

```bash
bash -n scripts/init-secrets.sh scripts/test_init_secrets.sh scripts/dev.sh \
  scripts/observability.sh scripts/test_compose_config.sh \
  scripts/test_runtime_security.sh scripts/smoke_api_contract.sh
APP_ENV=dev go run ./cmd/cli config check
make image VERSION=v0.4.0-test
docker image inspect jimu:v0.4.0-test >/dev/null
docker image inspect jimu:latest >/dev/null
```

Expected: 全部通过。

- [ ] **Step 8: 运行隔离与项目门禁**

Run:

```bash
make compose-check
make check-capabilities
make profiles-check
make test
make test-race
```

Expected: 全部通过；若本机环境缺少 Docker 或宿主资源，记录未执行项与原因。

- [ ] **Step 9: 最终差异检查**

Run:

```bash
git diff --check
git status --short
```

Expected: 无空白错误；只包含本任务相关文件，不包含用户原有无关改动。

- [ ] **Step 10: 停止点**

汇报变更、验证结果和剩余风险；未获明确指令不得提交或推送。
