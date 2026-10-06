# v0.3.4 文档收尾实施计划

> **执行方式：** 在当前会话 inline 按 task 顺序执行；每项完成后运行该项验证、勾选并单独提交。

**目标：** 清理旧规格目录、统一计划文档位置，并以当前代码事实重建稳定的项目设计文档和 README 入口。

**架构：** `docs/design` 维护不带日期的稳定设计事实，能力与形态清单仍以代码为准；README 保留使用说明和操作流程，详细架构内容链接到设计文档。旧计划只迁移位置并修复路径，不重写其历史结论。

**技术栈：** Markdown、Git 路径迁移、仓库源码与配置。

## 全局约束

- 删除仓库根目录 `.specify/` 与 `specs/`；保留 `docs/superpowers/specs/`。
- 迁移 `docs/plans/*.md` 到 `docs/superpowers/plans/`，保留文件名与内容，仅修复因位置变化失效的路径。
- 新设计文档按稳定架构主题命名，不在文件名中加入日期，不写死能力数量、Ungated 数量或门禁条数。
- 设计描述以当前分支源码为准；README/AGENTS 中的链接必须指向新文档。
- 每个 task 单独提交，提交信息遵循 Conventional Commits 英文格式。

---

### Task 1：删除旧规格工具与根规格目录

**文件：**
- 新增：`docs/superpowers/plans/2026-10-07-v034-documentation-cleanup.md`
- 删除：仓库根 `.specify/` 与 `specs/`
- 保留：`docs/superpowers/specs/`

- [x] 复核待删路径只包含根级旧规格工具和规格文件。
- [x] 删除这两个目录，确认 Superpowers 规格目录仍存在。
- [x] 运行 `git diff --check` 并检查工作区改动范围。
- [x] 提交：`chore(docs): remove legacy specification files`

### Task 2：迁移历史计划并修复路径

**文件：**
- 移动：`docs/plans/*.md` 到 `docs/superpowers/plans/`
- 修改：仓库内指向 `docs/plans` 或受迁移影响相对路径的文档与注释

- [x] 移动计划文件，保留文件名和历史内容。
- [x] 修复迁移后失效的 Markdown 链接、README/AGENTS/Dockerfile 注释及其他显式路径引用。
- [x] 检查 `docs/plans` 不再存在、计划文件数量不变、旧路径引用无残留。
- [x] 运行本地 Markdown 链接目标检查与 `git diff --check`。
- [x] 提交：`docs(plans): move plans under superpowers`

### Task 3：重建设计文档并精简 README

**文件：**
- 替换并重命名：`2026-09-18-capability-plugins-design.md` → `capability-architecture.md`
- 替换并重命名：`2026-09-29-agent-skills-design.md` → `agent-workflows.md`
- 新增：`docs/design/README.md` 作为设计文档索引与维护约定
- 新增：`docs/design/identity-and-tenancy.md`
- 新增：`docs/design/configuration-and-data-lifecycle.md`
- 新增：`docs/design/profiles-and-project-generation.md`
- 修改：`README.md`、`AGENTS.md`、`docs/releases/v0.3.4.md` 及相关设计链接

- [x] 按当前实现完善五篇稳定详细设计文档，统一使用“背景与范围、设计目标与非目标、设计方案、关键决策、约束与不变量、实现映射”体例。
- [x] 补充组件关系、核心流程、决策取舍、约束/失败处理和验证映射；在 Markdown 中直接维护关键流程图。
- [x] 新增设计文档索引，说明五篇的范围、关系与更新职责；将 `agent-workflows` 标明为工程协作设计。
- [x] 说明 `docs/design` 的维护规则：行为或边界变化时同步更新对应设计文档；具体清单和数量从代码生成或查阅，不在设计文档复制易漂移的计数。
- [x] README 保留快速使用、CLI 参数、配置入口和操作步骤；把详细的能力生命周期、身份边界、配置/数据归属与 profile/generator 内部机制改为链接。
- [x] 更新 AGENTS 与仍有效的历史发布文档中的设计文档链接；不重写冻结的历史结论。
- [x] 更新 v0.3.4 发布记录，说明设计文档与 README 整理。
- [x] 检查新设计文档和活跃文档中的链接，修复指向已删除设计文件或 README 旧锚点的链接；历史计划中的旧路径作为当时记录保留；设计文档不复制易漂移的固定计数。
- [x] 运行 README/设计文档本地链接目标与锚点检查、`git diff --check`。
- [x] 提交：`docs(architecture): document current project design`
