# 能力可插拔 P1.5：种子/迁移归属 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 迁移按能力目录组织、每个能力独立 goose 版本表（新增 `jimu migrate adopt-capabilities` 为存量实例登记基线）；种子从 `kernel/db` 下沉出内核，权限点改由能力 Descriptor 声明，消除最后一条 kernel→capabilities import。

**Architecture:** 迁移文件从全局 `migrations/{mysql,postgres}/` 拆到 `internal/capabilities/<name>/migrations/{mysql,postgres}/`，经 `go:embed` 进二进制；运行器用 goose v3.27.3 `Provider`（`WithTableName("goose_db_version_<capability>")`），按 catalog 拓扑序逐能力执行。种子整体迁到 `internal/app/seed.go`（组合根，允许 import 能力域模型），`basePermissions()` 拆进各能力 `Descriptor.Permissions`。

**Tech Stack:** Go 1.26 · goose v3.27.3（Provider + `WithTableName` + `fs.FS`）· gorm · sqlmock · testify

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md` §3.6（seed 行）、§4（违反 #1–#4 的处理）、§6.1（Descriptor）、§7（迁移归属与存量桥接）、§10（P1.5 行）；`docs/releases/v0.3.0.md`

## Global Constraints

- **禁止自动提交**：各任务的 commit 步骤仅在用户明确说"提交"后执行（AGENTS.md 最高优先级规则）
- 对外行为不变：`jimu migrate up/down/status/redo`、`jimu seed`、`make migrate*` / `make seed` / `make compose-migrate` / `make compose-seed` 命令面不变；HTTP 路由/响应/schema 不变
- **不新增第三方依赖**（goose/gorm/sqlmock/testify 均已在用）
- 全绿门禁：`gofmt -l .`（无输出）、`go build ./...`、`go vet ./...`、`golangci-lint run ./...`、`make check-log-usage`、`go test ./... -count=1`、`make release-check COMPOSE_ENV=.env.example`
- 分支：从 `release/v0.3.0` 切 `feature/capability-seed-migrations`（若有 issue 号按 `feature/<issue>-capability-seed-migrations`），PR 目标 `release/v0.3.0`
- 提交信息 Conventional Commits 轻量格式，全英文小写祈使句
- 迁移文件能力内版本号沿用**原全局编号**（见「设计裁定」，adopt 基线因此无需映射表）

## 设计裁定（执行前必读）

1. **能力内版本号 = 原全局编号**。设计说"能力内自行编号"，但把 `001_*.sql` 在各能力里重新从 1 编起会让 `adopt-capabilities` 需要一张"原全局版本 → 能力本地版本"映射表，且未来新增能力文件时两套编号语义打架。裁定：能力目录内的文件名沿用原全局编号（如 `user/migrations/mysql/001_users.sql`、`004_user_totp.sql`、`008_user_version.sql`），goose 版本号即文件名前缀。adopt 时"原全局已应用的最大版本 V"直接对每个能力插入 `version <= V` 的基线行，零映射。编号有空洞对 goose 无影响。
2. **表归属**（按 §7 + 现有 `TableName()` 实测）：

   | 迁移(原编号) | 内容 | 归属能力 |
   |---|---|---|
   | 001 | `users` | user |
   | 001 | `roles` + `user_roles` | role |
   | 001 | `permissions` + `role_permissions` | permission |
   | 002 | `audit_logs` | audit |
   | 002 | `outbox_events` | outbox |
   | 003 | `api_keys` | apikey |
   | 003 | `jobs`/`job_history`/`dead_letters`/`scheduled_jobs` | queue（`scheduled_jobs` 的消费者是 kernel/scheduler，表仍归 jobs 域；§7 未单列，归属 queue） |
   | 003 | `import_jobs` | dataops |
   | 003 | `user_oauth_bindings` | oauth |
   | 004 | `users.totp_*` 列 | user（mfa 从表是 P1.6 的事，现在列留在 users） |
   | 005 | `tenants` DDL + `users`/`roles.tenant_id` 列 + 存量回填 | tenant（§4 #1：tenant_id 列保留；tenancy 依赖 user/role，天然后行） |
   | 005 | `audit_logs.tenant_id` 列 | **audit 自己的迁移**（§4 #3 明文） |
   | 006 | `api_keys.tenant_id` 列 | **apikey 自己的迁移**（§4 #4 明文） |
   | 007 | `jobs`/`job_history`/`dead_letters`/`import_jobs.tenant_id` | jobs 三列归 queue，`import_jobs` 列归 dataops |
   | 008 | `version` 列 | users→user、roles→role、tenants→tenant（§4 #1：version 列保留） |
   | 009 | `search_documents` | search |
   | 010 | `audit_logs.prev_hash`/`entry_hash` + `audit_chain_head` | audit |
   | 011/012/013 | `login_histories`/`password_histories`/`trusted_devices` | auth |
   | 014 | `tenant_plans` | tenant |
   | 015 | `webauthn_credentials` | auth |

   拆分铁律：**一条 ALTER 只允许出现在一个能力的迁移里**，出现位置 = 被改列的 feature 归属（tenant_id→tenancy/apikey/audit 按上表）；capability 内文件按原编号命名，跨能力执行顺序由拓扑保证（tenant 在 user/role 之后——给 tenant 的 Descriptor 补 `Requires: ["user","role"]`，见 Task 4）。
3. **嵌入方式**：每个能力包内 `//go:embed migrations`，`Descriptor.Migrations` 携带 `fs.FS`（运行器 `fs.Sub(fsys, "migrations/"+dialect)`）。Dockerfile 不再需要 `COPY migrations/`（二进制自带），`migrations/` 顶层目录删除。无迁移的能力（admin）`Migrations` 为 nil，运行器跳过。embed 不存在的目录会编译失败，所以 Task 2（搬文件）与 Task 3（embed 声明）必须同 PR 内先搬后接。
4. **种子归位**：设计 §3.6 说"内核只留默认租户 + 超管角色 + Casbin 模型加载"。裁定偏差：种子编排整体迁到 `internal/app/seed.go`（组合根），`kernel/db/seed.go` 删除——内核连"默认租户"的种子也不保留（避免 kernel 用裸 SQL 重复 tenant 域模型的写入逻辑；能力关闭时 `capabilities.enabled` 不含 tenant 则不种默认租户，与"关闭的能力不 seed"语义一致）。Casbin 策略同步用 `kernel/access`（内核机制，保留）。`ponytail:` 若后续 profile 入口包需要免引 app 层，再把 seed 拆成 catalog 驱动的 capability Seeder 接口。
5. **权限点声明**：`contract.Descriptor` 增加 `Permissions []Permission`；`basePermissions()` 的 35 条按路由归属拆进 6 个 Descriptor（user/role/permission/audit/tenant/admin）。`/api/v1/roles/*/permissions`（角色分配权限）归 role；`/api/v1/permissions*` 归 permission；`/api/v1/admin/*` 4 条归 admin。种子循环改为遍历启用集 Descriptor 的 Permissions。
6. **adopt-capabilities 语义**：读全局 `goose_db_version` 当前最大已应用版本 V（表不存在 = 全新库，报错引导直接 `migrate up`）；对每个启用能力 `CREATE TABLE goose_db_version_<capability>`（goose Provider 首次 Up 自动建，故 adopt 用 `ApplyVersion(v, true)` 插基线行——只写版本记录不执行 SQL，详见 Task 5 实现）；插入该能力所有 `原编号 <= V` 的迁移版本；打印每个能力的基线明细。V=0（空表）同样走基线（插入 0 条，之后正常 Up）。

---

### Task 1: contract 增加 Permission 与 Descriptor.Permissions

**Files:**
- Modify: `internal/contract/capability.go`
- Test: `internal/contract/capability_test.go`

**Interfaces:**
- Produces: `type Permission struct { Name, Resource, Action string }`；`Descriptor.Permissions []Permission`。后续 Task 6 在能力包填充，Task 7 在 app/seed.go 消费。

- [ ] **Step 1: 写失败测试**

在 `internal/contract/capability_test.go` 追加：

```go
func TestDescriptorPermissionsPassedThrough(t *testing.T) {
	d := Descriptor{
		Name:        "user",
		Permissions: []Permission{{Name: "用户列表", Resource: "/api/v1/users", Action: "GET"}},
	}
	if len(d.Permissions) != 1 || d.Permissions[0].Resource != "/api/v1/users" {
		t.Fatalf("unexpected permissions: %+v", d.Permissions)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/contract/ -run TestDescriptorPermissionsPassedThrough -v`
Expected: FAIL（`undefined: Permission`）

- [ ] **Step 3: 最小实现**

`internal/contract/capability.go` 的 `Descriptor` 定义前加：

```go
// Permission 能力声明的权限点：种子阶段写入 permissions 表并授予超管角色。
type Permission struct {
	Name     string // 中文名称（permissions.name）
	Resource string // 资源路径（permissions.resource，支持 keyMatch 通配）
	Action   string // HTTP 动作（permissions.action）
}
```

`Descriptor` 结构体追加字段（放 `Mount` 之后）：

```go
	// Permissions 能力拥有的权限点；种子时由启用集聚合写入，未启用的能力不种。
	Permissions []Permission
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/contract/ -v`
Expected: PASS（全部用例）

- [ ] **Step 5: Commit（需用户明确指令）**

```bash
git add internal/contract/capability.go internal/contract/capability_test.go
git commit -m "feat(contract): add permission points to capability descriptors"
```

---

### Task 2: 迁移文件按能力拆分落位（纯文件搬迁，暂不接线）

**Files:**
- Create: 各能力 `internal/capabilities/<name>/migrations/{mysql,postgres}/*.sql`（明细见下）
- Delete: `migrations/` 顶层目录全部文件
- 注意: 本任务结束时 `go build ./...` 仍通过（没有代码引用被删目录，`MigrationDir()` 的 `findUp` 找不到目录时回落 `"migrations"` 相对路径——`kernel/db` 的迁移测试会挂，属预期，Task 3 修复；本任务与 Task 3 同一 PR 内先后提交）

**Interfaces:**
- Produces: 文件布局供 Task 3 的 embed 与运行器消费。文件名 = 原全局编号前缀 + 新后缀。

- [ ] **Step 1: mysql 侧拆分**

按下表从 `migrations/mysql/` 剪切内容（保持 `-- +goose Up/Down` 段完整，只删不增不改 SQL 语句）：

| 新文件 | 来自 | 剪切内容 |
|---|---|---|
| `capabilities/user/migrations/mysql/001_users.sql` | 001_core.sql | users 建表段（含注释） |
| `capabilities/role/migrations/mysql/001_roles.sql` | 001_core.sql | roles + user_roles 段 |
| `capabilities/permission/migrations/mysql/001_permissions.sql` | 001_core.sql | permissions + role_permissions 段 |
| `capabilities/audit/migrations/mysql/001_audit_logs.sql` | 002_audit_outbox.sql | audit_logs 段 |
| `capabilities/outbox/migrations/mysql/001_outbox_events.sql` | 002_audit_outbox.sql | outbox_events 段 |
| `capabilities/apikey/migrations/mysql/001_api_keys.sql` | 003_extensions.sql | api_keys 段 |
| `capabilities/queue/migrations/mysql/001_jobs.sql` | 003_extensions.sql | jobs + job_history + dead_letters + scheduled_jobs 段 |
| `capabilities/dataops/migrations/mysql/001_import_jobs.sql` | 003_extensions.sql | import_jobs 段 |
| `capabilities/oauth/migrations/mysql/001_user_oauth_bindings.sql` | 003_extensions.sql | user_oauth_bindings 段 |
| `capabilities/user/migrations/mysql/004_user_totp.sql` | 004_add_totp.sql | 整文件 |
| `capabilities/tenant/migrations/mysql/005_tenants.sql` | 005_add_tenants.sql | 整文件**除去** audit_logs 三行（ALTER/INDEX/Down 对应两行） |
| `capabilities/audit/migrations/mysql/002_audit_logs_tenant.sql` | 005_add_tenants.sql | audit_logs 的 ADD COLUMN + CREATE INDEX + Down 两行 |
| `capabilities/apikey/migrations/mysql/002_api_keys_tenant.sql` | 006_add_api_keys_tenant.sql | 整文件 |
| `capabilities/queue/migrations/mysql/007_jobs_tenant.sql` | 007_add_task_tenant.sql | jobs/job_history/dead_letters 段（4 组 ALTER+INDEX + 回填 UPDATE + Down 对应行） |
| `capabilities/dataops/migrations/mysql/007_import_jobs_tenant.sql` | 007_add_task_tenant.sql | import_jobs 段 |
| `capabilities/user/migrations/mysql/008_user_version.sql` | 008_add_optimistic_version.sql | users 一行 + Down 一行 |
| `capabilities/role/migrations/mysql/008_role_version.sql` | 008_add_optimistic_version.sql | roles 一行 + Down 一行 |
| `capabilities/tenant/migrations/mysql/008_tenant_version.sql` | 008_add_optimistic_version.sql | tenants 一行 + Down 一行 |
| `capabilities/search/migrations/mysql/009_search_documents.sql` | 009_add_search_documents.sql | 整文件 |
| `capabilities/audit/migrations/mysql/010_audit_chain.sql` | 010_add_audit_chain.sql | 整文件 |
| `capabilities/auth/migrations/mysql/011_login_histories.sql` | 011_add_login_histories.sql | 整文件 |
| `capabilities/auth/migrations/mysql/012_password_histories.sql` | 012_add_password_histories.sql | 整文件 |
| `capabilities/auth/migrations/mysql/013_trusted_devices.sql` | 013_add_trusted_devices.sql | 整文件 |
| `capabilities/tenant/migrations/mysql/014_tenant_plans.sql` | 014_add_tenant_plans.sql | 整文件 |
| `capabilities/auth/migrations/mysql/015_webauthn_credentials.sql` | 015_add_webauthn_credentials.sql | 整文件 |

- [ ] **Step 2: postgres 侧对等拆分**

对 `migrations/postgres/` 做同样拆分（文件名一致；postgres 文件里多出的 `COMMENT ON` 行跟随其表所在段走；`005` 里 `idx_audit_logs_tenant_id` 与 `COMMENT ON COLUMN audit_logs.tenant_id` 归 audit 的 `002_audit_logs_tenant.sql`；`001_core.sql` 里 `roles` 的 `CONSTRAINT/UNIQUE` 等按表归属同 mysql 规则）。

- [ ] **Step 3: 校验无遗漏**

Run: `bash -c 'diff <(grep -rh --include="*.sql" "^" migrations/mysql/ | grep -v "^--" | grep -v "^$") <(grep -rh --include="*.sql" "^" internal/capabilities/*/migrations/mysql/ | grep -v "^--" | grep -v "^$") | head -40'`
Expected: 仅注释/空行差异；语句级 diff 为空（逐语句比对）。postgres 同理跑一遍。

- [ ] **Step 4: 删除顶层 migrations/ 目录**

```bash
git rm -r migrations/
```

- [ ] **Step 5: 构建仍绿（迁移测试失败属预期，不修）**

Run: `go build ./...`
Expected: PASS。`go test ./internal/kernel/db/` 此刻 FAIL（目录没了），Task 3 修复。

- [ ] **Step 6: Commit（需用户明确指令；与 Task 3 分开提交会红 CI，实际提交顺序为 Task 2 → Task 3 连续两提交后统一 push）**

```bash
git add -A
git commit -m "refactor(db): split global migrations into per-capability directories"
```

---

### Task 3: 能力迁移运行器（goose Provider + embed）替换全局迁移

**Files:**
- Modify: `internal/contract/capability.go`（Descriptor.Migrations 字段）、`internal/kernel/db/migrate.go`、`internal/shared/testutil/testdb.go`（Migrate 走新运行器）、`internal/capabilities/{user,role,permission,tenant,audit,apikey,oauth,auth,queue,dataops,search,outbox}/module.go`（或各自新建 `migrations.go`，见 Step 5）+ 顶层包 `outbox.go`/`queue` 包根等实际 Descriptor 所在文件
- Test: 重写 `internal/kernel/db/migrate_test.go` 相关用例；`internal/kernel/db/migration_integration_test.go` 改造

**Interfaces:**
- Consumes: Task 2 的目录布局；goose `provider.NewProvider(dialect, db, fsys, provider.WithTableName(name))`
- Produces:
  - `db.MigrateEnabled(cfg config.DBConfig, caps []contract.Descriptor, direction string) error`（新核心；`Migrate`/`MigrateWithRetry` 保留旧签名，内部改为全能力清单调用）
  - `db.AdoptCapabilities(cfg config.DBConfig, caps []contract.Descriptor) (map[string][]int64, error)`（Task 5 用）
  - `contract.Descriptor.Migrations fs.FS`（nil = 无迁移）

- [ ] **Step 1: 写失败测试（单测，sqlite 不可用——goose Provider 需方言级 sql.DB；用 sqlmock 钉住"每能力独立版本表"的调用形状）**

`internal/kernel/db/migrate_test.go` 追加：

```go
func TestMigrationVersionTablesPerCapability(t *testing.T) {
	// 两个能力、各带一个迁移：期望 Provider 用 WithTableName 后，
	// 对 goose_db_version_user 与 goose_db_version_role 各自建表/查询。
	// sqlmock 侧按顺序期望两套 CREATE TABLE/SELECT —— 任何复用全局表名的实现会在这里失败。
	caps := []contract.Descriptor{
		{Name: "user", Migrations: os.DirFS(filepath.Join(testCapabilityMigRoot(), "user"))},
		{Name: "role", Migrations: os.DirFS(filepath.Join(testCapabilityMigRoot(), "role"))},
	}
	err := MigrateEnabled(testDBConfig(t), caps, "up")
	require.Error(t, err) // sqlmock 下执行到 SQL 即可，重点是表名断言
	require.Contains(t, mockCalls(t), "goose_db_version_user")
	require.Contains(t, mockCalls(t), "goose_db_version_role")
}
```

（`testCapabilityMigRoot`/`mockCalls` 为测试文件内 helper：前者指向 `internal/kernel/db/testdata/mig/{user,role}/mysql/001_x.sql` 新增的最小夹具；后者读取 sqlmock 记录。）

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/kernel/db/ -run TestMigrationVersionTablesPerCapability -v`
Expected: FAIL（`undefined: MigrateEnabled`）

- [ ] **Step 3: 实现 contract.Migrations 字段**

`internal/contract/capability.go`：import 增加 `"io/fs"`；`Descriptor` 追加：

```go
	// Migrations 能力自带迁移的嵌入文件系统（根下应有 mysql/ 与 postgres/ 子目录）；
	// nil 表示该能力无迁移。
	Migrations fs.FS
```

- [ ] **Step 4: 实现 MigrateEnabled**

`internal/kernel/db/migrate.go` 新增（`runMigration`/`MigrationDir`/`PostgresMigrationDir`/`migrationDir`/`findUp` 删除）：

```go
// MigrateEnabled 按能力拓扑序逐个执行迁移：每个能力用独立 goose Provider
// 与独立版本表 goose_db_version_<capability>，互不干扰；删除能力即删其表与记录。
func MigrateEnabled(cfg config.DBConfig, caps []contract.Descriptor, direction string) error {
	dialect := cfg.Dialect()
	gooseDialect := goose.DialectMySQL
	if dialect == "postgres" {
		gooseDialect = goose.DialectPostgres
	}
	sqlDB, driver, dsnStr, err := openSQLForMigrate(cfg) // 由 sqlDriverAndDSN 拆出：返回 *sql.DB
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()

	for _, cap := range caps {
		if cap.Migrations == nil {
			continue
		}
		fsys, err := fs.Sub(cap.Migrations, "migrations/"+dialectDir(dialect))
		if err != nil {
			return fmt.Errorf("capability %s migrations: %w", cap.Name, err)
		}
		if err := migrateOne(gooseDialect, sqlDB, fsys, "goose_db_version_"+cap.Name, direction); err != nil {
			return fmt.Errorf("capability %s: %w", cap.Name, err)
		}
	}
	return nil
}

func migrateOne(dialect goose.Dialect, sqlDB *sql.DB, fsys fs.FS, table, direction string) error {
	p, err := provider.NewProvider(dialect, sqlDB, fsys, provider.WithTableName(table))
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	defer func() { _ = p.Close() }()
	ctx := context.Background()
	switch direction {
	case "up":
		_, err = p.Up(ctx)
	case "status":
		st, serr := p.Status(ctx)
		if serr != nil {
			return serr
		}
		for _, s := range st {
			state := "Pending"
			if s.Applied {
				state = "Applied"
			}
			fmt.Printf("%s: %d %s (%s)\n", table, s.Source.Version, s.Source.Path, state)
		}
		return nil
	case "down":
		_, err = p.Down(ctx)
	case "redo":
		if _, err = p.Down(ctx); err == nil {
			_, err = p.Up(ctx)
		}
	default:
		return fmt.Errorf("unknown direction: %s", direction)
	}
	if err != nil {
		return err
	}
	return nil
}
```

`Migrate(cfg, direction)` 改为 `return MigrateEnabled(cfg, catalog.All(), direction)`（`internal/kernel/db` import `internal/capabilities/catalog`？——**不行**，那是 kernel→capabilities import，正是要消灭的）。裁定：`Migrate`/`MigrateWithRetry` 的旧签名保留但语义改为"仅内核基线"（当前无内核迁移 = no-op，status 打印空），生产调用方 `testutil.Migrate` 与 CLI 全部改传 `caps`。`MigrateWithRetry` 重试逻辑包住 `MigrateEnabled`。

- [ ] **Step 5: 各能力声明 embed**

每个有迁移的能力，在其 Descriptor 所在文件加：

```go
//go:embed migrations
var migrationsFS embed.FS
```

并在 `Descriptor` 字面量加 `Migrations: migrationsFS,`。涉及：user、role、permission、tenant、audit、apikey、oauth、auth、queue（Descriptor 在 queue 包根文件）、dataops、search、outbox。admin 不加（无迁移）。注意 `//go:embed` 必须放在包级 var 上方且文件属于该包。

- [ ] **Step 6: 接线调用方**

- `internal/shared/testutil/testdb.go` 的 `Migrate()`：`return db.MigrateEnabled(tdb.cfg, capsForMigrate(), "up")`；`capsForMigrate()` 从测试侧注入——`testutil` 不能 import catalog（同向违规）。裁定：`testutil` 增加 `SetMigrateCaps(caps []contract.Descriptor)` 包级注入点（默认空 = 无能力迁移，仅存量用户）；e2e/集成测试 setup 里传 `catalog.All()`。`ponytail:` 测试夹具注入用包级变量，profile 化后再收口。
- `internal/kernel/db/migration_integration_test.go`：改为对 mysql/postgres 真库各跑 `MigrateEnabled`（caps 来自硬编码的两个测试夹具能力），断言版本表 `goose_db_version_<name>` 存在且记录数正确、二次执行幂等。
- `internal/kernel/db/migrate_test.go`：删除 `TestMigrationDir`；`seed_test.go` 的 `repoRoot()` 改用 `runtime.Caller` 定位仓库根（或移到 Task 7 一起删）。

- [ ] **Step 7: 跑测试**

Run: `go test ./internal/kernel/db/ ./internal/shared/testutil/ ./internal/contract/ -count=1 -v`
Expected: PASS

Run: `go build ./... && go vet ./...`
Expected: PASS

- [ ] **Step 8: 真库集成验证（Docker/OrbStack 运行中）**

Run: `go test ./internal/kernel/db/ -run TestMigrationIntegration -tags integration -count=1 -v`（沿用该文件现有的 tag/环境变量约定）
Expected: mysql 与 postgres 均建出 12 张能力版本表、全部迁移应用成功、重跑幂等

- [ ] **Step 9: Commit（需用户明确指令）**

```bash
git add -A
git commit -m "feat(db): run migrations per capability with dedicated goose version tables"
```

---

### Task 4: tenant 依赖补齐与 catalog 顺序校验

**Files:**
- Modify: `internal/capabilities/tenant/module.go`
- Test: `internal/capabilities/catalog/catalog_test.go`

**Interfaces:**
- Consumes: Task 2 拆分后 tenant 迁移含对 users/roles 的 ALTER —— 执行顺序必须保证 user、role 先于 tenant
- Produces: tenant Descriptor `Requires: ["user", "role"]`

- [ ] **Step 1: 写失败测试**

`internal/capabilities/catalog/catalog_test.go` 追加：

```go
func TestTenantMigrationsRunAfterUserAndRole(t *testing.T) {
	// tenant 的 005_tenants.sql ALTER users/roles；执行顺序由 Resolve 返回序保证，
	// 因此 tenant.Requires 必须含 user 与 role。
	for _, d := range All() {
		if d.Name != "tenant" {
			continue
		}
		require.Contains(t, d.Requires, "user")
		require.Contains(t, d.Requires, "role")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/capabilities/catalog/ -run TestTenantMigrationsRunAfterUserAndRole -v`
Expected: FAIL

- [ ] **Step 3: 实现**

`internal/capabilities/tenant/module.go` 的 `Descriptor` 加 `Requires: []string{"user", "role"},`。

- [ ] **Step 4: 全量跑 catalog 与 e2e 受影响用例**

Run: `go test ./internal/capabilities/... ./internal/e2e/ -count=1`
Expected: PASS（e2e 若断言启用闭包文案，同步更新期望值——只有 tenant 依赖变化，影响面应仅限闭包用例）

- [ ] **Step 5: Commit（需用户明确指令）**

```bash
git add internal/capabilities/tenant/module.go internal/capabilities/catalog/catalog_test.go internal/e2e/
git commit -m "feat(capabilities): require user and role before tenant for migration order"
```

---

### Task 5: `jimu migrate adopt-capabilities` 存量基线登记

**Files:**
- Modify: `internal/kernel/db/migrate.go`（`AdoptCapabilities`）、`cmd/cli/main.go`（新子命令）
- Test: `internal/kernel/db/adopt_test.go`（单测 + 集成）

**Interfaces:**
- Consumes: Task 3 的 Provider 机制（`WithTableName`）与 Task 2 的文件布局
- Produces: `db.AdoptCapabilities(cfg config.DBConfig, caps []contract.Descriptor) (map[string][]int64, error)`；CLI `jimu migrate adopt-capabilities`

- [ ] **Step 1: 写失败测试（集成，真实库）**

`internal/kernel/db/adopt_test.go`：

```go
// TestAdoptCapabilities_NoRerunAfterAdopt 是设计 §7/P1 的验收主线：
// 存量库（全局版本表记到 V）adopt 后，各能力版本表已有基线行，
// 再次 MigrateEnabled(up) 不重放任何已应用迁移（建表语句 IF NOT EXISTS 之外无 DDL 执行）。
func TestAdoptCapabilities_NoRerunAfterAdopt(t *testing.T) {
	cfg := integrationDBConfig(t) // 复用 migration_integration_test.go 的环境变量约定
	// 1. 旧世界：把夹具迁移灌进全局 goose_db_version（模拟存量实例已应用到 V）
	seedLegacyGlobalVersionTable(t, cfg, 15) // 直接 INSERT 版本行 1..15，不执行 SQL
	// 2. adopt
	baseline, err := AdoptCapabilities(cfg, testCaps())
	require.NoError(t, err)
	require.NotEmpty(t, baseline["user"]) // user 应有 001/004/008 三条基线
	// 3. 新世界：再次 up 不得执行任何已应用迁移
	require.NoError(t, MigrateEnabled(cfg, testCaps(), "up"))
	// 4. 版本表行数不变
	require.Equal(t, len(baseline["user"]), countRows(t, cfg, "goose_db_version_user"))
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/kernel/db/ -run TestAdoptCapabilities -tags integration -v`
Expected: FAIL（`undefined: AdoptCapabilities`）

- [ ] **Step 3: 实现 AdoptCapabilities**

```go
// AdoptCapabilities 为存量实例登记各能力版本表基线：读取全局 goose_db_version
// 的最大已应用版本 V，对每个能力把原编号 <= V 的迁移版本直接写入
// goose_db_version_<capability>（is_applied=true，不执行 SQL）。
// 全新库（全局版本表不存在）返回错误并引导直接 migrate up。
func AdoptCapabilities(cfg config.DBConfig, caps []contract.Descriptor) (map[string][]int64, error) {
	// 1. 打开连接、读 goose_db_version 最大版本（表不存在 → 返回引导错误）
	// 2. 对每个 cap.Migrations != nil 的能力：
	//    a. 列出其迁移文件版本号（fs.ReadDir + 解析文件名前缀数字）
	//    b. Provider := NewProvider(dialect, sqlDB, fsys, WithTableName("goose_db_version_"+name))
	//    c. 对每个 <= V 的版本：p.ApplyVersion(ctx, v, true) —— 只登记不执行
	// 3. 返回 map[能力名][]已登记版本
}
```

注意：goose `Provider.ApplyVersion` 在版本表不存在时会先建表（复用 Task 3 的 `WithTableName`），`direction=true` 即登记为已应用。全局 `goose_db_version` 表本身保留不动（历史记录，不再被新运行器读写）。

- [ ] **Step 4: CLI 接线**

`cmd/cli/main.go` 在 `migrateRedoCmd` 后追加：

```go
var migrateAdoptCmd = &cobra.Command{
	Use:   "adopt-capabilities",
	Short: "Register per-capability version baselines for an existing database",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		log := logger.New(cfg.Log)
		baseline, err := db.AdoptCapabilities(cfg.DB, catalog.All())
		if err != nil {
			return fmt.Errorf("adopt failed: %w", err)
		}
		for _, name := range catalog.Names() {
			if vs, ok := baseline[name]; ok {
				fmt.Printf("%s: %d migrations adopted\n", name, len(vs))
			}
		}
		return nil
	},
}
```

`init()` 里 `migrateCmd.AddCommand(migrateAdoptCmd)`；import 增加 `"jimu/internal/capabilities/catalog"`。

- [ ] **Step 5: 跑测试与构建**

Run: `go test ./internal/kernel/db/ -tags integration -count=1 -v && go build ./...`
Expected: PASS；CLI 二进制含新命令（`go run ./cmd/cli migrate adopt-capabilities --help` 可见）

- [ ] **Step 6: 真库手工验收（设计 P1 验收主线）**

```bash
# 用现有 compose 环境（含既有数据）验证 adopt 后不重跑、后续新迁移正常应用
make compose-migrate   # 确认现有路径仍工作（旧世界）
make compose-seed
docker compose exec app ./jimu migrate adopt-capabilities
docker compose exec app ./jimu migrate status   # 各能力版本表显示基线已应用
docker compose exec app ./jimu migrate up       # 无待应用、无重放
```

Expected: adopt 输出各能力基线数量；status 全 Applied；up 幂等。

- [ ] **Step 7: Commit（需用户明确指令）**

```bash
git add internal/kernel/db/ cmd/cli/main.go
git commit -m "feat(cli): add migrate adopt-capabilities for existing databases"
```

---

### Task 6: 权限点拆进能力 Descriptor

**Files:**
- Modify: `internal/capabilities/{user,role,permission,audit,tenant,admin}/module.go`（Descriptor 加 Permissions）
- Delete: `internal/kernel/db/seed.go` 的 `basePermissions()`（整个文件在 Task 7 删，本任务先把数据搬走）

**Interfaces:**
- Consumes: Task 1 的 `contract.Permission`
- Produces: `catalog.All()` 聚合出的 35 条权限点与 `basePermissions()` 完全等价（Task 7 的测试钉住）

**权限点分配**（来源 `kernel/db/seed.go:145-182`，逐条搬，中文名不改）：

| Descriptor | 权限点（Resource + Action） |
|---|---|
| user | /api/v1/users GET/POST、/api/v1/users/* GET/PUT/DELETE（5 条） |
| role | /api/v1/roles GET/POST、/api/v1/roles/* GET/PUT/DELETE、/api/v1/roles/*/permissions POST（6 条） |
| permission | /api/v1/permissions GET/POST、/api/v1/permissions/* GET/PUT/DELETE（5 条） |
| audit | /api/v1/audits GET、/api/v1/audits/* GET（2 条；「审计导出 /api/v1/audits/export GET」若路由在 admin 能力则随 admin——执行时以 `capabilities/audit/interfaces/router.go` 与 admin 的审计导出路由为准，路由在哪个能力就归哪个） |
| tenant | /api/v1/tenants GET/POST、/api/v1/tenants/* GET/PUT/DELETE、/api/v1/tenant-plans GET/POST、/api/v1/tenant-plans/* GET/PUT/DELETE（8 条） |
| admin | /api/v1/admin/* GET/POST/PUT/DELETE（4 条） |

- [ ] **Step 1: 写失败测试（聚合等价性钉子）**

`internal/capabilities/catalog/catalog_test.go` 追加：

```go
func TestDescriptorPermissionsCoverBusinessRoutes(t *testing.T) {
	// 与原 kernel/db.TestBasePermissionsCoverBusinessRoutes 同一批必含项；
	// 迁移后权限点改由 Descriptor 声明，聚合结果必须覆盖同样的路由面。
	required := []struct{ resource, action string }{
		{"/api/v1/users", "GET"}, {"/api/v1/users", "POST"},
		{"/api/v1/users/*", "GET"}, {"/api/v1/users/*", "PUT"}, {"/api/v1/users/*", "DELETE"},
		{"/api/v1/roles", "GET"}, {"/api/v1/roles", "POST"},
		{"/api/v1/roles/*", "GET"}, {"/api/v1/roles/*", "PUT"}, {"/api/v1/roles/*", "DELETE"},
		{"/api/v1/roles/*/permissions", "POST"},
		{"/api/v1/permissions", "GET"}, {"/api/v1/permissions", "POST"},
		{"/api/v1/permissions/*", "GET"}, {"/api/v1/permissions/*", "PUT"}, {"/api/v1/permissions/*", "DELETE"},
		{"/api/v1/audits", "GET"}, {"/api/v1/audits/*", "GET"},
		{"/api/v1/tenants", "GET"}, {"/api/v1/tenants", "POST"},
		{"/api/v1/tenants/*", "GET"}, {"/api/v1/tenants/*", "PUT"}, {"/api/v1/tenants/*", "DELETE"},
	}
	got := map[string]bool{}
	for _, d := range All() {
		for _, p := range d.Permissions {
			got[p.Resource+" "+p.Action] = true
		}
	}
	for _, item := range required {
		if !got[item.resource+" "+item.action] {
			t.Fatalf("missing permission %s %s", item.action, item.resource)
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/capabilities/catalog/ -run TestDescriptorPermissionsCoverBusinessRoutes -v`
Expected: FAIL

- [ ] **Step 3: 填充各 Descriptor.Permissions**

按上表逐个能力加字段（例）：

```go
var Descriptor = contract.Descriptor{
	Name: "user",
	// ...
	Permissions: []contract.Permission{
		{Name: "用户列表", Resource: "/api/v1/users", Action: "GET"},
		{Name: "用户创建", Resource: "/api/v1/users", Action: "POST"},
		{Name: "用户详情", Resource: "/api/v1/users/*", Action: "GET"},
		{Name: "用户修改", Resource: "/api/v1/users/*", Action: "PUT"},
		{Name: "用户删除", Resource: "/api/v1/users/*", Action: "DELETE"},
	},
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/capabilities/... -count=1`
Expected: PASS

- [ ] **Step 5: Commit（需用户明确指令）**

```bash
git add internal/capabilities/ internal/contract/
git commit -m "feat(capabilities): declare permission points on capability descriptors"
```

---

### Task 7: 种子迁出内核（app/seed.go），删除 kernel/db/seed.go

**Files:**
- Create: `internal/app/seed.go`
- Delete: `internal/kernel/db/seed.go`、`internal/kernel/db/seed_test.go`
- Create: `internal/app/seed_test.go`（从 seed_test.go 改造迁移）
- Modify: `cmd/cli/main.go`（seed 命令改调 `app.RunSeed`）、`internal/e2e/api_contract_test.go`（两处 `db.RunSeed` → `app.RunSeed`）

**Interfaces:**
- Consumes: Task 6 的 `Descriptor.Permissions`；`kernel/access.NewEnforcer`；tenant/role/user 能力域模型（与现 seed.go 相同的 import，位置合法化——app 是组合根）
- Produces: `app.RunSeed(gdb *gorm.DB, caps []contract.Descriptor) error` 与 `app.RunSeedWithCasbin(gdb *gorm.DB, caps []contract.Descriptor) error`（caps 传 `catalog.Resolve` 的启用集；权限点从 caps 聚合，未启用能力的权限不种）

- [ ] **Step 1: 写失败测试（先立新测试再删旧实现）**

`internal/app/seed_test.go`：从 `kernel/db/seed_test.go` 迁移改造——`TestRunSeed_HappyPath` 等 sqlmock 用例的编排增加权限点来源变化（权限循环遍历 `caps` 聚合出的 35 条，数量断言改为 `len(catalog.All() 聚合)`）；`TestRunSeedWithCasbin`（sqlite 端到端）、`TestRunSeed_MissingAdminPassword` 原样搬；追加关键新用例：

```go
func TestRunSeed_SkipsDisabledCapabilityPermissions(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")
	gdb := newSeedSqliteDB(t) // 与原 seed_test.go 相同的 sqlite 内存库 helper
	// AutoMigrate 同原用例
	// 只启用 user+role+permission：audit/tenant/admin 的权限点不得出现
	caps, err := catalog.Resolve([]string{"user", "role", "permission"})
	require.NoError(t, err)
	require.NoError(t, RunSeed(gdb, caps))
	var count int64
	require.NoError(t, gdb.Table("permissions").Count(&count).Error))
	// user 5 + role 6 + permission 5 = 16；audit/tenant/admin 的 19 条不在
	assert.Equal(t, int64(16), count)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/app/ -run TestRunSeed -v`
Expected: FAIL（`undefined: RunSeed`）

- [ ] **Step 3: 实现 app/seed.go**

把 `kernel/db/seed.go` 整体搬来，改动三处：包名 `app`；`basePermissions()` 删除，权限循环改为：

```go
// 权限点来自启用集各能力的 Descriptor.Permissions（能力自声明，未启用不种）
var permissions []roleDomain.Permission
for _, d := range caps {
	for _, p := range d.Permissions {
		permissions = append(permissions, roleDomain.Permission{Name: p.Name, Resource: p.Resource, Action: p.Action})
	}
}
```

`RunSeed`/`SeedCasbinPolicies`/`RunSeedWithCasbin` 签名各加 `caps []contract.Descriptor` 参数；import 保持 `capabilities/{role,tenant,user}/domain` + `kernel/{access,tenant}`。默认租户/套餐/超管角色/管理员/角色分配逻辑逐行保留。

- [ ] **Step 4: 改调用方，删旧文件**

- `cmd/cli/main.go:153`：`app.RunSeedWithCasbin(dbConn, catalog.All())`（CLI 无配置开关语义之外的裁剪，用全量清单）
- `internal/e2e/api_contract_test.go:81,416`：`app.RunSeed(gdb, catalog.All())`
- `git rm internal/kernel/db/seed.go internal/kernel/db/seed_test.go`
- `kernel/db` 的 `newMockGormDB`（mysql_test.go）若仅 seed 测试使用则一并删除

- [ ] **Step 5: 跑全量相关测试**

Run: `go test ./internal/app/ ./internal/e2e/ ./internal/kernel/db/ ./cmd/... -count=1`
Expected: PASS；`go build ./...` PASS；`go vet ./...` PASS

- [ ] **Step 6: 验证反向依赖清零**

Run: `bash -c 'grep -rn "jimu/internal/capabilities" internal/kernel/ | grep -v _test.go'`
Expected: 无输出（P1.4 留下的两处 + seed 全部消除；如有残留逐条处理或记录到版本日志"已知残留"）

- [ ] **Step 7: Commit（需用户明确指令）**

```bash
git add -A
git commit -m "refactor(app): move seeding out of kernel with descriptor-declared permissions"
```

---

### Task 8: 脚手架、Dockerfile、文档与发布说明收尾

**Files:**
- Modify: `tools/generator/module.go`（迁移写入能力目录）、`Dockerfile`、`README.md`、`docs/releases/v0.3.0.md`、`docs/design/2026-09-18-capability-plugins-design.md`（P1.5 归位注记）、`.github/workflows/ci.yml` 与 `ci-docker.yml`（`migrations/**` 触发路径改为能力目录）

**Interfaces:**
- Consumes: Task 2–7 的全部产物
- Produces: 仓库级一致性；无功能变化

- [ ] **Step 1: generator 迁移落点**

`tools/generator/module.go`：
- `preflight` 的存在性检查 `migrations/mysql` 改为 `internal/capabilities`（已有）+ 能力目录 `migrations/mysql`
- `nextMigrationNumber(filepath.Join(root, "migrations", "mysql"))` 改为读该能力目录、取该目录内最大编号 +1（能力内自行编号从这里开始生效；存量编号沿用原全局值的裁定不变，新生成的能力从 001 起）
- 目标文件 `migrations/mysql/<n>_create_<table>.sql` 改为 `internal/capabilities/<name>/migrations/{mysql,postgres}/...`（两个方言都生成，模板用 mysql 语句并在注释里标注 postgres 差异待手工调整；`ponytail:` 模板不分支，复杂方言差异由生成者手改）
- 生成物同时新建 `migrations/postgres/` 空目录占位（`.gitkeep`）避免 embed 缺目录编译失败——若能力模板不含 migrations embed，则在生成的 `module.go` 模板里补 `//go:embed migrations` + Descriptor.Migrations

- [ ] **Step 2: Dockerfile**

删 `COPY migrations/ ./migrations/`（embed 已进二进制）。构建验证：`docker build -t jimu:p15 . && docker run --rm jimu:p15 ./jimu migrate status`（对空配置报 DB 连接错误即可，证明二进制不再依赖磁盘上的迁移目录）。

- [ ] **Step 3: CI 触发路径**

`ci.yml`/`ci-docker.yml` 的 `paths` 里 `migrations/**` 改为 `internal/capabilities/*/migrations/**`。

- [ ] **Step 4: README 同步**

- 目录树：`migrations/` 消失，能力目录下出现 `migrations/{mysql,postgres}/`
- 「数据库迁移」章节：双版本表机制（`goose_db_version_<capability>`）、执行顺序（catalog 拓扑序）、`adopt-capabilities` 用法与存量实例升级步骤（先 `migrate up` 到旧世界最新 → 升级二进制 → `migrate adopt-capabilities` → 之后正常 `migrate up`）
- CLI 命令表加 `jimu migrate adopt-capabilities`
- 「开发规范」：新增迁移写进能力目录、一条 ALTER 只属一个能力
- 种子章节：权限点来自能力 Descriptor，`jimu seed` 语义不变

- [ ] **Step 5: 发布说明**

`docs/releases/v0.3.0.md`：
- 「变更」两条"（进行中）"标记落地：能力自带迁移与配置条目改写为完成时态（配置归属仍未做，保留"配置段下沉"为未完成并注明）
- 新增条目：迁移按能力目录组织 + 独立版本表 + `adopt-capabilities`；种子迁出内核 + 权限点能力自声明
- 「验证」章节待全量回归后更新
- 「说明」补：顶层 `migrations/` 已删除，存量实例升级路径

- [ ] **Step 6: 设计文档注记**

`docs/design/2026-09-18-capability-plugins-design.md` §10 P1 行内追加 P1.5 完成注记（比照 P1.4 的写法），§3.6 seed 行、§7 表归属表加"已按原全局编号落位"注。

- [ ] **Step 7: 全量回归**

Run: `gofmt -l .`（无输出）`&& go build ./... && go vet ./... && golangci-lint run ./...`
Run: `make check-log-usage && go test ./... -count=1`
Run: `make release-check COMPOSE_ENV=.env.example`
Expected: 全绿（release-check 覆盖 compose-check 的隔离运行时校验与 API 契约冒烟；adopt 路径已在 Task 5 真库验证）

- [ ] **Step 8: Commit（需用户明确指令）**

```bash
git add -A
git commit -m "docs: record capability-owned migrations and seeding in readme and release notes"
```

---

## 任务依赖与执行顺序

Task 1（独立）→ Task 2 → Task 3（2、3 必须连续，中间态构建红）→ Task 4 → Task 5（依赖 3）→ Task 6 → Task 7（依赖 6）→ Task 8（收尾）。Task 5 的真库手工验收与 Task 8 的 release-check 最后统一过。

## Self-Review 记录

- **Spec 覆盖**：§3.6 seed 行（kernel 只留默认租户——以"种子全迁 app 层"偏差实现，裁定 #4 已注明）；§4 违反 #1（version/tenant_id 列保留、totp 留 user 待 P1.6）、#2（user_roles 归 role，access 拆分在 P1.6）、#3、#4 全覆盖；§6.1 Permissions 字段；§7 目录/版本表/执行顺序/adopt/表归属全覆盖（scheduled_jobs 归属为裁定 #2 补充）；§10 P1.5 目标（kernel→capabilities import 清零）在 Task 7 Step 6 钉住。
- **占位符扫描**：Task 3 Step 1 测试代码中的 `testCapabilityMigRoot`/`mockCalls` 是声明要新建的 helper，非未定义引用；Task 5 Step 3 实现以步骤散文+关键语义给出（SQL 交互经 Provider，无裸 SQL 可预先成文），其余任务代码完整。
- **类型一致性**：`contract.Permission{Name,Resource,Action}`、`Descriptor.Permissions []Permission`、`Descriptor.Migrations fs.FS`、`MigrateEnabled(cfg, caps, direction)`、`AdoptCapabilities(cfg, caps)`、`app.RunSeed(gdb, caps)` 在各任务间一致。
