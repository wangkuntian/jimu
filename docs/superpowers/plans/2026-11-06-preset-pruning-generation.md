# 按裁剪生成预设项目 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 `jimu new` 只保留一套"从主仓库裁剪"的生成引擎，提供 `minimal` / `auth` / `full` 三个预设，退役 `basic-api` / `auth-api`，并保证预设项目仍可继续 `capability add` 组合能力。

**Architecture:** 预设解析落在 `tools/generator/frameworkmanifest`（真正驱动生成的适配层），新增 `presets.go` 把预设名映射为「能力名 + 驱动集 + 支持级别」，复用现有 `capability.Compose` 闭包与 `completeSelection` 裁剪管线。内核侧先解耦 `seed.go` 的能力 domain 依赖（新增 `internal/kernel/seedmodel`），再补 nil-DB 护栏与内核 `/api/v1/ping`。最后删除旧模板渲染路径与模板树。

**Tech Stack:** Go 1.26、Cobra、`golang.org/x/tools/go/packages`、现有 manifest/render/workspace 事务管线、GitHub Actions、Docker。

**Spec:** `docs/superpowers/specs/2026-11-06-project-generation-by-pruning-design.md`

## Global Constraints

- **禁止自动 commit/push**：未经用户明确指令，只报告 diff 与验证结果，不执行 `git add` / `git commit` / `git push`。
- 生成失败必须在写目标目录前发生；不得留下半成品（沿用 `workspace.Create` 的 staging + 原子替换）。
- 不生成 `jimu-web`，不做全栈模板，不支持 Windows。
- 预设名固定为 `minimal` / `auth` / `full`；`basic-api` / `auth-api` 立即退役，无过渡期。
- `full` = 23 项能力 + 框架 `full` 形态的完整驱动集；不得用 `--with` 的"首项驱动"缺省替代。
- `minimal` = 0 能力 + 闭包裁剪内核，不产生迁移/DB/Redis 依赖。
- 支持级别：`minimal`/`auth` = `certified`，`full` = `validated`，`--with` = `experimental`。
- 文档正文中文；提交信息英文 Conventional Commits 且必须带 scope。
- 任何门禁都不允许"匹配不到用例却退出 0"。

## Review Focus

- **`minimal` 在设置了管理员密码环境变量时启动**：必须不 panic，且日志明确说明跳过 seed。这是当前代码的真实崩溃点。
- **`--template=minimal` 重复执行**：第二次必须幂等或明确拒绝，不能生成半棵树。
- **`full` 的驱动完整性**：`queue`/`storage`/`dataops` 必须带全部驱动目录，缺一个就是静默降级。
- **`minimal` 项目 `capability add`**：加完能力必须可 `go build`，且 manifest 记录可被 `upgrade` 复现。
- **旧 manifest 升级**：`basic-api`/`auth-api` 项目必须硬失败且不写任何字节，错误信息给出迁移路径。

## File Structure

**新增**

- `internal/kernel/seedmodel/seedmodel.go`：seed 所需 5 个最小持久化模型。
- `internal/kernel/seedmodel/seedmodel_test.go`：与能力 domain 的结构一致性断言。
- `tools/generator/frameworkmanifest/presets.go`：预设 → 能力集/驱动集/支持级别。
- `tools/generator/frameworkmanifest/presets_test.go`
- `tools/checkgateselectors/main.go`：门禁选择器非空检查器。

**修改**

- `internal/app/seed.go`：改 import seedmodel。
- `internal/profiles/profiles.go`：`StructuralSeed` 的 nil-DB 护栏。
- `internal/kernel/http/server.go`：`GET /api/v1/ping`。
- `tools/generator/frameworkmanifest/export.go`：模板分支改预设分支；`ExportForAdd` 只拒退役模板。
- `tools/generator/layout.go`：删除 `kernelRequiredDomains`。
- `tools/generator/frameworkmanifest/actions.go`：闭包裁剪内核目录。
- `Makefile`：`SCAFFOLD_OFFICIAL_TESTS` 换名 + `check-gate-selectors`。
- `docs/templates/v1/budgets.json`、`docs/templates/v1/matrix.yaml`。

**删除**

- `tools/generator/frameworkmanifest/basic.go`、`auth.go`
- `tools/generator/render/templates/basicapi/`、`authapi/`
- `tools/generator/testdata/upgrade/v0.5/`

---

### Task 1: 内核 seed 模型下沉

**Files:**
- Create: `internal/kernel/seedmodel/seedmodel.go`
- Create: `internal/kernel/seedmodel/seedmodel_test.go`
- Modify: `internal/app/seed.go`（import 块与全部类型引用）
- Test: `internal/app/seed_test.go`（既有测试必须继续通过，不改断言）

**Interfaces:**
- Produces（字段、`gorm`/`json` tag 逐字复制自对应 domain）：
  - `seedmodel.Permission`、`seedmodel.Role`、`seedmodel.Tenant`、`seedmodel.Plan`、`seedmodel.User`
  - 每个类型都有 `TableName() string`，返回值分别为 `permissions`、`roles`、`tenants`、`tenant_plans`、`users`
- `seedmodel.Role.Scope` 的类型是 `contract.RoleScope`

- [ ] **Step 1: 写失败的结构一致性测试**

```go
// internal/kernel/seedmodel/seedmodel_test.go
package seedmodel_test

import (
	"reflect"
	"testing"

	accessdomain "jimu/internal/capabilities/access/domain"
	tenantdomain "jimu/internal/capabilities/tenant/domain"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/kernel/seedmodel"
)

func TestSeedModelMatchesCapabilityDomains(t *testing.T) {
	pairs := []struct {
		name string
		seed any
		real any
	}{
		{"Permission", seedmodel.Permission{}, accessdomain.Permission{}},
		{"Role", seedmodel.Role{}, accessdomain.Role{}},
		{"Tenant", seedmodel.Tenant{}, tenantdomain.Tenant{}},
		{"Plan", seedmodel.Plan{}, tenantdomain.Plan{}},
		{"User", seedmodel.User{}, userdomain.User{}},
	}
	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			assertSamePersistedShape(t, reflect.TypeOf(pair.seed), reflect.TypeOf(pair.real))
		})
	}
}

func assertSamePersistedShape(t *testing.T, seed, real reflect.Type) {
	t.Helper()
	if seed.NumField() != real.NumField() {
		t.Fatalf("%s: field count %d != %d", real.Name(), seed.NumField(), real.NumField())
	}
	for i := 0; i < seed.NumField(); i++ {
		sf, rf := seed.Field(i), real.Field(i)
		if sf.Name != rf.Name {
			t.Fatalf("%s: field %d name %q != %q", real.Name(), i, sf.Name, rf.Name)
		}
		if sf.Type != rf.Type {
			t.Fatalf("%s.%s: type %v != %v", real.Name(), sf.Name, sf.Type, rf.Type)
		}
		if got, want := sf.Tag.Get("gorm"), rf.Tag.Get("gorm"); got != want {
			t.Fatalf("%s.%s: gorm tag %q != %q", real.Name(), sf.Name, got, want)
		}
	}
}

func TestSeedModelTableNamesMatchDomains(t *testing.T) {
	cases := []struct{ seed, real string }{
		{seedmodel.Permission{}.TableName(), accessdomain.Permission{}.TableName()},
		{seedmodel.Role{}.TableName(), accessdomain.Role{}.TableName()},
		{seedmodel.Tenant{}.TableName(), tenantdomain.Tenant{}.TableName()},
		{seedmodel.Plan{}.TableName(), tenantdomain.Plan{}.TableName()},
		{seedmodel.User{}.TableName(), userdomain.User{}.TableName()},
	}
	for _, c := range cases {
		if c.seed != c.real {
			t.Fatalf("table name %q != %q", c.seed, c.real)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/kernel/seedmodel/ -v`
Expected: FAIL —— `no Go files` 或 `undefined: seedmodel.Permission`

- [ ] **Step 3: 创建 seedmodel 包**

把 5 个模型的字段、`gorm` tag、`json` tag 从 `internal/capabilities/{access,tenant,user}/domain` 逐字复制；
不复制仓储接口、构造函数、业务方法或包级错误变量。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/kernel/seedmodel/ -v`
Expected: PASS

- [ ] **Step 5: 让 `seed.go` 改用 seedmodel**

三处能力 import 换成 `"jimu/internal/kernel/seedmodel"`；`domain.Permission` → `seedmodel.Permission`、
`domain.Role` → `seedmodel.Role`、`tenantDomain.Tenant` → `seedmodel.Tenant`、
`tenantDomain.Plan` → `seedmodel.Plan`、`userdomain.User` → `seedmodel.User`。
`ensureSystemRole`、`hasActivePlatformSuperAdmin`、`WarnIfNoAccounts` 的签名随之改类型。

- [ ] **Step 6: 验证内核不再依赖能力 domain**

Run: `go list -deps ./internal/app | grep 'jimu/internal/capabilities' || echo CLEAN`
Expected: `CLEAN`

Run: `go build ./... && go test ./internal/app/ -run TestRunSeed -v`
Expected: 编译通过，既有 seed 测试全绿

- [ ] **Step 7: 提交**

```bash
git add internal/kernel/seedmodel internal/app/seed.go
git commit -m "refactor(kernel): extract seed models from capability domains"
```

---

### Task 2: 空能力集的内核语义（seed / migrate / ping）

**Files:**
- Modify: `internal/profiles/profiles.go`（`StructuralSeed` 首行）
- Modify: `internal/kernel/http/server.go`（`SetupRouter` 的 `return r` 之前）
- Modify: `cmd/cli/activecaps.go`（`resolveActive` 与 `declaredDescriptors` 的空集语义）
- Test: `internal/profiles/profiles_test.go`、`internal/kernel/http/server_test.go`、`cmd/cli/activecaps_test.go`

**Interfaces:**
- Consumes: `assembly.Context.DB() *gorm.DB`、`SetupRouter(log, httpCfg, serverCfg, securityCfg, otelCfg, reporters...) *gin.Engine`
- Produces: `GET /api/v1/ping` → `200`，body 含 `pong`
- Produces: `resolveActive(profile, declared)` 与 `declaredDescriptors()` 在空声明集时返回空切片 + nil error

**为什么空集必须一起改**：`cmd/cli/main.go` 的六个子命令（`migrate up/down/status`、`seed` 等）
都走 `activeDescriptors()` → `resolveActive()`，它现在对空集返回 `errNoCatalogCapabilities`。
`minimal` 是 0 能力项目，会让这些命令全部失败 —— 一个"最小可运行服务"连 `make migrate` 都跑不了。
对 `minimal`，空集是**正常状态**，不是配置错误。

- [ ] **Step 1: 写三个失败测试**

```go
// internal/profiles/profiles_test.go 追加
func TestStructuralSeedSkipsWithoutDatabase(t *testing.T) {
	for _, tc := range []struct {
		name       string
		withSecret bool
	}{
		{"without admin secret", false},
		{"with admin secret", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.withSecret {
				t.Setenv("ADMIN_PASSWORD", "probepass")
				t.Setenv("ADMIN_PASSWORD_FILE", "")
			} else {
				t.Setenv("ADMIN_PASSWORD", "")
				t.Setenv("ADMIN_PASSWORD_FILE", "")
			}
			logPath := filepath.Join(t.TempDir(), "server.log")
			log := logger.New(config.LogConfig{Output: logPath})
			ctx, _, err := assembly.WireFor(
				&app.Container{Config: &config.Config{}, DB: nil, Logger: log},
				nil, nil, assembly.Assembly{Name: "no-db"}, nil,
			)
			require.NoError(t, err)
			require.NoError(t, profiles.StructuralSeed(ctx), "nil DB must not panic")
			out, err := os.ReadFile(logPath)
			require.NoError(t, err)
			require.Contains(t, string(out), "structural seed skipped")
		})
	}
}

// 该文件已 import os/filepath/logger/config/app/assembly/profiles/require，无需新增。

// internal/kernel/http/server_test.go 追加
func TestPingRouteServedInAllModes(t *testing.T) {
	for _, mode := range []string{"debug", "release"} {
		t.Run(mode, func(t *testing.T) {
			router := SetupRouter(testLogger(),
				config.HTTPConfig{Mode: mode}, config.ServerConfig{}, config.SecurityConfig{},
				observability.TracingConfig{})
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/api/v1/ping", nil))
			require.Equal(t, stdhttp.StatusOK, rec.Code)
			require.Contains(t, rec.Body.String(), "pong")
		})
	}
}

// cmd/cli/activecaps_test.go 追加
func TestResolveActiveAllowsEmptySelection(t *testing.T) {
	caps, err := resolveActive("minimal", map[string]bool{})
	require.NoError(t, err, "a 0-capability project has nothing to migrate; that is not an error")
	require.Empty(t, caps)
}

func TestResolveActiveStillRejectsNonEmptyWithoutMigrations(t *testing.T) {
	// 保留现有 fail-closed 语义：选了能力但一个迁移都没有，仍是配置错误。
	_, err := resolveActive("broken", map[string]bool{"apidocs": true})
	require.ErrorIs(t, err, errNoCatalogCapabilities)
}
```

`server_test.go` 已 import `stdhttp`、`httptest`、`require` 与 `testLogger()`，直接追加，
不要重复声明 `testLogger`。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/profiles/ -run TestStructuralSeedSkipsWithoutDatabase -v`
Expected: FAIL —— `panic: runtime error: invalid memory address or nil pointer dereference`（`with admin secret` 子用例）

Run: `go test ./internal/kernel/http/ -run TestPingRouteServedInAllModes -v`
Expected: FAIL —— 404

Run: `go test ./cmd/cli/ -run TestResolveActiveAllowsEmptySelection -v`
Expected: FAIL —— `profile declares no catalog capabilities`

- [ ] **Step 3: 实现三个改动**

`StructuralSeed` 第一行加：

```go
if ctx.DB() == nil {
	ctx.Logger().Warnw("structural seed skipped", "missing", "database")
	return nil
}
```

`SetupRouter` 在 `return r` 之前加：

```go
r.GET("/api/v1/ping", func(c *gin.Context) {
	response.OK(c, http.StatusOK, gin.H{"message": "pong"})
})
```

`cmd/cli/activecaps.go` 的 `resolveActive` 与 `declaredDescriptors` 各加一次空集判断
（放在构建 `declared` map 之后、调用 `catalog.MigrationSet` / `catalog.All()` 之前）：

```go
if len(declared) == 0 {
	return []contract.Descriptor{}, nil
}
```

保留原有的"非空但无迁移" fail-closed 分支不变。

`profiles.go` 与 `server.go` 都已 import `response`、`gin`、`net/http`，不需要新增 import；
`activecaps.go` 已 import `contract` 与 `catalog`。

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/profiles/ ./internal/kernel/http/ ./cmd/cli/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/profiles/profiles.go internal/profiles/profiles_test.go \
  internal/kernel/http/server.go internal/kernel/http/server_test.go \
  cmd/cli/activecaps.go cmd/cli/activecaps_test.go
git commit -m "fix(kernel): guard empty selection and seed without database"
```

---

### Task 3: 预设解析层

**Files:**
- Create: `tools/generator/frameworkmanifest/presets.go`
- Create: `tools/generator/frameworkmanifest/presets_test.go`
- Modify: `tools/generator/frameworkmanifest/export.go`（`ExportDetailed` 的模板分支）
- Modify: `tools/generator/frameworkmanifest/profiles.go`（新增 `resolvePresetSelection`）

**Interfaces:**
- Produces:
  - `type Preset struct { Name string; Capabilities []string; Drivers map[string][]string; Support string }`
  - `func PresetNames() []string` —— 固定顺序 `["minimal","auth","full"]`
  - `func LookupPreset(name string) (Preset, error)` —— 未知名字返回带三个可用名的错误
  - `func resolvePresetSelection(root string, preset Preset) (selection, error)`（包内）

- [ ] **Step 1: 写失败的预设测试**

```go
// tools/generator/frameworkmanifest/presets_test.go
package frameworkmanifest

import (
	"slices"
	"strings"
	"testing"

	"jimu/internal/profiles/full"
)

func TestPresetNamesAreStable(t *testing.T) {
	want := []string{"minimal", "auth", "full"}
	if got := PresetNames(); !slices.Equal(got, want) {
		t.Fatalf("PresetNames() = %v, want %v", got, want)
	}
}

func TestMinimalPresetHasNoCapabilities(t *testing.T) {
	p, err := LookupPreset("minimal")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Capabilities) != 0 || len(p.Drivers) != 0 {
		t.Fatalf("minimal = %+v, want empty", p)
	}
	if p.Support != "official" {
		t.Fatalf("minimal support = %q, want official", p.Support)
	}
}

func TestAuthPresetSelectsAuthTriple(t *testing.T) {
	p, err := LookupPreset("auth")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"user", "access", "auth"}; !slices.Equal(p.Capabilities, want) {
		t.Fatalf("auth capabilities = %v, want %v", p.Capabilities, want)
	}
	if p.Support != "official" {
		t.Fatalf("auth support = %q, want official", p.Support)
	}
}

func TestFullPresetCoversFrameworkFull(t *testing.T) {
	p, err := LookupPreset("full")
	if err != nil {
		t.Fatal(err)
	}
	framework := full.Assembly().Capabilities
	if len(p.Capabilities) != len(framework) {
		t.Fatalf("full preset has %d capabilities, framework full has %d", len(p.Capabilities), len(framework))
	}
	selected := map[string]bool{}
	for _, name := range p.Capabilities {
		selected[name] = true
	}
	for _, item := range framework {
		if !selected[item.Descriptor.Name] {
			t.Fatalf("full preset missing %q", item.Descriptor.Name)
		}
	}
}

func TestFullPresetDriversMatchFrameworkFull(t *testing.T) {
	p, err := LookupPreset("full")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range full.Assembly().Capabilities {
		got := p.Drivers[item.Descriptor.Name]
		if !slices.Equal(got, item.Drivers) {
			t.Fatalf("driver %s = %v, want %v", item.Descriptor.Name, got, item.Drivers)
		}
	}
	if p.Support != "validated" {
		t.Fatalf("full support = %q, want validated", p.Support)
	}
}

func TestUnknownPresetListsAvailable(t *testing.T) {
	_, err := LookupPreset("ghost")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, name := range PresetNames() {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("error %q does not mention %q", err, name)
		}
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./tools/generator/frameworkmanifest/ -run TestPreset -v`
Expected: FAIL —— `undefined: PresetNames`

- [ ] **Step 3: 实现 `presets.go`**

```go
package frameworkmanifest

import (
	"fmt"
	"slices"

	"jimu/internal/profiles/full"
)

type Preset struct {
	Name         string
	Capabilities []string
	Drivers      map[string][]string
	Support      string
}

var presetOrder = []string{"minimal", "auth", "full"}

func PresetNames() []string { return slices.Clone(presetOrder) }

func LookupPreset(name string) (Preset, error) {
	switch name {
	case "minimal":
		return Preset{Name: "minimal", Support: "official"}, nil
	case "auth":
		return Preset{Name: "auth", Capabilities: []string{"user", "access", "auth"}, Support: "official"}, nil
	case "full":
		return fullPreset(), nil
	default:
		return Preset{}, fmt.Errorf("unknown template %q (available: minimal, auth, full)", name)
	}
}

func fullPreset() Preset {
	preset := Preset{Name: "full", Drivers: map[string][]string{}, Support: "validated"}
	for _, item := range full.Assembly().Capabilities {
		preset.Capabilities = append(preset.Capabilities, item.Descriptor.Name)
		if len(item.Drivers) > 0 {
			preset.Drivers[item.Descriptor.Name] = slices.Clone(item.Drivers)
		}
	}
	return preset
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./tools/generator/frameworkmanifest/ -run 'TestPreset|TestMinimalPreset|TestAuthPreset|TestFullPreset|TestUnknownPreset' -v`
Expected: PASS

- [ ] **Step 5: 在 `profiles.go` 增加 `resolvePresetSelection`**

```go
func resolvePresetSelection(root string, preset Preset) (selection, error) {
	all, inCatalog := allDescriptors()
	composed := capability.Compose(all, capability.ComposeRequest{
		Capabilities:   preset.Capabilities,
		Drivers:        preset.Drivers,
		DefaultDrivers: false,
		Support:        contract.SupportLevel(preset.Support),
	})
	if err := composed.Err(); err != nil {
		return selection{}, err
	}
	selected := selection{
		shape:    preset.Name,
		template: preset.Name,
		support:  preset.Support,
		drivers:  composed.Drivers,
		ungated:  map[string]bool{},
	}
	if selected.drivers == nil {
		selected.drivers = map[string][]string{}
	}
	selected.requested = composed.Requested
	selected.autoAdded = composed.AutoAdded
	for _, descriptor := range composed.Resolved {
		selected.declared = append(selected.declared, descriptor.Name)
		if !inCatalog[descriptor.Name] {
			selected.ungated[descriptor.Name] = true
		}
	}
	selected.migration = composed.MigrationCarriers
	selected.assemblies = assemblyCapabilitiesFor(selected.declared, selected.drivers)
	return completeSelection(root, selected, all)
}
```

形态名固定取预设名：`--shape` 与 `--template` 互斥，现有校验已经拦掉，所以不需要额外参数。

`minimal` 会保留一个**空的** `internal/capabilities/catalog` 包（由 `catalog.go.tmpl` 渲染，
`entries` 为空切片）。保留它是因为 `internal/app` 与 `cmd/cli` 无条件 import catalog；
删掉它要同时改内核 import，属于 Task 1 没覆盖的额外耦合。这条写进代码注释，不要当成遗漏。

**不要改 `ParseCapabilitySet`。** 它服务 `RenderShape`，而 `RenderShape` 只在测试里调用；
真正驱动 `jimu new` 的是 `ExportDetailed`。预设必须落在生成路径上。

- [ ] **Step 6: 把 `ExportDetailed` 的模板分支接到预设**

把 `req.Template == "auth-api"` / `basicAPIDocument` 分支整体换成：

```go
if req.Template != "" {
	preset, err := LookupPreset(req.Template)
	if err != nil {
		return manifest.Document{}, Resolution{}, err
	}
		selected, err = resolvePresetSelection(root, preset)
	if err == nil {
		doc, err = buildDocument(root, req.Module, selected, req.Framework)
	}
}
```

保留 `req.Template == ""` 时的 `resolveSelection` 分支（`--with` / `--profile`）。

- [ ] **Step 7: 端到端验证三个预设能生成**

Run:
```bash
for p in minimal auth full; do
  go run ./cmd/jimu new "/tmp/jimu-preset-$p" --template=$p --no-tidy || exit 1
  python3 -c "import json;d=json.load(open('/tmp/jimu-preset-$p/.jimu/manifest.json'));print('$p',d['selection']['template'],d['selection']['shape'],d['composition']['support'],len(d['composition']['resolved']))"
done
```
Expected: 三行，support 依次 `certified`/`certified`/`validated`，resolved 依次 `0`/`3`/`23`

- [ ] **Step 8: 提交**

```bash
git add tools/generator/frameworkmanifest/presets.go tools/generator/frameworkmanifest/presets_test.go tools/generator/frameworkmanifest/export.go tools/generator/frameworkmanifest/profiles.go
git commit -m "feat(generator): add minimal, auth and full presets"
```

---

### Task 4: 预设项目的 `capability add`

**Files:**
- Modify: `tools/generator/frameworkmanifest/export.go`（`ExportForAdd` 的模板拒绝分支）
- Modify: `tools/generator/add_test.go`

**Interfaces:**
- Consumes: `LookupPreset`（Task 3）
- Produces: 预设项目 `capability add` 后 `selection.template == ""`、`composition.support == "experimental"`

- [ ] **Step 1: 写四个失败测试**

```go
// tools/generator/add_test.go 追加
func TestAddCapabilityToMinimalPresetProject(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	res, err := AddCapability(AddOptions{Dir: dir, From: FrameworkRoot(), Name: "tenant"})
	require.NoError(t, err)
	require.Contains(t, res.Changed, "internal/capabilities/tenant/module.go")

	doc, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	require.Empty(t, doc.Selection.Template, "added preset project must fall back to composition")
	require.Equal(t, manifest.CompositionSupportExperimental, doc.Composition.Support)
	require.Contains(t, doc.Composition.Requested, "tenant")
}

func TestAddCapabilityToAuthPresetProject(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "auth"})
	_, err := AddCapability(AddOptions{Dir: dir, From: FrameworkRoot(), Name: "mfa"})
	require.NoError(t, err)
	doc, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"user", "access", "auth", "mfa"}, doc.Composition.Requested)
	require.Equal(t, manifest.CompositionSupportExperimental, doc.Composition.Support)
}

func TestAddExistingCapabilityToFullPresetFails(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "full"})
	_, err := AddCapability(AddOptions{Dir: dir, From: FrameworkRoot(), Name: "tenant"})
	require.ErrorContains(t, err, "already present as a declared capability")
}

func TestAddCapabilityToRetiredTemplateFails(t *testing.T) {
	dir := writeManifestWithTemplate(t, "basic-api")
	_, err := AddCapability(AddOptions{Dir: dir, From: FrameworkRoot(), Name: "tenant"})
	require.ErrorContains(t, err, `template "basic-api" is retired`)
}
```

`writeManifestWithTemplate(t *testing.T, template string) string` 写成测试辅助函数：
在 `t.TempDir()` 里创建 `.jimu/manifest.json`，`Selection` 里填 `Shape`/`Template`/`Support` 与非空
`Capabilities`，`Composition` 与 `Rewrites`（`Kind: "module"`, `To: "example.com/legacy"`）填最小合法值，
最后用 `manifest.Write` 落盘（它会算 digest）。放在 `tools/generator/generator_test_helpers_test.go`。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./tools/generator/ -run 'TestAddCapabilityToMinimalPreset|TestAddCapabilityToAuthPreset' -v`
Expected: FAIL —— `capability add does not extend official template projects (minimal)`

- [ ] **Step 3: 改拒绝条件**

```go
switch previous.Selection.Template {
case "basic-api", "auth-api":
	return manifest.Document{}, fmt.Errorf(
		"template %q is retired and cannot be extended; regenerate with --template=minimal|auth|full or --with",
		previous.Selection.Template)
}
```

位置：`manifest.Validate(previous)` 之后，`for _, capability := range previous.Capabilities` 之前。

- [ ] **Step 4: 运行确认通过**

Run: `go test ./tools/generator/ -run TestAdd -v`
Expected: 全部 PASS，包括既有 `--with` 项目用例

- [ ] **Step 5: 验证加完能力能编译**

Run:
```bash
go run ./cmd/jimu new /tmp/jimu-add-min --template=minimal --no-tidy
cd /tmp/jimu-add-min && go build ./... && \
  go run /Users/king/Documents/Projects/Mine/Jimu/jimu/cmd/jimu capability add tenant && go build ./...
```
Expected: 两次 `go build` 都退出码 0

- [ ] **Step 6: 提交**

```bash
git add tools/generator/frameworkmanifest/export.go tools/generator/add_test.go tools/generator/generator_test_helpers_test.go
git commit -m "feat(generator): allow capability add on preset projects"
```

---

### Task 5: 内核闭包裁剪

**Files:**
- Modify: `tools/generator/frameworkmanifest/actions.go`（`kernelDirs` 循环改闭包）
- Modify: `tools/generator/layout.go`（删除 `kernelRequiredDomains`）
- Test: `tools/generator/kerneldomains_test.go`

**Interfaces:**
- Produces（包内）：
  - `func kernelCopyActions(root string, selected selection) ([]manifest.CopyAction, error)`
  - `func kernelPackages(root string, seeds []string) (map[string]bool, error)` —— 返回可达的 `internal/...` 相对目录集合
  - `func anyPackageUnder(reachable map[string]bool, dir string) bool`

**种子选择（重要，别照抄"从 cmd/server 出发"）**

实测：从 `./cmd/server ./cmd/cli` 求闭包会得到 **83 个能力包**，因为
`cmd/server → internal/profiles/active → internal/profiles/full → 全部 23 个能力`。
用它做裁剪等于什么都没删。

正确种子是**内核入口 + 已选能力根包**：

```
seeds = ["./internal/assembly", "./internal/capability"] + ["./internal/capabilities/"+name for name in selected.copy]
```

- `internal/assembly` 拉进 `kernel/*`、`app`、`config`、`contract`、`shared` 的真实子集
- `internal/capability` 拉进组合算法叶子
- 已选能力根包拉进该能力的全部子包
- `cmd/server` 与 `cmd/cli` 不参与闭包：它们的 import 由生成器**重写**（Task 3 的
  `selected.declared` 决定 `active` 与 `cli` 的内容），复制它们本身不需要闭包判断
- `internal/e2e` 只有测试文件，`Tests: false` 下永远不可达，因此 `minimal` 会自动丢掉它

实测（用这套种子、`selected.copy` 为空时）：`internal/kernel` 129 个 Go 文件里只有 16 个包的闭包可达，
`internal/e2e` 可达包数为 0。

- [ ] **Step 1: 写失败的裁剪测试**

```go
// tools/generator/kerneldomains_test.go 追加
func TestMinimalPresetDropsUnreachableKernelPackages(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	require.NoDirExists(t, filepath.Join(dir, "internal/e2e"),
		"internal/e2e is not reachable from cmd/server or cmd/cli")
	require.NoDirExists(t, filepath.Join(dir, "internal/capabilities"),
		"a 0-capability project must not ship any capability package")
	for _, kept := range []string{
		"internal/kernel/http",
		"internal/kernel/observability",
		"internal/assembly",
		"cmd/server",
		"cmd/cli",
	} {
		require.DirExists(t, filepath.Join(dir, kept), "%s is required by the entrypoint", kept)
	}
}

func TestMinimalPresetDropsOnlyUnreachableKernelSubpackages(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	require.NoDirExists(t, filepath.Join(dir, "internal/kernel", "cache"),
		"internal/kernel/cache is only reachable through capabilities")
	require.DirExists(t, filepath.Join(dir, "internal/kernel", "http"))
	require.DirExists(t, filepath.Join(dir, "internal/kernel", "db"))
}

func TestFullPresetKeepsEveryKernelPackage(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "full"})
	for _, kept := range []string{"internal/kernel/http", "internal/kernel/db", "internal/shared"} {
		require.DirExists(t, filepath.Join(dir, kept))
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./tools/generator/ -run 'TestMinimalPresetDrops|TestFullPresetKeepsEvery' -v`
Expected: FAIL —— `internal/e2e should not exist`

- [ ] **Step 3: 实现闭包裁剪**

在 `actions.go` 新增：

```go
func kernelCopyActions(root string, selected selection) ([]manifest.CopyAction, error) {
	seeds := []string{"./internal/assembly", "./internal/capability"}
	for _, name := range selected.copy {
		seeds = append(seeds, "./internal/capabilities/"+name)
	}
	reachable, err := kernelPackages(root, seeds)
	if err != nil {
		return nil, err
	}
	actions := make([]manifest.CopyAction, 0, len(kernelDirs)+len(kernelFiles))
	for _, dir := range kernelDirs {
		// conf/ 是数据目录，不是 Go 包；内核鉴权模型必须随包复制。
		if dir != "conf" && !anyPackageUnder(reachable, dir) {
			continue
		}
		actions = append(actions, manifest.CopyAction{
			Source: dir, Destination: dir,
			Include: []string{}, Exclude: kernelExcludesFor(dir),
		})
	}
	for _, file := range kernelFiles {
		actions = append(actions, manifest.CopyAction{Source: file, Destination: file, Include: []string{}, Exclude: []string{}})
	}
	return actions, nil
}

func kernelPackages(root string, seeds []string) (map[string]bool, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps,
		Dir:   root,
		Tests: false,
	}, seeds...)
	if err != nil {
		return nil, fmt.Errorf("load kernel packages: %w", err)
	}
	reachable := map[string]bool{}
	var loadErrors []string
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, e := range pkg.Errors {
			loadErrors = append(loadErrors, e.Error())
		}
		rel, ok := strings.CutPrefix(pkg.PkgPath, "jimu/")
		if !ok {
			return
		}
		reachable[rel] = true
	})
	if len(loadErrors) > 0 {
		return nil, fmt.Errorf("kernel package load errors: %s", strings.Join(loadErrors, "; "))
	}
	return reachable, nil
}

func anyPackageUnder(reachable map[string]bool, dir string) bool {
	if reachable[dir] {
		return true
	}
	prefix := dir + "/"
	for pkg := range reachable {
		if strings.HasPrefix(pkg, prefix) {
			return true
		}
	}
	return false
}
```

`kernelPackages` 里模块前缀固定用 `"jimu/"`（`frameworkModule` 常量），不用传入的 `module`：
闭包是在**框架仓库**里算的，与目标项目模块名无关。

`copyActions` 里原来的 `for _, source := range kernelDirs` 循环整体替换为：

```go
kernelActions, err := kernelCopyActions(root, module)
if err != nil {
	// copyActions 目前无 error 返回；改成 ([]manifest.CopyAction, error) 并由调用方传播
}
actions = append(actions, kernelActions...)
```

`copyActions` 的签名改为 `func copyActions(root string, selected selection) ([]manifest.CopyAction, error)`，
两个调用点同步加错误处理：

- `frameworkmanifest/actions.go` 的 `generatedFiles`（第 359 行附近）
- `frameworkmanifest/export.go` 的 `buildDocument` 里 `Copy: copyActions(...)`（第 239 行附近）

`actions.go` 需要新增 import：`"fmt"`、`"golang.org/x/tools/go/packages"`。

- [ ] **Step 4: 删除 `kernelRequiredDomains`**

删除 `layout.go` 的 `kernelRequiredDomains` 常量及其在 `CapabilityRoots` 里的循环；
`DomainOnly` 置空逻辑一并删除（Task 1 后 `internal/app` 不再 import 能力 domain）。
`tools/generator/kerneldomains_test.go` 里依赖该常量的断言同步删除，替换为 Step 1 的两个用例。

- [ ] **Step 5: 运行确认通过**

Run: `go test ./tools/generator/ -run 'TestMinimalPresetDrops|TestFullPresetKeepsEvery' -v`
Expected: PASS

Run:
```bash
go run ./cmd/jimu new /tmp/jimu-closure-min --template=minimal --no-tidy
find /tmp/jimu-closure-min -type f | wc -l
(cd /tmp/jimu-closure-min && go build ./...)
```
Expected: 文件数低于闭包裁剪前的 `--template=minimal` 产物；`go build` 退出码 0

- [ ] **Step 6: 提交**

```bash
git add tools/generator/frameworkmanifest/actions.go tools/generator/layout.go tools/generator/kerneldomains_test.go tools/generator/frameworkmanifest/export.go
git commit -m "refactor(generator): prune kernel packages by import closure"
```

---

### Task 6: 退役旧模板

**Files:**
- Delete: `tools/generator/frameworkmanifest/basic.go`、`tools/generator/frameworkmanifest/auth.go`
- Delete: `tools/generator/render/templates/basicapi/`、`tools/generator/render/templates/authapi/`
- Delete: `tools/generator/basicapi_test.go`
- Delete: `tools/generator/testdata/upgrade/v0.5/`
- Modify: `tools/generator/upgrade.go`（`upgradeRequest`）
- Modify: `tools/generator/upgrade_fixture_test.go`
- Modify: `docs/templates/v1/budgets.json`、`docs/templates/v1/matrix.yaml`

**Interfaces:**
- Produces: `var ErrRetiredTemplate = errors.New("template is retired")`
- `upgradeRequest` 对旧模板返回 `%w` 包裹 `ErrRetiredTemplate` 的错误，且发生在模块解析之前

- [ ] **Step 1: 写失败的退役测试**

```go
// tools/generator/upgrade_test.go 追加
func TestUpgradeRejectsRetiredTemplates(t *testing.T) {
	for _, template := range []string{"basic-api", "auth-api"} {
		t.Run(template, func(t *testing.T) {
			dir := writeManifestWithTemplate(t, template)
			before := hashTree(t, dir)
			_, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
			require.ErrorIs(t, err, ErrRetiredTemplate)
			require.Contains(t, err.Error(), template)
			require.Contains(t, err.Error(), "--template=")
			require.Equal(t, before, hashTree(t, dir), "retired template must not write")
		})
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./tools/generator/ -run TestUpgradeRejectsRetiredTemplates -v`
Expected: FAIL —— `undefined: ErrRetiredTemplate`

- [ ] **Step 3: 实现硬失败**

```go
var ErrRetiredTemplate = errors.New("template is retired")

func upgradeRequest(previous manifest.Document, root, projectDir string, framework *manifest.Framework) (frameworkmanifest.Request, error) {
	switch previous.Selection.Template {
	case "basic-api", "auth-api":
		return frameworkmanifest.Request{}, fmt.Errorf(
			"%w: project was generated with retired template %q and cannot be upgraded. "+
				"Regenerate with --template=minimal|auth|full (or --with) and merge your application code",
			ErrRetiredTemplate, previous.Selection.Template)
	}
	module := moduleFromManifest(previous)
	// ...其余逻辑不变
}
```

- [ ] **Step 4: 删除旧渲染路径与模板树**

删除上面 Files 列出的文件与目录，以及 `ExportDetailed` 中对 `basicAPIDocument`/`authAPIDocument` 的引用（Task 3 已换掉）。
同时删除 `make` 目标里对这些测试的引用（Task 7 处理 Makefile 变量）。

- [ ] **Step 5: 重写升级 fixture**

```go
// tools/generator/upgrade_fixture_test.go
const (
	upgradeFixtureRoot    = "testdata/upgrade/v0.10"
	upgradeFixtureVersion = "v0.10.0"
)

var upgradeFixtureCases = []upgradeFixtureCase{
	{template: "auth", module: "example.com/upgrade-auth", generatedFiles: 0},
}
```

保留 `TestUpgradeFixturesMatchV050` 的结构但改名为 `TestUpgradeFixturesMatchV010`，
断言 `selection.template == "auth"`、`composition.support == "certified"`。
`prepareUpgradeFixture` 改为从 `testdata/upgrade/v0.10/auth/` 复制（该目录由本步骤用
`--template=auth` 生成后提交）。`securityChanges` 字段按新模板实测填写。

- [ ] **Step 6: 更新预算与矩阵**

`budgets.json`：删除 `basic-api`/`auth-api`，新增三预设。
`matrix.yaml`：`default_template: minimal`；`templates` 段替换为三个预设；
`support_levels` 增加 `validated`。

预算数字必须实测，不能凭空写。先用宽松上界跑 `TestPresetBudgets` 得到实测值，再写回。

- [ ] **Step 7: 运行确认通过**

Run: `go build ./... && go test ./tools/generator/ -run 'TestUpgradeRejectsRetiredTemplates|TestUpgradeFixtures' -v`
Expected: PASS

- [ ] **Step 8: 提交**

```bash
git add -A tools/generator docs/templates/v1
git commit -m "feat(generator): retire basic-api and auth-api templates"
```

---

### Task 7: 门禁与文档

**Files:**
- Modify: `Makefile`（`SCAFFOLD_*_TESTS` 变量、新增 `check-gate-selectors`、`check` 目标）
- Create: `tools/checkgateselectors/main.go`、`tools/checkgateselectors/main_test.go`
- Modify: `tools/generator/templategate_test.go`（替换模板门禁测试）
- Modify: `.github/workflows/ci-scaffold.yml`（clean-room job 改用新预设；`heavy` 过滤补路径）
- Modify: `scripts/clean_room_e2e.sh`（新预设 + 去掉 Compose）
- Delete: `scripts/test_readme_contract.sh`、`scripts/test_release_workflow.sh`（孤儿脚本）
- Modify: `docs/releases/v0.10.0.md`、`docs/design/profiles-and-project-generation.md`、
  `docs/templates/v1/README.md`、`README.md`、`docs/PRODUCTION.md`

**Interfaces:**
- Produces: `make check-gate-selectors`，失败时非零退出
- `func requireMatches(regex, goListOutput string) error`（`checkgateselectors` 包内，供测试）

- [ ] **Step 1: 写失败的选择器测试**

```go
// tools/checkgateselectors/main_test.go
package main

import "testing"

func TestParseMakefileSelectors(t *testing.T) {
	src := "SCAFFOLD_OFFICIAL_TESTS := ^(TestA|TestB)$\nSCAFFOLD_UPGRADE_TESTS := ^(TestC)$\n"
	got, err := parseSelectors(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d selectors, want 2", len(got))
	}
	if got["SCAFFOLD_OFFICIAL_TESTS"] != "^(TestA|TestB)$" {
		t.Fatalf("official = %q", got["SCAFFOLD_OFFICIAL_TESTS"])
	}
}

func TestEmptyMatchIsAnError(t *testing.T) {
	if err := requireMatches("^TestNope$", "ok  \tjimu/tools/generator\t0.5s\n"); err == nil {
		t.Fatal("empty match must be an error")
	}
}

func TestNonEmptyMatchPasses(t *testing.T) {
	out := "TestAlpha\nTestBeta\nok  \tjimu/tools/generator\t0.5s\n"
	if err := requireMatches("^Test(Alpha|Beta)$", out); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./tools/checkgateselectors/ -v`
Expected: FAIL —— 包不存在

- [ ] **Step 3: 实现检查器**

`main.go`：

```go
package main

func parseSelectors(makefile string) (map[string]string, error)
func requireMatches(regex, output string) error
func main()
```

`parseSelectors` 逐行匹配 `^(SCAFFOLD_[A-Z_]+_TESTS)\s*:=\s*(.+)$`，返回 map。
`requireMatches` 扫描输出行，`strings.HasPrefix(line, "Test")` 计数；为 0 时返回错误。
`main` 读仓库根 `Makefile`（从 cwd 向上找含 `module jimu` 的 `go.mod`），
对每个选择器执行 `go test ./tools/generator/ -list <regex>`，把 stdout 交给 `requireMatches`；
任何一条失败即 `os.Exit(1)` 并打印选择器名与原始输出。

`go` 的路径用 `os.Getenv("GO")`，缺省 `"go"`。**不要**解析 `go test` 的退出码，
因为它在零匹配时也返回 0（已实测）。

- [ ] **Step 4: 运行确认通过**

Run: `go test ./tools/checkgateselectors/ -v && go run ./tools/checkgateselectors`
Expected: PASS，且真实 Makefile 的三个选择器都非空

- [ ] **Step 5: 更新 Makefile**

```make
SCAFFOLD_OFFICIAL_TESTS := ^(TestPresetBudgets|TestMinimalPresetStartsWithoutDependencies|TestAuthPresetEmptyDatabaseEndToEnd)$
SCAFFOLD_UPGRADE_TESTS := ^(TestUpgradeFixtures|TestUpgradeRejectsRetiredTemplates)$
SCAFFOLD_COMPOSITION_TESTS := ^(TestRepresentativeCompositionMatrix|TestCompositionMatrixBudget|TestCompositionNegativeMatrix)$

check-gate-selectors:
	@go run ./tools/checkgateselectors
```

把 `check-gate-selectors` 加入 `check` 目标的依赖列表。

- [ ] **Step 6: 写三个预设门禁测试**

在 `tools/generator/templategate_test.go` 替换已删除的两个模板测试：

- `TestPresetBudgets`：对 `minimal`/`auth`/`full` 复用现有度量逻辑（`ClosureSize`、`directDependencyCount`、
  `externalServiceCount`、`document.Report.Routes` 等），与 `budgets.json` 比较。
- `TestMinimalPresetStartsWithoutDependencies`：生成 `minimal`，`go run ./cmd/server` 起进程，
  轮询 `http://127.0.0.1:<mgmt>/readyz` 直到 200，再断言 `/api/v1/ping` 返回 `pong`，最后关停。
  两个子用例：设置与不设置 `ADMIN_PASSWORD`。
- `TestAuthPresetEmptyDatabaseEndToEnd`：生成 `auth`，用 `docker run` 起 MySQL 8.4 与 Redis 7，
  跑 `make migrate && make seed`，验 `/readyz` 与一次登录。端口用 `freePort(t)`，`t.Cleanup` 里
  `docker rm -f`。**不使用 `docker-compose.yml`**（新预设不生成）。

- [ ] **Step 7: 更新 CI 与 clean-room 脚本**

三处必须改，否则 CI 会在删模板后直接失败或静默变空。

**(a) `scripts/clean_room_e2e.sh`**：整个脚本从"basic + auth 双模板"改为"minimal + auth 双预设"。

- `basic_dir` / `basic_pid` 改名为 `min_dir` / `min_pid`（纯重命名，保持可读）
- `"$JIMU_BIN" new demo --template=basic-api` → `--template=minimal`
- `"$JIMU_BIN" new demo --template=auth-api` → `--template=auth`
- 删掉 `make up` / `make verify` / `docker compose logs` 三处：新预设不生成 `docker-compose.yml`。
  `auth` 段落改为：
  ```bash
  ( cd "$auth_dir" && "$JIMU_BIN" new demo --template=auth --no-tidy )
  cd "$auth_dir/demo"
  "$JIMU_BIN" check .
  "$JIMU_BIN" upgrade . --check
  make build
  DB_HOST=127.0.0.1 DB_PORT=13306 DB_USER=jimu DB_PASSWORD=jimu DB_NAME=jimu \
    REDIS_ADDR=127.0.0.1:16379 \
    APP_ENV=dev HTTP_PORT=18081 MANAGEMENT_PORT=19091 ./bin/server >"$auth_dir/server.log" 2>&1 &
  auth_pid=$!
  wait_http "http://127.0.0.1:19091/livez" || fail "auth did not become ready"
  curl -fsS "http://127.0.0.1:18081/api/v1/ping" >/dev/null || fail "auth ping failed"
  ```
  数据库由 CI 的 service container 提供（见 (b)），脚本本身不负责起库。
- 最后一行 `printf 'PASS: clean-room basic-api and auth-api\n'` 改为
  `printf 'PASS: clean-room minimal and auth presets\n'`
- `minimal` 段落保留 `/livez` `/readyz` `/api/v1/ping` 三条断言（`/api/v1/ping` 现在由内核提供，
  Task 2 已加）

**(b) `.github/workflows/ci-scaffold.yml`**：

- `clean-room` job 增加 MySQL 与 Redis service container：
  ```yaml
  services:
    mysql:
      image: mysql:8.4
      env:
        MYSQL_ROOT_PASSWORD: root
        MYSQL_DATABASE: jimu
        MYSQL_USER: jimu
        MYSQL_PASSWORD: jimu
      ports: ["13306:3306"]
      options: >-
        --health-cmd="mysqladmin ping -h 127.0.0.1 -uroot -proot"
        --health-interval=3s --health-timeout=3s --health-retries=30
    redis:
      image: redis:7-alpine
      ports: ["16379:6379"]
      options: >-
        --health-cmd="redis-cli ping"
        --health-interval=3s --health-timeout=2s --health-retries=30
  ```
- step 名称 `"🧪 Run clean-room basic-api and auth-api"` 改为
  `"🧪 Run clean-room minimal and auth presets"`
- 文件头注释里 `official：两个官方模板的预算、basic smoke 与 auth 空库 Compose E2E` 改为
  `official：三个预设的预算、minimal 无依赖 smoke 与 auth 空库 E2E`
- `changes` job 的路径过滤正则已经覆盖 `tools/`、`internal/`、`Makefile`、`scripts/`，
  不需要为新预设加路径；但**要加 `docs/templates/v1/`**（已在正则里，确认即可）

**(c) 删除两个孤儿契约脚本**

`scripts/test_readme_contract.sh` 与 `scripts/test_release_workflow.sh` 从创建起就没有被任何门禁调用
（已核对 `Makefile`、`ci.yml`、`ci-scaffold.yml`、`release.yml`，均无引用），
也没有可执行位（`100644`）。而且 `test_readme_contract.sh` 现在跑就是红的：

```
FAIL: docs/PRODUCTION.md is missing: P1-02
FAIL: docs/PRODUCTION.md is missing: 未完成
```

它断言的是早已过期的一次性状态，不是契约。**直接删除两个文件**，不修、不接入。

```bash
git rm scripts/test_readme_contract.sh scripts/test_release_workflow.sh
```

**删除后必须补上它们唯一有价值的断言。** 这两个脚本里唯一值得保留的是：

1. release 资产契约：CLI 四平台 + `template-bundle.json` + checksums（原 `test_release_workflow.sh`）
2. release 不再含 server 二进制（本次 Task 8 新增）

把这两条放进 `scripts/test_makefile_contract.sh`（它已被 `check-makefile` 调用，
而 `check-makefile` 在 `make check` 与 `CI_CHECKS` 里 —— 是真正生效的门禁）。
该文件已有 `assert_contains` / `assert_not_contains` 辅助函数，直接复用：

```bash
RELEASE_WORKFLOW="$ROOT/.github/workflows/release.yml"

# release 资产契约：CLI + bundle + checksums 必须在
for needle in 'build-cli:' './cmd/jimu' 'jimu.BundleVersion' 'template-bundle.json' \
              'sha256sum' 'jimu-${GITHUB_REF_NAME}-checksums.txt'; do
  assert_contains "$(cat "$RELEASE_WORKFLOW")" "$needle" "release workflow 缺少 $needle"
done

# server 二进制已退役（Task 8）：这三样不得再出现
for needle in './cmd/server' 'matrix.profile' 'jimu-${PROFILE}'; do
  assert_not_contains "$(cat "$RELEASE_WORKFLOW")" "$needle" "release workflow 仍含已退役的 $needle"
done
```

放在脚本末尾 `echo "✅ Makefile contract passed"` 之前。

**不要**把这套断言写成新的独立脚本 —— 那会重演"写了没人调用"的问题。
`check-makefile` 已经在门禁链上，这是唯一不需要额外接线的地方。

**为什么 `.github/workflows/ci.yml` 不用改**：它只跑 `ci` 目标（单元测试 + vet + lint），
不涉及生成模板。`release.yml` 的 `profile: full|minimal` 是**框架自身的构建形态**，
不是生成预设，与本次改动无关，保持不动。

- [ ] **Step 8: 更新文档**

- `docs/releases/v0.10.0.md`：新增「不兼容变更」段——三预设、旧模板退役、Compose 不再生成、`/api/v1/ping` 新增。
- `docs/templates/v1/README.md`：说明「内核路由不计入 `report.routes`」。
- `docs/design/profiles-and-project-generation.md`：产品入口改为三预设；`--profile=minimal` 的替代从 `auth-api` 改为 `--template=auth`。
- `README.md`：`jimu new` 示例改三预设。
- `scripts/clean_room_e2e.sh`：`--template=basic-api` → `--template=minimal`，`--template=auth-api` → `--template=auth`。

- [ ] **Step 9: 全量验证**

Run:
```bash
go build ./... && go test ./... -count=1
make check                 # 含 check-makefile → test_makefile_contract.sh（已含 release 契约）
make check-gate-selectors
make test-scaffold-official
bash -n scripts/clean_room_e2e.sh
```
Expected: 全部 PASS；`test-scaffold-official` 实际执行非零个用例。

clean-room 的完整链路（含 Docker service container）只在 CI 上跑；本地只验证脚本语法。
CI 侧在 `release/*` PR 上自动跑，不需要手动 `workflow_dispatch`。

删掉 `test_readme_contract.sh` 后，README 不再有自动契约检查。
这是有意的取舍：它的断言要么已过期（`P1-02`/`未完成`），要么可以并入
`test_makefile_contract.sh` 的 release 契约块。不要为了"检查 README 文本"再新建一个脚本。

- [ ] **Step 10: 提交**

```bash
git add -A Makefile tools/checkgateselectors tools/generator docs README.md scripts .github/workflows/ci-scaffold.yml
git commit -m "chore(gates): guard scaffold selectors and document presets"
```

---

### Task 8: Release 只发布 CLI

**Files:**
- Modify: `.github/workflows/release.yml`（删除 `build` job）
- Modify: `scripts/test_makefile_contract.sh`（release 资产契约 + 退役反向断言）
- Modify: `docs/releases/v0.10.0.md`（不兼容变更）
- Test: `make check-makefile`

**Interfaces:**
- 不产生新代码接口。产出是一条契约：release 资产集合 = 4 个 CLI + `template-bundle.json` + checksums。

**为什么**：Jimu 的交付物是代码。用户 `jimu new` 拿到工程后自行构建；
发布预构建的 `jimu-full-*` / `jimu-minimal-*` server 二进制与这个模型矛盾，
也与 ADR 0002（Release CLI 只做 scaffold）矛盾。

**已逐条核对的证据**（这是本任务的判断依据，执行时不要重新论证）：

1. `jimu-web/vite.config.ts` 只要求一个 HTTP 端点（`JIMU_API_PROXY_TARGET ?? 'http://localhost:8080'`），
   不关心端点由什么产生。
2. `jimu-web/README.md`「起后端」给的是 `cd ../jimu && make compose-up` —— Compose 从源码构建。
3. `scripts/demo.sh` 走 `make up` → `docker-compose.yml` 的 `build: context: .`，同样从源码构建。
4. 全仓当前文档没有任何一处让用户下载 `jimu-full-*` / `jimu-minimal-*`（只有历史 plan/spec 提到）。
5. **server 二进制不是自包含的**：`Dockerfile` runtime 阶段 COPY `conf/`、`configs/`、`docs/openapi/`；
   `internal/config` 缺 `app.yaml` 时硬失败（`failed to read app.yaml`），无内置默认配置。
   所以"下载一个二进制就能起服务"今天并不成立。

**因此 jimu-web 不需要这些资产**：它需要的是"跑起来的后端"，由 Compose 提供，链路已完整。
若将来确实要"下载即用"，正确载体是**容器镜像**（能带上 conf/configs/openapi），不是裸二进制 ——
那是独立工作，不在本任务内。

**依赖 Task 7 Step 7(c)**：那里已删除两个孤儿脚本，并把 release 资产契约搬进
`test_makefile_contract.sh`。本任务只在同一份断言里补上"server 二进制不得再出现"。

- [ ] **Step 1: 在 `test_makefile_contract.sh` 加反向断言**

Task 7 Step 7(c) 已在该脚本末尾加了正向断言（`build-cli:`、`./cmd/jimu`、
`template-bundle.json` 等）。本步只加反向部分，插在正向块之后：

```bash
# server 二进制已退役：release 只发 CLI（Jimu 交付物是代码，不是二进制）
for needle in './cmd/server' 'matrix.profile' 'jimu-${PROFILE}'; do
  assert_not_contains "$(cat "$RELEASE_WORKFLOW")" "$needle" \
    "release workflow 仍含已退役的 $needle"
done
```

- [ ] **Step 2: 运行确认失败**

Run: `make check-makefile`
Expected: FAIL —— `release workflow 仍含已退役的 ./cmd/server`

这是真正的红：`check-makefile` 在 `make check` 与 `CI_CHECKS` 里，不需要额外接线。

- [ ] **Step 3: 删除 `build` job**

删除 `.github/workflows/release.yml` 里整个 `build:` job（`name: Build (${{ matrix.profile }} ...)`，
含 8 条 `matrix.include`、`profileoverlay` 调用、`Upload jimu-${PROFILE}-...` step）。

同时把 `publish` 的依赖从 `needs: [verify, build, build-cli]` 改为 `needs: [verify, build-cli]`。

`publish` 里 `sha256sum jimu-* template-bundle.json` 现在只会匹配 4 个 CLI，
语义仍然正确；为可读性改成显式列表：

```bash
sha256sum jimu-linux-amd64 jimu-linux-arm64 jimu-darwin-amd64 jimu-darwin-arm64 \
  template-bundle.json > "jimu-${GITHUB_REF_NAME}-checksums.txt"
```

`Create Release` 的 `files:` 段同理改为显式 5 行。

顺手删掉 `build` job 上方那段解释"为什么必须并行 8 个矩阵"的注释 —— 它随 job 一起失去意义。
文件头注释 `发布物：full 与 minimal 两个形态 × 4 平台的 server 二进制，加 4 个 Release CLI 二进制` 改为
`发布物：4 个平台的 Release CLI、template-bundle.json 和 checksums`。

**不要改 `smoke` job**：它下载的是 `jimu-$(go env GOOS)-$(go env GOARCH)`，即 CLI
（已核对第 264、277 行），与 server 二进制无关。

- [ ] **Step 4: 运行确认通过**

Run: `make check-makefile`
Expected: `✅ Makefile contract passed`

Run: `grep -c 'cmd/server' .github/workflows/release.yml || echo 0`
Expected: `0`

- [ ] **Step 5: 写进 release note**

`docs/releases/v0.10.0.md` 的「不兼容变更」段追加：

```markdown
- **不再发布 server 二进制**：`jimu-full-<os>-<arch>` 与 `jimu-minimal-<os>-<arch>`
  共 8 个资产自 v0.10.0 起移除。Jimu 的交付物是代码：用 `jimu new` 生成工程后自行
  `go build`。Release 现在只含 4 个平台的 `jimu` CLI、`template-bundle.json` 与 checksums。
```

- [ ] **Step 6: 提交**

```bash
git add .github/workflows/release.yml scripts/test_makefile_contract.sh docs/releases/v0.10.0.md
git commit -m "chore(release): stop publishing server binaries"
```

---

## 执行顺序与依赖

- Task 1 → Task 2：纯内核，可各自独立合并。
- Task 3 依赖 Task 1（`minimal` 产物要能编译）。
- Task 4 依赖 Task 3。
- Task 5 依赖 Task 1（删 `kernelRequiredDomains` 的前提）。
- Task 6 依赖 Task 3、4、5。
- Task 7 依赖 Task 6。
- Task 8 **必须在 Task 7 之后**：它依赖 Task 7 Step 7(c) 已把 release 契约搬进
  `test_makefile_contract.sh`，并已删除两个孤儿脚本。

## 已知风险

- 删除 v0.5 fixture 会失去「v0.5 → 当前」的跨版本升级证明。新 fixture 只证明当前模板的事务正确性（备份/回滚/冲突），
  与「旧模板硬失败」的决策一致，但要在 release note 里明说。
- `minimal` 的预算数字必须实测后回填，不能凭空写。
- 本机 `go` 可能被包装器接管，`-list` 输出会被吞；`check-gate-selectors` 在 CI 上必须用真实二进制，
  本地被包装时应当报错而不是静默通过。
- Task 8 删掉 server 二进制是**资产层面的破坏性变更**。仓库内没有文档承诺这些资产
  （已核对：只有历史 plan/spec 提到），但 release note 必须显式声明。
