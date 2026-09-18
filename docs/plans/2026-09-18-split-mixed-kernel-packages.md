# 能力可插拔 P1-B2：内核混装包拆分 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `internal/kernel/db` 与 `internal/kernel/http` 两个混装包中"属于能力层"的文件拆出去，使 `kernel/db` 只剩连接/迁移/事务机制、`kernel/http` 只剩 server/管理面/内核中间件；**消除两条 kernel→capability 跨层边**。

**Architecture:** 五组文件搬迁（`git mv` + 包名对齐 + 引用方重写）。与 P1-A/P1-B1 不同，本次**不适用重建式验证**（拆包必然改变包结构与符号归属），判据换成：① 拆分后每个新包职责单一；② 全绿；③ 对外行为不变（`capabilities.enabled`、路由、配置键、schema 不变）；④ `kernel/*` 不再 import `capabilities/*`（净减少两条跨层边）。

**Tech Stack:** Go 1.26 · git mv · GORM plugin/callback · promauto · swaggo

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md`（§3.5.3 包归位表、§3.6 混装包、§10 P1）

## Global Constraints

- **零语义变化**：HTTP 路由/响应/配置键/schema/`capabilities.enabled` 语义不变；拆出去的代码逻辑不改，只改位置、包名与必要的可见性
- **本阶段不新增跨能力依赖**：拆分目标是**消除** kernel→capability 边；`capabilities/admin → capabilities/apikey` 之类的新边留给 P1-D
- `kernel/auth` 的拆分**不在本计划**（它的 apikey 部分会产生跨能力 import，与 P1-D 绑定；见「分期说明」）
- 全绿：`gofmt -l .`（无输出）、`go build ./...`、`go vet ./...`、`golangci-lint run ./...`、`make check-log-usage`、`go test ./... -count=1`、`make release-check COMPOSE_ENV=.env.example`
- 提交信息全英文（Conventional Commits）；每任务一提交；提交需用户明确指令
- 分支：从 `release/v0.3.0` 切 `feature/split-mixed-kernel-packages`，PR 目标 `release/v0.3.0`，squash 合并

## 分期说明（为什么本计划不含 `kernel/auth`）

`kernel/auth` 混装三块：auth 机制（jwt/session/limiter/lockout）、access（casbin/roles/permission_middleware）、apikey（apikey/apikey_middleware）。按 §3.5.3，后两块应分别归 `capabilities/access` 与 `capabilities/apikey`。但实测 `capabilities/admin` 现在直接引用 apikey 仓库与 casbin——拆出去之后 admin 必须跨能力 import，正是设计文档禁止的形态；正确解法是 P1-D 的 `contract` 端口（`Authorizer`/`APIKeyAuthenticator`）。因此 `kernel/auth` 的拆分与 P1-D 合并为一个计划（P1-B2b/P1-D），避免制造已知违规中间态。

## 拆分明细（唯一副本）

| 源文件 | 去向 | 消费者（实测） | 拆分后需要做的事 |
|---|---|---|---|
| `kernel/db/breaker.go` + `breaker_test.go` | `kernel/breaker/`（并入**现有**包，包名 `breaker`） | `kernel/db` 的 `Connect*` 内部调用 `attachBreaker` | 符号需导出（如 `AttachDBBreaker`）；`kernel/db` import `kernel/breaker` 并传参（同层引用，方向合法） |
| `kernel/db/encryption.go` + `encryption_test.go` | `capabilities/encryption/` | `app/container.go:131` | `RegisterEncryptionHooks` 改为 `encryption.RegisterHooks`；**消除 kernel→capability 边** |
| `kernel/db/retention.go` + `retention_test.go` + `retention_integration_test.go` + `cleanup.go` + `cleanup_test.go` | `capabilities/retention/` | `app/bootstrap.go:268,284` | 新包名 `retention`；`NewRetentionService`/`NewCleanupService` 等符号随之改名空间 |
| `kernel/http/swagger.go` | `capabilities/apidocs/` | `app/bootstrap.go:157` | 包名 `apidocs`；`RegisterSwagger` 导出不变 |
| `kernel/http/upload_handler.go` + `upload_handler_test.go` + `upload_handler_bench_test.go` + `clamav.go` + `clamav_integration_test.go` | `capabilities/uploadsec/` | `app/container.go:165-170`、`app/bootstrap.go` | 包名 `uploadsec`；`NewClamAVScanner`/`ClamAVConfig`/上传 handler 随之改名空间 |

不动的：`kernel/db/{concurrency,transaction,migrate,mysql,postgres,gorm_logger,snowflake,seed}.go`（seed 的下沉属 P1-B3）、`kernel/http/{server,management}.go`、`kernel/http/middleware/`（中间件归位属 P1-B2b）。

## 风险预置（写计划时已识别）

1. `kernel/db/breaker.go` 的 `isDBTransportError`/`breakerPlugin` 均为包内私有，且被 `kernel/db` 的连接初始化调用 → 移入 `kernel/breaker` 包后需导出，并确认不与现有 `kernel/breaker` 包的符号冲突（实测该包只有 167 行、无同名符号）。
2. `retention.go` 内含 `auditLogRow` 等 7 个 `TableName()` 最小模型 —— 它们引用的是**别的能力拥有的表**（audit_logs/jobs/…），属于"按表名读数据"的保留任务，不是跨能力代码依赖；搬进 `capabilities/retention/` 后应保持只按表名操作。
3. `retention_integration_test.go` 与 `retention_test.go` 依赖 `testutil`，搬迁后路径不变（testutil 未动）。
4. `swagger.go` 有 `_ "jimu/docs/openapi"` 空导入 —— 搬到 `capabilities/apidocs/` 后空导入必须跟随（否则 swagger 文档不注册）。
5. `upload_handler.go` 被 `capabilities/{oauth/provider,notification,tenant,breach,user}` 引用？—— 实测**没有**：这些目录 import 的是 `kernel/http` 的**其他**符号（server/middleware）。拆分后需逐一确认引用方是否需要改 import。

---

### Task 1: `kernel/db` 拆分（breaker / encryption / retention 三组）

**Files:**
- Move: `internal/kernel/db/{breaker.go,breaker_test.go}` → `internal/kernel/breaker/`
- Move: `internal/kernel/db/{encryption.go,encryption_test.go}` → `internal/capabilities/encryption/`
- Move: `internal/kernel/db/{retention.go,retention_test.go,retention_integration_test.go,cleanup.go,cleanup_test.go}` → `internal/capabilities/retention/`
- Modify: `internal/kernel/db/{mysql.go,postgres.go}`（breaker attach 调用点）、`internal/app/container.go`、`internal/app/bootstrap.go`

**Interfaces:**
- Produces:
  - `breaker.AttachDBBreaker(db *gorm.DB, cfg config.BreakerConfig) error`（原 `attachBreaker`，导出）
  - `encryption.RegisterHooks(g *gorm.DB, c *encryption.Cipher)`（原 `db.RegisterEncryptionHooks`；同包后包限定符消失）
  - `retention.NewRetentionService / NewRetentionServiceWithRules / DefaultRetentionRules / NewCleanupService / DefaultCleanupConfig`
  - `kernel/db` 不再含 encryption/retention/breaker 符号

- [ ] **Step 1: 移动文件**

```bash
git mv internal/kernel/db/breaker.go internal/kernel/breaker/db_plugin.go
git mv internal/kernel/db/breaker_test.go internal/kernel/breaker/db_plugin_test.go
git mv internal/kernel/db/encryption.go internal/capabilities/encryption/hooks.go
git mv internal/kernel/db/encryption_test.go internal/capabilities/encryption/hooks_test.go
mkdir -p internal/capabilities/retention
git mv internal/kernel/db/retention.go internal/capabilities/retention/retention.go
git mv internal/kernel/db/retention_test.go internal/capabilities/retention/retention_test.go
git mv internal/kernel/db/retention_integration_test.go internal/capabilities/retention/retention_integration_test.go
git mv internal/kernel/db/cleanup.go internal/capabilities/retention/cleanup.go
git mv internal/kernel/db/cleanup_test.go internal/capabilities/retention/cleanup_test.go
```

> `breaker.go` → `db_plugin.go` 避免与 `kernel/breaker` 包内已有文件名歧义；`encryption.go` → `hooks.go` 因目标包已可能有同名概念。

- [ ] **Step 2: 对齐包名与可见性**

- `internal/kernel/breaker/db_plugin.go`：包名已是 `breaker`；`attachBreaker` → `AttachDBBreaker` 并导出；确认无与现有包符号冲突。
- `internal/capabilities/encryption/hooks.go`：包名改为 `encryption`（并入现有包）；`RegisterEncryptionHooks` → `RegisterHooks`；检查与该包现有符号（`Cipher` 等）无冲突。
- `internal/capabilities/retention/*.go`：包名 `retention`。

- [ ] **Step 3: 修引用方**

- `internal/kernel/db/{mysql.go,postgres.go}`：`attachBreaker(...)` → `breaker.AttachDBBreaker(...)`，加 import。
- `internal/app/container.go:131`：`db.RegisterEncryptionHooks(dbConn, cipher)` → `encryption.RegisterHooks(dbConn, cipher)`，import 从 `kernel/db` 改为 `capabilities/encryption`（若 container 已 import encryption 包则只需改调用）。
- `internal/app/bootstrap.go:268,284`：`db.NewCleanupService` → `retention.NewCleanupService`、`db.NewRetentionService` → `retention.NewRetentionService`、`db.DefaultCleanupConfig` → `retention.DefaultCleanupConfig`。
- 全仓 `grep -rn 'RegisterEncryptionHooks\|NewRetentionService\|NewCleanupService\|attachBreaker'` 确认无漏网。

- [ ] **Step 4: 验证**

```bash
gofmt -l . && go build ./... && go vet ./...
go test ./internal/kernel/... ./internal/capabilities/encryption/ ./internal/capabilities/retention/ ./internal/app/ -count=1
go test ./... -count=1 2>&1 | tail -3
# 跨层边检查：kernel 下不得再 import capabilities
grep -rn 'jimu/internal/capabilities/' --include='*.go' internal/kernel/ ; echo "exit=$?"
```

Expected: 全绿；最后一条**无输出**（`kernel/db → capabilities/encryption` 这条边消除；`kernel/http → capabilities/storage` 属 Task 2）。

- [ ] **Step 5: 提交**（用户授权后执行）

```bash
git add -A && git commit -m "refactor(kernel): split db breaker, encryption hooks and retention into their owners"
```

---

### Task 1（续）：`kernel/http` 拆分 —— 与上文同属一个任务，一次评审覆盖

> **合并说明（用户裁定）**：原计划把 `kernel/db` 与 `kernel/http` 拆成两个任务，但两者同属"把混装包里属于能力层的文件拆出去"，合计仅 ~11 个文件、两轮评审流程过重。**实际执行为单个任务**：完成上文 Step 1–5 后继续下列 Step 6–9，最后一次性提交与评审。

- [ ] **Step 6: 移动 `kernel/http` 的两组文件**

```bash
mkdir -p internal/capabilities/apidocs internal/capabilities/uploadsec
git mv internal/kernel/http/swagger.go internal/capabilities/apidocs/swagger.go
git mv internal/kernel/http/upload_handler.go internal/capabilities/uploadsec/upload_handler.go
git mv internal/kernel/http/upload_handler_test.go internal/capabilities/uploadsec/upload_handler_test.go
git mv internal/kernel/http/upload_handler_bench_test.go internal/capabilities/uploadsec/upload_handler_bench_test.go
git mv internal/kernel/http/clamav.go internal/capabilities/uploadsec/clamav.go
git mv internal/kernel/http/clamav_integration_test.go internal/capabilities/uploadsec/clamav_integration_test.go
```

- [ ] **Step 7: 对齐包名与引用方**

- 两个新包的包名分别为 `apidocs`、`uploadsec`（包名 = 目录名，无冲突）。
- `internal/app/bootstrap.go:157`：`platformhttp.RegisterSwagger(...)` → `apidocs.RegisterSwagger(...)`（`platformhttp` 别名现指向 `kernel/http`，其余调用不动）。
- `internal/app/container.go:165-170`：`platformhttp.NewClamAVScanner` → `uploadsec.NewClamAVScanner`。
- `grep -rn 'kernel/http\.' internal/app/ | grep -i 'swagger\|upload\|clamav'` 确认清零。
- 逐一确认原引用 `kernel/http` 的能力包（`oauth/provider`、`notification`、`tenant/interfaces`、`breach`、`user/interfaces`）用的符号不属于 swagger/upload —— 实测如此，若发现例外按同样方式迁移或报告。

- [ ] **Step 8: 跨层边终检 + 全量回归（整个任务的验收）**

```bash
gofmt -l . && go build ./... && go vet ./... && golangci-lint run ./...
grep -rn 'jimu/internal/capabilities/' --include='*.go' internal/kernel/ ; echo "exit=$?"
make check-log-usage
go test ./... -count=1 2>&1 | tail -3
make bench-ci
make release-check COMPOSE_ENV=.env.example
```

Expected: 全绿；跨层边 grep **无输出**（两条边全部消除）；`release-check` 输出 `All checks passed`。

- [ ] **Step 9: 提交**（用户授权后执行）

```bash
git add -A && git commit -m "refactor(kernel): split mixed kernel packages into their capability owners"
```

---

## Self-Review

**1. Spec 覆盖**：§3.5.3 表中 `platform/db` 的三个待拆项（breaker→kernel、encryption→encryption、retention→retention）与 `platform/http` 的两个待拆项（swagger→apidocs、upload/clamav→uploadsec）全部覆盖；§3.6 的 `platform/auth` 拆分显式排除（见「分期说明」，与 P1-D 绑定）；§3.5.2 中间件归位不属本计划（P1-B2b）。§10 P1 的「`platform/` 消失」在 P1-B1 已达成；本计划完成后 `kernel/db`/`kernel/http` 不再是混装包。

**2. 占位符扫描**：无 TBD；映射表、`git mv` 命令、符号改名表齐全。

**3. 类型一致性**：导出符号改名表（`AttachDBBreaker`/`RegisterHooks`）与引用方重写一一对应；新包名 `apidocs`/`uploadsec`/`retention` 与设计文档 §3.4 一致。

**4. 风险**：① `kernel/breaker` 包并入后与现有 `Circuit` 类符号的命名冲突（实测无同名，执行时仍需 go build 兜底）；② `swagger.go` 的 `_ "jimu/docs/openapi"` 空导入必须随迁；③ `retention` 的 7 个 `TableName()` 模型按表名跨能力读数据，设计上接受（§3.6 已说明），拆分后不得引入对其他能力 domain 包的 import；④ e2e 的 PG 迁移测试路径在 `.github/workflows` 指向 `kernel/db`，拆分后仍正确（migrate 留在 kernel/db）。
