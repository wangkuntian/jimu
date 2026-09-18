# 能力可插拔 P0：契约与运行时开关 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在**不改变现有行为**的前提下，建立能力契约（`contract.Descriptor`）、全仓唯一能力清单（`catalog`）与运行时开关（`capabilities.enabled` + 依赖闭包校验），并去掉 `bootstrap` 里按能力名硬编码的挂载点特判。

**Architecture:** 每个能力包导出静态 `Descriptor`（名称 / 硬依赖 / 挂载点）；`internal/capabilities/catalog` 聚合这些描述符并提供 `Resolve(enabled)`，完成「未知能力报错 → 依赖闭包补齐 → 按清单顺序返回」；`cmd/server/main.go` 用解析结果过滤传给 `app.Bootstrap` 的能力切片；`registerHTTP` 只按 `Descriptor().Mount` 决策，不再看能力名。

**Tech Stack:** Go 1.26 · Gin · GORM · Viper · Zap · testify

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md`（§3.1 内核、§6.1 能力自描述、§6.4 层③运行时、§10 P0）

## Global Constraints

- **对外行为不变**：HTTP 路由、响应、配置键名、数据库 schema 与 master 完全一致；默认配置下 `full` 行为逐字节等价
- **不新增第三方依赖**
- **提交信息全英文**（Conventional Commits，`githooks/commit-msg` + CI 强制，禁止 CJK）
- **提交需用户明确指令**（本仓库 AGENTS.md 禁止自动提交）：计划中的 commit 步骤标注为「（用户授权后执行）」
- 分支：从 `release/v0.3.0` 切出 `feature/<issue>-capability-contract`，PR 目标 `release/v0.3.0`，squash 合并
- 每个任务结束必须通过：`gofmt -l .`（无输出）、`go build ./...`、`go vet ./...`、该任务的测试
- **简单优先**：P0 只做契约 + 清单 + 开关 + 挂载点；不动目录结构、不动迁移、不改各能力内部实现

## 本计划与 spec 的一处差异（需确认）

spec §10 的 P0 写有「建立 `internal/kernel/`；`internal/capabilities/` 下按现有 8 模块原样落位」。**本计划推迟这次目录搬迁**，理由：`internal/platform/` 的 29 个包并非全部属于内核 —— `captcha`/`breach`/`oauth`/`notification`/`storage`/`search`/`importer`/`exporter`/`ws`/`feature` 在 §3.4/§3.6 中归属**能力**，整目录改名为 `kernel/` 会先搬错再拆一次；逐包归位与 §3.6 的混装包拆分同属 P1，放在一起只需改一次 import 路径。目录搬迁移至 P1 第一个任务组。

## 文件结构

| 文件 | 职责 |
|---|---|
| `internal/contract/capability.go`（新建） | `MountPoint`、`Descriptor`、`Describable`、`Describe()` |
| `internal/contract/capability_test.go`（新建） | 挂载点归一化、`Describe` 对未实现者的回退 |
| `internal/capabilities/catalog/catalog.go`（新建） | **唯一能力清单** + `Resolve` 闭包解析 + `Names` |
| `internal/capabilities/catalog/catalog_test.go`（新建） | 表驱动：空=全部、未知报错、依赖补齐、顺序稳定、缺失依赖报错 |
| `internal/modules/<8>/module.go`（改） | 各导出 `Descriptor` 并实现 `Descriptor()` |
| `internal/config/config.go`（改） | `CapabilitiesConfig` + `Config.Capabilities` |
| `internal/config/validate.go`（改） | 能力名非空、去重校验 |
| `internal/config/config_test.go`（改） | YAML 映射与校验用例 |
| `configs/app.yaml`（改） | `capabilities.enabled` 段（注释说明空=全部） |
| `internal/app/bootstrap.go`（改） | `registerHTTP` 按 `Mount()` 决策；启动打印启用清单 |
| `internal/app/bootstrap_http_test.go`（新建） | 挂载点行为测试（受保护中间件只作用于 `MountProtected`） |
| `cmd/server/main.go`（改） | 按 `catalog.Resolve` 过滤能力 |
| `README.md`（改） | 新增「能力开关」章节 |
| `docs/releases/v0.3.0.md`（改） | P0 条目与验证 |

## 当前依赖图（`Requires` 的事实依据）

由 `internal/modules/*` 的 import 关系实测得出，P0 按此声明硬依赖；P1 会细化为 `SoftRequires`（例如 `auth` 对 `tenant` 仅在开通式注册开启时才需要）：

| 能力 | Requires | Mount |
|---|---|---|
| `user` | — | `MountProtected` |
| `role` | — | `MountProtected` |
| `permission` | `role` | `MountProtected` |
| `tenant` | — | `MountProtected` |
| `auth` | `user` `role` `tenant` | `MountSelfManaged`（内部自建受保护子组 + 公开子组） |
| `audit` | — | `MountProtected` |
| `admin` | `user` `audit` | `MountProtected`（组内自带 IP 白名单 + `AdminAuth`） |
| `oauth` | `auth` `user` | `MountPublic`（登录/回调，无鉴权） |

---

### Task 1: 能力描述契约

**Files:**
- Create: `internal/contract/capability.go`
- Test: `internal/contract/capability_test.go`

**Interfaces:**
- Consumes: 现有 `contract.Module`（`internal/contract/module.go`）
- Produces: `contract.MountPoint`（`MountPublic`/`MountProtected`/`MountSelfManaged`）、`contract.Descriptor{Name string; Requires []string; Mount MountPoint}`、`func (Descriptor) Normalized() MountPoint`、`type Describable interface{ Descriptor() Descriptor }`、`func Describe(m Module) Descriptor`

- [ ] **Step 1: 写失败测试**

创建 `internal/contract/capability_test.go`：

```go
package contract

import "testing"

// stubModule 只实现 Module 接口，用于验证 Describe 的回退行为。
type stubModule struct{ name string }

func (s stubModule) Name() string          { return s.name }
func (s stubModule) RegisterHTTP(r Router) {}
func (s stubModule) RegisterJobs(j JobRegistry) {}
func (s stubModule) RegisterEvents(e EventBus) {}

// describableStub 额外实现 Describable。
type describableStub struct {
	stubModule
	desc Descriptor
}

func (d describableStub) Descriptor() Descriptor { return d.desc }

func TestDescriptorNormalizedDefaultsToProtected(t *testing.T) {
	if got := (Descriptor{Name: "user"}).Normalized(); got != MountProtected {
		t.Fatalf("zero value mount = %q, want %q", got, MountProtected)
	}
	if got := (Descriptor{Name: "oauth", Mount: MountPublic}).Normalized(); got != MountPublic {
		t.Fatalf("explicit mount = %q, want %q", got, MountPublic)
	}
}

func TestDescribeFallsBackWhenNotDescribable(t *testing.T) {
	got := Describe(stubModule{name: "legacy"})
	if got.Name != "legacy" {
		t.Fatalf("name = %q, want %q", got.Name, "legacy")
	}
	if len(got.Requires) != 0 {
		t.Fatalf("requires = %v, want empty", got.Requires)
	}
	if got.Normalized() != MountProtected {
		t.Fatalf("mount = %q, want %q", got.Normalized(), MountProtected)
	}
}

func TestDescribeUsesDeclaredDescriptor(t *testing.T) {
	want := Descriptor{Name: "auth", Requires: []string{"user", "role", "tenant"}, Mount: MountSelfManaged}
	got := Describe(describableStub{stubModule: stubModule{name: "auth"}, desc: want})
	if got.Name != want.Name || got.Normalized() != want.Mount || len(got.Requires) != 3 {
		t.Fatalf("Describe() = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/contract/ -run 'TestDescriptor|TestDescribe' -v`
Expected: FAIL —— `undefined: Descriptor` / `undefined: MountProtected`

- [ ] **Step 3: 实现合约类型**

创建 `internal/contract/capability.go`：

```go
package contract

// MountPoint 决定能力的 HTTP 路由挂载方式。
type MountPoint string

const (
	// MountPublic 挂在 /api/v1 上且不套用受保护中间件（登录、回调、验证码等公开端点）。
	MountPublic MountPoint = "public"
	// MountProtected 挂在 /api/v1 上并套用受保护中间件（认证 + 租户注入 + 限流）。
	MountProtected MountPoint = "protected"
	// MountSelfManaged 由能力自行决定挂载与中间件（如 auth 内部同时有公开与受保护子组）。
	MountSelfManaged MountPoint = "self-managed"
)

// Descriptor 能力对外的静态描述：用于启用闭包校验与路由挂载决策。
// 静态声明（而非运行时推断）是"删除能力后仍能编译"的前提。
type Descriptor struct {
	Name     string     // 能力名，全仓唯一
	Requires []string   // 硬依赖：启用本能力必须同时启用这些能力
	Mount    MountPoint // 路由挂载方式；零值等价 MountProtected
}

// Normalized 返回归一化后的挂载点，空值按 MountProtected 处理。
func (d Descriptor) Normalized() MountPoint {
	if d.Mount == "" {
		return MountProtected
	}
	return d.Mount
}

// Describable 由能力实现以声明自身描述。
type Describable interface {
	Descriptor() Descriptor
}

// Describe 读取能力描述；未实现 Describable 时回退为"仅名称 + 受保护挂载"。
func Describe(m Module) Descriptor {
	if d, ok := m.(Describable); ok {
		return d.Descriptor()
	}
	return Descriptor{Name: m.Name()}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/contract/ -run 'TestDescriptor|TestDescribe' -v`
Expected: PASS（3 个用例）

- [ ] **Step 5: 提交（用户授权后执行）**

```bash
git add internal/contract/capability.go internal/contract/capability_test.go
git commit -m "feat(contract): add capability descriptor with mount points"
```

---

### Task 2: 能力清单与启用闭包解析

**Files:**
- Create: `internal/capabilities/catalog/catalog.go`
- Create: `internal/capabilities/catalog/catalog_test.go`

**Interfaces:**
- Consumes: Task 1 的 `contract.Descriptor`；Task 3 的各能力 `Descriptor` 变量（本任务先只写解析逻辑，清单在 Task 3 填齐）
- Produces: `catalog.All() []contract.Descriptor`、`catalog.Names() []string`、`catalog.Resolve(enabled []string) ([]contract.Descriptor, error)`

- [ ] **Step 1: 写失败测试**

创建 `internal/capabilities/catalog/catalog_test.go`（数据依赖的用例使用**测试夹具**，因此本任务结束时全绿，不依赖 Task 3）：

```go
package catalog

import (
	"strings"
	"testing"

	"jimu/internal/contract"
)

// withEntries 用测试夹具临时替换能力清单，测试结束自动恢复。
func withEntries(t *testing.T, ds ...contract.Descriptor) {
	t.Helper()
	old := entries
	entries = ds
	t.Cleanup(func() { entries = old })
}

// fixture 复刻 P0 八个能力的依赖形态（与真实清单拓扑一致）。
func fixture() []contract.Descriptor {
	return []contract.Descriptor{
		{Name: "user", Mount: contract.MountProtected},
		{Name: "role", Mount: contract.MountProtected},
		{Name: "permission", Requires: []string{"role"}, Mount: contract.MountProtected},
		{Name: "tenant", Mount: contract.MountProtected},
		{Name: "auth", Requires: []string{"user", "role", "tenant"}, Mount: contract.MountSelfManaged},
		{Name: "audit", Mount: contract.MountProtected},
		{Name: "admin", Requires: []string{"user", "audit"}, Mount: contract.MountProtected},
		{Name: "oauth", Requires: []string{"auth", "user"}, Mount: contract.MountPublic},
	}
}

func TestResolveEmptyMeansAll(t *testing.T) {
	withEntries(t, fixture()...)
	got, err := Resolve(nil)
	if err != nil {
		t.Fatalf("Resolve(nil) error: %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("len = %d, want 8 (all)", len(got))
	}
}

func TestResolveUnknownCapability(t *testing.T) {
	withEntries(t, fixture()...)
	_, err := Resolve([]string{"nope"})
	if err == nil {
		t.Fatal("expected error for unknown capability")
	}
	if !strings.Contains(err.Error(), "unknown capability") {
		t.Fatalf("error = %v, want unknown capability", err)
	}
}

func TestResolvePullsDependenciesAndKeepsOrder(t *testing.T) {
	withEntries(t, fixture()...)
	// permission 依赖 role，role 无依赖；顺序必须与清单一致（依赖在前）
	got, err := Resolve([]string{"permission"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if len(got) != 2 || got[0].Name != "role" || got[1].Name != "permission" {
		t.Fatalf("Resolve([permission]) = %v, want [role permission]", namesOf(got))
	}
}

func TestResolveClosureIsTransitive(t *testing.T) {
	withEntries(t, fixture()...)
	// oauth -> auth -> user/role/tenant
	got, err := Resolve([]string{"oauth"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	want := map[string]bool{"oauth": true, "auth": true, "user": true, "role": true, "tenant": true}
	if len(got) != len(want) {
		t.Fatalf("Resolve([oauth]) = %v, want %d entries", namesOf(got), len(want))
	}
	for _, d := range got {
		if !want[d.Name] {
			t.Fatalf("unexpected capability %q in closure %v", d.Name, namesOf(got))
		}
	}
}

func TestResolveIsSubsetOfAll(t *testing.T) {
	withEntries(t, fixture()...)
	got, err := Resolve([]string{"user"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "user" {
		t.Fatalf("Resolve([user]) = %v, want [user]", namesOf(got))
	}
}

func TestResolveRejectsUnknownDependency(t *testing.T) {
	withEntries(t, contract.Descriptor{Name: "broken", Requires: []string{"ghost"}})
	_, err := Resolve([]string{"broken"})
	if err == nil || !strings.Contains(err.Error(), `requires unknown capability "ghost"`) {
		t.Fatalf("error = %v, want requires unknown capability", err)
	}
}

// TestDescriptorsAreWellFormed 同时校验夹具与真实清单。Task 3 填齐清单后，
// 该用例对 catalog 的 8 个条目同样生效。
func TestDescriptorsAreWellFormed(t *testing.T) {
	lists := map[string][]contract.Descriptor{"fixture": fixture(), "catalog": All()}
	for label, list := range lists {
		known := make(map[string]bool, len(list))
		for _, d := range list {
			known[d.Name] = true
		}
		for _, d := range list {
			if d.Name == "" {
				t.Fatalf("%s: capability has empty name", label)
			}
			for _, dep := range d.Requires {
				if !known[dep] {
					t.Fatalf("%s: capability %q requires unknown capability %q", label, d.Name, dep)
				}
			}
			switch d.Normalized() {
			case contract.MountPublic, contract.MountProtected, contract.MountSelfManaged:
			default:
				t.Fatalf("%s: capability %q has invalid mount %q", label, d.Name, d.Mount)
			}
		}
	}
}

func namesOf(ds []contract.Descriptor) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return out
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/capabilities/catalog/ -v`
Expected: FAIL —— 包不存在（`no Go files` / `undefined: Resolve`）

- [ ] **Step 3: 实现清单与解析**

创建 `internal/capabilities/catalog/catalog.go`（**本步先建骨架**，`entries` 先空，Task 3 填齐后 `TestDescriptorsAreWellFormed` 才会真正覆盖真实清单）：

```go
// Package catalog 维护全仓库唯一的能力清单与启用集解析。
//
// 新增能力：在 entries 中追加一行（位置必须在它的依赖之后）。
// 删除能力：删掉该行与对应目录，其余代码无需改动 —— 这是"可插拔"的中心点。
package catalog

import (
	"fmt"
	"strings"

	"jimu/internal/contract"
)

// entries 是唯一的能力清单，顺序即默认启用顺序（同时是依赖拓扑序）。
var entries []contract.Descriptor

// All 返回清单中的全部能力描述（副本，调用方修改不影响清单）。
func All() []contract.Descriptor {
	return append([]contract.Descriptor(nil), entries...)
}

// Names 返回清单中的能力名，按清单顺序。
func Names() []string {
	out := make([]string, 0, len(entries))
	for _, d := range entries {
		out = append(out, d.Name)
	}
	return out
}

// Resolve 解析启用集：enabled 为空表示全部启用（向后兼容默认配置）；
// 未知能力报错；硬依赖自动补齐闭包；返回结果按清单顺序排列。
func Resolve(enabled []string) ([]contract.Descriptor, error) {
	if len(enabled) == 0 {
		return All(), nil
	}
	byName := make(map[string]contract.Descriptor, len(entries))
	for _, d := range entries {
		byName[d.Name] = d
	}
	on := make(map[string]bool, len(enabled))
	for _, name := range enabled {
		if _, ok := byName[name]; !ok {
			return nil, fmt.Errorf("unknown capability %q, available: %s", name, strings.Join(Names(), ", "))
		}
		on[name] = true
	}
	// 依赖闭包：反复补齐直到不再变化，保证传递依赖也被纳入。
	for changed := true; changed; {
		changed = false
		for name := range on {
			for _, dep := range byName[name].Requires {
				if _, ok := byName[dep]; !ok {
					return nil, fmt.Errorf("capability %q requires unknown capability %q", name, dep)
				}
				if !on[dep] {
					on[dep] = true
					changed = true
				}
			}
		}
	}
	out := make([]contract.Descriptor, 0, len(on))
	for _, d := range entries {
		if on[d.Name] {
			out = append(out, d)
		}
	}
	return out, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/capabilities/catalog/ -v`
Expected: PASS（7 个用例全绿；数据依赖用例使用测试夹具，不依赖 Task 3 的真实清单）

- [ ] **Step 5: 提交（用户授权后执行）**

```bash
git add internal/capabilities/catalog/
git commit -m "feat(catalog): add capability registry with dependency closure"
```

---

### Task 3: 八个能力声明各自描述符

**Files:**
- Modify: `internal/modules/user/module.go`、`role/module.go`、`permission/module.go`、`tenant/module.go`、`auth/module.go`、`audit/module.go`、`admin/module.go`、`oauth/module.go`
- Modify: `internal/capabilities/catalog/catalog.go`（填 `entries`）

**Interfaces:**
- Consumes: Task 1 的 `contract.Descriptor`、Task 2 的 `catalog.entries`
- Produces: 每个能力包导出 `var Descriptor contract.Descriptor` 并实现 `func (m *Module) Descriptor() contract.Descriptor`

- [ ] **Step 1: 为每个能力加描述符**

以 `internal/modules/user/module.go` 为例（其余七个同构，只改名称/依赖/挂载点，值取自本计划「当前依赖图」）：

```go
// Descriptor 声明用户能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:  "user",
	Mount: contract.MountProtected,
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }
```

各能力的取值：

```go
// role
var Descriptor = contract.Descriptor{Name: "role", Mount: contract.MountProtected}
// permission
var Descriptor = contract.Descriptor{Name: "permission", Requires: []string{"role"}, Mount: contract.MountProtected}
// tenant
var Descriptor = contract.Descriptor{Name: "tenant", Mount: contract.MountProtected}
// auth
var Descriptor = contract.Descriptor{Name: "auth", Requires: []string{"user", "role", "tenant"}, Mount: contract.MountSelfManaged}
// audit
var Descriptor = contract.Descriptor{Name: "audit", Mount: contract.MountProtected}
// admin
var Descriptor = contract.Descriptor{Name: "admin", Requires: []string{"user", "audit"}, Mount: contract.MountProtected}
// oauth
var Descriptor = contract.Descriptor{Name: "oauth", Requires: []string{"auth", "user"}, Mount: contract.MountPublic}
```

- [ ] **Step 2: 填齐 catalog 清单**

修改 `internal/capabilities/catalog/catalog.go` 的 import 与 `entries`（顺序即拓扑序：依赖在前）：

```go
import (
	"fmt"
	"strings"

	"jimu/internal/contract"
	adminmodule "jimu/internal/modules/admin"
	auditmodule "jimu/internal/modules/audit"
	authmodule "jimu/internal/modules/auth"
	oauthmodule "jimu/internal/modules/oauth"
	"jimu/internal/modules/permission"
	"jimu/internal/modules/role"
	tenantmodule "jimu/internal/modules/tenant"
	"jimu/internal/modules/user"
)

var entries = []contract.Descriptor{
	user.Descriptor,
	role.Descriptor,
	permission.Descriptor,
	tenantmodule.Descriptor,
	authmodule.Descriptor,
	auditmodule.Descriptor,
	adminmodule.Descriptor,
	oauthmodule.Descriptor,
}
```

> 注意：若某个模块包名与变量名冲突（例如包 `user` 与包内变量 `Descriptor`），保持 `包名.Descriptor` 形式即可；只有包名与 Go 关键字或本地变量冲突时才需要别名。
>
> **顺序是硬要求**：`Resolve` 不做拓扑排序，只按 `entries` 顺序输出（Task 2 评审确认）。必须保持「依赖在前」，即 `user`/`role` → `permission` → `tenant` → `auth` → `audit` → `admin` → `oauth`，与 Task 2 测试夹具 `fixture()` 的顺序一致。

- [ ] **Step 2b: 收紧 catalog 测试（承接 Task 2 评审的 2 条 Minor）**

在 `internal/capabilities/catalog/catalog_test.go` 中做两处加强：

1. `TestResolveClosureIsTransitive` 增加顺序断言（传递闭包的顺序对后续迁移执行有意义）：

```go
	wantOrder := []string{"user", "role", "tenant", "auth", "oauth"}
	if gotOrder := namesOf(got); !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("Resolve([oauth]) order = %v, want %v", gotOrder, wantOrder)
	}
```
（同时在测试文件 import 中加 `"reflect"`。）

2. `TestDescriptorsAreWellFormed` 增加重名检测与「依赖下标 < 被依赖者下标」断言：

```go
		index := make(map[string]int, len(list))
		for i, d := range list {
			if _, dup := index[d.Name]; dup {
				t.Fatalf("%s: duplicate capability name %q", label, d.Name)
			}
			index[d.Name] = i
		}
		for _, d := range list {
			for _, dep := range d.Requires {
				if index[dep] >= index[d.Name] {
					t.Fatalf("%s: capability %q at index %d must follow its dependency %q at index %d",
						label, d.Name, index[d.Name], dep, index[dep])
				}
			}
		}
```
（保留该用例原有的 name 非空、`Requires` 指向已知能力、`Mount` 合法三项检查。）

- [ ] **Step 3: 运行测试确认通过**

Run: `go test ./internal/capabilities/catalog/ -v`
Expected: PASS（7 个用例全绿；`TestDescriptorsAreWellFormed` 此时真正覆盖 catalog 的 8 个真实条目）

- [ ] **Step 4: 全量编译与格式检查**

Run: `gofmt -l internal/modules internal/capabilities && go build ./... && go vet ./...`
Expected: 无输出（gofmt）、构建与 vet 通过

- [ ] **Step 5: 提交（用户授权后执行）**

```bash
git add internal/modules internal/capabilities/catalog
git commit -m "feat(capabilities): declare descriptors for all eight capabilities"
```

---

### Task 4: `capabilities.enabled` 配置段

**Files:**
- Modify: `internal/config/config.go`（`Config` 结构体 + 新 `CapabilitiesConfig`）
- Modify: `internal/config/validate.go`（`validateCommon` 追加校验）
- Modify: `internal/config/config_test.go`
- Modify: `configs/app.yaml`

**Interfaces:**
- Consumes: 无（配置层独立）
- Produces: `config.CapabilitiesConfig{Enabled []string}`、`Config.Capabilities`（`mapstructure:"capabilities"`）

- [ ] **Step 1: 写失败测试**

追加到 `internal/config/config_test.go`：

```go
func TestCapabilitiesConfigFieldMapping(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	conf := `
capabilities:
  enabled:
    - user
    - auth
`
	if err := v.ReadConfig(strings.NewReader(conf)); err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	assert.Equal(t, []string{"user", "auth"}, cfg.Capabilities.Enabled)
}

func TestValidateCapabilitiesRejectsBlankName(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Capabilities.Enabled = []string{"user", " "}
	if err := cfg.Validate("dev"); err == nil {
		t.Fatal("expected error for blank capability name")
	}
}

func TestValidateCapabilitiesRejectsDuplicate(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Capabilities.Enabled = []string{"user", "user"}
	if err := cfg.Validate("dev"); err == nil {
		t.Fatal("expected error for duplicate capability name")
	}
}

func TestValidateCapabilitiesAllowsEmpty(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Capabilities.Enabled = nil
	if err := cfg.Validate("dev"); err != nil {
		t.Fatalf("empty enabled must be valid (means all), got %v", err)
	}
}
```

> `minimalValidConfig(t)` 是需要在同文件新增的测试助手：返回一份能通过 `Validate("dev")` 的配置副本。

```go
// minimalValidConfig 返回一份通过 dev 校验的配置副本，供能力校验用例复用。
func minimalValidConfig(t *testing.T) Config {
	t.Helper()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	return *cfg
}
```

> 用例中调用 `minimalValidConfig(t)`（`Load()` 在本仓库测试中已被 `TestLoad` 使用，依赖仓库内 `configs/app.yaml`；若某环境不可用，改为直接构造 `Config{}` 并只填 `validateCommon` 必填字段，参考 `TestValidateOutboxPublisher` 的写法）。判据：测试稳定通过。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/config/ -run 'TestCapabilities|TestValidateCapabilities' -v`
Expected: FAIL —— `cfg.Capabilities undefined`

- [ ] **Step 3: 加配置结构与校验**

`internal/config/config.go`：在 `Config` 结构体中 `GRPC` 之后追加字段：

```go
	Capabilities CapabilitiesConfig `mapstructure:"capabilities"`
```

并在 `GRPCConfig` 定义附近新增类型：

```go
// CapabilitiesConfig 能力启用开关（v0.3.0 可插拔能力）
type CapabilitiesConfig struct {
	// Enabled 启用的能力名清单；留空表示全部启用（保持向后兼容）
	Enabled []string `mapstructure:"enabled"`
}
```

`internal/config/validate.go` 的 `validateCommon()` 末尾（`captcha` 校验之前）追加：

```go
	if err := validateCapabilities(c.Capabilities); err != nil {
		return err
	}
```

并在文件末尾新增：

```go
// validateCapabilities 校验能力开关：名称非空且不重复（能力名是否存在由 catalog 解析时校验）
func validateCapabilities(cfg CapabilitiesConfig) error {
	seen := make(map[string]bool, len(cfg.Enabled))
	for i, name := range cfg.Enabled {
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("invalid capabilities.enabled[%d]: name must not be blank", i)
		}
		if seen[name] {
			return fmt.Errorf("duplicate capabilities.enabled entry: %q", name)
		}
		seen[name] = true
	}
	return nil
}
```

- [ ] **Step 4: 加配置样例段**

`configs/app.yaml` 末尾追加：

```yaml
# 能力开关（v0.3.0 可插拔能力；留空表示全部启用）
# 可选值见 internal/capabilities/catalog；留空时行为与本字段不存在时完全一致。
capabilities:
  enabled: []   # 例如 ["user", "role", "permission", "tenant", "auth", "audit", "admin"]
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/config/ -v`
Expected: PASS（含既有用例不回归）

- [ ] **Step 6: 提交（用户授权后执行）**

```bash
git add internal/config configs/app.yaml
git commit -m "feat(config): add capabilities.enabled switch with validation"
```

---

### Task 5: 挂载点去特例化

**Files:**
- Modify: `internal/app/bootstrap.go:377-410`（`registerHTTP`）
- Test: `internal/app/bootstrap_http_test.go`

**Interfaces:**
- Consumes: Task 1 的 `contract.Describe` / `Descriptor.Normalized()`
- Produces: `registerHTTP` 行为契约 —— `MountProtected` 的模块获得受保护中间件；`MountPublic` / `MountSelfManaged` 直接挂到根路由

- [ ] **Step 1: 写失败测试**

创建 `internal/app/bootstrap_http_test.go`：

```go
package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
)

// probeModule 记录是否被注册，并按声明的挂载点注册一个探针路由。
type probeModule struct {
	name       string
	desc       contract.Descriptor
	registered bool
}

func (p *probeModule) Name() string { return p.name }

func (p *probeModule) Descriptor() contract.Descriptor { return p.desc }

func (p *probeModule) RegisterHTTP(r contract.Router) {
	p.registered = true
	r.Group("/api/v1").GET("/"+p.name+"/probe", func(c *gin.Context) { c.Status(http.StatusNoContent) })
}

func (p *probeModule) RegisterJobs(contract.JobRegistry) {}

func (p *probeModule) RegisterEvents(contract.EventBus) {}

// protectedProvider 声明受保护中间件：命中时写入标记头。
type protectedProvider struct{ probeModule }

func (p *protectedProvider) ProtectedHTTPMiddleware() ([]gin.HandlerFunc, error) {
	return []gin.HandlerFunc{func(c *gin.Context) {
		c.Header("X-Protected", "1")
		c.Next()
	}}, nil
}

func TestRegisterHTTPAppliesProtectedMiddlewareByMountPoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 名称刻意不用 auth/oauth：当前实现按名称特判，本用例必须能在该实现下失败。
	self := &protectedProvider{probeModule{name: "probe-self", desc: contract.Descriptor{Name: "probe-self", Mount: contract.MountSelfManaged}}}
	prot := &probeModule{name: "probe-prot", desc: contract.Descriptor{Name: "probe-prot", Mount: contract.MountProtected}}
	pub := &probeModule{name: "probe-pub", desc: contract.Descriptor{Name: "probe-pub", Mount: contract.MountPublic}}

	if err := registerHTTP(router, nil, nil, self, prot, pub); err != nil {
		t.Fatalf("registerHTTP error: %v", err)
	}
	for _, m := range []*probeModule{&self.probeModule, prot, pub} {
		if !m.registered {
			t.Fatalf("module %q was not registered", m.name)
		}
	}

	cases := []struct {
		path      string
		protected bool
	}{
		{"/api/v1/probe-prot/probe", true},
		{"/api/v1/probe-self/probe", false},
		{"/api/v1/probe-pub/probe", false},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		router.ServeHTTP(rec, req)
		got := rec.Header().Get("X-Protected") == "1"
		if got != c.protected {
			t.Fatalf("path %s protected = %v, want %v", c.path, got, c.protected)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/app/ -run TestRegisterHTTPAppliesProtectedMiddlewareByMountPoint -v`
Expected: FAIL —— 当前实现按 `module.Name() != "auth" && module.Name() != "oauth"` 判定：`probe-self` 与 `probe-pub` 都会拿到受保护中间件，而用例期望它们不被套用，报错形如 `path /api/v1/probe-self/probe protected = true, want false`

- [ ] **Step 3: 按挂载点重写 registerHTTP**

将 `internal/app/bootstrap.go` 的 `registerHTTP`（第 377–410 行）替换为：

```go
func registerHTTP(router registerRouter, log *logger.Logger, extraProtected []gin.HandlerFunc, modules ...contract.Module) error {
	// 全局中间件：所有能力声明的前置中间件（如审计写入）
	for _, module := range modules {
		if provider, ok := module.(contract.HTTPMiddlewareProvider); ok {
			router.Use(provider.HTTPMiddleware()...)
		}
	}
	// 受保护中间件：必须恰好由一个能力提供。多个提供者时无法仅凭 catalog 顺序
	// 判定认证/租户注入/限流链的组合语义，因此拒绝启动而不是"首个提供者生效"。
	var protected []gin.HandlerFunc
	providers := make([]string, 0, 1)
	for _, module := range modules {
		provider, ok := module.(contract.ProtectedHTTPMiddlewareProvider)
		if !ok {
			continue
		}
		chain, err := provider.ProtectedHTTPMiddleware()
		if err != nil {
			return fmt.Errorf("configure protected middleware: %w", err)
		}
		providers = append(providers, contract.Describe(module).Name)
		protected = append(protected, chain...)
	}
	if len(providers) > 1 {
		return fmt.Errorf("multiple capabilities provide protected middleware (%s); an explicit ordering rule is required", strings.Join(providers, ", "))
	}
	// extraProtected 只含租户限流/幂等等补充中间件，不能替代认证与租户注入，
	// 因此「是否存在受保护中间件」必须在追加 extraProtected 之前判定。
	hasProtectedMiddleware := len(protected) > 0
	// 追加外部注入的受保护中间件（如租户维度限流），顺序在认证/租户注入之后
	protected = append(protected, extraProtected...)

	for _, module := range modules {
		desc := contract.Describe(module)
		if desc.Normalized() == contract.MountProtected {
			if !hasProtectedMiddleware {
				return fmt.Errorf("capability %q declares MountProtected but no enabled capability provides protected middleware; enable the capability that provides it (currently \"auth\")", desc.Name)
			}
			module.RegisterHTTP(router.Group("", protected...))
		} else {
			module.RegisterHTTP(router)
		}
		if log != nil {
			log.Infow("capability registered", "name", desc.Name, "mount", string(desc.Normalized()))
		}
	}
	return nil
}
```

> **终审修复波修订**：本 Task 原方案是"首个提供者生效"（找到第一个 `ProtectedHTTPMiddlewareProvider` 后 `break`），已被**单提供者规则**取代 —— 收集全部提供者，多于一个即拒绝启动并列出冲突能力名，直到 P1 设计出显式排序规则（spec §10 P0 要求删除"第一个中间件提供者"约定）。同时 `log` 参数由包内 `moduleLogger` 改为 `*logger.Logger`，消除 logcheck 的可见性盲区（见 Self-Review 表）；`hasProtectedMiddleware` 在追加 `extraProtected` 之前判定这一点被回归测试钉住。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/app/ -run TestRegisterHTTP -v && grep -n 'Name() != "auth"' internal/app/bootstrap.go`
Expected: 测试 PASS；`grep` 无输出（字符串特判已删除）

- [ ] **Step 4b: 钉住真实清单的挂载点（承接 Task 3 评审的 1 条 Minor）**

`registerHTTP` 的行为现在取决于 `catalog.entries` 里八个真实 `Mount` 值，但没有任何已提交的测试断言这些值。在 `internal/capabilities/catalog/catalog_test.go` 的 `TestDescriptorsAreWellFormed` 末尾（`for label, list := range lists` 循环之后）追加一行断言，把真实清单与夹具锁在一起（`reflect` 已在 Step 2b 引入）：

```go
	if !reflect.DeepEqual(All(), fixture()) {
		t.Fatalf("catalog descriptors drifted from the expected fixture:\n got %+v\nwant %+v", All(), fixture())
	}
```

理由：`fixture()` 已被 `Resolve` 的顺序与闭包用例覆盖，把它与真实清单绑定后，任何一个真实 `Mount`/`Requires`/顺序被改动都会立即失败。

Run: `go test ./internal/capabilities/catalog/ -v`
Expected: PASS（7 个用例全绿，新增断言使 `TestDescriptorsAreWellFormed` 同时钉住顺序、依赖与挂载点）

- [ ] **Step 5: 提交（用户授权后执行）**

```bash
git add internal/app/bootstrap.go internal/app/bootstrap_http_test.go
git commit -m "refactor(app): mount capabilities by declared mount point"
```

---

### Task 6: 启用集过滤与启动清单

**Files:**
- Modify: `cmd/server/main.go`（能力构造与过滤）
- Modify: `internal/app/bootstrap.go`（启动打印启用清单）

**Interfaces:**
- Consumes: Task 2 的 `catalog.Resolve`、Task 4 的 `cfg.Capabilities.Enabled`
- Produces: 传给 `app.Bootstrap` 的 `modules` 切片只含启用能力；启动日志输出启用清单

- [ ] **Step 1: 在 Bootstrap 中打印启用清单**

`internal/app/bootstrap.go` 的 `Bootstrap` 函数中，**在 `registerHTTP` 成功返回之后**打印（不能放在开头：fail-closed 校验失败时进程退出，提前打印等于宣称一组其实被拒绝的"已启用"集）：

```go
	names := make([]string, 0, len(modules))
	for _, module := range modules {
		names = append(names, contract.Describe(module).Name)
	}
	container.Logger.Infow("capabilities enabled", "count", len(names), "names", strings.Join(names, ","))
```

用 `contract.Describe(module).Name` 而不是 `module.Name()`，与 `registerHTTP` 的注册日志保持同一来源。并在文件 import 中加入 `"strings"`。

> 终审修复波修订：本条最初规定插入在 `Bootstrap` 开头并用 `module.Name()`；终审指出"先打印 enabled、随后被 fail-closed 拒绝"的阅读矛盾，故改为注册成功后打印。

- [ ] **Step 2: main.go 按启用集过滤**

`cmd/server/main.go`：把现有的 `app.Bootstrap(container, user.New(...), ...)` 改为先构造后过滤。替换 `tenantMod := tenantmodule.New(...)` 到 `application, err := app.Bootstrap(...)` 之间整段为：

```go
	// 租户套餐/配额：定义在 tenant 能力，注入到创建用户/角色/API Key 的路径
	tenantMod := tenantmodule.New(container.DB, *cfg)

	// 能力开关：capabilities.enabled 为空表示全部启用（向后兼容）
	caps, err := catalog.Resolve(cfg.Capabilities.Enabled)
	if err != nil {
		_ = container.Stop(context.Background())
		return fmt.Errorf("resolve capabilities: %w", err)
	}

	// 全部能力的实例：键为能力名，与 catalog 清单一一对应
	// 过渡实现（P0）：先构造再过滤；P1 引入显式 Deps 后改为按需构造
	all := map[string]contract.Module{
		"user":       user.New(container.DB, *cfg, container.Redis, container.Outbox),
		"auth":       authmodule.New(container.DB, container.Redis, cfg.Auth, cfg.HTTP.Mode == config.HTTPModeRelease, container.Captcha, cfg.Captcha, container.Outbox, container.Notification, container.Cipher, container.BreachChecker, tenantMod.Quota()),
		"role":       role.New(container.DB, tenantMod.Quota()),
		"permission": permission.New(container.DB),
		"tenant":     tenantMod,
		"audit":      auditmodule.New(container.DB, cfg.Audit, container.Logger),
		"admin": adminmodule.New(cfg.Version, cfg.Environment, container.Redis, container.DB, middleware.IPAllowlist(cfg.Security.AdminIPAllowlist), container.Scheduler, container.Storage, container.UploadScanner, container.FeatureFlag, container.EventBus,
			auth.NewWithRotation(cfg.Auth.JWTSecret, cfg.Auth.JWTPreviousSecret, cfg.Auth.Issuer, cfg.Auth.AccessExpireMin, cfg.Auth.RefreshExpireDay),
			tenantMod.Quota()),
		"oauth": oauthmodule.New(container.DB, container.Redis, cfg.OAuth, cfg.Auth, container.HTTPClient),
	}
	modules := make([]contract.Module, 0, len(caps))
	for _, d := range caps {
		module, ok := all[d.Name]
		if !ok {
			_ = container.Stop(context.Background())
			return fmt.Errorf("capability %q is declared in catalog but not wired in main", d.Name)
		}
		modules = append(modules, module)
	}

	application, err := app.Bootstrap(container, modules...)
```

并在 `main.go` import 中加入 `"jimu/internal/capabilities/catalog"` 与 `"jimu/internal/contract"`。

- [ ] **Step 2b: 消除 e2e 里重复的挂载特判（承接 Task 5 评审的 1 条 Minor）**

`internal/e2e/api_contract_test.go:130-137` 因 `registerHTTP` 不可导出而**复制了一份**旧的名称特判：

```go
	// 3) 路由注册（auth/oauth 公开，其余受保护）
	for _, m := range modules {
		if len(protected) > 0 && m.Name() != "auth" && m.Name() != "oauth" {
			m.RegisterHTTP(router.Group("", protected...))
		} else {
			m.RegisterHTTP(router)
		}
	}
```

若不改，e2e 的挂载策略会与生产静默漂移（真实能力改了 `Mount` 也不会失败）。替换为按描述符判定，与 `registerHTTP` 一致（该文件已 import `jimu/internal/contract`，若未 import 则补上）：

```go
	// 3) 路由注册（按能力声明的挂载点：受保护 / 公开或自管理）
	for _, m := range modules {
		if contract.Describe(m).Normalized() == contract.MountProtected && len(protected) > 0 {
			m.RegisterHTTP(router.Group("", protected...))
		} else {
			m.RegisterHTTP(router)
		}
	}
```

- [ ] **Step 2c: 两处小修（承接 Task 4 / Task 5 评审的 Minor）**

1. `internal/config/validate.go` 的 `validateCapabilities` 改为把归一化后的名字写回，避免 `" user"` 通过配置校验却在 `catalog.Resolve` 处报 unknown capability（与同文件 `Redis.Mode` 的归一化先例一致）。签名改为接收指针：

```go
	if err := validateCapabilities(&c.Capabilities); err != nil {
		return err
	}
```

```go
func validateCapabilities(cfg *CapabilitiesConfig) error {
	seen := make(map[string]bool, len(cfg.Enabled))
	for i, raw := range cfg.Enabled {
		name := strings.TrimSpace(raw)
		if name == "" {
			return fmt.Errorf("invalid capabilities.enabled[%d]: name must not be blank", i)
		}
		if seen[name] {
			return fmt.Errorf("duplicate capabilities.enabled entry: %q", name)
		}
		seen[name] = true
		cfg.Enabled[i] = name
	}
	return nil
}
```

2. 在 `internal/config/config_test.go` 增加一条混合空白的用例，锁住 trim 与写回：

```go
func TestValidateCapabilitiesTrimsAndDedupes(t *testing.T) {
	cfg := minimalValidConfig(t)
	cfg.Capabilities.Enabled = []string{"user", " user"}
	if err := cfg.Validate("dev"); err == nil {
		t.Fatal("expected duplicate detection after trimming")
	}
	cfg.Capabilities.Enabled = []string{" user ", "auth"}
	if err := cfg.Validate("dev"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.Equal(t, []string{"user", "auth"}, cfg.Capabilities.Enabled)
}
```

3. 在 `internal/app/bootstrap_http_test.go` 的 `TestRegisterHTTPAppliesProtectedMiddlewareByMountPoint` 循环内补状态码断言，避免「路由未匹配（404）」被误当成「未套受保护中间件」：

```go
		if rec.Code != http.StatusNoContent {
			t.Fatalf("path %s status = %d, want %d", c.path, rec.Code, http.StatusNoContent)
		}
```

- [ ] **Step 2d: MountProtected 必须 fail closed（承接 Task 6 评审的 Important 发现）**

评审证实：启用集里没有 `auth`（唯一的 `ProtectedHTTPMiddlewareProvider`）时，`registerHTTP` 会把声明为 `MountProtected` 的能力挂到**裸路由**上；而 `internal/modules/user/interfaces/handler.go` 的 `Get` 不读鉴权/租户上下文、`application/service.go` 的 `tenantAllowed` 在 `ctxTenant == 0` 时返回 true，于是未认证的 `GET /api/v1/users/:id` 会返回用户记录，启动日志却仍打印 `mount: "protected"`。

裁定：**启动失败**（不是静默跳过）。修改 `internal/app/bootstrap.go` 的注册循环。注意「是否真的存在受保护中间件」必须**在追加 `extraProtected` 之前**判定 —— `extraProtected` 只含租户限流/幂等等补充中间件（`security.idempotency_enabled` 默认开启，故它默认非空），用追加后的 `len(protected) == 0` 判定会 fail open：

```go
	// extraProtected 只含租户限流/幂等等补充中间件，不能替代认证与租户注入，
	// 因此「是否存在受保护中间件」必须在追加 extraProtected 之前判定。
	hasProtectedMiddleware := len(protected) > 0
	// 追加外部注入的受保护中间件（如租户维度限流），顺序在认证/租户注入之后
	protected = append(protected, extraProtected...)
	for _, module := range modules {
		desc := contract.Describe(module)
		if desc.Normalized() == contract.MountProtected {
			if !hasProtectedMiddleware {
				return fmt.Errorf("capability %q declares MountProtected but no enabled capability provides protected middleware; enable the capability that provides it (currently \"auth\")", desc.Name)
			}
			module.RegisterHTTP(router.Group("", protected...))
		} else {
			module.RegisterHTTP(router)
		}
		if log != nil {
			log.Infow("capability registered", "name", desc.Name, "mount", string(desc.Normalized()))
		}
	}
```

并在 `internal/app/bootstrap_http_test.go` 增加两个失败用例（该文件需 import `strings`）：`TestRegisterHTTPFailsClosedWithoutProtectedProvider`（无提供者时必须报错且不注册）与 `TestRegisterHTTPFailsClosedWhenExtraProtectedIsPresent`（即使 `extraProtected` 非空，仍必须报错 —— 这条正是防止上面的 fail-open 回归）：

```go
func TestRegisterHTTPFailsClosedWithoutProtectedProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	prot := &probeModule{name: "probe-prot", desc: contract.Descriptor{Name: "probe-prot", Mount: contract.MountProtected}}

	err := registerHTTP(router, nil, nil, prot)
	if err == nil {
		t.Fatal("expected error when a protected capability has no middleware provider")
	}
	if !strings.Contains(err.Error(), "MountProtected") {
		t.Fatalf("error = %v, want it to name the mount requirement", err)
	}
	if prot.registered {
		t.Fatalf("capability %q must not be registered when no middleware provider exists", prot.name)
	}
}

func TestRegisterHTTPFailsClosedWhenExtraProtectedIsPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	prot := &probeModule{name: "probe-prot", desc: contract.Descriptor{Name: "probe-prot", Mount: contract.MountProtected}}
	extra := []gin.HandlerFunc{func(c *gin.Context) { c.Header("X-Extra", "1"); c.Next() }}

	if err := registerHTTP(router, nil, extra, prot); err == nil {
		t.Fatal("expected error even when extraProtected is non-empty")
	}
	if prot.registered {
		t.Fatalf("capability %q must not be registered when only extra middleware exists", prot.name)
	}
}
```

- [ ] **Step 3: 编译并跑测试**

Run: `gofmt -l . && go build ./... && go vet ./... && go test ./internal/app/ ./internal/capabilities/... ./internal/contract/ -v`
Expected: gofmt 无输出；构建/vet 通过；测试全绿

- [ ] **Step 4: 手工验证关闭能力生效**

在临时环境变量下启动一次服务（不写仓库文件）：

```bash
CATALOG_CHECK=1 go run ./cmd/server 2>&1 | head -5
```
Expected: 日志出现 `capabilities enabled` 且 `count=8`（默认全部启用）。

随后用**合法**的过滤组合验证（`["user","auth"]` 经闭包补齐为 `user/role/tenant/auth`，共 4 个；用 `["user"]` 会因 Step 2d 的 fail-closed 而拒绝启动，那是预期行为）：

```bash
cat > /tmp/jimu-caps.yaml <<'EOF'
capabilities:
  enabled: ["user", "auth"]
EOF
go run ./cmd/server 2>&1 | head -5
```
Expected: `capabilities enabled` 的 `count=4`、`names=user,role,tenant,auth`；`/api/v1/permissions`、`/api/v1/audits`、`/api/v1/admin/*`、`/api/v1/oauth/*` 全部 404；`/api/v1/users` 仍返回 401（受保护中间件生效）。

再验证 fail-closed：

```bash
cat > /tmp/jimu-caps-bad.yaml <<'EOF'
capabilities:
  enabled: ["user"]
EOF
go run ./cmd/server 2>&1 | tail -3
```
Expected: 启动失败，错误信息指出声明了 `MountProtected` 但没有启用提供受保护中间件的能力。

> 执行提示：本仓库的配置加载器不支持 `JIMU_CAPABILITIES_ENABLED` 这类嵌套键的环境变量覆盖（`applyEnvOverrides` 只处理固定的扁平键、未启用 `AutomaticEnv`/`SetEnvKeyReplacer`），因此改用临时配置文件（放 `/tmp`，不写入仓库）或容器内挂载临时配置。本步骤的判据是：**过滤与 fail-closed 在运行时确实生效**。

- [ ] **Step 5: 提交（用户授权后执行）**

```bash
git add cmd/server/main.go internal/app/bootstrap.go
git commit -m "feat(server): filter capabilities by enabled set at startup"
```

---

### Task 7: 文档、版本日志与全量回归

**Files:**
- Modify: `README.md`（新增「能力开关」小节 + 项目结构说明）
- Modify: `docs/releases/v0.3.0.md`（P0 条目与验证）
- Modify: `AGENTS.md`（新增"能力边界"约束，指向 spec）

**Interfaces:**
- Consumes: Task 1–6 的全部产出
- Produces: 文档与发布说明；全量回归证据

- [ ] **Step 1: README 增加能力开关章节**

在 README「配置」章节之后插入（内容如下，注意保持单层代码块）：

````markdown
### 能力开关（v0.3.0）

后端由**能力**组成，可用 `capabilities.enabled` 选择启用哪些能力（留空 = 全部启用，行为与旧版本一致）：

```yaml
capabilities:
  enabled: ["user", "role", "permission", "tenant", "auth", "audit", "admin"]
```

- 硬依赖会自动补齐：只写 `["oauth"]` 会连带启用 `auth`/`user`/`role`/`tenant`
- 未启用的能力不挂路由、不注册定时任务与事件、不启动其后台组件
- **受保护能力需要认证器**：声明为受保护（`MountProtected`）的能力必须有模块提供受保护中间件（当前为 `auth`）；否则进程**启动即失败**并指出缺失的提供者，而不是把路由裸挂出去。因此 `enabled: ["user"]` 这类"有业务路由、无认证器"的配置会被拒绝
- 能力清单与依赖关系见 `internal/capabilities/catalog/catalog.go`；设计见 [能力可插拔设计](docs/design/2026-09-18-capability-plugins-design.md)
````

- [ ] **Step 2: 更新版本日志**

`docs/releases/v0.3.0.md` 的「变更」段追加：

```markdown
- **能力契约与运行时开关** — 每个能力导出静态 `Descriptor`（名称/硬依赖/挂载点），`internal/capabilities/catalog` 作为唯一能力清单提供启用集解析（未知能力报错、硬依赖自动补齐闭包、按拓扑序返回）；新增 `capabilities.enabled` 配置（留空 = 全部启用，向后兼容），未启用的能力不挂路由、不注册任务与事件、不启动后台组件；HTTP 挂载点由能力声明决定，去掉 `bootstrap` 中对 `auth`/`oauth` 的能力名特判
- **受保护能力 fail closed** — 启用集里若含声明为受保护的能力、却没有模块提供受保护中间件（当前为 `auth`），进程启动即失败并指出缺失提供者；此前挂载点判定会把这类路由裸挂出去（未认证即可访问），现在改为拒绝启动而不是降级运行
```

同时在「说明」段追加：

```markdown
- 启动日志中模块注册文案由 `module registered` 改为 `capability registered`（并附带实际生效的挂载点）
- 声明为受保护的能力需要同时启用提供受保护中间件的能力（当前为 `auth`）：`capabilities.enabled: ["user"]` 会被拒绝启动；合法的最小组合形如 `["user", "auth"]`（闭包自动补齐 `role`/`tenant`）
```

并在 `configs/app.yaml` 的 `capabilities` 段注释中补上该约束（与 README 一致）。

并在「验证」段写入本计划 Task 7 Step 4 的实际结果。

- [ ] **Step 3: AGENTS.md 增加能力边界约束**

在「架构约束」章节「租户体系」之前插入：

```markdown
### 能力边界（v0.3.0 起）

后端按**能力**组织，可插拔为正式能力（设计与清单见 [docs/design/2026-09-18-capability-plugins-design.md](docs/design/2026-09-18-capability-plugins-design.md)）：

- 能力清单只维护在 `internal/capabilities/catalog`，新增/删除能力只改该文件
- 每个能力导出静态 `Descriptor`（名称 / 硬依赖 `Requires` / 挂载点 `Mount`），依赖必须单向
- 能力之间只经 `contract` 端口调用，禁止 import 其他能力的内部包
- 挂载点由 `Descriptor.Mount` 声明，禁止按能力名做特判
```

- [ ] **Step 4: 全量回归**

Run: `gofmt -l . && go build ./... && go vet ./... && golangci-lint run ./... && make check-log-usage && go test ./...`
Expected: gofmt 无输出；构建/vet/lint 0 issues；log-usage 通过；`go test ./...` 全绿（覆盖率不低于 70%）

Run: `make release-check COMPOSE_ENV=.env.example`
Expected: 通过（含 fmt-check / vet / check-log-usage / test / govulncheck / compose-check）

- [ ] **Step 5: 提交（用户授权后执行）**

```bash
git add README.md AGENTS.md docs/releases/v0.3.0.md
git commit -m "docs(capabilities): document the capability switch and boundaries"
```

---

## Self-Review

**1. Spec 覆盖检查**

| spec 要求 | 覆盖任务 |
|---|---|
| §6.1 能力自描述（`Name`/`Requires`/`Mount`/`Permissions`/…） | Task 1（Name/Requires/Mount）、Task 3（8 个能力声明）；`Owns`/`Config`/`Migrations`/`Permissions` 字段属 P1/P2，见下方差异说明 |
| §6.4 层③ 运行时 `capabilities.enabled` + 依赖闭包校验 | Task 2（`Resolve`）、Task 4（配置）、Task 6（接线） |
| §3.5.2 / §5.2 `admin` 名特判移除 | Task 5 |
| §10 P0「去掉"第一个中间件提供者"约定」（终审修复波：改为单提供者 fail closed，多于一个提供者即拒绝启动） | Task 5 |
| §10 P0「`catalog` 显式清单」「去掉 `auth`/`oauth` 字符串特判」「关闭能力后路由/任务/事件消失」 | Task 2/3/5/6 |
| §10 P0「建立 `internal/kernel/`」「能力目录落位」 | **推迟到 P1**（见「本计划与 spec 的一处差异」） |
| §1 可量化验收（`compose-report`） | P2（本计划不做） |

**2. 占位符扫描**：无 TBD/TODO；每步含可执行命令或完整代码；Task 4 Step 1 的测试助手给了两种实现方式并明确判据（`Load()` 可用性），不是"自行决定"式空白。

**3. 类型一致性**：`Descriptor`/`MountPoint`/`Normalized()`/`Describe()`（Task 1）在 Task 2/3/5/6 中签名一致；`catalog.Resolve`/`All`/`Names`（Task 2）在 Task 6 中调用一致；`Config.Capabilities.Enabled`（Task 4）在 Task 6 中使用一致；8 个 `Descriptor` 变量名统一为 `Descriptor`。

**4. 已知遗留（记录在案，不在 P0 处理）**：P0 仍会构造全部能力实例（过滤发生在构造之后），因为显式 `Deps` 结构体属 P1；除 `audit` 的 worker（仅经 `Components()` 在启用时启动）外，各能力构造函数无副作用，因此"未启用 = 不生效"在 P0 已成立。
