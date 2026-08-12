<!--
Sync Impact Report
- Version: 0.1.0 → 1.0.0（MAJOR：原则从"业务模块项目"视角重定义为"通用后端库"视角，向后不兼容）
- 修改的原则:
  - I. 模块化分层 → III. 可组合模块与标准适配器（重定义）
  - II. 简单优先（YAGNI）→ IV. 简单优先与最少依赖（扩展：依赖传播约束）
  - III. 验证先于完成 → V. 验证与质量（更名）
  - IV. 统一约定 → 折叠进"架构与实现约束"章节（降级）
  - V. 可观测性与安全 → 折叠进"架构与实现约束"章节（降级）
- 新增原则: I. 通用性（业务无关）、II. API 稳定性与语义化版本、VI. 文档与示例
- 新增章节: 无（沿用模板结构）
- 移除章节: 无
- 待办: 无
-->

# Jimu Constitution

## Core Principles

### I. 通用性（业务无关）

Jimu 是通用后端库，MUST 保持业务无关；业务规则 MUST 由使用方应用承载，库不得内置具体业务逻辑。

- 所有能力 MUST 默认可选，通过配置或特性开关启用，不得强制挂载。
- 库组件 MUST 不引用业务实体、业务错误码或业务表结构。
- 非目标 MUST 被维护：不做多租户，数据表不得预留 `tenant_id`、`tenant` 字段或租户中间件。

理由: 通用库被多项目复用；掺入业务假设会让复用方被迫继承无关约束。

### II. API 稳定性与语义化版本

公共 API MUST 向后兼容；破坏性变更 MUST 仅在 MAJOR 版本发布，并附迁移说明。

- 导出符号（函数/类型/接口/配置键）的变更 MUST 遵循 SemVer：新增可选能力为 MINOR，修复与澄清为 PATCH。
- 废弃 API MUST 至少保留一个 MINOR 版本周期并输出弃用警告后再移除。
- 配置与契约变更 MUST 提供升级路径，不得静默改变既有行为。

理由: 复用方依赖稳定 API；兼容性崩溃是库信誉的根本损失。

### III. 可组合模块与标准适配器

能力 MUST 按可组合模块提供，遵循统一分层
（`domain` → `application` → `infrastructure` → `interfaces`）并实现 `contract.Module`。

- 外部依赖（存储/队列/通知/缓存等）MUST 通过标准接口抽象，并提供可插拔适配器。
- 模块 MUST 可独立启用或替换实现，不影响其他模块。
- 业务逻辑 MUST 依赖接口而非具体实现。

理由: 可组合性让复用方按需装配；标准适配器让实现可替换而不改调用方。

### IV. 简单优先与最少依赖

每个改动 MUST 只解决当前问题，不引入未被要求的抽象、配置项或扩展点。

- 库的第三方依赖 MUST 最小化，优先 stdlib；选型 MUST 以可维护性为先，而非功能最全。
- 可选组件的依赖 MUST 不泄漏到基础模块，避免把编译期与运行期负担强加给所有复用方。
- 接口、工厂、泛化 MUST 等第二个真实使用者出现后再抽象。

理由: 库的依赖会传播给所有复用方；每行代码与每个依赖都是长期维护负担。

### V. 验证与质量

任何改动声称完成前 MUST 通过 fmt、vet、lint、build 与相关测试。

- 集成测试 MUST 使用真实 MySQL，经 `testutil.SkipUnlessMysql` 自动跳过（数据库不可达不得阻塞开发）。
- 发布 MUST 通过 `make release-check` 与 CI 门禁，含 govulncheck 依赖与 Trivy 镜像扫描。
- 无法运行验证时 MUST 说明原因与剩余风险，不得声称已完成。

理由: 库被多项目依赖，缺陷成本被放大；无证据即无完成。

### VI. 文档与示例

每个公共能力 MUST 有文档与可运行示例。

- 导出 API MUST 带 godoc 注释；HTTP 接口 MUST 带中文 swagger 注解。
- README MUST 与代码同步：新增配置项、CLI 命令、API MUST 更新对应章节。
- 每个适配器或特性 MUST 提供使用示例或测试用例作参考。

理由: 库的价值取决于能否被理解；无示例的 API 等于不可用。

## 架构与实现约束

- 模块注册: 所有模块 MUST 实现 `contract.Module`（`Name`/`RegisterHTTP`/`RegisterJobs`/`RegisterEvents`），HTTP 路由统一注册在 `/api/v1` 前缀下。
- 统一响应: HTTP 状态码恒为 200，业务错误通过 `body.code` 表达；错误码按模块区间分配（1xxx 通用、2xxx 用户/认证）。
- 数据模型: 基础表 MUST 含 `id`/`created_at`/`updated_at`/`deleted_at`；主键由应用生成雪花 ID，不使用 `AUTO_INCREMENT`。
- 迁移: 使用 Goose，命名 `{seq}_create_{table}s.sql`，字段与表 MUST 带中文 COMMENT。
- 配置: 枚举字段（`http.mode`/`log.level`/`log.format`）MUST 启动校验非法值；敏感值支持 `_FILE` 文件注入；`APP_ENV` 切换配置文件。
- 可观测性: 结构化日志（Zap）+ OpenTelemetry 追踪 + Prometheus 指标作为可选能力提供标准接入点，不得强制复用方接入。
- 设计边界（非目标）: 租户隔离明确不做；新增表、接口、中间件前 MUST 对照非目标清单。

## 开发流程与质量门禁

- 分支模型: 采用 GitHub Flow，`master` 为唯一长期分支（保持可部署、CI 通过、禁止直接 push）；`release/x.y.z` 为集成与发布分支。
- 合并: 非 Dependabot 改动 MUST 经 PR 且至少 1 人 review；feature/fix/hotfix → release 用 squash merge，release → master 用 merge commit；合并后删除源分支。
- 发布: 发布当日合并后打 `vX.Y.Z` tag；tag 需 `make release-check` 通过；回滚用 revert commit 或 hotfix PR，master 不接受 force push。
- 提交信息: MUST 使用 Conventional Commits（`type(scope): summary`），summary 用英文、小写开头、祈使句、不加句号。
- 门禁: CI MUST 通过（`make release-check`）；golangci-lint 与 pre-commit hooks（fmt/vet/lint）为硬性门禁。

## Governance

宪法是本项目的最高协作规范，覆盖 CLAUDE.md 与各模块内部约定；冲突时以宪法为准。

- 修订流程: 任何原则新增、删除或修改 MUST 通过 PR 提交，附文档说明与迁移计划，经 review 通过后合并。
- 版本策略: 采用 SemVer。MAJOR — 向后不兼容的原则移除或重定义；MINOR — 新增原则/章节或实质性扩展；PATCH — 措辞澄清、笔误、非语义修正。每次修订 MUST 递增版本并更新 `Last Amended` 日期。
- 合规评审: PR review MUST 核对改动是否符合宪法（业务无关性、API 兼容、可组合性、依赖与门禁）；新增的复杂度或抽象 MUST 能引用宪法依据，否则应拒绝。

**Version**: 1.0.0 | **Ratified**: 2026-08-12 | **Last Amended**: 2026-08-12
