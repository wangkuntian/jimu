> 本文是 P1.2（#35 合并）与 P1.3（#37）之间的阶段间隙计划，无正式子阶段编号。

# 累积技术债清理 实现计划（v0.3.0 阶段间隙）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 清掉 P0/P1-A/P1-B1 三轮评审累积下来的可发布级技术债、一处测试偶发失败、以及两处仓库级不一致；不做任何新功能。

**Architecture:** 五个互不依赖的小任务，各自独立可测：① 契约与注册表加固（P0 遗留）② 装配与 e2e 夹具对齐 ③ e2e 偶发失败修复 ④ 仓库级 CI/Makefile 一致性 ⑤ 文档收尾（含 P1-B1 终审裁定的待落地修订）。全部改动都应有测试或可验证命令兜住。

**Tech Stack:** Go 1.26 · testify · GitHub Actions · Makefile

**Spec / 来源：** 三轮评审的 deferred-minor 与 parked 记录（见各阶段 SDD 账本），以及 `docs/design/2026-09-18-capability-plugins-design.md` 的既有约束。

## Global Constraints

- **不改对外行为**：HTTP 路由/响应/配置键/schema 不变；`capabilities.enabled` 语义不变
- **不新增第三方依赖**（`require.Eventually` 来自已在用的 testify）
- 全绿：`gofmt -l .`（无输出）、`go build ./...`、`go vet ./...`、`golangci-lint run ./...`、`make check-log-usage`、`go test ./... -count=1`、`make release-check COMPOSE_ENV=.env.example`
- 提交信息全英文（Conventional Commits）；每任务一提交；提交需用户明确指令
- 分支：从 `release/v0.3.0`（**P1-B1 合并后**）切 `chore/capability-minors`，PR 目标 `release/v0.3.0`
- 简单优先：每条只做最小必要改动，不顺手重构

## 任务来源清单（三轮评审的 deferred 项）

| # | 来源 | 内容 |
|---|---|---|
| 1 | P0 终审 Minor | `contract.Describe(nil)` 会 panic（`m.Name()` 空指针） |
| 2 | P0 终审 Minor | `catalog.All()` 的"副本"注释只对结构体成立，`Requires` 仍与包级描述符共享底层数组 |
| 3 | P0 终审 Minor | `catalog.Resolve` 的闭包补齐 `for name := range on` 使多处 dangling 依赖时错误文案随机 |
| 4 | P0 终审 Minor | `main.go` 的 `all` 映射与 `catalog.Names()` 的耦合只在启动时报错，无用例钉住 |
| 5 | P0 终审 Minor | `internal/e2e/api_contract_test.go` 自建路由仍保留旧的 `... && len(protected) > 0` 形状，与生产 fail-closed 语义不一致 |
| 6 | P0/P1 评审 | `internal/e2e` 的 `TestRoleAssignmentAndRBAC` 依赖"登录耗时超过 50ms 策略缓存 TTL"→ 时间敏感偶发失败 |
| 7 | P1-A/P1-B1 | `internal/kernel/db/retention.go:29` 注释仍写"kernel 不应反向依赖 modules" |
| 8 | P1-B1 终审 | CI 的 `CHANGELOG check` 因 `actions/checkout` 默认 `fetch-depth: 1` 可能取不到基线而静默通过 |
| 9 | P1-B1 终审 | 本地 `golangci-lint`（v2.12.2）与 CI 固定版本（v2.7.2）不一致，`make lint` 结果不可比 |
| 10 | P1-B1 终审裁定（待落地） | 版本日志恢复字面 `internal/platform/`、`oauth provider` 改写为 `oauth（provider 包）`（能力清单按 `/` 计数须为 14）；计划过滤器链补目录级例外（`grep -v '^docs/plans/'`、`grep -v '^docs/releases/'`） |

---

### Task 1: 契约与注册表加固（清单 #1–#3）

**Files:**
- Modify: `internal/contract/capability.go`、`internal/capabilities/catalog/catalog.go`
- Test: `internal/contract/capability_test.go`、`internal/capabilities/catalog/catalog_test.go`

**Interfaces:**
- Consumes: 现有 `Describe(Module) Descriptor`、`All()`、`Resolve([]string)`
- Produces: 同签名；仅行为加固（nil 安全、确定性错误、真副本）

- [ ] **Step 1: 写失败测试**

在 `internal/contract/capability_test.go` 追加：

```go
func TestDescribeNilReturnsEmptyDescriptor(t *testing.T) {
	if got := Describe(nil); got.Name != "" || len(got.Requires) != 0 || got.Mount != "" {
		t.Fatalf("Describe(nil) = %+v, want zero Descriptor", got)
	}
}
```

在 `internal/capabilities/catalog/catalog_test.go` 追加：

```go
func TestAllReturnsDeepCopyOfRequires(t *testing.T) {
	withEntries(t, contract.Descriptor{Name: "a", Requires: []string{"b"}}, contract.Descriptor{Name: "b"})
	got := All()
	got[0].Requires[0] = "mutated"
	if All()[0].Requires[0] != "b" {
		t.Fatal("All() must not expose the registry's Requires backing array")
	}
}

func TestResolveReportsFirstDanglingDependencyInListOrder(t *testing.T) {
	// 两个坏依赖：错误必须稳定指向清单顺序里的第一个，而不是 map 遍历的随机一个
	withEntries(t,
		contract.Descriptor{Name: "a", Requires: []string{"ghost-a"}},
		contract.Descriptor{Name: "b", Requires: []string{"ghost-b"}},
	)
	for i := 0; i < 20; i++ {
		_, err := Resolve([]string{"a", "b"})
		if err == nil || !strings.Contains(err.Error(), `"ghost-a"`) {
			t.Fatalf("run %d: error = %v, want the first dangling dependency in list order", i, err)
		}
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/contract/ -run TestDescribeNil -v && go test ./internal/capabilities/catalog/ -run 'TestAllReturnsDeepCopy|TestResolveReportsFirst' -v`
Expected: 三条全部 FAIL（`Describe(nil)` panic；`All()` 共享底层数组；错误文案随机而非 `ghost-a`）

- [ ] **Step 3: 实现**

`internal/contract/capability.go`：

```go
// Describe 读取能力描述；未实现 Describable 或传入 nil 时回退为零值描述。
func Describe(m Module) Descriptor {
	if m == nil {
		return Descriptor{}
	}
	if d, ok := m.(Describable); ok {
		return d
	}
	return Descriptor{Name: m.Name()}
}
```

`internal/capabilities/catalog/catalog.go` —— `All()` 深拷贝 `Requires`：

```go
// All 返回清单中全部能力的深拷贝（含 Requires），调用方修改不影响清单。
func All() []contract.Descriptor {
	out := make([]contract.Descriptor, len(entries))
	for i, d := range entries {
		out[i] = d
		out[i].Requires = append([]string(nil), d.Requires...)
	}
	return out
}
```

`Resolve` 的闭包补齐改为按清单顺序遍历（错误确定 + 输出顺序不变）：

```go
	for changed := true; changed; {
		changed = false
		for _, d := range entries {
			if !on[d.Name] {
				continue
			}
			for _, dep := range d.Requires {
				if _, ok := byName[dep]; !ok {
					return nil, fmt.Errorf("capability %q requires unknown capability %q", d.Name, dep)
				}
				if !on[dep] {
					on[dep] = true
					changed = true
				}
			}
		}
	}
```

> 注意：其余返回路径（`Resolve` 的输出切片）也应深拷贝 `Requires`，或明确注释为只读；二选一并在报告中说明。

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/contract/ ./internal/capabilities/... -count=1 -v`
Expected: 全绿（含既有用例不回归）

- [ ] **Step 5: 提交**（用户授权后执行）

```bash
git add internal/contract internal/capabilities && git commit -m "fix(contract): harden describe and the capability registry"
```

---

### Task 2: 装配名册钉住 + e2e 夹具对齐（清单 #4–#5）

**Files:**
- Modify: `cmd/server/main.go`（导出可供测试断言的装配名册）
- Create: `cmd/server/main_test.go`
- Modify: `internal/e2e/api_contract_test.go`（自建路由的挂载判定）
- Modify: `internal/kernel/db/retention.go`（注释措辞）

**Interfaces:**
- Consumes: `catalog.Names()`
- Produces: `var wiredCapabilities []string`（`cmd/server` 包级），由 `run()` 用于自检、由测试用于与 catalog 对账

- [ ] **Step 1: 写失败测试**

创建 `cmd/server/main_test.go`：

```go
package main

import (
	"sort"
	"testing"

	"jimu/internal/capabilities/catalog"
)

func TestWiredCapabilitiesMatchCatalog(t *testing.T) {
	wired := append([]string(nil), wiredCapabilities...)
	known := catalog.Names()
	sort.Strings(wired)
	sort.Strings(known)
	if len(wired) != len(known) {
		t.Fatalf("wired %d capabilities (%v), catalog declares %d (%v)", len(wired), wired, len(known), known)
	}
	for i := range wired {
		if wired[i] != known[i] {
			t.Fatalf("wired[%d] = %q, catalog[%d] = %q", i, wired[i], i, known[i])
		}
	}
}

func TestWiredCapabilitiesHaveNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, name := range wiredCapabilities {
		if seen[name] {
			t.Fatalf("duplicate wired capability %q", name)
		}
		seen[name] = true
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./cmd/server/ -v`
Expected: FAIL —— `undefined: wiredCapabilities`

- [ ] **Step 3: 在 `main.go` 声明名册并自检**

```go
// wiredCapabilities 是 main 装配的能力名册；必须与 catalog.Names() 一致。
// 单元测试（main_test.go）对账两者，run() 在启动时按它自检装配映射。
var wiredCapabilities = []string{
	"user", "role", "permission", "tenant", "auth", "audit", "admin", "oauth",
}
```

在 `run()` 里构造 `all` 映射之后、过滤之前插入：

```go
	if len(all) != len(wiredCapabilities) {
		_ = container.Stop(context.Background())
		return fmt.Errorf("%w: %d instances for %d declared names", errCapabilityWiringMismatch, len(all), len(wiredCapabilities))
	}
	for _, name := range wiredCapabilities {
		if _, ok := all[name]; !ok {
			_ = container.Stop(context.Background())
			return fmt.Errorf("%w: %q", errCapabilityNoInstance, name)
		}
	}
```

> **修订说明（执行记录）**：实际落地改用两个包装哨兵错误 `errCapabilityWiringMismatch` / `errCapabilityNoInstance`（`cmd/server/main.go` 包级声明），错误文本不变。

- [ ] **Step 4: 对齐 e2e 夹具的挂载判定**

`internal/e2e/api_contract_test.go` 的自建路由仍是旧形状，改为与生产 `registerHTTP` 一致（含 fail-closed 语义：无提供者却要挂受保护能力时应让测试失败而不是裸挂）：

```go
	// 3) 路由注册（按能力声明的挂载点；受保护能力必须存在中间件提供者）
	for _, m := range modules {
		desc := contract.Describe(m)
		if desc.Normalized() != contract.MountProtected {
			m.RegisterHTTP(router)
			continue
		}
		require.NotEmpty(t, protected, "capability %q declares MountProtected but no protected middleware provider is present", desc.Name)
		m.RegisterHTTP(router.Group("", protected...))
	}
```

- [ ] **Step 5: 更新 `retention.go` 注释措辞**

`internal/kernel/db/retention.go:29`：

```go
// 刻意不依赖业务能力的领域模型（kernel 不应反向依赖 capabilities）。
```

- [ ] **Step 6: 运行确认通过**

Run: `go vet ./... && go test ./cmd/server/ ./internal/e2e/ ./internal/kernel/db/ -count=1`
Expected: 全绿；`internal/e2e` 的契约用例仍通过（其模块列表恒含 auth，故不会触发新的 fail-closed 断言）

- [ ] **Step 7: 提交**（用户授权后执行）

```bash
git add cmd/server internal/e2e internal/kernel/db && git commit -m "chore(server): pin the wiring roster and align the e2e mount guard"
```

---

### Task 3: 修掉 e2e 的时间敏感偶发失败（清单 #6）

**Files:**
- Modify: `internal/e2e/api_contract_test.go`（`TestRoleAssignmentAndRBAC`）

**Interfaces:**
- Consumes: `platformauth.PolicyCacheTTL`
- Produces: 无新接口；把"依赖登录耗时超过 TTL"改成"有界轮询直到缓存过期"

**根因**：该用例把 `PolicyCacheTTL` 设为 50ms，然后**立即**断言权限已生效 —— 实际上只有当 bcrypt 登录耗时 >50ms 时缓存才过期；登录快于 50ms 时仍读到旧策略 → 403 → 偶发失败（评审实测首跑失败、隔离 10/10 通过）。

- [ ] **Step 1: 写"复现→修复"的替换**

把步骤 5 的即时断言改为有界轮询（缓存过期是异步行为，轮询是对它的正确表达）：

```go
	// 5) 该用户登录后能 GET /users（原 403 → 200）。
	// 策略缓存按 TTL 异步过期，故用有界轮询等待生效，而不是依赖"登录耗时 > TTL"。
	userToken := login(t, r, "rbacuser", "rbacpass123")
	require.Eventually(t, func() bool {
		w := doJSON(t, r, http.MethodGet, "/api/v1/users", userToken, "")
		return w.Code == http.StatusOK
	}, 3*time.Second, 25*time.Millisecond, "permission should take effect after the policy cache TTL")
```

- [ ] **Step 2: 验证稳定性**

**配方修订（执行时发现）**：初版写的 `-count=10` 判据**不可达**，且原因与本次 flake 无关 —— e2e 套件用 `file:e2e_contract?mode=memory&cache=shared` 且从不关闭，同一进程内连续 `-count` 次迭代会复用数据库状态，第 2 次即在"创建 rbacuser"处 409（修复前后同样如此，属既有的测试隔离缺陷）。因此稳定性判据改为**隔离进程 × N**（这也正是 CI 的真实形态：每个包一个进程、`-count=1`）：

```bash
pass=0; fail=0
for i in $(seq 1 30); do
  if go test ./internal/e2e/ -run TestRoleAssignmentAndRBAC -count=1 >/dev/null 2>&1; then pass=$((pass+1)); else fail=$((fail+1)); fi
done
echo "isolated_pass=$pass isolated_fail=$fail"
```

Expected: `isolated_fail=0`（修复前用同一命令应能观察到偶发失败：实测 30 次里 1 次在权限断言处 200→403）。

> 已登记为独立的 deferred 项：e2e 的内存 SQLite 未按测试关闭，使 `-count>1` 不可用；修复它需要改 `newTestAppWithDB` 与文件内所有用例，不属本任务范围。

- [ ] **Step 3: 提交**（用户授权后执行）

```bash
git add internal/e2e && git commit -m "test(e2e): wait for policy cache expiry instead of racing the clock"
```

---

### Task 4: 仓库级一致性（清单 #8–#9）

**Files:**
- Modify: `.github/workflows/ci.yml`、`Makefile`

**Interfaces:**
- Consumes: 无
- Produces: `make lint` 使用与 CI 相同的 golangci-lint 版本（`LINT_VERSION`），CHANGELOG 门禁拿到真实基线

- [ ] **Step 1: 给 CHANGELOG 门禁补足历史**

`.github/workflows/ci.yml` 的 checkout（lint job，即执行 "📝 CHANGELOG check" 的那个 job）加：

```yaml
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0
```

理由：该步骤用 `git diff origin/${{ github.base_ref }}...HEAD` 判定，默认浅克隆可能取不到基线 → `grep` 拿到空输入 → 门禁静默通过。

- [ ] **Step 2: 让 `make lint` 与 CI 版本一致**

`Makefile` 增加版本变量并改造 `lint`：

```make
LINT_VERSION ?= v2.7.2   # 与 .github/workflows/ci.yml 的 GOLANGCI_LINT_VERSION 保持一致

lint:
	@if command -v golangci-lint >/dev/null 2>&1 && golangci-lint version 2>/dev/null | grep -q "$(LINT_VERSION:v%=%)"; then \
		golangci-lint run ./...; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		echo "本地 golangci-lint 版本与 CI（$(LINT_VERSION)）不一致，改用 go run 固定版本"; \
		go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(LINT_VERSION) run ./...; \
	else \
		echo "golangci-lint 未安装且无法 go run，使用 go vet 替代"; \
		go vet ./...; \
	fi
```

> **配方修订（执行时发现）**：初版写的是 `grep -q "$(LINT_VERSION)"`（即匹配 `v2.7.2`），但 `golangci-lint version` 输出的版本号**不带 `v` 前缀**（形如 `2.12.2`），所以该判断永不成立 —— 本地二进制分支会变成死代码，并且每次都会打印"版本不一致"的假提示。正确写法是 `"$(LINT_VERSION:v%=%)"`（去掉前导 `v`）。两条分支都需实测（本地一致时走本地二进制、不一致时走 `go run`）。

- [ ] **Step 3: 验证**

```bash
make lint 2>&1 | tail -3
grep -n 'fetch-depth' .github/workflows/ci.yml
bash -n scripts/bench_ci.sh && echo "workflow yaml ok" && python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml')); print('ci.yml parses')"
```

Expected: `make lint` 输出 0 issues（走固定版本或本地一致版本）；两个 workflow 文件仍能被 YAML 解析

- [ ] **Step 4: 提交**（用户授权后执行）

```bash
git add .github/workflows/ci.yml Makefile && git commit -m "chore(ci): fetch full history and pin the local lint version"
```

---

### Task 5: 文档收尾（清单 #7、#10）

**Files:**
- Modify: `docs/releases/v0.3.0.md`（恢复字面路径 + `oauth provider` 改写为 `oauth（provider 包）`）
- Modify: `docs/plans/2026-09-18-platform-relocation.md`（过滤器链补例外 + 计数）
- Modify: `docs/design/2026-09-18-capability-plugins-design.md`（§3.1 标题下方加一条全局注记：文中 `platform/x` 指其 P1-B1 **搬迁前**的位置）

**Interfaces:**
- Consumes: P1-B1 的映射表
- Produces: 发布说明写上可检索的规范路径（发布后即冻结，这是唯一窗口）

- [ ] **Step 1: 恢复版本日志的字面路径**

`docs/releases/v0.3.0.md` 的「平台包归位」条目：把"`internal/` 下平台层"恢复为字面 `internal/platform/`，并把 `oauth provider` 改写为 `oauth（provider 包）`（能力清单以 `/` 分隔，若写成 `oauth/provider` 读者会数出 15 个名字，与条目声明的 14 个不符）。

- [ ] **Step 2: 计划过滤器链改为目录级例外**

`docs/plans/2026-09-18-platform-relocation.md` 的 Step 3a 过滤器链：把逐个计划文件的单文件例外（P0 计划 / namespace-move 计划 / 本计划）换成一条目录级例外，`docs/releases/` 目录例外与设计文档单文件例外保留，最终为三条过滤器：

```bash
  | grep -v '^docs/plans/' \
  | grep -v '^docs/releases/' \
  | grep -v '^docs/design/2026-09-18-capability-plugins-design.md'
```

并在该步骤的 3a 注释与「不留残余」约束里把例外写成「一处单文件例外（设计文档）+ 两个目录级例外（`docs/plans/`、`docs/releases/`）」与对应说明（逐个计划文件枚举例外已连续三次被新计划打破，故计划整体按目录排除；设计文档保留单文件粒度，使未来真正需要更新的设计文档仍会被扫出）。

- [ ] **Step 3: 设计文档加全局注记**

在 `docs/design/2026-09-18-capability-plugins-design.md` §3.1 标题下方加：

```markdown
> 路径注记：本文写于 P1-B1 之前，正文中的 `platform/x` 指该包**搬迁前**的位置；搬迁后**原平台层**的内核机制在 `internal/kernel/x`、能力实现在 `internal/capabilities/…`。§3.5.3/§3.6 的表格另有就地说明。
```

- [ ] **Step 4: 验证**

```bash
# 4a) 加宽模式零残留（三个例外：`docs/plans/`、`docs/releases/`、设计文档）
grep -rnE '(^|[^A-Za-z0-9_-])(internal/)?platform\b' README.md AGENTS.md Makefile docs/ specs/ configs/ scripts/ .github/ internal/ tools/ 2>/dev/null \
  | grep -v '^docs/plans/' \
  | grep -v '^docs/releases/' \
  | grep -v '^docs/design/2026-09-18-capability-plugins-design.md'

# 4b) 版本日志确实写了字面规范路径
grep -n 'internal/platform/' docs/releases/v0.3.0.md

# 4c) 能力清单按 `/` 分隔计数必须等于条目声明的 14（`oauth（provider 包）` 不额外贡献名字）
sed -n '19p' docs/releases/v0.3.0.md | sed 's/.*14 个能力实现：//; s/）；.*//' | awk -F'/' '{print NF}'

# 4d) 旧写法不得再出现在版本日志里（`oauth/provider` 会被读成第 15 个名字）
grep -nE 'oauth[/ ]provider' docs/releases/v0.3.0.md
```

Expected: 4a 无输出；4b 命中版本日志「平台包归位」条目（`docs/releases/v0.3.0.md:19`）；4c 输出 `14`；4d 无输出

- [ ] **Step 5: 提交**（用户授权后执行）

```bash
git add docs/ && git commit -m "docs(releases): keep the canonical platform path in the release body"
```

---

## Self-Review

**1. 覆盖检查**：上方「任务来源清单」10 项 → Task 1（#1–#3）、Task 2（#4–#5、#7）、Task 3（#6）、Task 4（#8–#9）、Task 5（#10 与设计文档注记）。**无遗漏项**。

**2. 占位符扫描**：无 TBD/TODO；每步含可执行代码/命令与期望输出。Task 4 Step 1 与 Task 5 Step 2 的编辑点是**位置描述 + 目标内容**（YAML 片段、过滤器行），执行者需按现文件定位——已在步骤里写明判据（`grep -n 'fetch-depth'`、Step 4 的验证命令）。

**3. 类型一致性**：`wiredCapabilities`（Task 2）只在 `cmd/server` 包内使用与断言；`Describe(nil)` 的零值语义与 `Descriptor` 零值一致（`Normalized()` 会把它归一为 `MountProtected`，但 nil 不会出现在 catalog 解析路径上）；`All()` 深拷贝后 `TestDescriptorsAreWellFormed` 的 `reflect.DeepEqual(All(), fixture())` 仍成立（深拷贝不改值）。

**4. 风险**：
- Task 2 的 e2e 断言 `require.NotEmpty(t, protected, ...)`：该文件的自建路由恒包含 auth 模块，故不会触发；若将来移除了 auth，该断言会**正确地**让测试失败。
- Task 3 的 `require.Eventually` 把"缓存过期"显式化为有界等待：3s 上限远大于 50ms TTL，且不再与 bcrypt 耗时耦合；若策略永不生效，用例仍会失败（不会静默通过）。
- Task 4 的 `go run ...@v2.7.2` 首次执行需联网下载模块；若 CI 或本地无网络且本地二进制版本不一致，`make lint` 会**硬失败**（这是刻意的：比起吐出不可比的 lint 结论，宁可响亮失败）；只有 `golangci-lint` 完全缺失时才会退化为 `go vet`。
- Task 5 的过滤器改动会改变 P1-B1 计划里"Expected: 无输出"的达成方式（从单文件例外改为目录例外），需同步更新该计划的说明文字，否则计划自相矛盾。
