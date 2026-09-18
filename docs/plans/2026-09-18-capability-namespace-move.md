# 能力可插拔 P1-A：命名空间搬迁 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把业务模块目录从 `internal/modules/` 搬到 `internal/capabilities/`，并同步全仓（Go 代码、脚手架、脚本、文档）的路径引用；**零逻辑变化**。

**Architecture:** 纯目录搬迁 + 路径字符串重写。目录移动用 `git mv` 保留历史；路径重写覆盖三种形态：`jimu/internal/modules/…`（import 路径）、`internal/modules/…`（文件头注释与模板）、`"internal", "modules"`（`filepath.Join` 拆分写法，sed 抓不到）。`internal/platform/*` **不在本计划内**：它的去向与"归属判定"绑定（哪些包属内核、哪些属能力、`platform/auth` 还要拆成三份），留给 P1-B 一次搬到位，避免同一批包搬两次。

**Tech Stack:** Go 1.26 · git mv · perl -pi（跨平台原地替换）· Makefile/release-check

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md`（§3.1 内核 / §3.2–3.4 能力命名空间 / §6.1 能力自描述 / §10 P1）

## Global Constraints

- **零逻辑变化**：只改文件位置与路径字符串；不改任何标识符、函数签名、包名、行为。搬迁后包名保持不变（`auth` 目录的包仍是 `authmodule` 等），因此**不需要改任何 `package` 子句**
- **不留残余**：Go / 脚本 / 配置 / Makefile 中的 `internal/modules` 引用必须归零（`docs/plans/2026-09-18-capability-plugins-p0.md` 是 P0 的历史执行记录，保留原样；设计文档 §3.6 表格单元与 §5 的"现状路径"表同理。注意：设计文档中**没有**任何含 `internal/modules` 的目录树块，§3.8 实为「非代码资产模块化」，因此本次**不改写前瞻布局**，只加历史注记）
- **全绿**：`gofmt -l .`（无输出）、`go build ./...`、`go vet ./...`、`go test ./...`、`make check-log-usage`、`make release-check COMPOSE_ENV=.env.example`
- **提交信息全英文**（Conventional Commits，`githooks/commit-msg` 拒绝 CJK）
- 分支：从 `release/v0.3.0` 切 `feature/capability-namespace`，PR 目标 `release/v0.3.0`，squash 合并
- 提交需用户明确指令（AGENTS.md 最高优先级规则）
- 简单优先：不做顺手重构；搬迁后功能、路由、配置键、schema 全部不变

## 文件结构

| 动作 | 对象 | 说明 |
|---|---|---|
| 移动（8 个目录） | `internal/modules/{user,role,permission,tenant,auth,audit,admin,oauth}` → `internal/capabilities/<同名>` | 包名不变；`internal/capabilities/catalog` 已存在 |
| 删除空目录 | `internal/modules/` | 搬完后必须为空 |
| 路径重写（Go） | 132 个引用文件（含 56 测试）+ 模块内 41 处非 import 引用 | 三种字符串形态 |
| 脚手架同步 | `tools/generator/module.go`、`templates.go`、`module_test.go`、`compile_test.go` | 新模块必须生成到 `internal/capabilities/` |
| 文档/脚本同步 | `README.md`（×2）、`docs/CONTRIBUTING.md`、`Makefile`、`scripts/bench_ci.sh`、`specs/001-jimu-framework-spec/{contracts/module.md,research.md}`、设计文档 §3.6 表格单元 + §5 标题下历史注记 | Task 2 |

## 现状基线（本计划编写时实测）

- 引用 `jimu/internal/modules/` 的 Go 文件：**132**（其中测试 56）
- `internal/modules/` 下 Go 文件：**178**
- 非 import 形式的引用：**41 处**，其中 `filepath.Join("internal", "modules", …)` 拆分写法 **23 处**（`tools/generator/module.go` 12 处 + `module_test.go` 11 处），另有引用号包裹的无尾斜杠形态 `"internal/modules"` **2 处**（`module_test.go`）
- 非 Go 引用：`README.md:953,974`、`docs/CONTRIBUTING.md:156`、`Makefile:288`、`scripts/bench_ci.sh:13`、`specs/…/{contracts/module.md:27,research.md:18}`

---

### Task 1: 目录搬迁与 Go 代码路径重写

**Files:**
- Move: `internal/modules/{user,role,permission,tenant,auth,audit,admin,oauth}` → `internal/capabilities/`
- Modify: 全部引用路径的 Go 文件（自动重写）+ `tools/generator/*.go`（含手工修正 `filepath.Join` 拆分写法）

**Interfaces:**
- Consumes: P0 已建立的 `internal/capabilities/catalog`（它 import 八个能力包）
- Produces: 新命名空间 `internal/capabilities/<name>`；包名与导出符号**完全不变**，故下游按路径引用即可

- [ ] **Step 1: 记录基线并搬迁目录**

```bash
git status --short          # 应干净；不干净则先与用户确认
git rev-parse HEAD          # 记为 BASE，写进报告
mkdir -p internal/capabilities
for m in user role permission tenant auth audit admin oauth; do
  git mv "internal/modules/$m" "internal/capabilities/$m"
done
rmdir internal/modules
ls internal/capabilities/
```

Expected: `internal/capabilities/` 下出现 `catalog` + 八个能力目录；`internal/modules/` 已不存在。

- [ ] **Step 2: 重写四种路径字符串形态**

```bash
grep -rl 'jimu/internal/modules/\|internal/modules/\|"internal", "modules"\|"internal/modules"' --include='*.go' . | while read -r f; do
  perl -pi -e 's|jimu/internal/modules/|jimu/internal/capabilities/|g; s|internal/modules/|internal/capabilities/|g; s|"internal", "modules"|"internal", "capabilities"|g; s|"internal/modules"|"internal/capabilities"|g' "$f"
done
```

> 顺序要紧：先替换带 `jimu/` 前缀的 import 路径，再替换其余（文件头注释、模板、脚手架测试里的裸路径），最后处理两种 `filepath.Join` 相关的形态。
>
> **配方修订（执行时发现）**：初版只列了三种形态，漏了**引用号包裹的无尾斜杠形态** `"internal/modules"`（`tools/generator/module_test.go` 的 `newTestRepository`/`assertNoGeneratedFiles` 用到）。Step 3 的全仓 grep 能兜住这类遗漏，但配方里应直接补齐 —— 上面已是四条。

- [ ] **Step 3: 确认 Go 侧零残余**

```bash
grep -rn 'internal/modules\|"internal", "modules"' --include='*.go' . ; echo "exit=$?"
```

Expected: 无输出（`grep` 退出码 1）。若有输出，逐条确认是路径引用还是无关字面量；路径引用必须一并改掉。

- [ ] **Step 4: 验证脚手架生成到新命名空间**

```bash
grep -n 'capabilities' tools/generator/module.go | head -5
go test ./tools/generator/... -count=1 -v 2>&1 | tail -20
```

Expected: `module.go` 的落盘目录与提示文案均已指向 `internal/capabilities/`；generator 两个测试用例通过（`module_test.go` 断言生成路径，`compile_test.go` 会生成一个 product 模块并 `go test ./internal/capabilities/product/...`）。

- [ ] **Step 5: 机械性验证（关键判据 —— 重建式）**

搬迁后**除路径字符串外不应有任何改动**。逐行过滤无法证明这一点，因此改用**重建验证**：把 BASE 的每个改动文件按替换式重写后与 HEAD 逐字节比对。

> 配方修订记录（执行时踩到两处，均已修正）：
> 1. 替换式必须**四条**：`jimu/internal/modules/`、`internal/modules/`、`"internal", "modules"`（`filepath.Join` 拆分写法）、以及 `"internal/modules"`（无尾斜杠形态，`tools/generator/module_test.go` 用到）。
> 2. 重命名文件在 `git diff --name-only` 里**只有新路径**，必须用 `--name-status` 拿到 `R<score> 旧路径 新路径` 配对，旧路径取 BASE、新路径取 HEAD，逐一比对。
> 3. `.go` 文件需先过 `gofmt`：路径变长会让 import 分组顺序变化（实测 24 个文件被 gofmt 重排），属格式化的确定性副作用。

```bash
git diff -M --name-status a09064b..HEAD > /tmp/p1a_ns.txt
renames=0; differs=0
while read -r st src dst; do
  case "$st" in
    R*) s="$src"; d="$dst"; renames=$((renames+1)) ;;
    M*) s="$src"; d="$src" ;;
    *)  continue ;;
  esac
  git show "a09064b:$s" \
    | perl -pe 's|jimu/internal/modules/|jimu/internal/capabilities/|g; s|internal/modules/|internal/capabilities/|g; s|"internal", "modules"|"internal", "capabilities"|g; s|"internal/modules"|"internal/capabilities"|g' \
    > /tmp/p1a_expected
  [ "${d##*.}" = "go" ] && gofmt /tmp/p1a_expected > /tmp/p1a_expected.fmt && mv /tmp/p1a_expected.fmt /tmp/p1a_expected
  git show "HEAD:$d" > /tmp/p1a_actual
  cmp -s /tmp/p1a_expected /tmp/p1a_actual || { echo "DIFFERS: $s -> $d"; differs=$((differs+1)); }
done < /tmp/p1a_ns.txt
echo "renames=$renames differs=$differs"
```

Expected: `differs=0`（实测 `renames=178 differs=0`，即 194 个改动文件全部由路径替换复原）。

- [ ] **Step 5b: 确认改动面只有预期文件**

```bash
git diff -M --stat a09064b..HEAD | tail -3
git diff -M --diff-filter=M --name-only a09064b..HEAD | wc -l     # 非纯重命名的文件数
git diff -M --name-only a09064b..HEAD | grep -v '^internal/' | grep -v '^tools/' ; echo "exit=$?"
```

Expected: 最后一条无输出（改动只落在 `internal/` 与 `tools/` 之下）。

- [ ] **Step 6: 编译、格式与全量测试**

```bash
gofmt -l .
go build ./...
go vet ./...
go test ./... -count=1 2>&1 | tail -5
```

Expected: `gofmt` 无输出；build/vet 通过；测试全绿（与搬迁前的包数与结果一致，无新增失败）。

- [ ] **Step 7: 提交（用户授权后执行）**

```bash
git add -A internal/capabilities internal/modules tools/generator $(git diff --name-only | tr '\n' ' ')
git commit -m "refactor(capabilities): move business modules into the capabilities namespace"
```

> 若 `git add` 的路径展开不便，用 `git add -A` 后以 `git status --short` 复核：只应包含能力目录的重命名与路径重写文件，不得包含无关改动。

---

### Task 2: 文档与脚本同步 + 全量回归

**Files:**
- Modify: `README.md:953,974`、`docs/CONTRIBUTING.md:156`、`Makefile:288`、`scripts/bench_ci.sh:13`、`specs/001-jimu-framework-spec/contracts/module.md:27`、`specs/001-jimu-framework-spec/research.md:18`
- Modify: `docs/design/2026-09-18-capability-plugins-design.md`（§3.6 表格单元 + §5 前置历史注记；**不含**前瞻目录形态改写——设计文档中本就没有含 `internal/modules` 的目录树块，§3.8 是「非代码资产模块化」）

**Interfaces:**
- Consumes: Task 1 的新路径
- Produces: 全仓引用一致；发布门禁在最终提交上通过

- [ ] **Step 1: 逐个替换非 Go 引用**

| 文件:行 | 原文 | 改为 |
|---|---|---|
| `README.md:953` | `internal/modules/user/domain/user.go` | `internal/capabilities/user/domain/user.go` |
| `README.md:974` | `internal/modules/{name}/` | `internal/capabilities/{name}/` |
| `docs/CONTRIBUTING.md:156` | `go test ./internal/modules/user/... -run Integration -v` | `go test ./internal/capabilities/user/... -run Integration -v` |
| `Makefile:288` | `./internal/modules/auth/application/...` | `./internal/capabilities/auth/application/...` |
| `scripts/bench_ci.sh:13` | `./internal/modules/auth/application/...` | `./internal/capabilities/auth/application/...` |
| `specs/001-jimu-framework-spec/contracts/module.md:27` | `internal/modules/{name}/` | `internal/capabilities/{name}/` |
| `specs/001-jimu-framework-spec/research.md:18` | `internal/modules/auth/interfaces/router.go` | `internal/capabilities/auth/interfaces/router.go` |

- [ ] **Step 2: 更新设计文档的历史说明**

> **配方修订（执行时发现）**：初版说「§3.8『最终目录形态』写的是 `internal/modules/`，需要更新」——**这个前提是错的**。实测设计文档中 `internal/modules` 出现 **0 次**：当前 §3.8 是「非代码资产模块化」，文档里也**没有**任何含 `internal/modules` 的目录树块（那份布局块属于已废弃的早期草稿）。因此本节只做「加历史注记」这一件事。

在 `docs/design/2026-09-18-capability-plugins-design.md` 的 §3.6 表格单元（`platform/grpc/userinfo_service.go` 行）与 §5「重点能力拆分」标题下各加一处历史注记（§5 的表格使用裸文件名，加注是为了让读者知道这些路径已随 P1-A 迁移）。§5 处实际落盘的措辞为：

```markdown
> 注：本节各表中的路径是**改造前的现状路径**（搬迁前位于 `internal/modules/` 下，表内多为相对写法）。P1-A 已把业务模块搬入 `internal/capabilities/…`，后续 P1 子计划继续拆分；表格保留原路径以便与当时的评审记录对照。
```

- [ ] **Step 3: 全仓零残余检查（加宽模式 + 引用路径可解析）**

命令必须用**加宽模式**：字面量 `internal/modules` 抓不到裸树节点（`│   └── modules/`）与缩写形态（`modules/audit/application/worker.go`），而这两类正是本阶段实际漏掉、后又补上的（见下方修订说明）。

```bash
grep -rnE '(^|[^A-Za-z0-9_./-])(internal/)?modules/' \
  --include='*.go' --include='*.md' --include='*.sh' --include='*.yml' --include='*.yaml' --include='*.json' \
  README.md Makefile docs/ specs/ configs/ scripts/ .github/ internal/ tools/ 2>/dev/null \
  | grep -v '^docs/plans/2026-09-18-capability-plugins-p0.md' \
  | grep -v '^docs/plans/2026-09-18-capability-namespace-move.md' \
  | grep -v '^docs/design/2026-09-18-capability-plugins-design.md' \
  | grep -v '^docs/releases/'
```

Expected: 无输出。**四处例外都是刻意的**：

1. `docs/plans/2026-09-18-capability-plugins-p0.md` —— P0 的历史执行记录（当时的路径就是 `internal/modules/…`）；
2. `docs/plans/2026-09-18-capability-namespace-move.md` —— 本计划自身，它记录的正是这次搬迁，天然大量出现旧路径；
3. `docs/design/2026-09-18-capability-plugins-design.md` —— §3.6 表格单元与 §5 标题下已加历史注记，明确标注为改造前路径；
4. `docs/releases/` —— 发布记录属冻结文本：`docs/releases/v0.2.0.md` 是已发布版本的记录，**不得回改**；`docs/releases/v0.3.0.md` 的本版「变更」条目记载的正是这次搬迁。

> **更强的判据（本次必须补做）**：grep 只能证明"没有旧字符串"，**不能证明文档引用的路径真实存在**。两种 grep 形态都放过"前缀写对、路径本身不存在"的引用，也无法保证缩写形态补全后正确。因此本步除 grep 外，还要做一次**引用路径可解析检查**：把文档中出现的 `internal/…` 代码路径逐个 `test -e` 确认落盘存在；对缩写形态（如 `modules/audit/application/worker.go`）先补全成 `internal/capabilities/audit/application/worker.go` 再检查。**每个文档引用的路径都必须在磁盘上存在**，这是比 grep 更强的要求。

> **修订说明（执行记录）**：初版只 grep 字面量 `internal/modules`，结构性抓不到裸节点与缩写形态，实际漏了两处 —— ① `README.md` 目录树里的裸子节点 `│   └── modules/`（同一轮还暴露了删节点后 `shared/` 仍用 `├──` 的连接符问题）；② `specs/001-jimu-framework-spec/research.md:27` 与 `:41` 的缩写引用 `modules/audit/application/worker.go`、`modules/admin/module.go`。两处均在修复轮 `b9aa8fd` 补上；`research.md` 的这两行现已补全为 `internal/capabilities/…` 全路径。
>
> 初版还只列了两处例外，漏了**本计划文档自身**（它必然包含大量旧路径）与**发布记录**；字面上的"无输出"因此不可能达成。

- [ ] **Step 4: 全量回归（发布门禁）**

```bash
gofmt -l .
go build ./...
go vet ./...
golangci-lint run ./...
make check-log-usage
make release-check COMPOSE_ENV=.env.example
```

Expected: `gofmt` 无输出；lint 0 issues；logcheck 仅剩既有 2 条 R3 提示且退出码 0；`make release-check` 输出 `All checks passed`（含 fmt-check / vet / check-log-usage / `go test ./...` / govulncheck / compose-check 全新栈从 001 迁移到 015）。

- [ ] **Step 5: 提交（用户授权后执行）**

```bash
git add README.md docs/CONTRIBUTING.md Makefile scripts/bench_ci.sh specs/ docs/design/2026-09-18-capability-plugins-design.md
git commit -m "docs(capabilities): update path references for the capabilities namespace"
```

---

## Self-Review

**1. Spec 覆盖**

| spec 要求 | 覆盖 |
|---|---|
| §10 **P0**「内核归位」行中的「`internal/capabilities/` 下按现有 8 模块原样落位」 | Task 1（业务模块）。注意：该行属设计文档的 **P0** 阶段，「P1-A」的阶段拆分只存在于本计划，设计文档中并无此概念 |
| §3.8 最终目录形态与 `internal/kernel/` 命名 | 设计文档 §3.8 实为「非代码资产模块化」，全文**没有**含 `internal/modules` 的目录树块，故本计划**不改写前瞻布局**；只在 §3.6 表格单元与 §5 标题下加历史注记。`platform/*` → `kernel/` 的**实际搬迁**属 P1-B |
| §6.1 能力清单唯一维护在 `internal/capabilities/catalog` | 已由 P0 满足；Task 1 保持其路径不变 |
| 脚手架产出新形态骨架 | Task 1 Step 4 只保证**路径**正确；骨架内容（`capability.go`、postgres 迁移等）属 P3 |

**2. 占位符扫描**：无 TBD/TODO；每步含可执行命令与期望输出；Task 2 的文件-行-新旧文本逐条列出。

**3. 类型一致性**：本计划不引入新标识符；`catalog`、`contract.Descriptor`、各能力 `Descriptor` 变量均由 P0 定义且不改名。

**4. 已知取舍**：`internal/platform/*` 留待 P1-B 一次搬到位（含 `platform/auth` 拆分），因此本计划结束后仓库仍有 `internal/platform/` 与 `internal/kernel/` 的命名缺口 —— 已在 Task 2 的文档更新中显式标注为"待 P1-B"，不让文档承诺尚未发生的事。

**5. 风险**：改动的绝对行数很大（132 文件的 import + 41 处非 import 引用），但全部是字符串替换；Step 5 的「非路径行零改动」判据是防"顺手改坏"的关键闸门，评审应以此为准而非逐行读 diff。另外 **P1-B 将搬迁 `internal/platform/**`，其缩写形态 `platform/…` 遍布 docs/specs，因此 P1-B 的残余门禁必须在开工前就按上面 Step 3 的加宽模式写死**（沿用 `internal/modules` 那种字面量 grep 会重演本次的漏检）。
