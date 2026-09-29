# AI Agent skills（v0.3.0）设计

**状态**：设计已确认，待转实现计划
**关联**：[AGENTS.md](../../AGENTS.md)（协作规范）、[docs/CONTRIBUTING.md](../CONTRIBUTING.md)（开发流程）、[README.md](../../README.md)（能力模型与门禁）

## 1. 背景与目标

v0.3.0 的能力模型（`Descriptor` 单一元数据来源、5 个形态、驱动级可插拔、非代码资产归属、四道门禁、`jimu new` 脚手架）已经冻结，但**操作它的知识只散落在 README / AGENTS.md / 设计文档里**：AI Agent 接手本仓时要么全量读 README（上下文成本高），要么漏掉关键铁律（禁止自动提交、能力间只经 `contract`、一条 ALTER 只属一个能力、门禁必绿）。

**目标**：把「在本仓改代码」的操作知识做成**可安装的 Agent skill** —— 装一次，Agent 下次会话就能按正确流程新增能力/形态/驱动、写迁移、跑门禁、排障，而不必先读完整个 README。

**成功判据**：

1. `make skills-install` 一条命令后，Claude Code 等 Agent 能发现 `jimu` skill（软链可解析）；
2. skill 覆盖六条工作流：新增能力、新增形态/驱动、迁移与 adopt、门禁与报告排障、运行时降级排障、脚手架用法；
3. `make check-skills` 能挡住 frontmatter 失配与断链；
4. 不改 Go 源码、不改生成器、不改 `go.mod`、不改 `configs/*.yaml`。

## 2. 非目标

- **不随 `jimu new` 生成项目出货**（生成器零改动）；生成项目要不要自带 skill 是后续独立议题
- **不做可安装包形态**（`npx skills add` / plugin marketplace / 独立仓）
- **不接 `make ci` / `make release-check`**：`check-skills` 先作为可手动目标观察（与 `check-templates` 当初的路径一致）
- **不提供卸载目标**（`make skills-uninstall`）；软链是幂等重建的，手工 `rm` 即可
- **不改 `.gitignore` 对 `.claude/`、`.agents/` 的忽略**（本仓「AI 工具目录不入库」的既有取舍）

## 3. 现状与约束（调研结论）

| 事实 | 影响 |
|---|---|
| `.gitignore` 第 45–46 行忽略 `.claude/`、`.agents/` | skill **不能**住在 Agent 发现路径里，必须有入库的事实源目录 + 安装步骤 |
| 本工作区 `.agents/skills/` 已装第三方 superpowers 包（含 `superpowers.json`） | 安装必须是**逐 skill 软链**，不得重写或清理该目录 |
| 生成项目的 Makefile 来自 `templates/project/Makefile.tmpl`（`tools/generator/buildfiles.go`），**不是**从框架 Makefile 派生 | 给框架 Makefile 加 `skills-install` 不会泄漏进 `jimu new` 产物，生成器无需改动 |
| `ci.yml` 的 CHANGELOG 门禁只对 `^(internal/\|cmd/\|configs/)` 的改动生效 | `skills/** + Makefile + scripts/**` 不触发该门禁；版本日志仍按规范人工更新 |
| 本仓 AI 工具已读 `AGENTS.md`（`CLAUDE.md` 是它的软链） | AGENTS.md 只加一行索引指向 skill，避免规范正文重复两份 |
| Agent Skills 约定：`SKILL.md` + YAML frontmatter（`name` 必须与目录同名、`description` 用于命中） | 事实源目录结构直接采用该约定，安装即「软链到发现路径」 |

## 4. 方案：单入口 + references（渐进式披露）

### 4.1 目录结构

```
skills/
└── jimu/
    ├── SKILL.md                       # 入口：铁律 + 六条工作流索引
    └── references/
        ├── capability.md              # 新增能力全流程
        ├── profile-driver.md          # 新增形态 / 新增驱动
        ├── migration.md               # 迁移编写与 adopt
        ├── gates.md                   # 门禁与报告排障
        ├── runtime.md                 # 运行时降级与排障
        └── scaffold.md                # 脚手架用法
scripts/check_skills.sh                # skills 校验（make check-skills 的实现）
```

### 4.2 frontmatter 契约

```yaml
---
name: jimu
description: 在 jimu 框架仓内改动时使用：新增/删除能力、新增形态（profile）或驱动、编写能力内迁移、运行四道能力门禁与报告、排查运行时降级。Use when adding capabilities, profiles, drivers, migrations, or running capability gates in the jimu repo.
---
```

- `name` 必须等于目录名，模式 `^[a-z0-9]+(-[a-z0-9]+)*$`
- `description` 非空、单行、含触发关键词（能力/形态/驱动/迁移/门禁 + 英文同义词），因为它是 Agent 唯一的命中依据
- 正文用中文（与本仓文档一致）

### 4.3 入口 `SKILL.md` 内容骨架

1. **铁律**（不重复规范正文，只列最易踩的）：未经明确指令禁止 commit/push；简单优先；保护工作区（先 `git status --short`）；改代码同步 README 与 `docs/releases/<version>.md`；能力间只经 `contract` 端口；一条 ALTER 只属一个能力；改完门禁必绿
2. **本仓结构速记**：`internal/{kernel,capabilities,catalog,assembly,profiles,contract,config,app}`、`cmd/{server,cli}`、`tools/*` 各是什么（各一行）
3. **六条工作流的索引表**：何时读哪份 reference、读完的验收命令

示例索引行：

| 我要做的事 | 读 | 验收 |
|---|---|---|
| 新增一个能力 | `references/capability.md` | `make check-capabilities` 5 条 ✅ + `make profiles-check` |
| 新增一个形态 | `references/profile-driver.md` | `make profiles-check` + `make check-capabilities` |
| 让某能力支持新驱动 | `references/profile-driver.md` | `make check-capabilities` + `make compose-report` |
| 加/改迁移 | `references/migration.md` | `go test ./internal/kernel/db/...` + `make migrate-status` |
| 门禁红了 | `references/gates.md` | 对应目标复绿 |
| 能力没生效 / 降级 | `references/runtime.md` | `GET /capabilities` + 启动日志 |

### 4.4 六份 reference 的内容大纲

每份统一四段：**何时读** / **步骤** / **验收命令与期望输出** / **常见红与定位**。

| reference | 核心内容 | 对应权威来源 |
|---|---|---|
| `capability.md` | `Descriptor` 逐字段（`Requires`/`SoftRequires`/`Owns`/`Configs`/`Permissions`/`Mount`/`Migrations`/`Drivers`/`Assets`）→ `wire.go` 自装配 → `catalog` 登记 → 形态清单加入（非 catalog 标 `Ungated`）→ 权限点/配置段 | README「开发规范 › 新增能力 / 驱动」 |
| `profile-driver.md` | 形态包 + `registry` 登记 + `scripts/check_profiles.sh` 的 `EXPECTED_<name>` golden 同步 + 选点包不变式；驱动独立成包 + `Register` + `Descriptor.Drivers`/`assembly.Capability.Drivers` 两处声明 + `drivers.go` blank import | README「形态（profile）」「驱动级可插拔（P2.5）」 |
| `migration.md` | 迁移写进 `internal/capabilities/<name>/migrations/<方言>/`、能力内编号取最大 +1、一条 ALTER 只属一个能力、schema 依赖（`user`/`access` → `tenant`）、`adopt-capabilities` 存量路径、`down`/`redo` 按反向能力序与「不等深度 drain 会失败」的已知限制 | README「数据库迁移」、release note「说明」 |
| `gates.md` | `check-capabilities` 5 条断言分别管什么、`profiles-check` 的 golden 闭包、`compose-report-check` 的平台相关列掩码、`check-templates` 与 `JIMU_HEAVY_MATRIX`；每条「红了先看什么」 | README「质量门禁」、Makefile 命令表 |
| `runtime.md` | `capabilities.enabled` 闭包语义、`SoftRequires` 降级的「声明层静态比对、可能多报」、`GET /capabilities`、`capability degraded` 日志、形态启动冒烟 `JIMU_PROFILES_SMOKE=1`、受保护中间件单提供者规则 | README「能力开关（v0.3.0）」、release note「说明」 |
| `scaffold.md` | `jimu new`（`--profile`/`--with`/`--shape`/`--module`/`--dry-run`/`--force`/`--no-tidy`/`--report`）与 `jimu capability add`、生成项目 vs 形态两条裁剪路径的区别（前者才减小 `go.mod`）、`jimu module create` 只落骨架不改注册点 | README「生成项目」「CLI 工具」 |

**写作纪律**：reference 只写「怎么做 + 怎么验 + 红了看哪」，**不复制** README/设计文档的规范正文，改为链接到具体章节；数字（路由数、表数、闭包计数）只引用 `docs/profiles/compose-report.md`，不在 skill 里写死，避免报告更新后 skill 陈旧。

## 5. 安装机制：`make skills-install`

```
src = skills/<name>/                 （仓库内事实源）
dst = .claude/skills/<name>          （Claude Code 项目级发现路径）
      .agents/skills/<name>          （本工作区既有的 Agent skills 路径）
```

行为细则：

1. 遍历 `skills/*/`（含 `SKILL.md` 的目录才是 skill），目标父目录 `mkdir -p`
2. `ln -sfn <绝对路径 src> <dst>`，**幂等**：重复执行结果不变、exit 0
3. 冲突处理：`dst` 存在且不是指向 `skills/` 的软链（真实目录或外来软链）→ 打印冲突与原因、非零退出，**不覆盖**（保护第三方 skill 包）
4. 成功时逐个打印 `installed <name> -> <dst>`，末尾打印一句「重启 Agent 会话即生效」
5. 只装「本仓 `skills/` 下的 skill」，不清理目标目录里的任何其它条目

## 6. 门禁：`make check-skills`

`scripts/check_skills.sh`（bash，无 Go 依赖），逐 skill 断言：

1. `skills/<name>/SKILL.md` 存在，且 frontmatter 以 `---` 起止
2. `name:` 等于目录名且匹配 `^[a-z0-9]+(-[a-z0-9]+)*$`
3. `description:` 存在且非空
4. `SKILL.md` 中每个 `references/<file>` 引用都指向存在的文件
5. `references/` 下每个 `.md` 都被 `SKILL.md` 引用（无孤儿）
6. `skills/` 下不存在没有 `SKILL.md` 的目录（半成品）

任一失败打印「哪个 skill / 哪条断言 / 怎么修」并非零退出；全绿打印一行 `✅ check-skills: N 个 skill 契约完整`。

**接入范围**：只加 Makefile 目标与 help 行，**不加入** `make ci` / `make release-check`（保持聚合目标的发布语义不变）。

## 7. 文档更新

| 文件 | 改动 |
|---|---|
| `README.md` | 新增「AI Agent skills（v0.3.0）」一节：`make skills-install` / `make check-skills` 用法、事实源是 `skills/`、`.claude/`、`.agents/` 不入库的原因与既有约定 |
| `AGENTS.md` | 「开发前必读」加一行：本仓有 `skills/jimu`（`make skills-install` 安装），改动前先读入口 SKILL.md |
| `Makefile` | 新增 `skills-install`、`check-skills` 两个目标 + `.PHONY` + help 行 |
| `docs/releases/v0.3.0.md` | 「新增」段加一条（能力模型与 skill 配套） |
| `docs/design/2026-09-18-capability-plugins-design.md` | **不改**：skill 不进 §10 的 P0–P3 口径，只由版本日志记录 |

## 8. 测试与验收

**正向**

1. `make check-skills` → `✅ check-skills: 1 个 skill 契约完整`
2. `make skills-install` → 打印 installed；`.claude/skills/jimu/SKILL.md` 与 `.agents/skills/jimu/SKILL.md` 均为软链且 `readlink -f` 解析到仓库内 `skills/jimu/SKILL.md`
3. 重复执行 `make skills-install` → exit 0 且 `ls -l` 结果不变（幂等）
4. `git status --short` → 只出现预期新增（`skills/**`、`scripts/check_skills.sh`、Makefile/README/AGENTS.md/版本日志），**不出现** `.claude/`、`.agents/`
5. README/AGENTS.md 新增的相对链接全部落地——用一次性 `python3` 脚本遍历 `](path)` 断言目标存在（与本次 P3 文档同步同一做法），**不**做成常驻门禁

**反向（门禁必须真的挡得住）**

6. 把某 skill 的 `name:` 改成与目录不一致 → `make check-skills` 非零退出且指到该 skill
7. 从 `SKILL.md` 删掉一个 reference 的引用 → 该 reference 成为孤儿 → 非零退出
8. 把某个 reference 链接指向不存在的文件 → 非零退出
9. 在 `.claude/skills/jimu` 处放一个真实目录后执行 `make skills-install` → 非零退出且原目录未被改动

## 9. 版本归属与发布影响

**建议进 v0.3.0**：改动面是 `skills/**` + `scripts/check_skills.sh` + Makefile 两个目标 + 三处文档，**无 Go 源码、无生成器、无 `go.mod`/`configs` 改动**，四道能力门禁与 `compose-report` 数字不受影响。v0.3.0 冻结的能力模型正是 skill 要教的内容，配套发布最自然。

发布前仍需按流程在 release tip 重跑 `make release-check`（Makefile 与脚本属非 md 改动，按纪律复跑一次），并把结果并入版本日志「验证」段的最终条目。

## 10. 风险与未决

| 风险 | 缓解 |
|---|---|
| skill 内容与 README/设计文档漂移（两处描述同一流程） | reference 不复制正文、只链接章节；数字只引用 `compose-report.md`；`check-skills` 管结构，内容漂移靠 review 与「改代码同步文档」的既有规范 |
| `skills-install` 覆盖第三方 skill 包 | 冲突即非零退出、不覆盖；只软链本仓 `skills/` 下的名字 |
| 不同 Agent 的发现路径不一致（Claude Code vs 其它） | 事实源中立（`skills/`），装到 `.claude/skills/` 与 `.agents/skills/` 两处；后续要支持别的工具只需在安装目标里加一行 |
| skill 描述命中率低（Agent 不知道何时用） | `description` 同时含中英文关键词；AGENTS.md 加索引行兜底 |
| 未决：未来是否让 `jimu new` 也把 skill 拷进生成项目 | 本轮明确非目标（生成器零改动）；若要做，走独立设计 + 生成器漂移门禁 |
