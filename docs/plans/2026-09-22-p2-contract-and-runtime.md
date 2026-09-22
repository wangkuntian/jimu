# 能力可插拔 P2.2 + P2.3：能力自描述契约与运行时降级 实现计划

> 本文件是 P2 子阶段 P2.2/P2.3 的详细计划；P2 三层机制总纲见
> [2026-09-21-p2-three-layer-mechanism.md](2026-09-21-p2-three-layer-mechanism.md)，
> P2.1（配置归属）执行记录见 [2026-09-21-config-ownership.md](2026-09-21-config-ownership.md)。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补齐 `contract.Descriptor` 的 `SoftRequires`/`Owns` 两项自描述，令 catalog 能表达「可选依赖缺失时降级」，并让这两项声明**可被门禁校验**（不是装饰性元数据）；同时收口 P2.3 的运行时降级验证与启用清单对外可见。

**Architecture:** 静态声明放在各能力的 `Descriptor`（`internal/capabilities/<name>/module.go` 或 `migrations.go`）；`internal/capabilities/catalog` 负责声明校验、硬依赖闭包（不变）与软依赖降级报告；`internal/app` 在启动日志与管理端点输出最终启用清单与降级项。`kernel/*` 不得 import `capabilities/*`，管理端点经 `HealthRouter` 的可变注册器由 `internal/app` 注入。

**Tech Stack:** Go 1.26 · `io/fs`（embed 迁移只读扫描）· `regexp`（门禁静态检查）· testify

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md` §6.1（能力自描述）、§6.4（层③运行时）、§9（门禁）、§10 P2

## 裁定（执行前必读，已与用户确认）

1. **不加 `Tags`**：`Tags` 目前没有任何消费者（设计里只在 §6.1 出现一次，profile 按显式子集而非 tag 过滤）。等 P2.4/P2.7 出现真实消费者再加，避免无用元数据。
2. **不 catalogize**：`storage`/`notification`/`retention`（P2.1 ③ 挂账）与 `ws`/`grpc`/`apidocs`/`encryption` 保持非 catalog、沿用组合根显式装配；是否 catalogize 与 P2.5（驱动级拆分）/P2.6（非代码资产）一起定。**catalog 项数保持 18**，`catalog_test.go` 的 `TestResolveEmptyMeansAll` 的 `18`、`wiredCapabilities`、`main_test.go` 装配断言均不动。
3. **不改名**：`contract.Descriptor` 保持现名（改 `Capability` 涉及 106 处引用、18 个字面量、22 个测试文件，独立提交另议）。
4. **`Migrations` 保持 `fs.FS`**：不改回路径形态（P1.5 为 embed 进二进制的既定选择，改回会推翻并牵动 `kernel/db/migrate.go` 的 `fs.Stat`/`fs.Sub`）。
5. **带最小 `check-capabilities`**：只为让 `Owns`/`SoftRequires` 有校验者。完整四道门禁（含跨能力 import 一致性、`capabilities/A → capabilities/B/internal` 越界、`kernel → capabilities` 反向依赖）仍属 P2.8。
6. **P2.3 与本阶段同分支**：P2.3 依赖 P2.2，且其剩余量小（P0 已做运行时门控、`bootstrap.go:185` 已打印启用清单；`configs` 装配级回归已在 P2.1 完成）。
7. **I1 降级语义（终审裁定，已执行）**：`catalog.Degraded` 的报告是**声明层**的静态比对 —— 只比较 `Descriptor.SoftRequires` 与已解析启用集，**不观测运行时装配**。组合根当前仍无条件注入多数依赖（`cmd/server/main.go:176-178`/`:205`、`internal/app/container.go:217-220`，且 `main.go:182` 标注为「过渡实现（P0）」），故在组合根改为按启用集驱动（P1 显式 `Deps`）之前可能多报。**保留声明**（声明即契约，收窄到今天的中转装配会把实现残留固化成契约），改为把语义写进 `Degraded`/`Degradation` 文档，并在启动日志同时打印「已装配模块集」（`capabilities enabled`）与「已解析启用集」（`capabilities resolved`，键只用 `count`/`names`，避免 R3 告警）。
8. **I2 `access` → `tenant` 软依赖（终审裁定，已执行）**：`access` 的 `RoleService` 有真实的可选配额注入（`application/service.go:18` `quota TenantQuota // nil = 未启用租户配额`，`module.go:25-33` 可选注入），且 `tenant` 硬依赖 `access`，故只能声明为 `SoftRequires`。声明取值表与「真实 SoftRequires」枚举同步加入 `access`→`tenant`。

## Global Constraints

- **禁止自动提交**：commit 仅在用户明确指令后执行（AGENTS.md 最高优先级）
- **对外 YAML 键逐一不变**：本阶段不动配置段，`git diff configs/` 必须为 0 行
- **catalog 项数保持 18**：不新增/删除能力；`wiredCapabilities` 不动
- 能力之间只经 `contract` 端口调用；`internal/kernel/**` 不得 import `internal/capabilities/**`
- 全绿门禁：`gofmt -l .`、`go build ./...`、`go vet ./...`、`golangci-lint run ./...`、`make check-log-usage`、`make test-cover`+`test-coverage-check`、`make test-race`、`make swagger-check`、`make bench-ci`
- 分支从 `release/v0.3.0` 切出；完成后 PR 目标 `release/v0.3.0`（squash merge）

## 声明取值表（唯一事实来源，Task 2 直接照抄）

`Requires` 为现状**不变**；`SoftRequires` 只登记**代码里真实存在可选注入/nil 降级**的耦合；`Owns` 取自各能力迁移里 `CREATE TABLE` 的实际表名（13 个能力有表、5 个无表）。

| 能力 | Requires（不变） | SoftRequires（新增） | Owns（新增） | Soft 依据 |
|---|---|---|---|---|
| `user` | — | `access`, `tenant` | `users` | `admin_service.go:220 s.roles==nil`、`:146 s.quota!=nil`；且 access/tenant 硬依赖 user，反向硬声明成环 |
| `access` | `user` | `tenant` | `roles`, `permissions`, `role_permissions`, `user_roles` | `service.go:18 quota TenantQuota // nil = 未启用租户配额`；`module.go:25-33` 可选注入 |
| `tenant` | `user`, `access` | — | `tenants`, `tenant_plans` | — |
| `mfa` | `user` | `auth` | `user_mfa`, `trusted_devices` | auth 硬依赖 mfa，反向硬声明成环；JWT 参数由装配期从 auth 段传入 |
| `auth` | `user`, `access`, `tenant`, `mfa` | `captcha`, `breach` | `login_histories`, `password_histories` | `handler.go:280 h.captcha==nil` 跳过校验；`service.go:43 breachChecker nil = 未启用` |
| `passkey` | `user`, `auth` | — | `webauthn_credentials` | — |
| `audit` | — | — | `audit_logs`, `audit_chain_head` | 设计 §3.2 标 tenancy 可空，但代码只经 `kernel/tenant` 上下文、无能力端口耦合 → 不登记 |
| `console` | `auth`, `access` | — | — | — |
| `oauth` | `auth`, `user` | — | `user_oauth_bindings` | — |
| `apikey` | — | `tenant` | `api_keys` | `apikeys_service.go:57 if s.quota != nil`；`module.go:19 quota 可选` |
| `queue` | — | — | `jobs`, `job_history`, `dead_letters`, `scheduled_jobs` | — |
| `outbox` | — | `queue` | `outbox_events` | `publisher=mq` 才需要队列（组合根跨能力校验），`event_bus` 时降级 |
| `dataops` | — | — | `import_jobs` | 设计 §3.3 标 jobs/user，但模块只接 db → 不登记 |
| `search` | — | — | `search_documents` | — |
| `captcha` | — | — | — | 无表 |
| `feature` | — | — | — | 无表 |
| `uploadsec` | — | — | — | 无表；依赖 `storage`，但 storage 非 catalog，不可作为 SoftRequires 取值 |
| `breach` | — | — | — | 无表 |

**结构不变量**（Task 1 的校验与 Task 4 的门禁都按此判定）：
- `SoftRequires` 每项必须是 catalog 能力名；不得自引用；`SoftRequires ∩ Requires = ∅`
- `Requires` 决定的依赖图必须无环，且 catalog 清单顺序满足「依赖在前」（现有 `TestDescriptorsAreWellFormed` 已钉住）
- `Owns` 每张表在全部能力的迁移里**恰好被 CREATE 一次**，且只能由 `Owns` 声明者创建

---

## Task 1: 契约加 `SoftRequires` / `Owns` + catalog 声明校验

**Files:**
- Modify: `internal/contract/capability.go`
- Modify: `internal/capabilities/catalog/catalog.go`
- Test: `internal/contract/capability_test.go`、`internal/capabilities/catalog/catalog_test.go`

**Interfaces:**
- Produces: `contract.Descriptor.SoftRequires []string`、`contract.Descriptor.Owns []string`
- Produces: `catalog.ValidateDeclarations() error`
- Produces: `catalog.Degraded(caps []contract.Descriptor) []catalog.Degradation`（Task 3 用；本任务只定义类型与函数骨架亦可，但建议一并实现，见 Task 3 步骤说明——**本任务先只做校验**）

- [ ] **Step 1: 写失败测试（contract 字段存在 + catalog 校验拒绝非法声明）**

在 `internal/capabilities/catalog/catalog_test.go` 追加：

```go
// TestValidateDeclarationsRejectsBadSoftRequires 声明结构不合法必须报错。
func TestValidateDeclarationsRejectsBadSoftRequires(t *testing.T) {
	cases := []struct {
		name string
		ds   []contract.Descriptor
	}{
		{"unknown soft dep", []contract.Descriptor{{Name: "a", SoftRequires: []string{"ghost"}}}},
		{"self soft dep", []contract.Descriptor{{Name: "a", SoftRequires: []string{"a"}}}},
		{"soft overlaps hard", []contract.Descriptor{
			{Name: "b"},
			{Name: "a", Requires: []string{"b"}, SoftRequires: []string{"b"}},
		}},
		{"duplicate soft dep", []contract.Descriptor{{Name: "b"}, {Name: "a", SoftRequires: []string{"b", "b"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withEntries(t, tc.ds...)
			if err := ValidateDeclarations(); err == nil {
				t.Fatal("expected declaration error")
			}
		})
	}
}

// TestValidateDeclarationsAcceptsCurrentCatalog 真实清单必须通过声明校验。
func TestValidateDeclarationsAcceptsCurrentCatalog(t *testing.T) {
	if err := ValidateDeclarations(); err != nil {
		t.Fatalf("catalog declarations invalid: %v", err)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/capabilities/catalog/ -run TestValidateDeclarations -count=1`
Expected: FAIL —— `undefined: ValidateDeclarations`

- [ ] **Step 3: contract 加字段**

`internal/contract/capability.go` 的 `Descriptor` 增加（放在 `Requires` 之后、`Owns` 放在 `Migrations` 之前）：

```go
	// SoftRequires 可选依赖：目标能力缺失时本能力降级运行（不自动补齐、不参与拓扑序）。
	// 取值必须是 catalog 能力名，且不得与 Requires 重叠或自引用。
	SoftRequires []string

	// Owns 本能力拥有的表名；这些表只能由本能力的迁移 CREATE。
	// 空表示本能力不拥有表。
	Owns []string
```

- [ ] **Step 4: catalog 实现声明校验**

`internal/capabilities/catalog/catalog.go` 增加：

```go
// ValidateDeclarations 校验清单内的自描述声明是否自洽：
// SoftRequires 必须是清单内能力名、不得自引用、不得与 Requires 重叠、不得重复。
// 依赖图无环与「清单顺序满足依赖在前」由 TestDescriptorsAreWellFormed 钉住。
func ValidateDeclarations() error {
	known := make(map[string]bool, len(entries))
	for _, d := range entries {
		known[d.Name] = true
	}
	for _, d := range entries {
		hard := make(map[string]bool, len(d.Requires))
		for _, dep := range d.Requires {
			hard[dep] = true
		}
		seen := make(map[string]bool, len(d.SoftRequires))
		for _, dep := range d.SoftRequires {
			if !known[dep] {
				return fmt.Errorf("capability %q soft-requires unknown capability %q", d.Name, dep)
			}
			if dep == d.Name {
				return fmt.Errorf("capability %q soft-requires itself", d.Name)
			}
			if hard[dep] {
				return fmt.Errorf("capability %q declares %q in both Requires and SoftRequires", d.Name, dep)
			}
			if seen[dep] {
				return fmt.Errorf("capability %q duplicates soft requirement %q", d.Name, dep)
			}
			seen[dep] = true
		}
	}
	return nil
}
```

并在 `All()` 与 `Resolve()` 的深拷贝里补 `SoftRequires`/`Owns`：

```go
		out[i].SoftRequires = append([]string(nil), d.SoftRequires...)
		out[i].Owns = append([]string(nil), d.Owns...)
```

同时把 `Resolve` 开头改为先校验：

```go
func Resolve(enabled []string) ([]contract.Descriptor, error) {
	if err := ValidateDeclarations(); err != nil {
		return nil, err
	}
	...
```

- [ ] **Step 5: 同步测试夹具**

`catalog_test.go` 的 `fixture()` 现在有 18 项且被 `TestDescriptorsAreWellFormed` 与 `All()` 逐值比较：本任务字段全空，**无需要改**（Step 6 会验证）。新增深拷贝断言：

```go
func TestAllReturnsDeepCopyOfSoftRequiresAndOwns(t *testing.T) {
	withEntries(t, contract.Descriptor{
		Name:         "a",
		SoftRequires: []string{"b"},
		Owns:         []string{"t1"},
	}, contract.Descriptor{Name: "b"})
	got := All()
	got[0].SoftRequires[0] = "mutated"
	got[0].Owns[0] = "mutated"
	if All()[0].SoftRequires[0] != "b" || All()[0].Owns[0] != "t1" {
		t.Fatal("All() must not expose registry backing arrays")
	}
}
```

- [ ] **Step 6: 验证**

Run: `go test ./internal/contract/ ./internal/capabilities/catalog/ -count=1`
Expected: PASS（含既有 `TestDescriptorsAreWellFormed`）

- [ ] **Step 7: 提交**（用户授权后）`feat(contract): add soft requirements and owned tables to descriptors`

---

## Task 2: 18 个能力补齐 `SoftRequires` / `Owns` 声明

**Files:**
- Modify: `internal/capabilities/{user,access,tenant,mfa,auth,passkey,audit,console,oauth,apikey,queue,outbox,dataops,search,captcha,feature,uploadsec,breach}` 的 `Descriptor` 定义文件（`module.go` 或 `migrations.go`）
- Test: `internal/capabilities/catalog/catalog_test.go`（夹具）、各能力 `module_test.go`（仅新增字段断言，不重排）

**Interfaces:**
- Consumes: `contract.Descriptor.SoftRequires`/`Owns`（Task 1）
- Produces: 18 个能力的声明与「声明取值表」逐字一致

- [ ] **Step 1: 写失败测试（Owns 形态 + 软依赖形态）**

`catalog_test.go` 追加：

```go
// TestCatalogOwnsShape 钉住各能力拥有的表（空 = 不拥有表）。
func TestCatalogOwnsShape(t *testing.T) {
	want := map[string][]string{
		"user":    {"users"},
		"access":  {"roles", "permissions", "role_permissions", "user_roles"},
		"tenant":  {"tenants", "tenant_plans"},
		"mfa":     {"user_mfa", "trusted_devices"},
		"auth":    {"login_histories", "password_histories"},
		"passkey": {"webauthn_credentials"},
		"oauth":   {"user_oauth_bindings"},
		"apikey":  {"api_keys"},
		"queue":   {"jobs", "job_history", "dead_letters", "scheduled_jobs"},
		"outbox":  {"outbox_events"},
		"dataops": {"import_jobs"},
		"search":  {"search_documents"},
		"audit":   {"audit_logs", "audit_chain_head"},
	}
	got := map[string][]string{}
	for _, d := range All() {
		if len(d.Owns) > 0 {
			got[d.Name] = d.Owns
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog Owns drifted:\n got %v\nwant %v", got, want)
	}
}

// TestCatalogSoftRequiresShape 钉住可选依赖（只登记代码里真实可降级的耦合）。
func TestCatalogSoftRequiresShape(t *testing.T) {
	want := map[string][]string{
		"user":   {"access", "tenant"},
		"access": {"tenant"},
		"mfa":    {"auth"},
		"auth":   {"captcha", "breach"},
		"apikey": {"tenant"},
		"outbox": {"queue"},
	}
	got := map[string][]string{}
	for _, d := range All() {
		if len(d.SoftRequires) > 0 {
			got[d.Name] = d.SoftRequires
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog SoftRequires drifted:\n got %v\nwant %v", got, want)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/capabilities/catalog/ -run 'TestCatalogOwnsShape|TestCatalogSoftRequiresShape' -count=1`
Expected: FAIL —— `got map[]` 与期望不符

- [ ] **Step 3: 逐个补声明**

按「声明取值表」修改 18 个 `Descriptor` 字面量，例如 `internal/capabilities/auth/module.go`：

```go
var Descriptor = contract.Descriptor{
	Name:         "auth",
	Migrations:   migrationsFS,
	Requires:     []string{"user", "access", "tenant", "mfa"},
	SoftRequires: []string{"captcha", "breach"},
	Owns:         []string{"login_histories", "password_histories"},
	Mount:        contract.MountSelfManaged,
	Configs: []contract.ConfigSpec{
		{Section: ConfigKey, New: func() any { return &Config{} }},
	},
}
```

`internal/capabilities/queue/migrations.go`：

```go
var Descriptor = contract.Descriptor{
	Name:       "queue",
	Mount:      contract.MountProtected,
	Migrations: migrationsFS,
	Owns:       []string{"jobs", "job_history", "dead_letters", "scheduled_jobs"},
	Configs: []contract.ConfigSpec{
		{Section: ConfigKey, New: func() any { return &Config{} }},
		{Section: SchedulerConfigKey, New: func() any { return &SchedulerConfig{} }},
	},
}
```

其余 16 个照表逐字补齐（无表、无软依赖的能力显式留空，不写字面量）。

- [ ] **Step 4: 同步 `fixture()`**

`catalog_test.go` 的 `fixture()` 每一项补上同样的 `SoftRequires`/`Owns`（`TestDescriptorsAreWellFormed` 会逐值比较）。例如：

```go
		{Name: "auth", Requires: []string{"user", "access", "tenant", "mfa"},
			SoftRequires: []string{"captcha", "breach"},
			Owns:         []string{"login_histories", "password_histories"},
			Mount:        contract.MountSelfManaged},
```

- [ ] **Step 5: 各能力补最小断言（防漂移回到本地）**

在 `internal/capabilities/mfa/module_test.go` 的 `TestModuleDescriptor` 追加：

```go
	assert.Equal(t, []string{"auth"}, d.SoftRequires)
	assert.ElementsMatch(t, []string{"user_mfa", "trusted_devices"}, d.Owns)
```

对 `auth`、`user`、`apikey`、`outbox` 同样各加两行断言（值取「声明取值表」）。

- [ ] **Step 6: 验证**

Run: `go test ./internal/capabilities/... -count=1`
Expected: PASS

- [ ] **Step 7: 提交** `feat(capabilities): declare soft requirements and owned tables`

---

## Task 3: catalog 软依赖降级报告 + 启动日志

**Files:**
- Modify: `internal/capabilities/catalog/catalog.go`
- Modify: `internal/app/bootstrap.go`（启用清单日志后追加降级日志）
- Modify: `internal/app/container.go`（`Container` 持有 `Capabilities []contract.Descriptor`）
- Modify: `cmd/server/main.go`（`NewContainer` 传入 `caps`）
- Test: `internal/capabilities/catalog/catalog_test.go`

**Interfaces:**
- Consumes: `contract.Descriptor.SoftRequires`（Task 1）、`Container.Capabilities`
- Produces: `catalog.Degraded(caps []contract.Descriptor) []catalog.Degradation`
- Produces: `type catalog.Degradation struct { Capability string; Missing []string }`

- [ ] **Step 1: 写失败测试**

```go
// TestDegradedListsMissingSoftDeps 只报告缺失的软依赖，硬依赖缺失由 Resolve 报错。
func TestDegradedListsMissingSoftDeps(t *testing.T) {
	caps := []contract.Descriptor{
		{Name: "user", SoftRequires: []string{"access", "tenant"}, Owns: []string{"users"}},
		{Name: "auth", SoftRequires: []string{"captcha", "breach"}, Owns: []string{"login_histories"}},
		{Name: "captcha"},
		{Name: "access", Requires: []string{"user"}},
	}
	got := Degraded(caps)
	require.Len(t, got, 1)
	assert.Equal(t, "auth", got[0].Capability)
	assert.Equal(t, []string{"breach"}, got[0].Missing)
}

// TestDegradedEmptyWhenAllSoftDepsPresent 依赖齐全时无降级项。
func TestDegradedEmptyWhenAllSoftDepsPresent(t *testing.T) {
	caps := []contract.Descriptor{
		{Name: "auth", SoftRequires: []string{"captcha"}},
		{Name: "captcha"},
	}
	assert.Empty(t, Degraded(caps))
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/capabilities/catalog/ -run TestDegraded -count=1`
Expected: FAIL —— `undefined: Degraded`

- [ ] **Step 3: 实现 `Degraded`**

```go
// Degradation 描述一个能力因缺失软依赖而降级运行。
type Degradation struct {
	Capability string
	Missing    []string
}

// Degraded 计算已解析启用集里的降级项：SoftRequires 中不在集合内的目标。
// 软依赖不会自动补齐（设计 §6.4），缺失只降级、不报错。
func Degraded(caps []contract.Descriptor) []Degradation {
	present := make(map[string]bool, len(caps))
	for _, d := range caps {
		present[d.Name] = true
	}
	out := make([]Degradation, 0, len(caps))
	for _, d := range caps {
		var missing []string
		for _, dep := range d.SoftRequires {
			if !present[dep] {
				missing = append(missing, dep)
			}
		}
		if len(missing) > 0 {
			out = append(out, Degradation{Capability: d.Name, Missing: missing})
		}
	}
	return out
}
```

- [ ] **Step 4: `Container` 持有启用集描述符并在启动日志输出降级**

`internal/app/container.go`：`Container` 增字段 `Capabilities []contract.Descriptor`；`NewContainer` 签名在 `capCfgs` 后加 `caps []contract.Descriptor`，并写入结构体。`cmd/server/main.go` 调用改为 `app.NewContainer(cfg, sections, capCfgs, caps, enabled)`。

`internal/app/bootstrap.go` 在 `capabilities enabled` 日志后追加：

```go
	for _, d := range catalog.Degraded(container.Capabilities) {
		container.Logger.Warnw("capability degraded", "name", d.Capability, "missing", strings.Join(d.Missing, ","))
	}
```

（`bootstrap.go` 需 import `jimu/internal/capabilities/catalog`。执行记录：原稿的 `"missing"` 键当时不在 `tools/logcheck` 标准词汇表（R3），实现先改用已登记的 `"names"`；经确认后反过来决定**登记**该 key —— `tools/logcheck` 词汇表新增 `"missing": "缺失项清单（如缺失的可选依赖名）"`，日志改用更精确的 `"missing"`。）

- [ ] **Step 5: 验证**

Run: `go build ./... && go test ./internal/app/ ./internal/capabilities/catalog/ -count=1`
Expected: PASS

- [ ] **Step 6: 提交** `feat(catalog): report degraded capabilities for missing soft requirements`

---

## Task 4: 最小 `check-capabilities` 门禁（校验 Owns ↔ 迁移，防装饰性声明）

**Files:**
- Create: `tools/checkcapabilities/main.go`
- Create: `tools/checkcapabilities/main_test.go`
- Modify: `Makefile`（新增 `check-capabilities` 目标；**暂不接 CI**，P2.8 统一接入）

**Interfaces:**
- Consumes: `catalog.ValidateDeclarations()`、`catalog.All()`（含 `Migrations fs.FS`）
- Produces: `make check-capabilities`

- [ ] **Step 1: 写失败测试（表归属判定是纯函数，先测它）**

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOwnershipViolations(t *testing.T) {
	// 声明了 Owns 但迁移里没建 → 违规
	declared := map[string][]string{"user": {"users"}}
	created := map[string][]string{} // cap -> tables
	require.Error(t, checkOwnership(declared, created))

	// 迁移建了但没声明 → 违规
	declared = map[string][]string{}
	created = map[string][]string{"user": {"users"}}
	require.Error(t, checkOwnership(declared, created))

	// 两张能力都声明同一张表 → 违规
	declared = map[string][]string{"user": {"users"}, "access": {"users"}}
	created = map[string][]string{"user": {"users"}, "access": {"users"}}
	require.Error(t, checkOwnership(declared, created))

	// 一致 → 通过
	declared = map[string][]string{"user": {"users"}}
	created = map[string][]string{"user": {"users"}}
	assert.NoError(t, checkOwnership(declared, created))
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./tools/checkcapabilities/ -count=1`
Expected: FAIL —— `undefined: checkOwnership`

- [ ] **Step 3: 实现门禁**

`tools/checkcapabilities/main.go`：

```go
// Command checkcapabilities 校验能力自描述与实际迁移一致（P2.8 门禁的第一块）。
// 当前范围：Owns 的表必须由且仅由该能力的迁移 CREATE（表归属唯一）。
// 完整门禁（跨能力 import 一致性、internal 越界、kernel→capabilities 反向依赖）留待 P2.8。
package main

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"

	"jimu/internal/capabilities/catalog"
)

var createTableRe = regexp.MustCompile(`(?i)CREATE TABLE(?: IF NOT EXISTS)?\s+[` + "`" + `"]?([a-z_]+)`)

// checkOwnership 比较「声明拥有的表」与「迁移实际建的表」，返回违规说明。
func checkOwnership(declared, created map[string][]string) error {
	owner := map[string]string{} // 表 -> 能力
	for capName, tables := range declared {
		for _, t := range tables {
			if prev, dup := owner[t]; dup {
				return fmt.Errorf("table %q owned by both %q and %q", t, prev, capName)
			}
			owner[t] = capName
		}
	}
	for capName, tables := range created {
		for _, t := range tables {
			if owner[t] != capName {
				return fmt.Errorf("table %q created by %q but owned by %q", t, capName, owner[t])
			}
		}
	}
	for t, capName := range owner {
		if !contains(created[capName], t) {
			return fmt.Errorf("capability %q declares Owns %q but its migrations do not create it", capName, t)
		}
	}
	return nil
}

func main() {
	if err := catalog.ValidateDeclarations(); err != nil {
		fmt.Fprintln(os.Stderr, "❌ check-capabilities:", err)
		os.Exit(1)
	}
	declared := map[string][]string{}
	created := map[string][]string{}
	for _, d := range catalog.All() {
		if len(d.Owns) > 0 {
			declared[d.Name] = d.Owns
		}
		tables, err := createdTables(d)
		if err != nil {
			fmt.Fprintln(os.Stderr, "❌ check-capabilities:", err)
			os.Exit(1)
		}
		if len(tables) > 0 {
			created[d.Name] = tables
		}
	}
	if err := checkOwnership(declared, created); err != nil {
		fmt.Fprintln(os.Stderr, "❌ check-capabilities:", err)
		os.Exit(1)
	}
	fmt.Println("✅ check-capabilities: 能力自描述与迁移归属一致")
}

// createdTables 从能力嵌入的 mysql 迁移里提取 CREATE TABLE 的表名（去重排序）。
func createdTables(d contract.Descriptor) ([]string, error) {
	if d.Migrations == nil {
		return nil, nil
	}
	set := map[string]bool{}
	err := fs.WalkDir(d.Migrations, "migrations/mysql", func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || !strings.HasSuffix(path, ".sql") {
			return err
		}
		b, err := fs.ReadFile(d.Migrations, path)
		if err != nil {
			return err
		}
		for _, m := range createTableRe.FindAllStringSubmatch(string(b), -1) {
			set[m[1]] = true
		}
		return nil
	})
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, err
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
```

（`createdTables` 的形参类型写 `contract.Descriptor`，需 import `jimu/internal/contract`。）

`Makefile` 增加：

```make
## check-capabilities: 校验能力自描述（Owns）与迁移归属一致（P2.8 门禁第一块）
check-capabilities:
	@go run ./tools/checkcapabilities
```

- [ ] **Step 4: 验证（含真实清单必须通过）**

Run: `go test ./tools/checkcapabilities/ -count=1 && make check-capabilities`
Expected: 单测 PASS；`✅ check-capabilities: 能力自描述与迁移归属一致`

- [ ] **Step 5: 提交** `feat(tools): validate owned tables against capability migrations`

---

## Task 5: P2.3 运行时降级验证 + 启用清单管理端点

**Files:**
- Modify: `internal/kernel/http/management.go`（`HealthRouter` 加可变注册器）
- Modify: `internal/app/bootstrap.go`（注册 `/capabilities`）
- Create: `internal/app/capabilities_handler_test.go`
- Test: `internal/capabilities/catalog/catalog_test.go`（P2.3 降级逐条）

**Interfaces:**
- Consumes: `catalog.Degraded`（Task 3）、`Container.Capabilities`（Task 3）
- Produces: `GET /capabilities`（管理端口）返回 `{"enabled":[...],"degraded":[{"capability":"auth","missing":["breach"]}]}`

- [ ] **Step 1: 写失败测试（catalog 降级逐条 + 端点 JSON）**

`catalog_test.go` 用真实清单钉住「auth 缺 captcha/breach」「outbox 缺 queue」「apikey 缺 tenant」的降级行为：

```go
// TestDegradedWithRealCatalog 关闭软依赖后对应能力出现在降级清单里。
func TestDegradedWithRealCatalog(t *testing.T) {
	// 只启用 auth 的硬依赖闭包，captcha/breach 缺席
	caps, err := Resolve([]string{"auth"})
	require.NoError(t, err)
	got := Degraded(caps)
	require.Len(t, got, 1, "只有 auth 声明了缺失的软依赖")
	assert.Equal(t, "auth", got[0].Capability)
	assert.ElementsMatch(t, []string{"captcha", "breach"}, got[0].Missing)
}

// TestDegradedNoneOnFullCatalog 全部启用（默认）时无降级项。
func TestDegradedNoneOnFullCatalog(t *testing.T) {
	assert.Empty(t, Degraded(All()))
}
```

`internal/app/capabilities_handler_test.go`：

```go
package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/contract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilitiesHandlerReportsEnabledAndDegraded(t *testing.T) {
	caps := []contract.Descriptor{
		{Name: "user", Requires: []string{}, SoftRequires: []string{"access"}},
		{Name: "auth", SoftRequires: []string{"captcha"}},
	}
	rec := httptest.NewRecorder()
	capabilitiesHandler(caps)(rec, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Enabled  []string `json:"enabled"`
		Degraded []struct {
			Capability string   `json:"capability"`
			Missing    []string `json:"missing"`
		} `json:"degraded"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, []string{"user", "auth"}, body.Enabled)
	require.Len(t, body.Degraded, 2)
	assert.Equal(t, "auth", body.Degraded[1].Capability)
	assert.Equal(t, []string{"captcha"}, body.Degraded[1].Missing)
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/app/ -run TestCapabilitiesHandler -count=1`
Expected: FAIL —— `undefined: capabilitiesHandler`

- [ ] **Step 3: 实现端点（内核不 import capabilities）**

`internal/kernel/http/management.go` 的 `HealthRouter` 改为可变注册器：

```go
// HealthRouter 组装管理端口路由。extra 用于注入能力清单等由组合根提供的路由，
// 避免 kernel 反向 import capabilities。
func HealthRouter(readiness *observability.Readiness, enablePprof bool, extra ...func(*http.ServeMux)) http.Handler {
	mux := http.NewServeMux()
	observability.RegisterHealth(mux, readiness)
	mux.Handle("/metrics", promhttp.Handler())
	for _, register := range extra {
		register(mux)
	}
	if enablePprof {
		... // 原样保留
	}
	return mux
}
```

`internal/app/bootstrap.go` 增加 handler 与注册：

```go
// capabilitiesHandler 输出最终启用清单与降级项（设计 §6.4）。
func capabilitiesHandler(caps []contract.Descriptor) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		names := make([]string, 0, len(caps))
		for _, d := range caps {
			names = append(names, d.Name)
		}
		degraded := catalog.Degraded(caps)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"enabled": names, "degraded": degraded})
	}
}
```

（执行记录：原稿的 `map[string]any` 已改为按字段声明顺序编码的结构体 `capabilitiesResponse{Enabled, Degraded}` —— map 的键按字母序 `degraded`/`enabled` 序列化，与字节钉住的契约不符。）

`HealthRouter` 调用改为：

```go
		platformhttp.HealthRouter(readiness, cfg.Management.EnablePprof, func(mux *http.ServeMux) {
			mux.HandleFunc("/capabilities", capabilitiesHandler(container.Capabilities))
		}),
```

（`bootstrap.go` 需 import `encoding/json`、`net/http`。）

- [ ] **Step 4: 验证（含既有运行时门控回归）**

Run: `go test ./internal/app/... ./internal/capabilities/catalog/ ./internal/kernel/http/ -count=1`
Expected: PASS

- [ ] **Step 5: 提交** `feat(app): expose enabled and degraded capabilities on the management port`

---

## Task 6: 收口 —— 全量门禁与文档

**Files:**
- Modify: `README.md`（能力清单章节：`SoftRequires`/`Owns` 语义、`make check-capabilities`、`/capabilities` 管理端点）
- Modify: `docs/design/2026-09-18-capability-plugins-design.md`（§10 标记 P2.2/P2.3 完成）
- Modify: `docs/plans/2026-09-21-p2-three-layer-mechanism.md`（P2.2/P2.3 状态与裁定记录）
- Modify: `docs/releases/v0.3.0.md`（变更条目 + 验证结果）

- [ ] **Step 1: 文档更新**（上述四处，键与语义照 Task 1–5 的最终实现写）
- [ ] **Step 2: 全量回归**

Run: `gofmt -l . && go build ./... && go vet ./... && golangci-lint run ./... && make check-log-usage && make test-cover && make test-coverage-check && make test-race && make swagger-check && make bench-ci && make check-capabilities`
Expected: 全部通过；`git diff configs/` 为 0 行

- [ ] **Step 3: 提交** `docs(capabilities): record soft requirements and owned tables`

---

## Self-Review

**1. Spec 覆盖**：§6.1 的 `SoftRequires`/`Owns` 落地（`Tags` 经确认推迟）；§6.4「启用闭包 + 关闭能力不生效」在 P0 已建，本计划补 SoftRequires 降级与启用清单可见；§9 的 `check-capabilities` 落地第一块（Owns↔迁移），完整四道门禁留 P2.8；§10 的 P2.2「catalog 闭包扩展」以「硬依赖闭包不变 + 软依赖降级报告」实现。

**2. 占位符扫描**：无 TBD；每个 Task 给出文件、接口、测试代码与验证命令。

**3. 类型一致性**：`contract.Descriptor.SoftRequires/Owns`、`catalog.ValidateDeclarations`、`catalog.Degraded`/`catalog.Degradation`、`Container.Capabilities`、`capabilitiesHandler`、`checkOwnership` 在所有 Task 间命名与签名一致。

**4. 风险**：
- `Container.Capabilities` 是容器新字段，`NewContainer` 签名变更的唯一调用点是 `cmd/server/main.go:111`（无测试调用点），Task 3 Step 5 的编译会兜住。
- `HealthRouter` 改可变参数：调用点仅 `bootstrap.go:198` 与 `internal/kernel/http/management_test.go:35`，变参向后兼容，预期零改动。
- `internal/capabilities/catalog/catalog_test.go` 目前未 import testify；Task 1/2/3/5 新增用例用到 `require`/`assert`，需在该文件补 `github.com/stretchr/testify/{assert,require}` 导入（或用 `t.Fatalf` 改写）。
- `check-capabilities` 用正则提取 `CREATE TABLE`，对 `ALTER TABLE ... ADD` 不敏感（Owns 只认 CREATE，符合裁定）；PostgreSQL 侧不单独扫描（两方言表名一致，mysql 为准）。
