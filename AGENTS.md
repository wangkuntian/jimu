# AGENTS.md

本项目的 AI agent 协作规范。面向人类贡献者的开发流程（分支/PR/发布/集成测试手册/Release Note 模板）见 [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md)，项目结构与配置见 [README.md](README.md)。

`CLAUDE.md` 是指向本文件的软链接（Claude Code 经它读取同一份规范）。更新规范只改本文件；请勿把 `CLAUDE.md` 改回实体文件或删除重建，以免软链失效。

默认使用中文回复用户，除非用户明确要求使用其他语言。

## 禁止自动提交

**严禁在未经用户明确指令的情况下执行 `git commit`、`git add` + `git commit`、`git push` 或任何创建 commit 的操作。** 包括但不限于：完成一个改动后自动提交、连续多个改动时自动分批提交、任何形式的"顺手提交"。

只有用户明确说"提交"、"commit"、"推送"等指令时才可以执行。此规则优先级最高，覆盖其他所有规范。

## 开发前必读

- 项目结构、技术栈、API 等见 [README.md](README.md)
- 修改代码后，必须同步更新 README.md 相关章节
- 创建分支、开 PR、发版遵循 [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) 的命名与流程约定（分支名 `feature/<issue>-<slug>` 等小写短横线格式）
- 本仓有可安装的 Agent skill：事实源 `skills/jimu/`，`make skills-install` 装到 `.claude/skills/` 与 `.agents/skills/`；改动前先读入口 `skills/jimu/SKILL.md`（铁律 + 六条工作流索引）

## 架构约束

### 模块结构

每个能力根包必须导出静态 `Descriptor` 与 `Wire(*assembly.Context) (contract.Module, error)`；按职责设置 Clean Architecture 分层目录，不强制四层或 `module.go`。仅提供端口或迁移时 `Wire` 可返回 nil；HTTP 路由统一注册在 `/api/v1` 前缀下。详见 README「开发规范 · 模块结构」。

### 能力边界（v0.3.0 起）

后端按**能力**组织，可插拔为正式能力（设计与清单见 [docs/design/2026-09-18-capability-plugins-design.md](docs/design/2026-09-18-capability-plugins-design.md)）：

- 能力清单只维护在 `internal/capabilities/catalog`；新增/删除能力需同步 `catalog` 与需要它的形态清单 `internal/profiles/<name>/assembly.go`（非 catalog 条目标 `Ungated`）。唯一入口 `cmd/server` 只调 `assembly.Run(active.Assembly())`，当前形态由选点包 `internal/profiles/active` 决定，**不在这里逐个装配能力**
- 每个能力导出静态 `Descriptor`（名称 / 硬依赖 `Requires` / 软依赖 `SoftRequires` / 自有表 `Owns` / 配置段 `Configs` / 权限点 `Permissions` / 挂载点 `Mount` / 迁移 `Migrations` / 驱动 `Drivers` / 资产 `Assets`），依赖必须单向
- 能力之间只经 `contract` 端口调用；能力内部代码（含测试与驱动）禁止跨能力 import，只有 `catalog` 作为能力树内的组合根例外
- 挂载点由 `Descriptor.Mount` 声明，禁止按能力名做特判
- Casbin RBAC 机制位于内核 `internal/kernel/access`（强制器/策略/权限中间件），API Key 签发/校验位于 `internal/capabilities/apikey`；`kernel/auth` 只保留 JWT/Session/限流/登录失败锁定机制与 API Key 上下文助手（`apikey_context.go`）
- 原 `auth` 已拆为 `auth`（会话/凭证/登录历史/密码历史）、`mfa`（TOTP + 可信设备，含自有 `totp/` 实现与 `user_mfa` 表）、`passkey`（WebAuthn）；`breach`/`captcha` 为独立能力；开通式注册（provisioned registration）属 `tenant` 能力。auth 经 `contract.MFAVerifier`/`TenantProvisioner`/`BreachChecker`/`CaptchaVerifier` 消费它们，passkey 经 `contract.LoginFinalizer` 复用 auth 的登录收尾；TOTP 状态存 `user_mfa`（迁移 016），`users` 表不再有 `totp_*` 列
- 原 `admin` 已拆散（P1.7）：`role` + `permission` 合并为 `access`（roles/permissions/role_permissions/user_roles 四表 + 用户角色分配）；管理端 `/api/v1/admin/*` 路由按用例归还各能力（用户→`user`、任务与调度→`queue`、API Key→`apikey`、用户导入→`dataops`、审计列表→`audit`、文件上传→`uploadsec`），平台级视图与**管理端准入中间件**归新能力 `console`；`/api/v1/admin/*` 通配权限点由 `console` 声明。`user` 管理面与自助面共用同一 repository/配额，角色分配经 `contract.UserRoleAssigner` 委托 `access`
- **形态（profile）** — `full`/`minimal`/`saas`/`enterprise`/`machine` 五个；形态名与清单的唯一来源是 `internal/profiles/registry`（它 import 全部形态包，**不得**被 `cmd/server` 或选点包 import）。形态只裁剪编进二进制的包与符号（`go.mod` 不变），真正减小依赖的是层① `jimu new` 生成的独立项目
- **驱动级可插拔** — 第三方驱动独立成包，能力在 `Descriptor.Drivers` 声明**可用集**、形态在 `assembly.Capability.Drivers` 声明**选中集**并在 `internal/profiles/<name>/drivers.go` blank import；核心包只留接口 + 注册表，未注册即 fail-closed
- **非代码资产** — `Descriptor.Assets` 声明能力拥有的资产（仓库相对路径），内核运维/观测资产用具名资产组 `ops`/`observability`；归属按最长前缀匹配
- **门禁** — 改能力/形态/驱动/资产后跑 `make check-capabilities`（能力声明、驱动、形态装配、资产归属与生成器边界）与 `make profiles-check`（golden 依赖闭包），报告漂移用 `make compose-report-check`；三者已进 `make ci`/`release-check` 与 CI 的 `Capability Gates` job。新增能力/形态/驱动的完整步骤见 README「[开发规范 › 新增能力 / 驱动](README.md#新增能力--驱动)」，生成独立项目的流程见 README「[生成项目](README.md#生成项目jimu-new--jimu-capability-add)」

### 租户体系

多租户（v0.2.0 起）为正式能力，以下不变量修改时不得偏离（完整设计见 README「租户体系 / 开通式注册」）：

- **单归属** — 用户/角色/审计日志归属唯一租户（`tenant_id` 列）；角色名租户内唯一，用户名/邮箱全局唯一。
- **上下文来源** — 租户身份只来自 JWT claim（`tid`），经中间件注入 request context，业务层从 `internal/kernel/tenant.FromContext(ctx)` 读取；**禁止**从 header/query 接受租户标识（旧实现因此被废弃）。
- **默认租户** — `id=1`、`code=default`（`internal/kernel/tenant.DefaultTenantID`），不可删除；上下文无租户（tid=0）时创建资源归默认租户、查询不过滤（平台级视角）。
- **编码** — 校验与归一化（统一小写）只在 `internal/kernel/tenant`（`ValidCode`/`NormalizeCode`）；编码不可变。

## 编码约束

错误码分段、配置校验、数据库迁移规范、日志调用规范等开发约束统一维护在 README「[开发规范](README.md#开发规范)」章节，修改相关代码前先阅读并遵守。

## 文档维护

修改代码后，必须同步更新 README.md 相关章节（配置表 / CLI 命令 / API 示例 / 目录树 / Makefile 速查），新增 API 使用中文 swagger 注解。

版本日志按 release 版本号记录在 `docs/releases/<version>.md`，该文件同时作为 GitHub Release body（`release.yml` 读取）。改动源码的 PR 必须同步更新对应版本文件（CI 强制检查）；模板与规则见 [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md)。

## 简单优先

只写当前任务需要的最少代码。

- 不添加未被要求的功能、配置项、扩展点或抽象。
- 不为一次性代码创建接口、工厂或通用框架。
- 能删除本次改动造成的无用代码时直接删除。
- 不重构无关代码，不顺手改格式、命名或注释。

如果实现规模明显超过需求，先停下来简化方案。

## 保护工作区

工作区可能包含用户未提交改动。

- 修改前查看 `git status --short`。
- 不覆盖、回滚或格式化无关文件。
- 不运行 `git reset hard`、`git checkout -- .`、批量删除等破坏性命令，除非用户明确要求。
- 提交前再次查看 `git status --short`，只纳入当前任务相关文件。
- 除非用户明确要求，否则不要创建 commit。

如果无关本地改动影响当前任务，说明冲突并尽量绕开。

## Commit Message

使用 Conventional Commits 轻量格式 `type(scope): summary`；`type` 取 `feat` / `fix` / `docs` / `test` / `refactor` / `chore`。summary 与正文全部使用英文：小写开头、祈使句、不加句号；一个 commit 只包含一个清晰主题。由 `githooks/commit-msg` 强制检查（CI 兜底）。完整规则与示例见 [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md)。

## 回复格式

长任务中给出简洁进度，说明正在收集什么上下文、准备改什么、验证结果是什么。

最终回复包含：

- 改了什么。
- 跑了什么验证。
- 还有什么风险或未做事项。

## codebase-memory-mcp

本项目使用 codebase-memory-mcp（tree-sitter 知识图谱 MCP）做代码库结构分析，已替代原
graphify。索引存全局 `~/.cache/codebase-memory-mcp/`，仓库内不落文件（`.codebase-memory/`
已 gitignore）。

规则：

- 遇到代码库结构问题（调用链/影响面/架构），优先用 MCP 工具查询，而不是逐文件
  grep/read：`trace_path`（调用链）、`search_graph`（结构/语义搜索）、
  `get_architecture`（架构总览）、`detect_changes`（未提交改动的符号影响映射）、
  `query_graph`（Cypher 只读查询）、`get_code_snippet`（按限定名取源码）。
- 图谱未索引时先建索引：`codebase-memory-mcp cli index_repository --repo-path .`
  （或在 agent 会话里说 "index this project"）；之后后台 watcher 自动跟随 git 变更。
- 修改代码后无需手动刷新索引（watcher 自动同步）。
