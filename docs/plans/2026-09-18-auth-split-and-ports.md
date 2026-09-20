# 能力可插拔 P1.4：kernel/auth 拆分与端口 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 拆分 `internal/kernel/auth` 三块（auth 机制 / access RBAC / apikey），建立 `contract` 端口消除两处反向依赖，并完成 §3.5.2 的中间件归位。

**Architecture:** 三步走：① 纯搬迁（casbin/roles/permission_middleware → 新包 `internal/kernel/access`；apikey.go/apikey_middleware.go → 新包 `internal/capabilities/apikey`），消费者 import 路径同步；② 端口化：`internal/contract/ports.go` 新增 `APIKeyVerifier`/`UploadScanner` 端口，`kernel/grpc/userinfo_service.go` 改吃端口并由装配注入，消除 kernel→capability 反向依赖；③ 中间件归位：`kernel/http/middleware/ratelimit_dimension.go` 随 apikey 迁到 `capabilities/apikey/middleware`。`kernel/auth` 保留 JWT/session/limiter/lockout/AuthMiddleware（认证机制，被 kernel/http/middleware 与多能力引用）。

**Tech Stack:** Go 1.26 · Casbin v3 · GORM · gin

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md`（§3.5.2 中间件归属、§3.5.3/§3.6 混装包、§3.7 端口、§10 P1.4）

## Global Constraints

- **零语义变化**：HTTP 路由/响应/配置键/schema/`capabilities.enabled` 语义不变；搬迁的代码逻辑不改
- **消除两条 kernel→capability 反向依赖**：`kernel/auth/apikey.go → capabilities/admin/domain`（随 apikey.go 迁出，该引用本就指错方向）、`kernel/grpc/userinfo_service.go → capabilities/user/domain`（改 contract 端口）。`kernel/db/seed.go → capabilities/{role,tenant,user}` 留给 P1.5（种子下沉）
- **保留 `APIKeyAuthMiddleware`**（用户裁定：`machine` 形态 §3.4 需要它，虽然当前生产调用方为 0）
- **不新增跨能力 import**：`capabilities/apikey` 不得 import 其他能力的内部包
- 全绿：`gofmt -l .`、`go build ./...`、`go vet ./...`、`golangci-lint run ./...`、`make check-log-usage`、`go test ./... -count=1`、`make bench-ci`、`make release-check COMPOSE_ENV=.env.example`
- 提交信息全英文（Conventional Commits）；三任务各一提交；提交需用户明确指令
- 分支：从 `release/v0.3.0` 切 `feature/auth-split-and-ports`，PR 目标 `release/v0.3.0`

## 侦察结论（codebase-memory 图谱，2026-09-18 索引 6769 节点/34421 边）

| 符号组 | 文件 | 非测试消费者 |
|---|---|---|
| auth 机制 | jwt.go session.go middleware.go limiter.go lockout.go | capabilities/{auth,oauth,ws,admin}、kernel/http/middleware、app、cmd/server |
| access（RBAC） | casbin.go roles.go permission_middleware.go | capabilities/auth（`NewPathEnforcer`/`DBAuthorizationStore`/`Policy`）、kernel/db/seed.go（`NewEnforcer`，P1.5 处理）、e2e |
| apikey | apikey.go apikey_middleware.go | app（`NewAPIKeyVerifier`/`NewDBAPIKeyStore`）、kernel/http/middleware（`APIKeyFromContext`/`APIKeyRateLimitMiddleware`）、cmd/server；**`APIKeyAuthMiddleware` 生产调用方 0**（仅包内测试，保留备用） |
| 反向依赖 | kernel/auth/apikey.go→capabilities/admin/domain；kernel/grpc/userinfo_service.go→capabilities/user/domain | 随迁移/端口消除 |

**中间件归位（§3.5.2）实测**：`kernel/http/middleware/ratelimit_dimension.go` 含 `UserRateLimit`/`TenantRateLimit`/`APIKeyRateLimit` 三个维度且 import `kernel/auth`+`kernel/tenant` —— 随 apikey 迁入 `capabilities/apikey/middleware/`（`TenantRateLimitMiddleware` 依赖 kernel/tenant 为机制引用，合法；`UserRateLimitMiddleware` 消费方为 auth 能力，跨能力调用留待端口，本阶段只迁文件不改调用语义）。

---

### Task 1: access 与 apikey 搬迁（含消费者 import 同步）

**Files:**
- Move: `internal/kernel/auth/{casbin.go,roles.go,permission_middleware.go}` (+`roles_test.go`、`permission_middleware_test.go`、`casbin_test.go` 如有) → `internal/kernel/access/`
- Move: `internal/kernel/auth/{apikey.go,apikey_middleware.go}` (+`apikey_db_test.go`、`apikey_middleware_test.go`、`apikey_test.go` 如有) → `internal/capabilities/apikey/`
- Move: `internal/kernel/http/middleware/ratelimit_dimension.go`(+test) → `internal/capabilities/apikey/middleware/`
- Modify: 全部引用方（`platformauth`/无别名 import 同步）、`internal/kernel/auth` 包内交叉引用

**Interfaces:**
- Produces:
  - `internal/kernel/access`：`NewEnforcer`/`NewPathEnforcer`/`PermissionMiddleware`/`AuthorizationStore`/`NewDBAuthorizationStore`/`Policy`
  - `internal/capabilities/apikey`：`APIKey`/`APIKeyStore`/`APIKeyVerifier`/`NewAPIKeyVerifier`/`HashKey`/`APIKeyAuthMiddleware`/`RequireScope`/`APIKeyFromContext`/`ContextWithAPIKey`/`middleware.UserRateLimitMiddleware` 等
  - `kernel/auth` 不再含 casbin/roles/apikey 符号

- [ ] **Step 1: 搬迁 access 三文件**

```bash
mkdir -p internal/kernel/access
git mv internal/kernel/auth/casbin.go internal/kernel/access/casbin.go
git mv internal/kernel/auth/roles.go internal/kernel/access/roles.go
git mv internal/kernel/auth/roles_test.go internal/kernel/access/roles_test.go 2>/dev/null || true
git mv internal/kernel/auth/permission_middleware.go internal/kernel/access/permission_middleware.go
ls internal/kernel/auth/*_test.go   # 确认 access 相关测试名后逐个 git mv
```

- [ ] **Step 2: 搬迁 apikey 两文件 + 维度限流中间件**

```bash
mkdir -p internal/capabilities/apikey/middleware
git mv internal/kernel/auth/apikey.go internal/capabilities/apikey/apikey.go
git mv internal/kernel/auth/apikey_middleware.go internal/capabilities/apikey/apikey_middleware.go
git mv internal/kernel/auth/apikey_db_test.go internal/capabilities/apikey/apikey_db_test.go 2>/dev/null || true
git mv internal/kernel/auth/apikey_middleware_test.go internal/capabilities/apikey/apikey_middleware_test.go 2>/dev/null || true
git mv internal/kernel/http/middleware/ratelimit_dimension.go internal/capabilities/apikey/middleware/ratelimit_dimension.go
git mv internal/kernel/http/middleware/ratelimit_dimension_test.go internal/capabilities/apikey/middleware/ratelimit_dimension_test.go 2>/dev/null || true
```

- [ ] **Step 3: 修包名、可见性与引用方**

- 两个新包包名 = 目录名（`access`、`apikey`、`middleware`）。
- 按编译错误逐个修引用方：`platformauth.NewPathEnforcer` → `access.NewPathEnforcer`（capabilities/auth）、`platformauth.NewDBAPIKeyStore` → `apikey.NewDBAPIKeyStore`（app/container）、`middleware.APIKeyRateLimitMiddleware` 引用方改 `apikeymw.` 等。**注意**：`ratelimit_dimension.go` 内部 `auth.APIKeyFromContext` 要改为 `apikey.APIKeyFromContext`（其消费的 apikey 包就在同一能力的父目录，直接引用合法）。
- `kernel/auth` 包内若有对已迁符号的内部引用（如 middleware.go 引用 roles），改为 import `kernel/access`。
- 全仓确认：`grep -rn 'platformauth\.\(APIKey\|NewAPIKeyVerifier\|NewPathEnforcer\|NewDBAuthorizationStore\|PermissionMiddleware\)' --include='*.go' .` 为空。

- [ ] **Step 4: 验证**

```bash
gofmt -l . && go build ./... && go vet ./... && golangci-lint run ./...
go test ./... -count=1 2>&1 | tail -3
grep -rn 'casbin\|roles\|apikey' internal/kernel/auth/ --include='*.go' | grep -v 'middleware.go' ; echo "exit=$?"
```

Expected: 全绿；`kernel/auth` 下除 `middleware.go`（AuthMiddleware）外无已迁符号残留。

- [ ] **Step 5: 提交**（用户授权后执行）

```bash
git add -A && git commit -m "refactor(auth): split access and apikey out of the kernel auth package"
```

---

### Task 2: contract 端口与反向依赖消除

**Files:**
- Move: `internal/kernel/grpc/userinfo_service.go` 的依赖改端口（文件不动）
- Move: `internal/capabilities/admin/domain/apikey.go` → `internal/capabilities/apikey/domain/`（模型搬迁）
- Modify: `internal/contract/ports.go`（新增端口）、`internal/kernel/auth/apikey_context.go`（接收 context 助手回迁）
- Modify: `internal/capabilities/admin/{module.go,infrastructure/apikey_repository.go}`、`internal/app/container.go`、`cmd/server/main.go`

**Interfaces:**
- Produces:
  - `contract.UserinfoSource`（`kernel/grpc/userinfo_service.go` 需要的 user 数据：由 user 能力提供实现，替代其对 `capabilities/user/domain` 的直接引用）
  - `container.APIKeyVerifier` 已经持有 `*apikey.APIKeyVerifier`（Task 1 迁移后类型路径变化，装配同步）

- [ ] **Step 1: 盘点 `userinfo_service.go` 对 user/domain 的真实依赖**

```bash
grep -n 'userdomain\.' internal/kernel/grpc/userinfo_service.go
```

按其用到的字段/方法在 `internal/contract/ports.go` 定义最小端口（例如 `UserinfoSource.GetByID(ctx, id) (*contract.Userinfo, error)`，`contract.Userinfo` 是端口自有视图结构，不含 gorm 标签），由 `internal/capabilities/user/` 提供实现并在 `main.go` 装配期注入 `kernel/grpc`。

- [ ] **Step 2: 处理 Task 1 遗留的两条边（评审发现，控制者裁定）**

**2-1 `kernel/http/middleware/ratelimit_user.go → capabilities/apikey`**（评审 Important #1）：`APIKeyFromContext`/`ContextWithAPIKey` 是**纯机制** context 助手，应留在 kernel。把这两个函数从 `capabilities/apikey/apikey.go` 移回 `internal/kernel/auth/apikey_context.go`（保留在 kernel/auth 的机制块内），`capabilities/apikey` 引用之；`kernel/http/middleware` 的 import 改回 `kernel/auth`。

**2-2 `capabilities/apikey → capabilities/admin/domain`**（评审 Important #2，Task 1 原声称"已消除"实为"随迁"）：按设计 §5.2，`api_keys` 表终局归属 apikey 能力。把 `APIKey` GORM 模型（`internal/capabilities/admin/domain/apikey.go`）整体搬到 `internal/capabilities/apikey/domain/apikey.go`（包名 `domain`），`admin` 的 `apikey_repository.go` 与引用方改引新路径，`adminapi` import 随之消失。**注意**：admin 的 apikeys CRUD API 与测试同步改引用；这是模型搬迁不是端口，与 P1.7 拆散 admin 的方向一致。

- [ ] **Step 2b: 端口化 userinfo**

1. `internal/contract/ports.go` 增加端口与视图结构。
2. `internal/kernel/grpc/userinfo_service.go`：删除 `userdomain` import，改用端口；构造函数签名改为接受端口。
3. `internal/capabilities/user/` 新增端口实现（薄适配：调用现有 repository/service）。
4. `internal/app/container.go`：构造并暴露该实现；`cmd/server/main.go` 注入 grpc server。

- [ ] **Step 3: 验证**

```bash
gofmt -l . && go build ./... && go vet ./...
grep -rn 'capabilities/' --include='*.go' internal/kernel/ ; echo "exit=$?"
go test ./... -count=1 2>&1 | tail -3
```

Expected: 全绿；`internal/kernel/` 对 `capabilities/` 的 import 仅剩 `kernel/db/seed.go`（P1.5 处理）与 `kernel/db/seed_test.go` 同理；`capabilities/apikey → capabilities/admin` 必须为零；`kernel/auth` 新增 `apikey_context.go`（纯机制）。

> **计划修订记录（Task 1 评审发现）**：初版声称 Task 1"已消除 `kernel/auth/apikey.go→admin/domain` 反向依赖"——实际是**随文件迁走**（`capabilities/apikey/apikey.go:12` 仍引 `adminapi.APIKey`），且 Task 1 还产生了第三条 kernel→capability 边（`ratelimit_user.go → capabilities/apikey`）。两条边按上面 Step 2 处理；原"Task 2 只处理 grpc"的范围声明作废。

- [ ] **Step 4: 提交**（用户授权后执行）

```bash
git add -A && git commit -m "refactor(grpc): consume userinfo through a contract port"
```

---

### Task 3: 中间件归位收尾 + 全量回归

**Files:**
- Modify: `README.md`（目录树与 auth 相关归属描述）、`AGENTS.md`（如 kernel/auth 路径引用）、`docs/design/2026-09-18-capability-plugins-design.md`（§3.5.3/§3.6 表格行更新）
- 验证为主

- [ ] **Step 1: 文档同步**

- README 项目结构树：`kernel/` 下 auth 描述更新（“JWT + Session + 限流 + 登录失败锁定（RBAC 在 kernel/access，API Key 在 capabilities/apikey）”）、`capabilities/` 增 `apikey/`、`kernel/` 增 `access/`。
- AGENTS.md：若引用 `kernel/auth` 的具体路径则同步（grep 确认）。
- 设计文档：§3.5.3 表 `platform/auth` 行标注拆分结果；§3.5.2 表格行标注完成。

- [ ] **Step 2: 全量回归**

```bash
gofmt -l . && go build ./... && go vet ./... && golangci-lint run ./...
make check-log-usage && go test ./... -count=1 && make bench-ci && make release-check COMPOSE_ENV=.env.example
```

Expected: `All checks passed`。

- [ ] **Step 3: 提交**（用户授权后执行）

```bash
git add -A && git commit -m "docs(auth): record the auth split and middleware placement"
```

---

## Self-Review

**1. Spec 覆盖**：§3.5.3 `platform/auth` 三块拆分（auth 机制留 kernel、access→kernel/access、apikey→capabilities/apikey）；§3.5.2 维度限流随 apikey、管理端准入随 console（console 独立能力在 P1.7，`admin_auth.go` 留 kernel/http/middleware 并在 Task 3 报告）；§3.7 端口；§10 P1.4 的反向依赖消除（2 处，seed 留 P1.5）。`APIKeyAuthMiddleware` 保留（用户裁定）。

**2. 占位符扫描**：Task 2 Step 1 的端口字段按 `userinfo_service.go` 实际依赖盘点后填写（命令已给，结构已定）；其余无 TBD。

**3. 类型一致性**：`access`/`apikey`/`middleware` 包名与目录一致；端口视图结构体不带 gorm 标签；消费者限定符改名逐文件处理。

**4. 风险**：① `kernel/auth` 包内 33 个引用文件中 auth 机制与 access/apikey 符号可能同文件混用（如 auth interfaces 同时用 JWT 与 NewPathEnforcer）→ 拆分后该文件需同时 import 两个包，属预期；② `ratelimit_dimension.go` 迁移后其包名 `middleware` 与 `kernel/http/middleware` 同名不同包，引用方限定符需精确；③ Casbin enforcer 的初始化顺序（kernel/db/seed.go 仍会调用 `access.NewEnforcer`，跨包同层合法）；④ swagger 注解的路径不涉及包名，不受影响。
