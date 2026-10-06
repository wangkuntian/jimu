# AI Agent skills 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 jimu 框架仓加一套可安装的 Agent skill（事实源 `skills/jimu/`，一条 `make skills-install` 软链到 `.claude/skills/` 与 `.agents/skills/`），让 AI Agent 装上后能按正确流程新增能力/形态/驱动、写迁移、跑门禁与排障。

**Architecture:** 单入口 + references 的渐进式披露：`SKILL.md` 只放铁律、本仓结构速记与六条工作流索引，细节按工作流拆进 `references/*.md`；reference 只写「怎么做 + 怎么验 + 红了看哪」并链接 README 既有章节，不复制规范正文、不写死会漂移的数字。安装与校验各一个 shell 脚本 + 一个 Makefile 目标，均不接入 `make ci`/`release-check`。

**Tech Stack:** Markdown（Agent Skills 约定：`SKILL.md` + YAML frontmatter）、bash、GNU Make、现有门禁工具（`make check-capabilities` / `profiles-check` / `compose-report-check`）

**当前 skill 事实源：** [skills/jimu/SKILL.md](../../../skills/jimu/SKILL.md)。本计划保留当时的实现方案。

## Global Constraints

- **无 Go 源码改动**：不改 `internal/**`、`cmd/**`、`tools/**`、`go.mod`/`go.sum`、`configs/**`、`templates/**`
- **不改 `.gitignore`**：`.claude/`、`.agents/` 维持忽略（第 45–46 行）；skill 事实源放顶层 `skills/`
- **不接入聚合门禁**：`check-skills` 只作为可手动目标，不得加进 `make ci` / `make release-check`
- **不覆盖第三方 skill**：安装目标已存在且不是指向本仓 `skills/<name>` 的软链时必须非零退出
- **reference 写作纪律**：只写「何时读 / 步骤 / 验收命令与期望输出 / 常见红与定位」四段；链接 README 章节而不是复制正文；数字只引用 `docs/profiles/compose-report.md` 或引用命令本身，不写死
- **frontmatter 契约**：`name` 必须等于目录名且匹配 `^[a-z0-9]+(-[a-z0-9]+)*$`；`description` 非空单行、含中英文触发关键词
- **相对链接基准**：`skills/jimu/references/*.md` 指向仓库根用 `../../../`；`skills/jimu/SKILL.md` 指向同目录 references 用 `references/<file>.md`
- **版本归属**：本轮进 v0.3.0；结束时按纪律在 release tip 复跑一次 `make release-check`
- **提交纪律**：未经用户明确指令不得 commit/push；本计划各任务的「提交」步骤一律标注为「待用户授权后执行」

---

### Task 1: 校验器 + Makefile 目标 + 入口 SKILL.md + 旗舰 reference

**Files:**
- Create: `scripts/check_skills.sh`
- Create: `skills/jimu/SKILL.md`
- Create: `skills/jimu/references/capability.md`
- Modify: `Makefile`（`.PHONY` 行、`help` 目标、新增 `check-skills` 目标）

**Interfaces:**
- Consumes: 无（首个任务）
- Produces: `make check-skills`（退出码 0 = 全部 skill 契约完整；非 0 = 有问题，错误行形如 `❌ skills/<name>: <原因>`，全绿打印 `✅ check-skills: N 个 skill 契约完整`）；`skills/jimu/SKILL.md` 的「工作流索引」表格（后续任务逐行追加）

- [ ] **Step 1: 写校验脚本（先写它，后续每个任务的验收都靠它）**

创建 `scripts/check_skills.sh`：

```bash
#!/usr/bin/env bash
# skills 契约校验：SKILL.md frontmatter（name/description）与 reference 引用完整性。
#
# 逐 skill 断言：
#   ① skills/<name>/SKILL.md 存在，且以 YAML frontmatter（首行 --- 起）开头
#   ② frontmatter 的 name 等于目录名，且匹配 ^[a-z0-9]+(-[a-z0-9]+)*$
#   ③ description 存在且非空
#   ④ SKILL.md 里每个 references/<file>.md 引用都指向存在的文件
#   ⑤ references/*.md 无孤儿（每个都被 SKILL.md 引用）
#   ⑥ skills/ 下不存在没有 SKILL.md 的目录；且至少有一个合法 skill
#
# 接入范围：只作 `make check-skills`，不加入 make ci / make release-check（设计 §6）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILLS_DIR="$ROOT/skills"

fail=0
err() { printf '❌ %s\n' "$1" >&2; fail=1; }

[ -d "$SKILLS_DIR" ] || { echo "❌ 缺少目录 skills/" >&2; exit 1; }

count=0
for dir in "$SKILLS_DIR"/*/; do
	[ -d "$dir" ] || continue
	name="$(basename "$dir")"
	entry="$dir/SKILL.md"

	if [ ! -f "$entry" ]; then
		err "skills/$name 缺少 SKILL.md"
		continue
	fi
	count=$((count + 1))

	# frontmatter：首个 --- 与下一个 --- 之间的块
	fm="$(awk 'NR==1 && $0=="---" {inside=1; next} inside && $0=="---" {exit} inside {print}' "$entry")"
	if [ -z "$fm" ]; then
		err "skills/$name/SKILL.md 缺少 YAML frontmatter（首行必须是 ---）"
	fi

	fmname="$(printf '%s\n' "$fm" | sed -n 's/^name:[[:space:]]*//p' | head -1)"
	fmdesc="$(printf '%s\n' "$fm" | sed -n 's/^description:[[:space:]]*//p' | head -1)"

	if [ "$fmname" != "$name" ]; then
		err "skills/$name: frontmatter name='$fmname' 与目录名不一致"
	fi
	if ! printf '%s' "$fmname" | grep -Eq '^[a-z0-9]+(-[a-z0-9]+)*$'; then
		err "skills/$name: name 必须匹配 ^[a-z0-9]+(-[a-z0-9]+)*$"
	fi
	if [ -z "${fmdesc//[[:space:]]/}" ]; then
		err "skills/$name: description 为空"
	fi

	# ④ SKILL.md 提到的 reference 必须存在
	while IFS= read -r ref; do
		[ -n "$ref" ] || continue
		[ -f "$dir/$ref" ] || err "skills/$name: 引用了不存在的 $ref"
	done < <(grep -oE 'references/[A-Za-z0-9._-]+\.md' "$entry" | sort -u)

	# ⑤ references/ 下不得有孤儿
	if [ -d "$dir/references" ]; then
		for f in "$dir"/references/*.md; do
			[ -f "$f" ] || continue
			rel="references/$(basename "$f")"
			grep -qF "$rel" "$entry" || err "skills/$name: $rel 未被 SKILL.md 引用（孤儿）"
		done
	fi
done

if [ "$count" -eq 0 ]; then
	err "skills/ 下没有合法的 skill 目录"
fi
if [ "$fail" -ne 0 ]; then
	echo "❌ check-skills: 见上方错误（修法：frontmatter 的 name 与目录同名、description 非空、引用与文件一一对应）" >&2
	exit 1
fi
echo "✅ check-skills: $count 个 skill 契约完整"
```

- [ ] **Step 2: 加 Makefile 目标与帮助行**

在 `Makefile` 的 `.PHONY` 段（`check-log-usage check-capabilities check-templates compose-report-check` 那一行之后）追加一行：

```make
.PHONY: check-skills skills-install
```

在 `help` 目标里，紧接 `make check-templates` 那行之后插入：

```make
	@echo "  make check-skills        校验 skills/** 的 frontmatter 与 reference 引用完整性"
	@echo "  make skills-install      把 skills/<name>/ 软链到 .claude/skills/ 与 .agents/skills/"
```

在 `## check-capabilities: ...` 目标块之前或之后（保持与相邻目标同格式）新增：

```make
## check-skills: 校验 skills/** 的 frontmatter 与 reference 引用完整性
check-skills:
	@./scripts/check_skills.sh
```

赋权：`chmod +x scripts/check_skills.sh`

- [ ] **Step 3: 建入口 SKILL.md（此时索引表只列 capability.md）**

创建 `skills/jimu/SKILL.md`（下方「工作流索引」表**暂时只保留第一行**，Task 2–6 各追加一行）：

````markdown
---
name: jimu
description: 在 jimu 框架仓内改动时使用：新增/删除能力、新增形态（profile）或驱动、编写能力内迁移与存量库 adopt、运行四道能力门禁与报告、排查运行时降级、使用 jimu new 脚手架生成项目。Use when adding capabilities, profiles, drivers, migrations, or running capability gates in the jimu repo.
---

# jimu 框架仓操作指南

本仓 v0.3.0 起按**能力**组织，能力在三个时机可组合：建项目时（`jimu new`）、编译时（形态 profile）、运行时（`capabilities.enabled`）。动手前先读本文件，再按「工作流索引」读对应 reference；reference 里链接的 README 章节是权威口径。

## 铁律（先看这段）

- **禁止自动提交**：没有用户明确指令，不得 `git commit` / `git push` / 创建任何 commit（[AGENTS.md](../../../AGENTS.md) 里此条优先级最高）
- **简单优先**：只写当前任务需要的最少代码；不顺手重构、不改无关格式与命名
- **保护工作区**：改前先 `git status --short`；不覆盖、不回滚、不格式化无关文件
- **文档同步**：改代码同步更新 `README.md` 对应章节；改源码的 PR 同步更新 `docs/releases/<version>.md`
- **能力边界**：能力之间只经 `internal/contract` 端口调用，禁止 import 其他能力的内部包；`contract.Descriptor` 是能力元数据的唯一来源
- **迁移纪律**：一条 ALTER 只属于一个能力，迁移写进该能力自己的目录
- **门禁必绿**：收工前跑 `make check-capabilities` 与 `make profiles-check`；报告相关改动还要 `make compose-report-check`

## 本仓结构速记

| 路径 | 是什么 |
|---|---|
| `internal/kernel/` | 内核（不可勾选）：JWT/Session/限流机制、db、租户上下文、Casbin 强制器、logger 等 |
| `internal/capabilities/<name>/` | 能力实现：静态 `Descriptor` + `contract.Module` + `wire.go` 自装配 |
| `internal/capabilities/catalog/` | 能力清单（唯一真源，18 项 + 非 catalog 条目标 `Ungated`） |
| `internal/capability/` | 描述符解析叶子包（`Resolve` / `ValidateDeclarations` / `Degraded`） |
| `internal/assembly/` | 装配与生命周期（`Assembly` / `Capability` / `Run` / `ProbeAssembly`） |
| `internal/profiles/<name>/` | 形态清单（`full`/`minimal`/`saas`/`enterprise`/`machine`）+ `registry`（形态总表）+ `active`（选点包） |
| `internal/contract/` | 能力间端口与 `Descriptor` 定义 |
| `cmd/server/` | 唯一入口：只 import `internal/assembly` 与选点包 `internal/profiles/active` |
| `cmd/cli/` | `jimu` CLI（`migrate` / `seed` / `new` / `capability` / `module` / `apikey`） |
| `tools/*` | 门禁与报告工具（`checkcapabilities` / `composereport` / `profileoverlay` / `profileassets` / `generator`） |
| `configs/*.yaml` | 内核段之外的能力配置段由各能力 `Descriptor.Configs` 声明并按启用集加载 |

## 工作流索引

| 我要做的事 | 读 |
|---|---|
| 新增 / 删除一个能力 | `references/capability.md` |

## 最常用的验收命令

```bash
make check-capabilities     # 5 条汇总行：自描述 ↔ 迁移、驱动、入口/选点包、资产归属
make profiles-check         # 5 形态 overlay 构建 + golden 依赖闭包
make compose-report-check   # 入库报告 == 本次实测（平台相关列掩码后比对）
make check-skills           # 本 skill 自身的契约校验
```

`make` 目标全表、含义与平台相关性说明见 README「[Makefile 命令](../../../README.md#makefile-命令)」。
````

- [ ] **Step 4: 写旗舰 reference `capability.md`（后续五份 reference 的风格样板）**

创建 `skills/jimu/references/capability.md`：

````markdown
# 新增 / 删除一个能力

**何时读**：往本仓加一个新能力（或删掉一个），或给能力补权限点、配置段、迁移归属、资产声明时。

**权威口径**：README「[开发规范 › 新增能力 / 驱动](../../../README.md#新增能力--驱动)」、「[开发规范 › 模块结构](../../../README.md#模块结构)」、「[能力清单](../../../README.md#能力清单)」。

## 步骤

1. **建能力目录**：`internal/capabilities/<name>/`，按需分层（`domain/` / `application/` / `infrastructure/` / `interfaces/`）。用脚手架打底：`./bin/jimu module create <name>`（只落骨架，**不改任何注册点**）
2. **导出静态 `Descriptor`**（`contract.Descriptor`，字段见 `internal/contract/capability.go`）：
   - `Name`：能力名，全仓唯一，必须与 `catalog` 清单里的名字一致
   - `Requires`：**硬依赖** —— 启用本能力必须同时启用这些能力，参与闭包补齐与拓扑序
   - `SoftRequires`：**软依赖** —— 目标缺失时本能力降级运行，不补齐、不参与拓扑序
   - `Owns`：本能力迁移 `CREATE` 的表名（**只认 `CREATE TABLE`**，`ALTER ... ADD` 不算归属）
   - `Migrations`：能力自带迁移的 `fs.FS`（根下有 `mysql/` 与 `postgres/`）
   - `Configs`：本能力拥有的配置段（`contract.ConfigSpec{Section, New}`），可多段
   - `Permissions`：能力拥有的权限点（种子时按启用集聚合写入）
   - `Mount`：路由挂载方式（零值等价 `MountProtected`，特权路由不会被裸挂到根路由）
   - `Drivers`：能力支持的**驱动包名**（有第三方驱动时才写）
   - `Assets`：能力拥有的非代码资产（仓库相对路径；有资产时才写）
3. **实现 `contract.Module`** 并写 `wire.go` 自装配（能力内部只依赖 `contract` 端口；禁止 import 其他能力的内部包）
4. **在 `internal/capabilities/catalog` 登记**该能力
5. **在需要它的形态清单 `internal/profiles/<name>/assembly.go` 里加入**；非 catalog 条目按 `Ungated` 声明。唯一入口 `cmd/server` 只调 `assembly.Run(active.Assembly())`，**不要**去 `cmd/server` 里逐个装配能力
6. **迁移**：写进 `internal/capabilities/<name>/migrations/{mysql,postgres}/`，能力内编号取该目录当前最大 +1；细节见 `references/migration.md`
7. **配置段**：新增配置段时同步 README「[配置项](../../../README.md#配置项)」与 `configs/*.yaml`（`configs/*.yaml` 是内核段之外的落地文件，键一经发布不得随意改动）
8. **资产**：声明 `Assets` 后跑门禁确认归属唯一（见 `references/gates.md`）
9. **删除能力**：从形态清单与 `catalog` 摘除 → 迁移与表按「运行时不删表」的既有取舍保留（README「[能力开关（v0.3.0）](../../../README.md#能力开关v030)」）→ 跑门禁确认没有悬空引用

## 验收命令与期望输出

```bash
make check-capabilities     # 必须是 5 条 ✅（① 自描述与 Owns ↔ 迁移归属 ② 驱动 ③ 形态只 import 已声明驱动 ④ 唯一入口与选点包 ⑤ 资产归属）
make profiles-check         # 5 形态 overlay 构建 + golden 依赖闭包通过
go test ./internal/... -count=1   # 能力自带单测全绿
```

## 常见红与定位

| 症状 | 定位 |
|---|---|
| `check-capabilities` ① 号红：声明的表没有迁移创建 / 一张表被两个能力声明 | `Owns` 与实际 `CREATE TABLE` 逐值对齐；`ALTER TABLE ... ADD` 不参与归属 |
| `check-capabilities` ④ 号红：选点包 import 了不止一个形态 | 只改形态包，**不要**动 `internal/profiles/active` 与 `cmd/server` |
| 能力声明了但没生效 | 该能力不在当前形态清单里，或 `capabilities.enabled` 排除了它 —— 见 `references/runtime.md` |
| 启动报「多个受保护中间件提供者」 | 同一启用集只允许一个能力**真实**提供受保护链，见 `references/runtime.md` |
````

- [ ] **Step 5: 跑校验，确认绿**

Run: `make check-skills`
Expected: `✅ check-skills: 1 个 skill 契约完整`

- [ ] **Step 6: 反向验证（门禁必须真的挡得住）**

```bash
# a) name 与目录不一致 → 必须红
sed -i.bak 's/^name: jimu$/name: wrongname/' skills/jimu/SKILL.md
make check-skills; echo "exit=$?"       # 期望：❌ … name='wrongname' 与目录名不一致；exit=1
mv skills/jimu/SKILL.md.bak skills/jimu/SKILL.md

# b) 断链 → 必须红
cp skills/jimu/SKILL.md /tmp/skill.bak
printf '\n见 `references/nope.md`。\n' >> skills/jimu/SKILL.md
make check-skills; echo "exit=$?"       # 期望：❌ … 引用了不存在的 references/nope.md；exit=1
cp /tmp/skill.bak skills/jimu/SKILL.md

# c) 孤儿 → 必须红
printf '# 孤儿\n' > skills/jimu/references/orphan.md
make check-skills; echo "exit=$?"       # 期望：❌ … references/orphan.md 未被 SKILL.md 引用；exit=1
rm skills/jimu/references/orphan.md

make check-skills                       # 还原后必须绿
```

- [ ] **Step 7: 提交（待用户授权后执行）**

```bash
git add scripts/check_skills.sh skills/jimu/SKILL.md skills/jimu/references/capability.md Makefile
git commit -m "feat(skills): add the jimu agent skill entry and its checker"
```

---

### Task 2: reference `profile-driver.md`（新增形态 / 新增驱动）

**Files:**
- Create: `skills/jimu/references/profile-driver.md`
- Modify: `skills/jimu/SKILL.md`（「工作流索引」表追加一行）

**Interfaces:**
- Consumes: `make check-skills`（Task 1）；SKILL.md 的「工作流索引」表
- Produces: 索引行文本 `| 新增一个形态（profile）或给能力加驱动 | `references/profile-driver.md` |`

- [ ] **Step 1: 写 `skills/jimu/references/profile-driver.md`**

四段结构（`何时读 / 步骤 / 验收命令与期望输出 / 常见红与定位`），权威口径链接 README「[形态（profile）](../../../README.md#形态profile)」及[形态与项目生成设计](../../design/profiles-and-project-generation.md)。内容必须覆盖：

**新增形态（profile）**
1. 新建 `internal/profiles/<name>/`（`assembly.go` 能力清单 + 可选 `drivers.go` blank import）
2. 在 `internal/profiles/registry` 的 `names`（固定顺序）与 `All` 里登记 —— `tools/profileoverlay -list`、`checkcapabilities`、`composereport`、`scripts/check_profiles.sh` 都从它派生，漏登记则门禁与报告看不到该形态
3. 同步 `scripts/check_profiles.sh` 的 `EXPECTED_<name>` / `FORBIDDEN_<name>` golden 闭包清单（逐值锁定，不更新即 `make profiles-check` 失败）
4. 断言不变式：`internal/profiles/active` 必须**恰好**只 import 一个形态；`registry` **不得**被 `cmd/server` 或选点包 import（`registry` import 全部形态包，被引用会让裁剪失效）
5. 构建与验证：`PROFILE=<name> make build-server`、`JIMU_PROFILES_SMOKE=1 make profiles-check`（需 DB+Redis）

**给能力新增第三方驱动（设计 §3.7 五步）**
1. 驱动独立成包 `internal/capabilities/<cap>/<driver>/`，包内 `init()` 调能力核心的 `Register`；核心只留接口 + 注册表（未注册即 **fail-closed**，不静默回退）；核心**不得** import 任何驱动包或第三方重型依赖
2. `Descriptor.Drivers` 加驱动**包名**（一个包可覆盖多个配置取值，如 `s3` 包覆盖 `s3`/`oss`/`minio`）
3. 形态 `assembly.Capability.Drivers` 列出选中子集（装配期强制 `⊆` 可用集）
4. `internal/profiles/<name>/drivers.go` blank import（`_ "jimu/internal/capabilities/<cap>/<driver>"`），并与第 3 步逐值一致
5. 门禁复核：驱动段 + `make compose-report` 的「重型依赖」列确实随形态消失

**验收命令**：`make check-capabilities`（5 条 ✅）、`make profiles-check`、`make compose-report-check`、`go run ./tools/profileassets <name>`（资产面）

**常见红**：golden 闭包不匹配（→ 更新 `scripts/check_profiles.sh` 期望集）、选点包 import 了两个形态、驱动忘声明（集合比较看不见未声明项，唯一捕获点是「驱动包只被 `internal/profiles/*` import」这条断言）

- [ ] **Step 2: 在 SKILL.md 索引表追加本行**

```markdown
| 新增一个形态（profile）或给能力加驱动 | `references/profile-driver.md` |
```

- [ ] **Step 3: 验收**

Run: `make check-skills`
Expected: `✅ check-skills: 1 个 skill 契约完整`（references 由 2 份变 2 份且无孤儿）

- [ ] **Step 4: 提交（待用户授权后执行）**

```bash
git add skills/jimu/references/profile-driver.md skills/jimu/SKILL.md
git commit -m "docs(skills): document profile and driver workflows"
```

---

### Task 3: reference `migration.md`（迁移编写与 adopt）

**Files:**
- Create: `skills/jimu/references/migration.md`
- Modify: `skills/jimu/SKILL.md`（索引表追加一行）

**Interfaces:**
- Consumes: `make check-skills`；SKILL.md 索引表
- Produces: 索引行 `| 编写 / 修改迁移、存量库 adopt | `references/migration.md` |`

- [ ] **Step 1: 写 `skills/jimu/references/migration.md`**

权威口径：README「[数据库迁移](../../../README.md#数据库迁移)」、[docs/releases/v0.3.0.md「说明」](../../../docs/releases/v0.3.0.md)。内容必须覆盖：

- **目录与编号**：`internal/capabilities/<name>/migrations/{mysql,postgres}/`；能力内版本号取该目录当前最大 +1（脚手架 `jimu module create` 自动完成）；**一条 ALTER 只属于一个能力** —— 它改的表归谁，迁移就写谁
- **两个方言都要写**：门禁只扫 mysql 判定归属，但 PostgreSQL 迁移必须同步（表名与 mysql 一致）
- **schema 依赖**：迁移集 = 形态声明集 ∪ schema 依赖（`user`/`access` → `tenant`）；`users`/`roles.tenant_id` 只由 tenant 的迁移创建，不含 tenant 的形态也会一并迁移它，否则建出写不进去的 schema
- **运行方式**：`jimu migrate up|down|status|redo` 与 `adopt-capabilities` 都跟随**编译期形态**（`capabilities.enabled` 不参与）；`full` 行为逐值不变
- **存量库升级路径**：旧二进制 `migrate up` 到旧世界最新 → 部署 v0.3.0 → `jimu migrate adopt-capabilities` 登记各能力版本表基线 → 之后正常 `migrate up`
- **`down`/`redo` 语义**：按**反向能力序**迭代，每能力每轮回滚其最后一条迁移；**已知限制** —— 各能力迁移深度不一时，drain-to-empty 可能因跨能力表依赖失败（如 `tenant` 005 Down 引用 `roles`），整库清空推荐重建库
- **运维警告**：同一数据库不要混用形态做迁移（`goose_db_version_<capability>` 会与实际 schema 错配）
- **运行时不删表**：关闭能力不删表与数据（刻意设计）

**验收命令**：`go test ./internal/kernel/db/... -count=1`（真实库集成用例无库时按设计 SKIP）、`make migrate-status`、`PROFILE=minimal make migrate-status`（表数下降）、`make check-capabilities`（① 号断言校验 `Owns` ↔ 迁移归属）

**常见红**：① 号断言报「表有迁移创建但没被任何能力声明」→ 补 `Owns` 或改归属；同一能力内重复版本号 → 重排编号

- [ ] **Step 2: 在 SKILL.md 索引表追加本行**

```markdown
| 编写 / 修改迁移、存量库 adopt | `references/migration.md` |
```

- [ ] **Step 3: 验收**

Run: `make check-skills`
Expected: `✅ check-skills: 1 个 skill 契约完整`

- [ ] **Step 4: 提交（待用户授权后执行）**

```bash
git add skills/jimu/references/migration.md skills/jimu/SKILL.md
git commit -m "docs(skills): document migration and adopt workflow"
```

---

### Task 4: reference `gates.md`（门禁与报告排障）

**Files:**
- Create: `skills/jimu/references/gates.md`
- Modify: `skills/jimu/SKILL.md`（索引表追加一行）

**Interfaces:**
- Consumes: `make check-skills`；SKILL.md 索引表
- Produces: 索引行 `| 门禁红了、报告漂移了 | `references/gates.md` |`

- [ ] **Step 1: 写 `skills/jimu/references/gates.md`**

权威口径：README「[质量门禁](../../../README.md#质量门禁)」、「[Makefile 命令](../../../README.md#makefile-命令)」、[docs/profiles/compose-report.md](../../../docs/profiles/compose-report.md)。内容必须覆盖：

- **`make check-capabilities` 5 条汇总行逐条解释**：① 能力自描述与 `Owns` ↔ mysql 迁移归属（单表唯一归属、无孤儿表、无未声明建表）② 驱动可用集 ↔ 目录存在 + 能力核心生产闭包零驱动零重型依赖 + 形态选中集 == 形态生产 import 闭包（集合比较）③ 形态生产代码只 import 已声明驱动 ④ 唯一入口 `cmd/server` 只 import `assembly` 与选点包，选点包恰好选一个形态且不 import `registry` ⑤ 资产归属唯一且无未声明资产（声明路径非空/存在/在资产根内、无重复所有者、无未声明资产、每形态覆盖全部内核资产组）
- **`make profiles-check`**：5 形态 overlay 构建 + golden 依赖闭包（`scripts/check_profiles.sh` 的 `EXPECTED_*`/`FORBIDDEN_*`）；`JIMU_PROFILES_SMOKE=1` 才做启动与 `/readyz` 冒烟（需 DB+Redis），未设时逐形态打印 SKIP 不算通过
- **`make compose-report-check`**：重新实测并比对入库报告的平台无关列（路由/迁移/表/本仓闭包文件数与代码行/重型依赖/直接依赖数）；**二进制大小列平台相关**（darwin 与 linux 不同），掩码后比对、只打印归档
- **`make check-templates`** 与重型矩阵：真实生成项目 + `go build/vet/test/run` 的用例统一由 `JIMU_HEAVY_MATRIX=1` 门控；本地入口 `make test-scaffold-matrix`；CI 在独立 workflow `.github/workflows/ci-scaffold.yml`，默认不在 PR 跑（`push release/**`、tag、`workflow_dispatch`、PR 打 `heavy-ci` 标签）
- **聚合关系**：`make ci` / `make release-check` 各含哪些目标；`make check-skills` **不在**聚合目标内（刻意）
- 每条门禁的「红了先看什么」：check-capabilities 看具体断言号 → 对应 reference；profiles-check 看哪个形态的闭包集合差异 → `scripts/check_profiles.sh` 期望集；compose-report-check 看首个差异行属于哪一列（平台相关列不参与）

**验收命令**：四个门禁各自单跑一次，期望输出逐条摘录（`✅ check-capabilities:` ×5、`✅ 各形态（overlay 构建 cmd/server）+ 依赖闭包裁剪门禁通过`、`✅ compose-report-check: …`）

- [ ] **Step 2: 在 SKILL.md 索引表追加本行**

```markdown
| 门禁红了、报告漂移了 | `references/gates.md` |
```

- [ ] **Step 3: 验收**

Run: `make check-skills && make check-capabilities`
Expected: 前者 `✅ check-skills: 1 个 skill 契约完整`；后者 5 条 `✅`

- [ ] **Step 4: 提交（待用户授权后执行）**

```bash
git add skills/jimu/references/gates.md skills/jimu/SKILL.md
git commit -m "docs(skills): document capability gates and report troubleshooting"
```

---

### Task 5: reference `runtime.md`（运行时降级与排障）

**Files:**
- Create: `skills/jimu/references/runtime.md`
- Modify: `skills/jimu/SKILL.md`（索引表追加一行）

**Interfaces:**
- Consumes: `make check-skills`；SKILL.md 索引表
- Produces: 索引行 `| 能力没生效、启动降级、形态起不来 | `references/runtime.md` |`

- [ ] **Step 1: 写 `skills/jimu/references/runtime.md`**

权威口径：README「[能力开关（v0.3.0）](../../../README.md#能力开关v030)」、「[形态（profile）](../../../README.md#形态profile)」、[docs/releases/v0.3.0.md「说明」](../../../docs/releases/v0.3.0.md)。内容必须覆盖：

- **三层时机区分**：建项目（`jimu new`）改 `go.mod`；编译期（profile）改编进二进制的包与符号（`go.mod` 不变）；运行时（`capabilities.enabled`）只决定能力是否生效，**不删表不删数据**
- **`enabled` 是完整解析集**：除 catalog 硬依赖闭包外，恒含七个非 catalog（`Ungated`）条目 —— `encryption`/`storage`/`notification`/`retention`/`apidocs`/`grpc`/`ws`
- **硬依赖 vs 软依赖**：`Requires` 补齐并参与拓扑序；`SoftRequires` 缺失只降级；降级报告是**声明层静态比对**（只读 `Descriptor`），可能多报
- **可观测点**：启动日志 `capabilities enabled` / `capabilities resolved`（含无 Module 实例的能力）、每个降级项的 `capability degraded` warn（字段 `name`/`missing`）、管理端口只读不鉴权的 `GET /capabilities` → `{"enabled":[…],"degraded":[{"capability":…,"missing":[…]}]}`
- **受保护中间件单提供者规则**：启用集含 `auth` 时由它提供（JWT + RBAC），无 `auth` 时由 `apikey` 提供（`X-API-Key` + `api:access`）；两个都非空会启动失败并列出冲突方；都没有而能力声明为受保护 → 拒绝启动。合法最小组合：`["auth"]` 或 `["user","access","apikey"]`
- **形态启动冒烟**：`JIMU_PROFILES_SMOKE=1 make profiles-check`（需 DB+Redis，端口 `JIMU_PROFILES_HTTP_PORT`/`JIMU_PROFILES_MGMT_PORT` 可覆盖）
- **`machine` 形态例外**：无 `auth`，`/api/v1/admin/apikeys` 不可自助签发，首把 Key 用 `PROFILE=machine make build-cli && ./bin/jimu-cli-machine apikey issue --name=first`

**验收命令**：`make profiles-check`、`JIMU_PROFILES_SMOKE=1 make profiles-check`（有 DB/Redis 时）、启动后 `curl -s localhost:<mgmt>/capabilities`

**常见红**：能力声明了却没路由 → 不在当前形态清单；降级报告里的缺失项实际装配期已被注入 → 静态比对的已知多报；`capabilities.enabled` 写了非 catalog 名字 → `catalog.ValidateDeclarations` 报错

- [ ] **Step 2: 在 SKILL.md 索引表追加本行**

```markdown
| 能力没生效、启动降级、形态起不来 | `references/runtime.md` |
```

- [ ] **Step 3: 验收**

Run: `make check-skills`
Expected: `✅ check-skills: 1 个 skill 契约完整`

- [ ] **Step 4: 提交（待用户授权后执行）**

```bash
git add skills/jimu/references/runtime.md skills/jimu/SKILL.md
git commit -m "docs(skills): document runtime capability degradation"
```

---

### Task 6: reference `scaffold.md`（脚手架用法）

**Files:**
- Create: `skills/jimu/references/scaffold.md`
- Modify: `skills/jimu/SKILL.md`（索引表追加一行）

**Interfaces:**
- Consumes: `make check-skills`；SKILL.md 索引表
- Produces: 索引行 `| 用脚手架生成项目 / 追加能力 | `references/scaffold.md` |`

- [ ] **Step 1: 写 `skills/jimu/references/scaffold.md`**

权威口径：README「[生成项目（`jimu new` / `jimu capability add`）](../../../README.md#生成项目jimu-new--jimu-capability-add)」、「[CLI 工具](../../../README.md#cli-工具)」。内容必须覆盖：

- **两条裁剪路径的区别**：`jimu new` 生成独立 module（`go mod tidy` 后 `go.mod` 真的变小）；形态（profile）只裁编进二进制的包与符号（`go.mod` 不变）
- **`jimu new <dir>`**：`--profile=<name>` 与 `--with=<cap>[:<drv>][,…]` 二选一（互斥）；`--with` 的能力名取自 catalog 18 ∪ Ungated 7，驱动默认取该能力 `Descriptor.Drivers` 首项，可用 `<cap>:<drv>` 覆盖；`--shape` 覆盖生成项目的形态名；`--module` 重写 module path（不改框架运行期名字/CLI 名/镜像名）；`--dry-run` 只打印计划；`--force` 只覆盖带 `.jimu-generated` 标记的产物；`--no-tidy` 跳过 `go mod tidy`；`--report` 写 `<dir>/docs/profiles/generated-report.md`
- **默认自检**：生成后跑 `go build ./...` 与生成项目自己的 `check-capabilities`，任一失败整体回滚
- **`jimu capability add <name>`**：在已生成项目上增量追加能力（确定性重渲染、幂等）；已有 `--report` 产物时同批刷新，不存在则不创建
- **`jimu module create <name>`**：**只**在本仓 `internal/capabilities/<name>/` 落骨架，不改任何注册点（注册见 `references/capability.md`）
- **生成项目与形态的边界**：生成项目的 Makefile/Dockerfile/`scripts/check_profiles.sh` 由 `templates/project/*.tmpl` 渲染（无 `PROFILE=`、无 overlay），框架 Makefile 的改动**不会**进产物
- **本仓门禁**：`make check-templates`（生成最小项目并真构建 + 跑生成项目门禁；被 `JIMU_HEAVY_MATRIX` 门控）、`make test-scaffold-matrix`（重型矩阵入口）

**验收命令**：`./bin/jimu new /tmp/sk-demo --profile=minimal --module=example.com/demo --dry-run`（只打印计划）、去掉 `--dry-run` 后 `cd /tmp/sk-demo && go build ./... && go run ./tools/checkcapabilities && go test ./... -count=1`

**常见红**：`frameworkRoot` 找不到（从过深目录执行，`configs` 向上搜索有 `SearchDepthUp` 上限，超限 fail-closed）、`go mod tidy` 失败、生成项目门禁红 → 模板漂移，跑 `make check-templates`

- [ ] **Step 2: 在 SKILL.md 索引表追加本行**

```markdown
| 用脚手架生成项目 / 追加能力 | `references/scaffold.md` |
```

- [ ] **Step 3: 验收（六份 reference 齐了，索引表应有六行）**

Run: `make check-skills && ls skills/jimu/references/`
Expected: `✅ check-skills: 1 个 skill 契约完整`；六个 `.md` 文件

- [ ] **Step 4: 提交（待用户授权后执行）**

```bash
git add skills/jimu/references/scaffold.md skills/jimu/SKILL.md
git commit -m "docs(skills): document the scaffolding commands"
```

---

### Task 7: 安装机制 `make skills-install`

**Files:**
- Create: `scripts/install_skills.sh`
- Modify: `Makefile`（新增 `skills-install` 目标；`.PHONY` 与 help 行已在 Task 1 加好）

**Interfaces:**
- Consumes: `skills/jimu/`（Task 1–6）
- Produces: `make skills-install`（幂等；成功打印 `installed <rel>/<name> -> <src>` / `unchanged <rel>/<name>`，末尾 `重启 Agent 会话即生效（事实源：skills/）`；冲突非零退出且不覆盖）

- [ ] **Step 1: 写安装脚本**

创建 `scripts/install_skills.sh`：

```bash
#!/usr/bin/env bash
# 把仓库内 skills/<name>/ 软链到各 Agent 的项目级发现路径。
#
# 事实源：skills/<name>/（入库）
# 目标：  .claude/skills/<name>（Claude Code 项目级）
#         .agents/skills/<name>（本工作区既有的 Agent skills 路径）
# 两者都在 .gitignore 内（本仓「AI 工具目录不入库」的既有约定）。
#
# 冲突策略：目标已存在且不是指向本仓 skills/<name> 的软链（真实目录/文件，或外来软链）
#           → 打印冲突并非零退出，**绝不覆盖**（保护 .agents/skills 里的第三方 skill 包）。
# 幂等：已指向本仓同一事实源 → 打印 unchanged 并跳过。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILLS_DIR="$ROOT/skills"
TARGETS=(".claude/skills" ".agents/skills")

[ -d "$SKILLS_DIR" ] || { echo "❌ 缺少目录 skills/" >&2; exit 1; }

changed=0
found=0
for dir in "$SKILLS_DIR"/*/; do
	[ -f "$dir/SKILL.md" ] || continue
	name="$(basename "$dir")"
	src="${dir%/}"
	found=$((found + 1))

	for rel in "${TARGETS[@]}"; do
		dst_dir="$ROOT/$rel"
		dst="$dst_dir/$name"
		mkdir -p "$dst_dir"

		if [ -L "$dst" ]; then
			current="$(readlink "$dst")"
			if [ "$current" = "$src" ]; then
				printf 'unchanged %s/%s\n' "$rel" "$name"
				continue
			fi
			echo "❌ $rel/$name 已是指向 $current 的软链，拒绝覆盖（手工处理后再跑）" >&2
			exit 1
		fi
		if [ -e "$dst" ]; then
			echo "❌ $rel/$name 已存在且不是软链，拒绝覆盖（第三方 skill 包请保留）" >&2
			exit 1
		fi

		ln -sfn "$src" "$dst"
		printf 'installed %s/%s -> %s\n' "$rel" "$name" "$src"
		changed=$((changed + 1))
	done
done

if [ "$found" -eq 0 ]; then
	echo "❌ skills/ 下没有带 SKILL.md 的 skill" >&2
	exit 1
fi
if [ "$changed" -eq 0 ]; then
	echo "no changes（全部已安装）"
fi
echo "重启 Agent 会话即生效（事实源：skills/）"
```

赋权：`chmod +x scripts/install_skills.sh`

- [ ] **Step 2: 加 Makefile 目标**

```make
## skills-install: 把 skills/<name>/ 软链到 .claude/skills/ 与 .agents/skills/（幂等，不覆盖非本仓条目）
skills-install:
	@./scripts/install_skills.sh
```

- [ ] **Step 3: 正向验证（幂等）**

```bash
make skills-install                      # 期望：installed .claude/skills/jimu -> … 与 installed .agents/skills/jimu -> …
ls -l .claude/skills/jimu/SKILL.md .agents/skills/jimu/SKILL.md
python3 -c 'import os,sys;print(os.path.realpath(sys.argv[1]))' .claude/skills/jimu/SKILL.md  # 期望：<repo>/skills/jimu/SKILL.md
make skills-install                      # 期望：两条 unchanged + no changes，exit 0
git status --short                       # 期望：不出现 .claude/ 与 .agents/
```

- [ ] **Step 4: 反向验证（冲突必须挡住且不破坏）**

```bash
rm .claude/skills/jimu
mkdir -p .claude/skills/jimu && echo keep > .claude/skills/jimu/marker
make skills-install; echo "exit=$?"      # 期望：❌ … 已存在且不是软链；exit=1
cat .claude/skills/jimu/marker           # 期望：keep（未被破坏）
rm -rf .claude/skills/jimu
ln -s /tmp .claude/skills/jimu
make skills-install; echo "exit=$?"      # 期望：❌ … 已是指向 /tmp 的软链；exit=1
rm .claude/skills/jimu
make skills-install                      # 还原后重新装好
```

- [ ] **Step 5: 提交（待用户授权后执行）**

```bash
git add scripts/install_skills.sh Makefile
git commit -m "feat(skills): add the skills-install target"
```

---

### Task 8: 文档接入（README / AGENTS.md / 版本日志）

**Files:**
- Modify: `README.md`（新增「AI Agent skills（v0.3.0）」一节）
- Modify: `AGENTS.md`（「开发前必读」加一行）
- Modify: `docs/releases/v0.3.0.md`（「新增」段加一条）

**Interfaces:**
- Consumes: `make skills-install` / `make check-skills`（Task 1、7）
- Produces: 面向人的入口文档；README 该节的锚点为 `#ai-agent-skillsv030`（供其它文档引用）

- [ ] **Step 1: README 加一节**

插入位置：「形态（profile）」相关章节之后、「项目结构」之前（`## 生成项目（jimu new / jimu capability add）` 与 `## 数据库迁移` 之间的位置亦可，保持与相邻章节同级 `##`）。内容：

````markdown
## AI Agent skills（v0.3.0）

本仓把「怎么改这个代码库」的操作知识做成可安装的 Agent skill：事实源是**受版本控制的** `skills/jimu/`（入口 `SKILL.md` + `references/*.md`），安装即把它软链到各 Agent 的项目级发现路径。

```bash
make skills-install     # 软链到 .claude/skills/ 与 .agents/skills/（幂等；已存在且不是本仓软链时拒绝覆盖）
make check-skills       # 校验 frontmatter（name 与目录同名、description 非空）与 reference 引用完整性
```

覆盖的工作流：新增/删除能力、新增形态（profile）与驱动、迁移编写与存量库 adopt、门禁与报告排障、运行时降级排障、`jimu new` 脚手架用法。

- **`.claude/` 与 `.agents/` 不入库**（`.gitignore` 的「AI」段）：Agent 的发现路径随工具而变，因此事实源独立放 `skills/`，安装只是软链；换工具时在 `scripts/install_skills.sh` 的目标列表里加一行即可
- `make check-skills` **不在** `make ci` / `make release-check` 的成员里（刻意如此，聚合目标的发布语义不变）
- 生成项目（`jimu new` 的产物）**不带** skill；脚手架相关用法见上面的「生成项目」章节
````

- [ ] **Step 2: AGENTS.md 加一行索引**

在 `## 开发前必读` 列表末尾追加：

```markdown
- 本仓有可安装的 Agent skill：事实源 `skills/jimu/`，`make skills-install` 装到 `.claude/skills/` 与 `.agents/skills/`；改动前先读入口 `skills/jimu/SKILL.md`（铁律 + 六条工作流索引）
```

- [ ] **Step 3: 版本日志加一条（「新增」段）**

在 `docs/releases/v0.3.0.md` 的 `## 新增` 段追加：

```markdown
- **AI Agent skills（v0.3.0）** — 把「在本仓改代码」的操作知识做成可安装的 Agent skill：事实源是受版本控制的 `skills/jimu/`（入口 `SKILL.md`：铁律 + 本仓结构速记 + 六条工作流索引，配 `references/{capability,profile-driver,migration,gates,runtime,scaffold}.md`），`make skills-install` 逐 skill 软链到 `.claude/skills/` 与 `.agents/skills/`（幂等；目标已存在且不是指向本仓事实源的软链时拒绝覆盖，保护第三方 skill 包），`make check-skills` 校验 frontmatter（`name` 与目录同名、`description` 非空）与 reference 引用完整性（无断链、无孤儿）。`.claude/`、`.agents/` 维持 gitignore（AI 工具目录不入库），故事实源独立成目录、安装只做软链。**不接入** `make ci`/`make release-check`；生成项目不带 skill（生成器零改动）；无 Go 源码与配置改动
```

- [ ] **Step 4: 校验相对链接与门禁**

```bash
python3 - <<'PY'
import re, os
bad = []
for f in ("README.md", "AGENTS.md", "docs/releases/v0.3.0.md"):
    base = os.path.dirname(f)
    for m in re.finditer(r'\]\(([^)\s]+)\)', open(f, encoding="utf-8").read()):
        t = m.group(1)
        if t.startswith(("http://", "https://", "#", "mailto:")):
            continue
        p = os.path.normpath(os.path.join(base, t.split("#")[0]))
        if not os.path.exists(p):
            bad.append((f, t))
print("broken links:", bad or "none")
PY
make check-skills
```

Expected: `broken links: none`；`✅ check-skills: 1 个 skill 契约完整`

- [ ] **Step 5: 提交（待用户授权后执行）**

```bash
git add README.md AGENTS.md docs/releases/v0.3.0.md
git commit -m "docs: document the installable agent skill"
```

---

### Task 9: 端到端验收（含 release-check 复跑）

**Files:**
- Modify: `docs/releases/v0.3.0.md`（「验证」段的最终条目，把本轮结果并入）

**Interfaces:**
- Consumes: Task 1–8 的全部产物
- Produces: 可直接用于发布的验证记录

- [ ] **Step 1: 全量结构验收**

```bash
make check-skills                                    # ✅ check-skills: 1 个 skill 契约完整
make check-capabilities                              # 5 条 ✅
make profiles-check                                  # 5 形态 overlay 构建 + golden 闭包通过
make compose-report-check                            # ✅ 与入库报告一致
ls -l .claude/skills/jimu/SKILL.md .agents/skills/jimu/SKILL.md
python3 -c 'import os,sys;[print(os.path.realpath(p)) for p in sys.argv[1:]]' .claude/skills/jimu/SKILL.md .agents/skills/jimu/SKILL.md
git status --short                                   # 只出现预期改动；无 .claude/、.agents/
```

Expected: 全部 ✅；两条软链都解析到 `<repo>/skills/jimu/SKILL.md`；`git status` 无被忽略目录

- [ ] **Step 2: 复跑发布门禁**

Run: `make release-check`
Expected: `All checks passed`（`test-scaffold-matrix` 约 9–10 分钟；本机 Docker/OrbStack 需运行中，`compose-check` 依赖它）

- [ ] **Step 3: 把结论并入版本日志「验证」段**

在 `docs/releases/v0.3.0.md` 现有的「**发布前最终验证（release tip …）**」块末尾追加一条：

```markdown
- **AI Agent skills 落地后复跑**：`make check-skills` → `✅ check-skills: 1 个 skill 契约完整`；`make skills-install` 幂等（重复执行打印 `unchanged`、exit 0），`.claude/skills/jimu` 与 `.agents/skills/jimu` 均软链到 `<repo>/skills/jimu`；冲突路径（真实目录 / 外来软链）非零退出且未覆盖；`make check-capabilities` 5 条 ✅、`make profiles-check` 与 `make compose-report-check` 通过；`make release-check` → `All checks passed`
```

- [ ] **Step 4: 提交（待用户授权后执行）**

```bash
git add docs/releases/v0.3.0.md
git commit -m "docs(release): record the agent skill verification"
```

---

## Self-Review

**1. Spec coverage**

| Spec 章节 | 对应任务 |
|---|---|
| §4.1 目录结构（`skills/jimu/` + 六份 reference + `scripts/check_skills.sh`） | T1（SKILL.md + capability.md + check_skills.sh）、T2–T6（其余五份） |
| §4.2 frontmatter 契约 | T1 Step 3（`name: jimu` + 中英 `description`）、T1 Step 1 断言 ②③ |
| §4.3 入口内容骨架（铁律 / 结构速记 / 索引表 / 验收速查） | T1 Step 3 |
| §4.4 六份 reference 内容大纲 | T1（capability）、T2、T3、T4、T5、T6 |
| §4.4 写作纪律（不复制正文、数字不写死） | Global Constraints + 每份 reference 的「权威口径」链接要求 |
| §5 安装机制（幂等、冲突不覆盖、打印、只软链本仓） | T7 |
| §6 门禁 6 条断言 + 不接聚合目标 | T1 Step 1、Step 6；T7 Step 4 |
| §7 文档更新（README / AGENTS.md / Makefile / 版本日志 / 设计文档不改） | T1 Step 2、T7 Step 2、T8；设计文档刻意未改（Global Constraints 与 T8 未涉及它） |
| §8 验收 1–9（正向 1–5、反向 6–9） | T1 Step 5–6、T7 Step 3–4、T8 Step 4、T9 Step 1–3 |
| §9 版本归属与发布影响（进 v0.3.0 + 复跑 release-check） | Global Constraints、T9 Step 2 |
| §10 风险（漂移 / 覆盖第三方 / 工具差异 / 命中率） | 漂移→写作纪律；覆盖→T7 Step 4；工具差异→T7 目标列表；命中率→T1 的 description 与 T8 Step 2 索引行 |

无遗漏项。

**2. Placeholder scan**：无 TBD/TODO；五份非旗舰 reference 用「必须覆盖的事实清单 + 权威链接」表达，事实全部是本仓已存在的路径/命令/常量（`hollow` 的步骤不存在）；脚本与 Makefile 片段均为可直接粘贴的完整文本。

**3. Type consistency**：`make check-skills`、`make skills-install`、`scripts/check_skills.sh`、`scripts/install_skills.sh`、`skills/jimu/SKILL.md`、`references/*.md` 在 T1–T9 中命名一致；`✅ check-skills: N 个 skill 契约完整` 的输出文案在 T1 定义、T4/T6/T7/T8/T9 引用一致。

## 执行记录与偏差（2026-09-29 执行完毕）

| 项 | 偏差 |
|---|---|
| T1 Step 1 的脚本 | **多了一条断言 ⑦**：`SKILL.md` 与 `references/*.md` 的相对链接可解析（`http(s)`/`mailto`/纯锚点跳过，不校验锚点片段）。原因是实现中真踩到：`SKILL.md` 上溯仓库根应为 `../../`，初稿写成 `../../../` 会跳出仓库，而 ①–⑥ 全绿看不见。设计 §6 已同步增补该条 |
| T1 Step 1 / T9 Step 1 的 `readlink -f` | 换成 `python3 -c 'import os,sys;print(os.path.realpath(sys.argv[1]))'` —— 纯 macOS 无 `readlink -f`（本机因装了 coreutils 才能跑，属侥幸） |
| T8 Step 3 的插入位置 | 版本日志**没有**「新增」段（分类为 亮点/变更/修复/验证/说明），条目落到 `## 变更` 末尾（遵循 CONTRIBUTING 模板「没有内容的分类可以省略」） |
| T1–T6 的分任务提交 | 未按计划逐任务提交，改为交付后按主题合三条：`feat(skills)`（skill + 两脚本 + Makefile）/ `docs`（README + AGENTS）/ `docs(release)`（版本日志 + 本计划）。逐任务提交需重放中间态，属人为历史 |
| 全部 9 个任务 | 其余步骤按计划原样执行；验收命令、期望输出与反向验证（含 T1 Step 6 三种破坏、T7 Step 4 两种冲突、T9 复跑 `make release-check`）全部实跑通过 |
