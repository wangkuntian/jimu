---
name: jimu
description: 在 jimu 框架仓内改动时使用：新增/删除能力、新增形态（profile）或驱动、编写能力内迁移与存量库 adopt、运行四道能力门禁与报告、排查运行时降级、使用 jimu new 脚手架生成项目。Use when adding capabilities, profiles, drivers, migrations, or running capability gates in the jimu repo.
---

# jimu 框架仓操作指南

本仓 v0.3.0 起按**能力**组织，能力在三个时机可组合：建项目时（`jimu new`）、编译时（形态 profile）、运行时（`capabilities.enabled`）。动手前先读本文件，再按「工作流索引」读对应 reference；reference 里链接的 README 章节是权威口径。

## 铁律（先看这段）

- **禁止自动提交**：没有用户明确指令，不得 `git commit` / `git push` / 创建任何 commit（[AGENTS.md](../../AGENTS.md) 里此条优先级最高）
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
| `internal/capabilities/<name>/` | 能力实现：静态 `Descriptor` + `Wire` 入口，按职责实现 `contract.Module` 与分层目录 |
| `internal/capabilities/catalog/` | 能力清单（唯一真源；18 项 catalog + 非 catalog 条目标 `Ungated`） |
| `internal/capability/` | 描述符解析叶子包（`Resolve` / `ValidateDeclarations` / `Degraded`） |
| `internal/assembly/` | 装配与生命周期（`Assembly` / `Capability` / `Run` / `ProbeAssembly`） |
| `internal/profiles/<name>/` | 形态清单（`full`/`minimal`/`saas`/`enterprise`/`machine`）+ `registry`（形态总表）+ `active`（选点包） |
| `internal/contract/` | 能力间端口与 `Descriptor` 定义 |
| `cmd/server/` | 唯一入口：只 import `internal/assembly` 与选点包 `internal/profiles/active` |
| `cmd/cli/` | `jimu` CLI（`migrate` / `seed` / `new` / `capability create/add` / `apikey`） |
| `tools/*` | 门禁与报告工具（`checkcapabilities` / `composereport` / `profileoverlay` / `profileassets` / `generator`） |
| `configs/*.yaml` | 内核段之外的能力配置段由各能力 `Descriptor.Configs` 声明并按启用集加载 |

## 工作流索引

| 我要做的事 | 读 |
|---|---|
| 新增 / 删除一个能力 | `references/capability.md` |
| 新增一个形态（profile）或给能力加驱动 | `references/profile-driver.md` |
| 编写 / 修改迁移、存量库 adopt | `references/migration.md` |
| 门禁红了、报告漂移了 | `references/gates.md` |
| 能力没生效、启动降级、形态起不来 | `references/runtime.md` |
| 用脚手架生成项目 / 追加能力 | `references/scaffold.md` |

## 最常用的验收命令

```bash
make check-capabilities     # 6 条汇总行：自描述 ↔ 迁移、驱动、入口/选点包、资产归属、跨能力 import
make profiles-check         # 5 形态 overlay 构建 + golden 依赖闭包
make compose-report-check   # 入库报告 == 本次实测（平台相关列掩码后比对）
make check-skills           # 本 skill 自身的契约校验
```

`make` 目标全表、含义与平台相关性说明见 README「[Makefile 命令](../../README.md#makefile-命令)」。
