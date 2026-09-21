# 能力可插拔 P1.7：admin 拆散 + user 双写合并 + access 合并 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 拆掉 `admin`（5175 行含测试 / 2430 行非测试）这个「路由命名空间」能力，把 7 类用例归还各能力，剩余平台级视图落成新能力 `console`；同时把 `/api/v1/admin/users` 并入 `user` 能力，并按设计 §5.4 把 `role` + `permission` 合并为 `access`。对外 URL/响应/schema 不变。

**Architecture:** `/api/v1/admin/*` 前缀保留，但按用例归属拆分到各能力的 `RegisterHTTP`（各自 `r.Group("/api/v1/admin")` + 管理端准入中间件）。管理端准入（`AdminAuth` + `AdminIPAllowlist`）按设计 §3.5.2 随新能力 `console`，经 `contract.AdminGuard` 端口提供给其他能力。`user` 能力新增管理面用例并复用同一 repository/quota/缓存不变量，消除与 `admin` 的重复实现。`access` = 原 `role` + `permission`（含 `user_roles` 分配）合并。

**Tech Stack:** Go 1.26 · gin · gorm · goose v3.27.3 · testify · miniredis · sqlite（e2e）

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md` §5.2（admin 拆散表）、§5.3（user 双写合并）、§5.4（access 定义）、§3.5.2（AdminAuth/AdminIPAllowlist 随 console）、§7（表归属）、§10（P1.7 行）

## Global Constraints

- **禁止自动提交**：commit 步骤仅在用户明确说「提交」后执行（AGENTS.md 最高优先级）
- **零语义变化**：HTTP 路由与路径、响应 JSON、状态码、错误码、配置键、`capabilities.enabled` 语义全部不变；只搬代码与改归属
- 能力清单只维护在 `internal/capabilities/catalog`；新增/删除能力同步 `cmd/server/main.go` 装配与 `wiredCapabilities`
- **能力之间只经 `contract` 端口调用**，禁止 import 其他能力的内部包；`kernel/` 不得 import `capabilities/`
- 全绿门禁：`gofmt -l .`、`go build ./...`、`go vet ./...`、`golangci-lint run ./...`、`make check-log-usage`、`go test ./... -count=1`、`make bench-ci`、`make release-check COMPOSE_ENV=.env.example`
- 分支：从 `release/v0.3.0` 切 `feature/admin-split-user-merge`，PR 目标 `release/v0.3.0`
- 迁移文件沿用原全局编号（role 001/008、permission 001、queue 001/007 等）；能力内编号不变，`access` 承接 role/permission 的既有编号

## 设计裁定（执行前必读）

1. **去向映射**（§5.2 落实，含文件名与行数）：

   | 现文件 | 去向 | 处理 |
   |---|---|---|
   | `interfaces/users.go` + `application/users_service.go` | `user` | 管理面用例并入 `user/application`；`AssignRoles` 归 `access`（user_roles 所有者） |
   | `interfaces/jobs.go` + `application/tasks_service.go` + `infrastructure/{job,dead_letter,job_history}_repository.go` | `queue` | 能力名保留 `queue`（P1.5 已裁定 jobs/job_history/dead_letters/scheduled_jobs 表归 queue，不新造 `jobs` 能力） |
   | `interfaces/apikeys.go` + `application/apikeys_service.go` + `infrastructure/apikey_repository.go` | `apikey` | API Key 管理面 |
   | `interfaces/import.go` + `application/import_service.go` + `domain/import_job.go` + `infrastructure/import_job_repository.go` | `dataops` | 导入任务与模板 |
   | `interfaces/audit.go` | `audit` | 复用 audit 仓储读 audit_logs |
   | `interfaces/features.go` | `feature` | 能力名保留 `feature`（目录已存在，不新造 `featureflag`） |
   | `interfaces/config.go` + `application/config_service.go` | `console`（新） | 配置热更新视图 |
   | `interfaces/monitoring.go` + `application/monitoring_service.go` | `console` | 平台级监控视图 |
   | `interfaces/ws.go` | `console` | 经 ws hub 端口 |
   | `interfaces/ratelimit.go` | `console` | 限流状态可视化 |
   | `interfaces/middleware.go`（AdminAuthMiddleware 旧实现，已废弃） | `console` | 统一改用 kernel `middleware.AdminAuth()`（见裁定 #3） |
   | `interfaces/handler.go`（error-codes） | `console` | 错误码文档端点 |
   | `application/service.go`（version/env/startTime） | `console` | 只被 error-codes/monitoring 使用 |

   `admin/` 目录最终删除。

2. **`console` 能力**：新增 `internal/capabilities/console/`，`Descriptor{Name:"console", Requires:["auth"], Mount:MountSelfManaged}`（自管 `/api/v1/admin` 组并套准入中间件）。它不拥有任何表、无迁移。权限点承接原 admin 的 4 条 `/api/v1/admin/*` 通配（GET/POST/PUT/DELETE）。

   > **权限点归属注意**：`/api/v1/admin/*` 通配权限点继续由 `console` 声明（保持拆分前语义：管理端整体由该通配覆盖）。各能力拆分后的 `/api/v1/admin/...` 路由仍在同一 Casbin 策略覆盖下，故权限点无需按能力再拆。

3. **管理端准入中间件**：`kernel/http/middleware/admin_auth.go` 的 `AdminAuth()` 是内核机制（读 `c.Get("roles")`），**不搬迁**；`console` 在 `RegisterHTTP` 中引用它。IP 白名单用内核 `middleware.IPAllowlist(cfg.Security.AdminIPAllowlist)`（现状即如此），也不搬迁。设计 §3.5.2 的「随 console」在本实现中指**挂载与生命周期由 console 拥有**，机制仍在 kernel（与 `auth.AuthMiddleware` 同模式）。

4. **user 双写合并**（§5.3）：
   - 管理面用例（List/Get/Create/Update/Disable）并入 `user/application`，与自助面**共享同一 repository 与 quota 校验**；
   - `AssignRoles`（user_roles 写）归 `access` 能力（表所有者），经 `contract.UserRoleAssigner` 端口供 user 管理面 handler 调用；
   - 公开 URL 不变：`/api/v1/admin/users*` 由 `user` 能力注册（`r.Group("/api/v1/admin")` + 准入中间件），`/api/v1/users*` 由 `user` 能力注册（现状）。

5. **access 能力**（§5.4）：`role` + `permission` 合并为 `internal/capabilities/access/`：
   - 表：`roles`、`role_permissions`、`permissions`、`user_roles`（迁移 role 001/008 + permission 001 移入 `access/migrations`，文件名与编号不变）
   - 路由：`/api/v1/roles*`、`/api/v1/permissions*`（不变）
   - `user_roles` 分配：`POST /api/v1/admin/users/:id/roles`（原 admin，归 access 提供，或由 user 经端口调用——**裁定**：该端点由 `access` 直接注册，因其写的是 user_roles）
   - `Requires`: `["user"]`（角色/权限归属用户所属租户）
   - `role`/`permission` 两个 catalog 条目与目录删除

6. **装配顺序**（catalog 拓扑序）：
   ```text
   user, access, tenant, mfa, auth, passkey, audit, console, oauth,
   apikey, queue, outbox, dataops, search, captcha, breach, feature
   ```
   - `access` Requires `["user"]`
   - `console` Requires `["auth"]`（准入中间件依赖 AuthMiddleware 注入的 `user_id`/`roles`，而 roles 由 access 的 AuthorizationMiddleware 注入 → 实际 Requires 需含 `access`）

   **修正**：`console` Requires `["auth", "access"]`（AdminAuth 读 `roles`，roles 由 access 的受保护中间件注入）。

7. **不动项**：`kernel/auth`、`kernel/access`（RBAC 机制）、`kernel/http/middleware/admin_auth.go`、`IPAllowlist`、audit 的 middleware 与 worker、`/api/v1/audits*` 路由。`admin` 的 `module_route_test.go` 路由清单断言改造为覆盖新归属。

---

### Task 1: `console` 能力（平台级视图 + 管理端准入）

**Files:**
- Create: `internal/capabilities/console/{module.go,application/{config_service.go,monitoring_service.go},interfaces/{handler.go,config.go,monitoring.go,ratelimit.go,ws.go}}`（+ 对应 `*_test.go`）
- Move: `internal/capabilities/admin/{application/config_service.go,application/monitoring_service.go,interfaces/config.go,interfaces/monitoring.go,interfaces/ratelimit.go,interfaces/ws.go,interfaces/handler.go,application/service.go}` → console 对应层

**Interfaces:**
- Produces: `console.Module`、`console.Descriptor`（Name=console, Requires=["auth","access"], Mount=MountSelfManaged, Permissions=4 条 `/api/v1/admin/*`）
- Consumes: `kernel/http/middleware.AdminAuth`、`IPAllowlist`、`capabilities/ws`、`contract.EventBus`

- [ ] **Step 1: 建 console 包骨架并搬文件**

`git mv` 上表 8 个文件到 `internal/capabilities/console/` 对应层，改 package 名与 import 路径（`admin/application` → `console/application`）。

- [ ] **Step 2: console/module.go**

```go
var Descriptor = contract.Descriptor{
    Name:     "console",
    Requires: []string{"auth", "access"},
    Mount:    contract.MountSelfManaged,
    Permissions: []contract.Permission{
        {Name: "管理后台读取", Resource: "/api/v1/admin/*", Action: "GET"},
        {Name: "管理后台写入", Resource: "/api/v1/admin/*", Action: "POST"},
        {Name: "管理后台修改", Resource: "/api/v1/admin/*", Action: "PUT"},
        {Name: "管理后台删除", Resource: "/api/v1/admin/*", Action: "DELETE"},
    },
}

func (m *Module) RegisterHTTP(r contract.Router) {
    admin := r.Group("/api/v1/admin")
    if m.ipAllowlist != nil { admin.Use(m.ipAllowlist) }
    admin.Use(middleware.AdminAuth())
    admin.GET("/error-codes", interfaces.NewHandler(m.service).GetErrorCodes)
    // monitoring/ratelimit/config/ws 端点（原样搬迁）
}
```

- [ ] **Step 3: 验证**

```bash
gofmt -l . && go build ./... && go vet ./internal/capabilities/console/...
go test ./internal/capabilities/console/... -count=1
```

- [ ] **Step 4: 提交**（用户授权后）

`git add -A && git commit -m "feat(console): add the platform console capability"`

---

### Task 2: `access` 能力（role + permission 合并）

**Files:**
- Create: `internal/capabilities/access/{module.go}`；`git mv internal/capabilities/role/* internal/capabilities/access/`（application/domain/infrastructure/interfaces/migrations），`git mv internal/capabilities/permission/{application,infrastructure,interfaces,migrations}/* internal/capabilities/access/*`
- Modify: `internal/capabilities/access/` 内 package 名与 import（`role`/`permission` → `access`）
- Move: `internal/capabilities/admin/application/users_service.go` 的 `AssignRoles` → `access/application/user_roles.go`
- Modify: `internal/capabilities/catalog/catalog.go`（删 role/permission，加 access）

**Interfaces:**
- Produces: `access.Module`（`Descriptor{Name:"access", Requires:["user"], Mount:MountProtected, Migrations:roles+permissions, Permissions:role6+permission5+user_roles1}`）
- Produces: `contract.UserRoleAssigner`（`AssignRoles(ctx, userID uint64, roleNames []string) error`），供 user 管理面调用

- [ ] **Step 1: git mv 两能力文件到 access，统一 package 名**

- [ ] **Step 2: 解决包内命名冲突**

role 与 permission 各有 `application/dto.go`/`errors.go`/`service.go`、`interfaces/handler.go`/`router.go`、`infrastructure/mysql_repository.go` —— 重命名为 `role_*.go` / `permission_*.go`，包内类型名加前缀避免冲突（`RoleService`/`PermissionService` 已不同名，检查 `dto`/`errors` 内类型）。

- [ ] **Step 3: user_roles 分配迁入（唯一所有者）**

`POST /api/v1/admin/users/:id/roles` 由 `access` 注册：

```go
admin := r.Group("/api/v1/admin")
admin.Use(middleware.AdminAuth())
admin.POST("/users/:id/roles", handler.AssignRole)
```

- [ ] **Step 4: catalog 更新**

```go
user.Descriptor, access.Descriptor, tenantmodule.Descriptor, ...
```

- [ ] **Step 5: 验证**

```bash
gofmt -l . && go build ./... && go test ./internal/capabilities/access/... -count=1
```

- [ ] **Step 6: 提交**（用户授权后）

`git add -A && git commit -m "refactor(access): merge role and permission into the access capability"`

---

### Task 3: user 双写合并（管理面并入 user）

**Files:**
- Move: `internal/capabilities/admin/application/users_service.go` → `internal/capabilities/user/application/admin_service.go`（去掉 AssignRoles，改走端口）
- Move: `internal/capabilities/admin/interfaces/users.go` → `internal/capabilities/user/interfaces/admin_users.go`
- Modify: `internal/capabilities/user/module.go`（注册 `/api/v1/admin/users*`）
- Modify: `internal/capabilities/user/application/service.go`（复用 quota/repo；删除与新管理面重复的逻辑）

**Interfaces:**
- Consumes: `contract.UserRoleAssigner`（access 提供）、`TenantQuota`（tenant 提供）
- Produces: `/api/v1/admin/users*` 6 条路由由 user 能力注册

- [ ] **Step 1: 管理面用例迁入 user，统一 quota 与租户可见性**

自查并统一：原 `admin` 的 `tenantVisible` 与 `user` 的 `tenantAllowed` 语义（`kernel/tenant.Visible` 为唯一实现），删除重复副本。

- [ ] **Step 2: AssignRoles 经端口**

user 的 `AssignRole` handler 调 `m.roles.AssignRoles(ctx, id, req.Roles)`（`m.roles` 为注入的 `contract.UserRoleAssigner`）。

- [ ] **Step 3: user/module.go 注册管理面路由**

```go
func (m *Module) RegisterHTTP(r contract.Router) {
    interfaces.RegisterUserRoutes(r.Group("/api/v1"), m.service, m.rdb)
    admin := r.Group("/api/v1/admin")
    admin.Use(middleware.AdminAuth())
    interfaces.RegisterAdminUserRoutes(admin, m.adminService, m.roles)
}
```

> 若 access 已注册 `POST /admin/users/:id/roles`，user 不得重复注册（gin 重复路由 panic）。**裁定**：`/admin/users/:id/roles` 只由 `access` 注册；user 管理面只注册 list/get/create/update/disable。

- [ ] **Step 4: 验证 + 路由对齐**

新增/改 e2e 路由清单测试，断言 `/api/v1/admin/*` 全量路由与拆分前一致（逐条、无重复）。

- [ ] **Step 5: 提交**（用户授权后）

`git add -A && git commit -m "refactor(user): merge the admin user management into the user capability"`

---

### Task 4: jobs/tasks → queue

**Files:**
- Move: `internal/capabilities/admin/interfaces/{jobs.go,tasks.go}` → `internal/capabilities/queue/interfaces/`
- Move: `internal/capabilities/admin/application/tasks_service.go` → `internal/capabilities/queue/application/`
- Move: `internal/capabilities/admin/infrastructure/{job_repository.go,dead_letter_repository.go,job_history_repository.go}` → `internal/capabilities/queue/infrastructure/`
- Modify: `internal/capabilities/queue/`（新增 `interfaces/`、`application/` 层与 `RegisterHTTP`）

**Interfaces:**
- Produces: queue 注册 `/api/v1/admin/tasks*`（4 条）与 `/api/v1/admin/jobs*`（6 条）
- queue 从「仅迁移」升级为有 Module 实例的能力（`New(db, sched)`）

- [ ] **Step 1: 搬文件并加 Module**

- [ ] **Step 2: queue Descriptor 增 Permissions**（原 admin 通配已覆盖，此处不新增权限点，仅挂路由）

- [ ] **Step 3: 验证 + 提交**（用户授权后）

`git add -A && git commit -m "refactor(queue): own the job and task admin endpoints"`

---

### Task 5: apikeys → apikey

**Files:**
- Move: `internal/capabilities/admin/interfaces/apikeys.go` → `internal/capabilities/apikey/interfaces/`
- Move: `internal/capabilities/admin/application/apikeys_service.go` → `internal/capabilities/apikey/application/`
- Move: `internal/capabilities/admin/infrastructure/apikey_repository.go` → `internal/capabilities/apikey/infrastructure/`
- Modify: `internal/capabilities/apikey/migrations.go`（补 Module/RegisterHTTP）

- [ ] **Step 1: 搬文件，apikey 增加 Module 实例**
- [ ] **Step 2: 注册 `/api/v1/admin/apikeys*`（4 条）**
- [ ] **Step 3: 验证 + 提交**（用户授权后）

`git add -A && git commit -m "refactor(apikey): own the API key admin endpoints"`

---

### Task 6: import → dataops

**Files:**
- Move: `internal/capabilities/admin/interfaces/import.go` → `internal/capabilities/dataops/interfaces/`
- Move: `internal/capabilities/admin/application/import_service.go` → `internal/capabilities/dataops/application/`
- Move: `internal/capabilities/admin/domain/import_job.go` → `internal/capabilities/dataops/domain/`
- Move: `internal/capabilities/admin/infrastructure/import_job_repository.go` → `internal/capabilities/dataops/infrastructure/`
- Modify: `internal/capabilities/dataops/`（补 Module/RegisterHTTP；`bindImportFile` 随迁）

- [ ] **Step 1: 搬文件**
- [ ] **Step 2: 注册 `/api/v1/admin/users/import*`（4 条）**
- [ ] **Step 3: 验证 + 提交**（用户授权后）

`git add -A && git commit -m "refactor(dataops): own the user import endpoints"`

---

### Task 7: audit / feature 收口 + 删除 admin + 文档

**Files:**
- Move: `internal/capabilities/admin/interfaces/audit.go` → `internal/capabilities/audit/interfaces/admin_audit.go`（改走 audit 的 service，不直连 infrastructure）
- Move: `internal/capabilities/admin/interfaces/features.go` → `internal/capabilities/feature/interfaces/`（feature 补 Module）
- Delete: `internal/capabilities/admin/`（含 module.go 与全部残留）
- Modify: `cmd/server/main.go`、`internal/capabilities/catalog/catalog.go`、`internal/capabilities/catalog/catalog_test.go`
- Modify: `README.md`、`AGENTS.md`、`docs/design/…`（§10 P1.7 标注完成）、`docs/releases/v0.3.0.md`

- [ ] **Step 1: audit 管理端列表改走 service 端口**
- [ ] **Step 2: feature 能力加 Module 并注册 `/api/v1/admin/features*`（2 条）**
- [ ] **Step 3: 删除 admin 目录，清理 catalog/main/e2e**

`wiredCapabilities` = `user, access, tenant, mfa, auth, passkey, audit, console, oauth, apikey, queue, captcha, feature`

- [ ] **Step 4: catalog 夹具与权限点对齐**（13→17→P1.7 后总数按实际逐值钉死）
- [ ] **Step 5: 全量回归**

```bash
gofmt -l . && go build ./... && go vet ./... && golangci-lint run ./...
make check-log-usage && go test ./... -count=1 && make bench-ci && make release-check COMPOSE_ENV=.env.example
```

- [ ] **Step 6: 提交**（用户授权后）

`git add -A && git commit -m "refactor(capabilities): remove the admin namespace and document P1.7"`

---

## Self-Review

**1. Spec 覆盖**：§5.2 全部 11 行去向落实（admin 目录删除、console 新增）；§5.3 user 双写合并（管理面并入 user、共享 repo/quota、AssignRoles 归 access）；§5.4 access = role+permission+Casbin+鉴权中间件+user_roles（Casbin 机制已在 kernel/access，本次合并能力层）；§3.5.2 AdminAuth/IPAllowlist 由 console 挂载；§7 表归属不变（迁移编号沿用）。

**2. 占位符扫描**：Task 2 Step 2 的命名冲突需实现时按实际类型名处理（已给策略）；Task 3 Step 3 的重复路由冲突已裁定（`:id/roles` 只由 access 注册）。无 TBD。

**3. 类型一致性**：`contract.UserRoleAssigner.AssignRoles(ctx, userID, roleNames) error` 在 Task 2 定义、Task 3 消费；`console.Descriptor.Requires=["auth","access"]` 与裁定 #6 一致。

**4. 风险**：① 7 个能力同时挂 `/api/v1/admin` 前缀，gin 路由重复注册会 panic —— 必须保证每条路径只注册一次（e2e 路由清单测试钉死）；② `access` 合并时 role/permission 的 `dto.go`/`errors.go`/`service.go` 同名文件需重命名，易漏改 import；③ user 双写合并要确保管理面与自助面的租户可见性/配额语义统一（`kernel/tenant.Visible` 唯一实现），否则引入越权；④ `feature` 与 `queue` 从「仅迁移/库」升级为有实例的能力，`wiredCapabilities` 与 main 装配需同步，漏改会导致启动期 fail-closed 报错。
