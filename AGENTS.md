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

## 架构约束

### 模块结构

每个业务模块必须遵循 Clean Architecture 分层与 `contract.Module` 注册规范，详见 README「开发规范 · 模块结构」；HTTP 路由统一注册在 `/api/v1` 前缀下。

### 能力边界（v0.3.0 起）

后端按**能力**组织，可插拔为正式能力（设计与清单见 [docs/design/2026-09-18-capability-plugins-design.md](docs/design/2026-09-18-capability-plugins-design.md)）：

- 能力清单只维护在 `internal/capabilities/catalog`；P0 阶段新增/删除能力还需同步 `cmd/server/main.go` 的实例装配（P1 起改由 profile 入口包承担）
- 每个能力导出静态 `Descriptor`（名称 / 硬依赖 `Requires` / 挂载点 `Mount`），依赖必须单向
- 能力之间只经 `contract` 端口调用，禁止 import 其他能力的内部包
- 挂载点由 `Descriptor.Mount` 声明，禁止按能力名做特判
- Casbin RBAC 机制位于内核 `internal/kernel/access`（强制器/策略/权限中间件），API Key 签发/校验位于 `internal/capabilities/apikey`；`kernel/auth` 只保留 JWT/Session/限流/登录失败锁定机制

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
