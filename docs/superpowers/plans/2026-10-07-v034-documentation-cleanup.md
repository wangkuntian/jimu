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
- [ ] 提交：`chore(docs): remove legacy specification files`

### Task 2：迁移历史计划并修复路径

**文件：**
- 移动：`docs/plans/*.md` 到 `docs/superpowers/plans/`
- 修改：仓库内指向 `docs/plans` 或受迁移影响相对路径的文档与注释

- [ ] 移动计划文件，保留文件名和历史内容。
- [ ] 修复迁移后失效的 Markdown 链接、README/AGENTS/Dockerfile 注释及其他显式路径引用。
- [ ] 检查 `docs/plans` 不再存在、计划文件数量不变、旧路径引用无残留。
- [ ] 运行本地 Markdown 链接目标检查与 `git diff --check`。
- [ ] 提交：`docs(plans): move plans under superpowers`

### Task 3：重建设计文档并精简 README

**文件：**
- 删除：`docs/design/` 中现有带日期设计文档
- 新增：`docs/design/capability-architecture.md`
- 新增：`docs/design/identity-and-tenancy.md`
- 新增：`docs/design/configuration-and-data-lifecycle.md`
- 新增：`docs/design/profiles-and-project-generation.md`
- 修改：`README.md`、`AGENTS.md` 及相关设计链接

- [ ] 按当前 `Descriptor`、`catalog`、`contract`、`assembly`、能力配置/表所有权、profile 和 generator 实现写四篇设计事实文档。
- [ ] 说明 `docs/design` 的维护规则：行为或边界变化时同步更新对应设计文档；具体清单和数量从代码生成或查阅，不在设计文档复制易漂移的计数。
- [ ] README 保留快速使用、CLI 参数、配置入口和操作步骤；把详细的能力生命周期、身份边界、配置/数据归属与 profile/generator 内部机制改为链接。
- [ ] 更新 AGENTS 与仍有效的历史发布文档中的设计文档链接；不重写冻结的历史结论。
- [ ] 检查新文档链接、旧日期设计路径与易漂移固定计数残留，运行 `git diff --check`。
- [ ] 提交：`docs(architecture): document current project design`
