# 能力可插拔 P1.2（P1-B1）：平台包归位 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `internal/platform/` 下 29 个包按归属整体迁到 `internal/kernel/`（内核机制，15 个）与 `internal/capabilities/`（能力实现，14 个），同步全仓引用；**除了一个包名（`oauth`→`provider`）外零语义变化**。

**Architecture:** 整目录搬迁 + 29 条固定映射的 import 重写。与 P1-A 的关键差别：这次不是"一个命名空间换个名字"，而是**每个包按归属分流到两个命名空间**，所以替换规则是一张 29 行的映射表（下面给出唯一副本，实现与验证共用），不能靠一条通配替换完成。搬迁结束后 `internal/platform/` 目录消失。

**Tech Stack:** Go 1.26 · git mv · perl -pi · 重建式验证（见 Task 1 Step 5）

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md`（§3.1 内核清单、§3.2–3.4 能力清单、§3.5.3 包归位表、§3.6 混装包、§10 P1）

## Global Constraints

- **零语义变化**：只改文件位置、import 路径、注释/文档中的路径文本；不改任何导出符号、函数签名、行为。**唯一允许的包名变更**是 `platform/oauth` → `capabilities/oauth/provider`（包名 `oauth` → `provider`，为避让已存在的 `capabilities/oauth` 模块目录）。其余 28 个包**包名不变**
- **能力名（`Descriptor.Name`）本阶段一律不改**：`tenant`/`notification`/`feature` 的最终能力名（`tenancy`/`notify`/`featureflag`）属 P1-E；本阶段只决定**代码放在哪个目录**，不碰 catalog 与配置值
- **不留残余**：Go / 脚本 / 工作流 / Makefile / README / AGENTS.md / specs / configs 中不得再有 `internal/platform` 引用。例外类只有一种——**主题就是搬迁本身的文档**：一处单文件例外（设计文档 `docs/design/2026-09-18-capability-plugins-design.md`，其 §3.5.3/§3.6 归位表属改造前现状、需加历史注记）+ 两个目录级例外（`docs/plans/`、`docs/releases/`）。**已发布的版本日志是冻结记录，永不重写**。逐个计划文件枚举例外已连续三次被下一份计划打破（P0 计划 → +namespace-move 计划 → +relocation 计划 → +本阶段计划），每加一份阶段计划都会再造一次，故计划整体按目录排除；设计文档保留单文件粒度，使未来真正需要更新的设计文档仍会被扫出
- **残余检查必须用加宽模式**（P1-A 的教训，已由评审实测）：`grep -rnE '(^|[^A-Za-z0-9_-])(internal/)?platform\b'` —— 它同时命中 `jimu/internal/platform/…`、`./internal/platform/…`、裸 `platform/…`、`"internal/platform"`，且不误伤 `xplatform/foo`。**并保留更强判据：文档引用的每个路径都必须在磁盘上存在**
- **全绿**：`gofmt -l .`（无输出）、`go build ./...`、`go vet ./...`、`golangci-lint run ./...`、`make check-log-usage`、`go test ./... -count=1`、`make bench-ci`、`make release-check COMPOSE_ENV=.env.example`
- **提交信息全英文**（Conventional Commits）；提交需用户明确指令
- 分支：从 `release/v0.3.0` 切 `feature/platform-relocation`，PR 目标 `release/v0.3.0`，squash 合并
- 简单优先：不拆包、不改内部结构、不顺手重构（混装包拆分属 P1-B2，反向依赖消除属 P1-B3）

## 归属映射表（实现与验证的唯一副本）

`internal/platform/<包>` → 目标目录（29 行；`→` 右侧即新的 import 路径片段）：

```text
# ---- 内核机制（kernel）：15 个 ----
auth          → internal/kernel/auth            # P1-B2 再拆出 access/apikey
breaker       → internal/kernel/breaker
cache         → internal/kernel/cache
db            → internal/kernel/db               # P1-B2 再拆出 breaker/encryption/retention
event         → internal/kernel/event
http          → internal/kernel/http             # P1-B2 再拆出 apidocs/uploadsec
httpclient    → internal/kernel/httpclient
logger        → internal/kernel/logger
mask          → internal/kernel/mask
observability → internal/kernel/observability
redis         → internal/kernel/redis
reporter      → internal/kernel/reporter
scheduler     → internal/kernel/scheduler
tenant        → internal/kernel/tenant           # Ruling：租户上下文是机制，不是能力实现（见下）
tlsconf       → internal/kernel/tlsconf

# ---- 能力实现（capabilities）：14 个 ----
breach        → internal/capabilities/breach
captcha       → internal/capabilities/captcha
encryption    → internal/capabilities/encryption
exporter      → internal/capabilities/dataops/exporter
feature       → internal/capabilities/feature
grpc          → internal/capabilities/grpc
importer      → internal/capabilities/dataops/importer
notification  → internal/capabilities/notification
oauth         → internal/capabilities/oauth/provider   # 包名 oauth → provider
outbox        → internal/capabilities/outbox
queue         → internal/capabilities/queue
search        → internal/capabilities/search
storage       → internal/capabilities/storage
ws            → internal/capabilities/ws
```

## Ruling: `platform/tenant` 归内核，不归 tenancy 能力

设计文档 §3.6 把 `platform/tenant`（上下文注入 + 编码校验/normalize）归到 `tenancy` 能力，并说"内核同时提供 no-op 实现"。实测该包被 **33 个能力文件 + 9 个内核文件**引用，它是**机制**（把租户身份从 JWT 中间件读到 context），与 `platform/auth` 的 JWT 中间件同类；若放进 `capabilities/tenancy`，则每个能力都要 import 另一个能力，且 `tenancy` 一关就全线编译失败 —— 与"可插拔"目标相反。
裁定：`platform/tenant` → `internal/kernel/tenant`；设计文档 §3.6 该行措辞在 Task 2 同步修正（"上下文机制在内核；tenancy 能力提供租户实体/套餐/配额/开通式注册"）。
代价（若判断错误）：设计文档与代码的归属表述需要再对一次；`tenancy` 关闭时租户上下文仍存在（返回默认/零租户），这正是 P0 已确立的语义。

## 现状基线（本计划编写时实测）

- 引用 `jimu/internal/platform/` 的 Go 文件：**166**（其中测试 67）
- 非 Go 引用（tracked，排除计划/设计文档）：`Makefile:288`、`scripts/bench_ci.sh:13`、`scripts/govulncheck.sh:5`（注释）、`.github/workflows/ci.yml:178`、`.github/workflows/release.yml:89`（**PG 迁移测试的包路径，改错即废掉该步骤**）、`AGENTS.md:41,42,43`、`README.md`（15 行）、`specs/**`（5 文件 15 行）、`docs/releases/v0.3.0.md`（1 行）
- 历史例外：`docs/releases/` 整目录 —— **已发布的版本日志是冻结记录，永不重写**（其中的旧路径原样保留，只与当时的发布内容对照；过滤链按目录而非按单文件排除）
- 子目录情况：`platform/oauth` 为扁平结构；`platform/grpc` 含子包（如 `userinfopb`），前缀替换会一并覆盖

---

### Task 1: 目录搬迁与 Go import 重写

**Files:**
- Move: `internal/platform/<29 个包>` → 按映射表
- Modify: 166 个引用文件（自动重写）+ `internal/capabilities/oauth/provider/*.go`（包名 `oauth`→`provider`）

**Interfaces:**
- Consumes: 映射表（上文唯一副本）
- Produces: 新命名空间 `internal/kernel/*` 与 `internal/capabilities/*`；除 `provider` 外包名与导出符号不变

- [ ] **Step 1: 搬迁目录**

```bash
git status --short          # 应干净
git rev-parse HEAD          # 记为 BASE
mkdir -p internal/kernel internal/capabilities/dataops internal/capabilities/oauth/provider
for p in auth breaker cache db event http httpclient logger mask observability redis reporter scheduler tenant tlsconf; do
  git mv "internal/platform/$p" "internal/kernel/$p"
done
for p in breach captcha encryption feature grpc notification outbox queue search storage ws; do
  git mv "internal/platform/$p" "internal/capabilities/$p"
done
git mv internal/platform/exporter internal/capabilities/dataops/exporter
git mv internal/platform/importer internal/capabilities/dataops/importer
# oauth 最后搬：目标是已存在目录下的新子目录
git mv internal/platform/oauth internal/capabilities/oauth/provider
rmdir internal/platform
ls internal/platform 2>&1   # 应报 No such file or directory
ls internal/kernel internal/capabilities
```

- [ ] **Step 2: 重写 import 路径（29 条映射，长路径优先）**

```bash
grep -rl 'jimu/internal/platform/' --include='*.go' . | while read -r f; do
  perl -pi -e '
    s|jimu/internal/platform/auth|jimu/internal/kernel/auth|g;
    s|jimu/internal/platform/breaker|jimu/internal/kernel/breaker|g;
    s|jimu/internal/platform/cache|jimu/internal/kernel/cache|g;
    s|jimu/internal/platform/db|jimu/internal/kernel/db|g;
    s|jimu/internal/platform/event|jimu/internal/kernel/event|g;
    s|jimu/internal/platform/httpclient|jimu/internal/kernel/httpclient|g;
    s|jimu/internal/platform/http|jimu/internal/kernel/http|g;
    s|jimu/internal/platform/logger|jimu/internal/kernel/logger|g;
    s|jimu/internal/platform/mask|jimu/internal/kernel/mask|g;
    s|jimu/internal/platform/observability|jimu/internal/kernel/observability|g;
    s|jimu/internal/platform/redis|jimu/internal/kernel/redis|g;
    s|jimu/internal/platform/reporter|jimu/internal/kernel/reporter|g;
    s|jimu/internal/platform/scheduler|jimu/internal/kernel/scheduler|g;
    s|jimu/internal/platform/tenant|jimu/internal/kernel/tenant|g;
    s|jimu/internal/platform/tlsconf|jimu/internal/kernel/tlsconf|g;
    s|jimu/internal/platform/breach|jimu/internal/capabilities/breach|g;
    s|jimu/internal/platform/captcha|jimu/internal/capabilities/captcha|g;
    s|jimu/internal/platform/encryption|jimu/internal/capabilities/encryption|g;
    s|jimu/internal/platform/exporter|jimu/internal/capabilities/dataops/exporter|g;
    s|jimu/internal/platform/feature|jimu/internal/capabilities/feature|g;
    s|jimu/internal/platform/grpc|jimu/internal/capabilities/grpc|g;
    s|jimu/internal/platform/importer|jimu/internal/capabilities/dataops/importer|g;
    s|jimu/internal/platform/notification|jimu/internal/capabilities/notification|g;
    s|jimu/internal/platform/oauth|jimu/internal/capabilities/oauth/provider|g;
    s|jimu/internal/platform/outbox|jimu/internal/capabilities/outbox|g;
    s|jimu/internal/platform/queue|jimu/internal/capabilities/queue|g;
    s|jimu/internal/platform/search|jimu/internal/capabilities/search|g;
    s|jimu/internal/platform/storage|jimu/internal/capabilities/storage|g;
    s|jimu/internal/platform/ws|jimu/internal/capabilities/ws|g;
  ' "$f"
done
```

> 顺序要紧：`httpclient` 必须排在 `http` 之前，`importer`/`exporter` 等长路径也必须先于任何可能作为前缀的规则。上面已按此排序。

- [ ] **Step 3: 改 oauth provider 的包名**

**好消息（实测）**：`platform/oauth` 的 5 个引用方**全部使用显式别名** `oauthplatform "jimu/internal/platform/oauth"`（共 18 处 `oauthplatform.Provider` 等），因此 **import 路径改完后限定符一个都不用动** —— 显式别名与包名无关。包内 8 个文件也**没有**自引用 `oauth.` 限定符。

所以本步只需改包声明：

```bash
grep -rl '^package oauth$' internal/capabilities/oauth/provider/*.go | while read -r f; do
  perl -pi -e 's|^package oauth$|package provider|' "$f"
done
grep -rn '^package ' internal/capabilities/oauth/provider/*.go | sed 's|.*/||'
go build ./... 2>&1 | head -20   # 应无输出
```

> 注意：`internal/capabilities/oauth/`（登录模块）自身的包名是 `oauth`，**不要**动它；只有 `provider/` 子目录的 8 个文件改。若 `go build` 报出任何限定符错误，说明有引用方没用别名 —— 按报错逐个修正并计入 Step 5 的 `DIFFERS` 解释。

> **配方修订（执行时发现）**：包名替换式必须**限定作用域**。写成全局 `s|^package oauth$|package provider|` 会连带把 `capabilities/oauth/`（登录模块）的包声明也改掉，破坏该模块。正确做法是只对目标目录执行：
>
> ```bash
> grep -rl '^package oauth$' internal/capabilities/oauth/provider/*.go | while read -r f; do
>   perl -pi -e 's|^package oauth$|package provider|' "$f"
> done
> ```
>
> 同理，Step 5 的重建式验证里该规则也必须只作用于 `internal/capabilities/oauth/provider/` 下的文件。

- [ ] **Step 3b: 检查"字符串被内嵌为长度前缀"的生成文件**

**这是执行时踩到的真破坏**：`internal/capabilities/grpc/userinfopb/userinfo.pb.go` 是 protobuf 生成文件，其内部把 `jimu/internal/platform/grpc/userinfopb` 作为**长度前缀的原始描述符字节**内嵌（前面有单字节长度）。纯文本替换会改变字符串长度而不同步长度字节，导致运行时 panic（`slice bounds out of range [-4:]`）并使 `internal/e2e` 变红。

处理步骤：

```bash
# 1) 找出所有可能内嵌路径的生成文件
grep -rln 'jimu/internal/' --include='*.pb.go' --include='*_gen.go' .

# 2) 用生成器重生成并比对，确认替换没有破坏描述符
make proto
git diff --stat -- '*.pb.go'
```

Expected: `userinfo.pb.go` 的原始描述符字节与长度前缀自洽（`make proto` 后无差异），且 `.proto` 的 `go_package` 已指向新路径。**若无法重生成（缺 protoc 工具链），必须手工修正长度字节并用 `go test ./internal/e2e/ -count=1` 证明 panic 消失**，并在报告中写明验证方式。

- [ ] **Step 4: 确认 Go 侧零残余**

```bash
grep -rnE '(^|[^A-Za-z0-9_-])(internal/)?platform\b' --include='*.go' . ; echo "exit=$?"
gofmt -l .
go build ./...
go vet ./...
go test ./... -count=1 2>&1 | tail -5
```

Expected: 第一条无输出（exit 1）；gofmt 无输出；build/vet 通过；测试全绿（包数与 P1-A 后一致）。

- [ ] **Step 5: 重建式验证（关键判据）**

搬迁后**除路径字符串与 `provider` 包名外不应有任何改动**：

```bash
# 把 BASE 的每个改动文件按 29 条映射重写后与 HEAD 逐字节比对
git diff -M --name-status BASE..HEAD > /tmp/b1_ns.txt
renames=0; differs=0
while read -r st src dst; do
  case "$st" in
    R*) s="$src"; d="$dst"; renames=$((renames+1)) ;;
    M*) s="$src"; d="$src" ;;
    *)  continue ;;
  esac
  git show "BASE:$s" | perl -pe '<与 Step 2 完全相同的 29 条替换式；追加 s|^package oauth$|package provider|>; s|oauth\.|provider.|g 仅限 provider 引用方>' > /tmp/b1_expected
  [ "${d##*.}" = "go" ] && gofmt /tmp/b1_expected > /tmp/b1_expected.fmt && mv /tmp/b1_expected.fmt /tmp/b1_expected
  git show "HEAD:$d" > /tmp/b1_actual
  cmp -s /tmp/b1_expected /tmp/b1_actual || { echo "DIFFERS: $s -> $d"; differs=$((differs+1)); }
done < /tmp/b1_ns.txt
echo "renames=$renames differs=$differs"
```

Expected: `differs` 应能收敛到 0，但**必须按下列程序处理非空结果**（执行时实测：只跑 29 条映射 + provider 包名规则会得到 30 处 `DIFFERS`，全部是计划未列出的路径文本与一处生成文件修正）：

对每个 `DIFFERS` 文件，逐条归入下列三类之一，并在报告中给出该文件与 BASE 的 diff 片段：

1. **注释/breadcrumb 路径文本**（实测 29 处，跨 28 文件）：文件头 `// internal/platform/...` 之类的面包屑、以及 `见 platform/httpclient` 之类正文引用。**判据：改写后引用的路径必须在磁盘上存在。**
2. **测试夹具中的功能路径**（实测 1 处：`tools/generator/compile_test.go` 的合成模块目录）：它不是装饰——`tools/generator/templates.go` 生成的 router 会 `import "jimu/internal/kernel/http/middleware"`，夹具必须让该路径存在，测试才能编译。**判据：与 `templates.go` 的生成产物一致。**
3. **生成文件的长度前缀修正**（见 Step 3b）：protobuf 描述符里内嵌路径的长度字节。

**不允许存在第 ④ 类**。若出现无法归入三类的差异，说明有真实语义变化，必须停下来报告而不是继续。

> 配方可复现性说明（评审提出）：本步骤不打印"扩展替换式"的字面正则，因为那会随发现过程而变；可复现的判据是上面这条**分类程序** —— 任何验证者跑完 29 条映射后，都应按三类逐条核对，并拒绝第四类。

- [ ] **Step 6: 提交（用户授权后执行）**

```bash
git add -A     # 随后 git status --short 复核：只应含 platform 目录的重命名与路径/包名重写
git commit -m "refactor(platform): relocate platform packages to kernel and capabilities"
```

---

### Task 2: 非 Go 引用同步 + 全量回归

**Files:**
- Modify: `Makefile:288`、`scripts/bench_ci.sh:13`、`scripts/govulncheck.sh:5`、`.github/workflows/ci.yml:178`、`.github/workflows/release.yml:89`、`AGENTS.md:41-43`、`README.md`（15 行，含目录树）、`specs/**`（5 文件 15 行）、`migrations/**`（7 文件，仅注释）、`docs/releases/v0.3.0.md`（1 行）、`docs/design/2026-09-18-capability-plugins-design.md`（§3.6 的 tenant 归属措辞 + §3.5.3 历史注记）

**Interfaces:**
- Consumes: Task 1 的新路径
- Produces: 全仓引用一致；发布门禁在最终提交上通过

- [ ] **Step 1: 逐个替换（按映射表）**

| 文件:行 | 原文 | 改为 |
|---|---|---|
| `.github/workflows/ci.yml:178` | `./internal/platform/db/...` | `./internal/kernel/db/...` |
| `.github/workflows/release.yml:89` | `./internal/platform/db/...` | `./internal/kernel/db/...` |
| `Makefile:288` | `./internal/platform/notification/...` `./internal/platform/http/...` `./internal/platform/queue/...` | `./internal/capabilities/notification/...` `./internal/kernel/http/...` `./internal/capabilities/queue/...` |
| `scripts/bench_ci.sh:13` 附近的三行 | 同上三个 | 同上三个 |
| `scripts/govulncheck.sh:5` | `internal/platform/importer` | `internal/capabilities/dataops/importer` |
| `AGENTS.md:41` | `internal/platform/tenant.FromContext(ctx)` | `internal/kernel/tenant.FromContext(ctx)` |
| `AGENTS.md:42` | `internal/platform/tenant.DefaultTenantID` | `internal/kernel/tenant.DefaultTenantID` |
| `AGENTS.md:43` | `internal/platform/tenant` | `internal/kernel/tenant` |
| `README.md` | 目录树 `├── platform/` 整块 + 正文 15 行 | 拆成 `├── kernel/`（15 个内核包）与并入 `├── capabilities/`（14 个能力包）；正文路径同步 |
| `specs/001-jimu-framework-spec/*.md`（5 文件 15 行） | 按映射表逐条替换（含裸/缩写形态） | — |
| `docs/releases/v0.3.0.md`（1 行） | 按映射表替换 | — |
| `migrations/{mysql,postgres}/{005,006,007}_add_*_tenant.sql`、`{mysql,postgres}/009_add_search_documents.sql`（共 7 文件） | 注释里的 `platform/tenant.DefaultTenantID`、`platform/search` | `kernel/tenant.DefaultTenantID`、`capabilities/search` |
| `docs/plans/2026-09-18-platform-relocation.md`（本计划，未跟踪） | — | **纳入本次提交**（与 P1-A 一致：计划文档随仓库留存） |

> 迁移文件的引用是**注释**（`-- 与 platform/tenant.DefaultTenantID 保持一致`），且 `internal/kernel/db/migrate.go` 不做校验和比对，故改写安全；但为达成"不留残余"仍应一并改掉。

- [ ] **Step 2: 设计文档同步**

1. §3.6 表中 `platform/tenant` 一行的归属改为：`kernel/tenant`（上下文机制；tenancy 能力负责租户实体/套餐/配额/开通式注册），并删去"内核同时提供 no-op 实现"的旧措辞。
2. §3.5.3 表前加历史注记：该表是**改造前的现状路径**（`platform/…`），P1-B1 已迁至 `kernel/…` 与 `capabilities/…`。

- [ ] **Step 3: 零残余检查（加宽模式 + 路径存在性）**

```bash
# 3a) 加宽模式的残留：只允许命中一处单文件例外（设计文档）+ 两个目录级例外（`docs/plans/`、`docs/releases/`）
grep -rnE '(^|[^A-Za-z0-9_-])(internal/)?platform\b' \
  --include='*.go' --include='*.md' --include='*.sh' --include='*.yml' --include='*.yaml' --include='*.json' \
  README.md AGENTS.md Makefile docs/ specs/ configs/ scripts/ .github/ internal/ tools/ 2>/dev/null \
  | grep -v '^docs/plans/' \
  | grep -v '^docs/releases/' \
  | grep -v '^docs/design/2026-09-18-capability-plugins-design.md'

# 3b) 路径存在性：文档里引用的 internal/... 路径必须真实存在
grep -rhoE '(jimu/)?internal/[a-z_/]+' README.md AGENTS.md docs/CONTRIBUTING.md \
  | sed 's|^jimu/||' | sort -u | while read -r p; do
    [ -e "$p" ] || echo "MISSING: $p"
  done
```

Expected: 3a 无输出；3b 只应报告"指向文件而非目录"或"示例/将来态"的已知情况（执行者需逐条解释，不得留未解释项）。

- [ ] **Step 4: 全量回归**

```bash
gofmt -l . && go build ./... && go vet ./... && golangci-lint run ./...
make check-log-usage
go test ./... -count=1
make bench-ci
make release-check COMPOSE_ENV=.env.example
```

Expected: 全部通过；`make release-check` 输出 `All checks passed`；`make bench-ci` 输出性能门禁通过（Task 1 改了 bench 目录）。

- [ ] **Step 5: 提交（用户授权后执行）**

```bash
git add README.md AGENTS.md Makefile scripts/ .github/ specs/ migrations/ docs/ && git commit -m "docs(platform): update path references for the platform relocation"
```

---

## Self-Review

**1. Spec 覆盖**

| spec 要求 | 覆盖 |
|---|---|
| §3.1 内核清单（db/redis/cache/http/logger/observability/breaker/event/tlsconf/mask/httpclient/reporter…） | Task 1 映射表 kernel 段（15 个） |
| §3.2–3.4 能力清单对应的平台实现（captcha/breach/notification/outbox/queue/storage/search/grpc/ws/encryption…） | Task 1 映射表 capabilities 段（14 个） |
| §3.5.3 `importer`/`exporter` → `dataops` 能力内部 | Task 1（`capabilities/dataops/{importer,exporter}`） |
| §3.6 混装包拆分（`platform/auth`→3、`platform/db`→6 类、`platform/http`→3 类） | **不在本计划**：属 P1-B2（本阶段只整包搬到位，避免一次改动同时做"搬家"与"拆包"） |
| §3.6 `platform/tenant` 归 tenancy + 内核 no-op | **本计划改为归 kernel**（见 Ruling），Task 2 同步设计文档 |
| §10 P1「`platform/` 下不再有跨能力的混装包」 | 部分：搬迁完成后 `platform/` 消失；混装包内部拆分属 P1-B2 |
| §10 P1「`platform → module` 反向依赖消除」 | 属 P1-B3（本阶段只搬，反向依赖仍在） |

**2. 占位符扫描**：无 TBD/TODO；映射表、替换式、逐文件替换表均为完整可执行内容；Step 5 的验证脚本中标注了两处必须与 Step 2 保持一致的占位说明（替换式与 provider 改名规则），执行者需照抄 Step 2 的代码块。

**3. 类型一致性**：本计划不引入新标识符；`contract.Descriptor`、各能力 `Descriptor`、`catalog` 均不改动。唯一符号层面变化是 `oauth` 包名→`provider`（含其 5 处引用方的限定符）。

**4. 已知取舍**：`kernel/auth`、`kernel/db`、`kernel/http` 在本阶段仍是混装包（含将分给 access/apikey/apidocs/uploadsec 的文件），这是刻意的——P1-B2 的拆分需要新包边界与端口设计，与搬迁混在一起会让 diff 无法用"重建式验证"判定。

**5. 风险**：`platform/tenant` 被 33 个能力文件引用，映射表中它的替换式必须排在正确位置（无前缀冲突）；`platform/oauth` → `capabilities/oauth/provider` 是唯一包名变更，是重建验证中唯一允许的 `DIFFERS` 来源，执行者必须逐条解释。

**6. 新增跨层边（终审发现）**

- `internal/kernel/db` → `internal/capabilities/encryption`：base 时消费方与被消费方同在平台层内，本阶段 `encryption` 按归属落入 `capabilities/` 而消费方留在内核，遂成为 kernel→capability 边（实测 `internal/kernel/db/encryption.go` 引用它）。
- `internal/kernel/http` → `internal/capabilities/storage`：同理（实测 `internal/kernel/http/upload_handler.go` 引用它）；这两条边的预期解法是 P1-B2 按设计文档 §3.6 拆分（`platform/db/encryption.go` → `encryption` 能力、`platform/http/upload_handler.go` → `uploadsec`），反向依赖消除由 P1-B3 跟踪。
