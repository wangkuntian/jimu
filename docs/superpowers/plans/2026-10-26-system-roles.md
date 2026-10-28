# 系统角色与租户开通角色 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立四个稳定系统角色，让平台 seed 与租户开通事务自动创建正确角色，并阻止系统角色篡改和最后一个平台超管被移除。

**Architecture:** `contract` 定义跨能力共享的系统角色目录和 provisioning 角色结构；`access` 拥有角色元数据、用户角色和最后超管约束；`tenant` 在租户创建事务中调用 access 的 `ProvisioningRoleStore`；`user` 与 `audit` 按平台/租户范围执行可见性。

**Tech Stack:** Go、Gin、GORM、MySQL/PostgreSQL goose migrations、SQLite 单测、Casbin、swag。

**Spec:** `docs/superpowers/specs/2026-10-26-system-roles-design.md`

## Global Constraints

- 系统角色 code 固定为 `platform_super_admin`、`platform_auditor`、`tenant_admin`、`tenant_member`。
- 自定义角色的 `code` 为空；首版不允许配置自定义 provisioning code。
- 平台系统角色只存在于默认租户；默认租户不创建租户系统角色。
- 非平台身份不得管理默认租户用户，也不得读取或授予平台角色。
- 系统角色禁止通过 RoleService 改名、停用或替换权限。
- 有效超管必须满足用户 `status=1`、`deleted_at IS NULL`、角色 `status=1`、角色未软删除。
- MySQL 与 PostgreSQL 迁移必须同时存在；迁移编号在 access 能力内递增。
- 能力间只经 `internal/contract` 协作，禁止跨能力内部 import。
- 未经用户明确指令不得 commit 或 push；本计划结束在工作树修改与验证。

## Review Focus

- 伪造 role ID 是否能绕过租户边界获得平台角色。
- 两个并发请求是否可能同时移除最后两个平台超管。
- 软删除或停用的 admin 是否被错误计为“有效超管”。
- 系统角色重复 seed 是否精确收敛权限且不会累积旧权限。
- `POST /tenants` 或开通式注册中角色创建失败时，是否完整回滚租户与用户。
- 平台审计员或租户管理员读审计详情时，是否可能跨租户读取。

---

### Task 1: 角色元数据、系统角色目录与迁移

**Files:**
- Create: `internal/contract/systemroles.go`
- Create: `internal/contract/systemroles_test.go`
- Modify: `internal/capabilities/access/domain/role.go`
- Modify: `internal/capabilities/access/application/dto.go`
- Modify: `internal/capabilities/access/application/dto_test.go`
- Create: `internal/capabilities/access/migrations/mysql/009_role_system_metadata.sql`
- Create: `internal/capabilities/access/migrations/postgres/009_role_system_metadata.sql`
- Create: `internal/capabilities/access/system_metadata_migration_test.go`

**Interfaces:**
- Produces: `contract.RoleScope`, `contract.RoleScopePlatform`, `contract.RoleScopeTenant`
- Produces: `contract.RoleCodePlatformSuperAdmin`、`RoleCodePlatformAuditor`、`RoleCodeTenantAdmin`、`RoleCodeTenantMember`
- Produces: `contract.SystemRoleSpec`、`contract.PlatformSystemRoles() []SystemRoleSpec`、`contract.TenantSystemRoles() []SystemRoleSpec`
- Produces: `domain.Role.Code string`、`domain.Role.Scope contract.RoleScope`、`domain.Role.IsSystem bool`
- Produces: `application.RoleResponse.Code/Scope/IsSystem`

- [ ] **Step 1: 写系统角色目录失败测试**

在 `systemroles_test.go` 断言四个 code 恰好各出现一次，两个平台角色 scope 为 `platform`，两个租户角色 scope 为 `tenant`，且 `tenant_admin` 权限集合逐值等于设计文档列出的 `(resource, action)` 集合。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/contract -run TestSystemRoleCatalog -count=1`

Expected: FAIL，提示 `PlatformSystemRoles` 或 `TenantSystemRoles` 未定义。

- [ ] **Step 3: 定义 contract 系统角色目录**

定义字符串常量与 `SystemRoleSpec{Code, Name, Description, Scope, AllPermissions, Permissions}`。平台超管使用 `AllPermissions=true`；平台审计员和租户管理员使用显式权限；租户成员权限为空。

- [ ] **Step 4: 写 RoleResponse 元数据失败测试**

在 `dto_test.go` 构造一个系统角色，断言 JSON 含 `code=tenant_admin`、`scope=tenant`、`is_system=true`；自定义角色断言 code 为空、scope 为 tenant、is_system 为 false。

- [ ] **Step 5: 运行测试确认失败**

Run: `go test ./internal/capabilities/access/application -run TestRoleResponseCarriesSystemMetadata -count=1`

Expected: FAIL，提示 RoleResponse 缺少字段。

- [ ] **Step 6: 扩展 Role 与 RoleResponse**

为 domain/adapter 增加三字段并更新 `ToRoleResponse`。GORM tag 使用 `column:code;size:64;not null;default:''`、`column:scope;size:16;not null;default:'tenant'`、`column:is_system;not null;default:false`。

- [ ] **Step 7: 写迁移内容测试**

在 `system_metadata_migration_test.go` 从 `Descriptor.Migrations` 读取 mysql/postgres 的 `009_role_system_metadata.sql`，断言包含三列、两个索引以及 Down 删除列。

- [ ] **Step 8: 运行测试确认失败**

Run: `go test ./internal/capabilities/access -run TestRoleSystemMetadataMigration -count=1`

Expected: FAIL，提示迁移文件不存在。

- [ ] **Step 9: 写两个方言迁移**

Up 增加 `code`、`scope`、`is_system` 与 `idx_roles_code`、`idx_roles_scope`；Down 先删索引再删列。

- [ ] **Step 10: 运行聚焦测试**

Run: `go test ./internal/contract ./internal/capabilities/access/... -count=1`

Expected: PASS。

---

### Task 2: Seed 平台系统角色

**Files:**
- Modify: `internal/app/seed.go`
- Modify: `internal/app/seed_test.go`

**Interfaces:**
- Consumes: `contract.PlatformSystemRoles()` 与 Role 元数据字段
- Produces: `ensureSystemRole(tx, tenantID, spec) (*domain.Role, error)`
- Produces: `assignRolePermissions(tx, roleID, permissionIDs) error`
- Produces: `ensureAdminRole(tx, adminUserID, roleID) error`

- [ ] **Step 1: 写全新库 seed 失败测试**

用 SQLite 执行迁移所需表结构后 seed，断言默认租户恰好存在两个平台系统角色；超管拥有全部 seeded permissions；审计员只拥有设计文档列出的只读权限；admin 只有一个 `platform_super_admin` 关联；默认租户不存在租户系统角色。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/app -run TestRunSeedCreatesPlatformSystemRoles -count=1`

Expected: FAIL，当前只有按名称创建“超级管理员”。

- [ ] **Step 3: 写无效超管 fail-closed 失败测试**

预置已停用或已软删除的 admin 用户及超管关联，执行 seed 断言返回包含“no active platform superadmin”的错误，且不静默恢复账号。

- [ ] **Step 4: 运行测试确认失败**

Run: `go test ./internal/app -run TestRunSeedRequiresActivePlatformSuperAdmin -count=1`

Expected: FAIL，当前 seed 会成功提交。

- [ ] **Step 5: 实现按 code 幂等 seed**

按 code 查系统角色，缺失时创建。精确替换系统角色权限并确保 admin `user_roles`。开发阶段不处理旧数据 adoption。

- [ ] **Step 6: 实现有效超管校验**

在同一 seed 事务内统计满足用户有效、角色有效和关联存在的 `platform_super_admin` 用户数；为 0 时返回错误。不要自动恢复已停用或软删除账号。

- [ ] **Step 7: 更新 seed 失败路径与 sqlmock 期望**

保留原事务失败覆盖，并更新新增的 `code`、`scope`、`is_system` 列相关 SELECT/INSERT 期望。

- [ ] **Step 8: 运行 seed 测试**

Run: `go test ./internal/app -run TestRunSeed -count=1`

Expected: PASS。

---

### Task 3: 租户开通总是创建 tenant_admin 与 tenant_member

**Files:**
- Modify: `internal/contract/ports.go`
- Modify: `internal/capabilities/access/provisioning_port.go`
- Modify: `internal/capabilities/access/provisioning_port_test.go`（不存在则创建）
- Modify: `internal/capabilities/tenant/application/provisioning.go`
- Modify: `internal/capabilities/tenant/application/service.go`
- Modify: `internal/capabilities/tenant/application/service_test.go`
- Modify: `internal/capabilities/tenant/application/provisioning_test.go`
- Modify: `internal/capabilities/tenant/config.go`
- Modify: `internal/capabilities/tenant/wire_test.go`
- Modify: `internal/capabilities/tenant/module.go`
- Modify: `internal/capabilities/tenant/wire.go`
- Modify: `internal/capabilities/tenant/interfaces/handler_test.go`
- Modify: `internal/e2e/api_contract_test.go`

**Interfaces:**
- Produces: `contract.ProvisioningRole{Code, Scope, System, Name, Description, Permissions}`
- Produces: `contract.ProvisioningRoles.OwnerRole` 按 code 优先、name 次之匹配
- Produces: `application.TenantService.WithRoles(store contract.ProvisioningRoleStore) *TenantService`
- Produces: `application.TenantService.WithDB(db *gorm.DB) *TenantService`

- [ ] **Step 1: 写 provisioner 默认角色失败测试**

在 provisioning 测试中启用空 `roles`，断言开通后租户有且仅有 `tenant_admin`、`tenant_member`，owner 绑定 `tenant_admin`，成员角色无权限。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/capabilities/tenant/application -run TestProvisionCreatesTenantSystemRoles -count=1`

Expected: FAIL，当前空 roles 不创建角色。

- [ ] **Step 3: 写自定义角色兼容测试**

配置一个自定义 name/permissions 的角色与 `owner_role`，断言两个内置角色仍创建，自定义角色无 code，owner 绑定自定义角色。

- [ ] **Step 4: 运行测试确认失败**

Run: `go test ./internal/capabilities/tenant/application -run TestProvisionAppendsCustomRole -count=1`

Expected: FAIL，默认角色未合并。

- [ ] **Step 5: 实现默认租户角色合并与校验**

从 `contract.TenantSystemRoles()` 构造内置模板，再追加配置角色。配置允许空 roles；拒绝内置名称重复；`owner_role` 只允许 `tenant_admin` 或自定义角色 name。

- [ ] **Step 6: 扩展 access provisioning store**

创建角色时持久化 code/scope/is_system；owner 匹配先 code 后 name；权限仍按 `(resource, action)` 查找，缺失跳过。

- [ ] **Step 7: 写 POST /tenants 原子创建角色失败测试**

用 SQLite 和真实 store 调 `TenantService.Create`，断言租户与两个内置角色同时存在；让角色创建报错时断言租户未落库。

- [ ] **Step 8: 运行测试确认失败**

Run: `go test ./internal/capabilities/tenant/application -run TestTenantServiceCreateProvisionsRoles -count=1`

Expected: FAIL，TenantService 尚未调用角色 store。

- [ ] **Step 9: 在 TenantService 创建事务中调用 ProvisionRoles**

`Create` 有 DB 时使用事务和 `dbutil.WithTransaction`，先创建租户再 provision tenant roles；无 DB 的 fake-repo 单测保留原路径。Wire 从 `access.provisioning` 注入 store。

- [ ] **Step 10: 运行租户与 e2e 聚焦测试**

Run: `go test ./internal/capabilities/tenant/... ./internal/e2e/... -count=1`

Expected: PASS。

---

### Task 4: Role API 元数据、角色 options 与系统角色不可变

**Files:**
- Modify: `internal/capabilities/access/domain/role.go`
- Modify: `internal/capabilities/access/infrastructure/mysql_repository.go`
- Modify: `internal/capabilities/access/infrastructure/mysql_repository_test.go`
- Modify: `internal/capabilities/access/application/service.go`
- Modify: `internal/capabilities/access/application/dto.go`
- Modify: `internal/capabilities/access/application/service_test.go`
- Modify: `internal/capabilities/access/interfaces/handler.go`
- Modify: `internal/capabilities/access/interfaces/handler_test.go`
- Modify: `internal/capabilities/access/interfaces/router.go`
- Modify: `internal/capabilities/access/module.go`
- Modify: `internal/capabilities/access/module_test.go`
- Modify: `internal/profiles/full/routes_golden_test.go`
- Modify: `internal/profiles/registry/routes_golden_test.go`

**Interfaces:**
- Produces: `RoleRepository.ListOptions(ctx, tenantID) ([]Role, error)`
- Produces: `RoleService.Options(ctx, requestedTenantID uint64) ([]RoleOptionResponse, error)`
- Produces: `GET /api/v1/roles/options`
- Produces: permission `{Resource: "/api/v1/roles/options", Action: "GET"}`

- [ ] **Step 1: 写 options 范围失败测试**

覆盖：租户身份只得到当前租户且 scope=tenant 的启用角色；平台身份可按 tenant_id 读取；跨租户 tenant_id 返回 403；停用角色不返回。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/capabilities/access/application -run TestRoleServiceOptions -count=1`

Expected: FAIL，Options 未定义。

- [ ] **Step 3: 实现 repository、service 与 DTO**

返回 `RoleOptionResponse{ID, Code, Name, Scope, IsSystem}`，固定 `is_system desc, id asc`；不暴露权限列表。

- [ ] **Step 4: 写系统角色不可变失败测试**

分别调用 Update、Delete、AssignPermissions，断言系统角色返回 403 且 repo 未被写。

- [ ] **Step 5: 运行测试确认失败**

Run: `go test ./internal/capabilities/access/application -run TestSystemRoleIsImmutable -count=1`

Expected: FAIL，当前可以修改系统角色。

- [ ] **Step 6: 实现系统角色保护和 options 路由**

在三个写方法中检查 `role.IsSystem`。`/roles/options` 必须在 `RequirePlatformScope` 之前独立注册；Descriptor 增加权限点。

- [ ] **Step 7: 更新 handler、路由与权限矩阵测试**

断言响应字段、options 路由只注册一次、权限总数增加 1，并更新 full/registry route goldens。

- [ ] **Step 8: 运行 access 与路由测试**

Run: `go test ./internal/capabilities/access/... ./internal/profiles/full ./internal/profiles/registry ./internal/capabilities/catalog -count=1`

Expected: PASS。

---

### Task 5: 默认租户边界与最后平台超管保护

**Files:**
- Modify: `internal/contract/ports.go`
- Modify: `internal/capabilities/access/application/user_roles.go`
- Modify: `internal/capabilities/access/application/user_roles_test.go`
- Modify: `internal/capabilities/access/interfaces/user_roles_test.go`
- Modify: `internal/capabilities/user/application/service.go`
- Modify: `internal/capabilities/user/application/service_test.go`
- Modify: `internal/capabilities/user/module_test.go`

**Interfaces:**
- Produces: `contract.UserRoleAssigner.EnsureCanDeactivateUser(ctx, userID) error`
- Produces: RoleService 之外的 access guard，返回 `CodeConflict` 当目标是最后一个有效超管

- [ ] **Step 1: 写非平台管理默认租户用户失败测试**

用户服务的 Get/List/Create/Update/Delete 在非平台上下文操作 tenant_id=1 时返回 403 或 404；非平台 role assignment 对默认租户用户返回 404。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/capabilities/user/... ./internal/capabilities/access/application -run TestNonPlatformCannotManageDefaultTenant -count=1`

Expected: FAIL。

- [ ] **Step 3: 实现默认租户平台边界**

增加共享语义 helper：平台范围可访问默认租户；租户范围访问 tenant_id=1 时视为不存在/禁止。用于用户列表、详情、创建、更新、删除和角色替换。

- [ ] **Step 4: 写最后一个超管移除失败测试**

SQLite 预置两个超管，移除其中一个成功；移除最后一个返回 409。停用或删除最后一个同样返回 409；第二名有效超管存在时允许。

- [ ] **Step 5: 运行测试确认失败**

Run: `go test ./internal/capabilities/access/application ./internal/capabilities/user/application -run TestLastPlatformSuperAdmin -count=1`

Expected: FAIL。

- [ ] **Step 6: 实现有效超管计数与事务锁**

查询计入 `users.status=1`、`users.deleted_at IS NULL`、`roles.status=1`、`roles.deleted_at IS NULL`。角色替换在现有事务内锁 `platform_super_admin` 角色行后检查。停用/删除通过 `EnsureCanDeactivateUser` 调用同一 guard。

- [ ] **Step 7: 运行 access/user 聚焦测试**

Run: `go test ./internal/capabilities/access/... ./internal/capabilities/user/... -count=1`

Expected: PASS。

---

### Task 6: 审计读取按平台/租户范围收敛

**Files:**
- Modify: `internal/capabilities/audit/application/service.go`
- Modify: `internal/capabilities/audit/application/service_test.go`
- Modify: `internal/capabilities/audit/application/export.go`
- Modify: `internal/capabilities/audit/application/export_test.go`
- Modify: `internal/capabilities/audit/interfaces/handler_test.go`

**Interfaces:**
- Produces: `auditTenantID(ctx) uint64`，平台范围返回 0，租户范围返回 JWT tenant
- Produces: `AuditService.Get` 对非平台上下文执行 `log.TenantID == ctxTenant` 校验

- [ ] **Step 1: 写平台审计范围失败测试**

平台上下文 List 应请求 repository tenantID=0；租户上下文请求自身 tenantID。平台 Get 可读任意租户；租户 Get 跨租户返回 404。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/capabilities/audit/application -run TestAuditScope -count=1`

Expected: FAIL，当前 List/Get 只读 `tenant.FromContext` 且 Get 无隔离。

- [ ] **Step 3: 实现统一审计范围 helper**

List、Get、Export 全部使用 helper。Get 在加载后校验租户归属；不匹配按 not found。

- [ ] **Step 4: 更新导出与 handler 测试**

断言平台导出全部、租户导出本租户、跨租户详情 404。

- [ ] **Step 5: 运行审计测试**

Run: `go test ./internal/capabilities/audit/... -count=1`

Expected: PASS。

---

### Task 7: 契约、文档、golden 与全量门禁

**Files:**
- Modify: `README.md`
- Modify: `docs/design/identity-and-tenancy.md`
- Modify: `docs/releases/v0.4.0.md`
- Regenerate: `docs/openapi/{docs.go,swagger.json,swagger.yaml}`
- Modify: `internal/contract/openapi_test.go`
- Modify: `tools/generator/frameworkmanifest/testdata/minimal.json`
- Modify: `tools/generator/frameworkmanifest/testdata/with_user_access_queue.json`
- Regenerate: `docs/profiles/compose-report.md`

**Interfaces:**
- Consumes all prior tasks.
- Produces final frontend contract and reproducible gates.

- [ ] **Step 1: 更新 OpenAPI 契约测试**

断言 RoleResponse 元数据字段、RoleOption 类型、`GET /roles/options`、字符串 ID、权限点与路由存在。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/contract -run 'TestOpenAPI.*Role' -count=1`

Expected: FAIL，生成物未更新。

- [ ] **Step 3: 更新 README、设计文档与 Release Note**

写清四个系统角色、租户创建路径、系统角色不可变、默认租户边界、最后超管保护和审计范围。

- [ ] **Step 4: 重新生成 OpenAPI**

Run: `make swagger`

Expected: `docs/openapi` 更新且命令 exit 0。

- [ ] **Step 5: 更新生成器 golden**

Run: `UPDATE_GOLDENS=1 go test ./tools/generator/frameworkmanifest -run TestExportMatchesGoldens -count=1`

Expected: PASS，两个 golden 更新。

- [ ] **Step 6: 更新编译面报告**

Run: `make compose-report`

Expected: 路由、文件数、代码行数按实测更新。

- [ ] **Step 7: 运行聚焦回归**

Run: `go test ./internal/capabilities/access/... ./internal/capabilities/tenant/... ./internal/capabilities/user/... ./internal/capabilities/audit/... ./internal/app ./internal/contract ./internal/e2e -count=1`

Expected: PASS。

- [ ] **Step 8: 运行能力与配置门禁**

Run: `make check-capabilities && make profiles-check && make compose-report-check && make check-config`

Expected: 全部 exit 0。

- [ ] **Step 9: 运行全量验证**

Run: `go test ./... -count=1 && go vet ./... && make lint`

Expected: 全部 exit 0，lint `0 issues`。

- [ ] **Step 10: 验证 OpenAPI 可复现**

Run: 临时目录执行 swag 生成，再与 `docs/openapi` 做递归 diff。

Expected: 无差异。

- [ ] **Step 11: 汇总未提交结果**

不 commit、不 push。最终报告列出改动、验证结果、残余风险，并等待用户明确提交指令。
