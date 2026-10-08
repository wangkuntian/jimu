# Release Issue 自动化补全实施计划

**目标：** 补齐已确认的五项发布流程，直接在 `release/v0.3.5` 工作，每项完成后独立英文 commit。

**设计：** [自动化补全设计](../specs/2026-10-08-release-cycle-completion-design.md)。沿用单个 Release Dependency Automation workflow，扫描与写入隔离 token，脚本承担可测试的数据处理。

## 任务与验收

- [x] 修复托管 Dependabot 作者匹配。提交 `7b3ef94`；merge contract 与 actionlint 通过。
- [x] 添加 Release Issue 模板。提交 `7182e3a`；模板校验通过。
- [x] 接入官方 Dependabot CLI 扫描与 App PR 创建。
  - 三类 JSON job 配置供 CLI 读取（JSON 是合法 YAML），运行前注入 repo 与固定分支 commit。
  - Python 脚本生成输入和消费 JSONL 事件；REST Git API 创建 blob/tree/commit/ref/PR，验证文件路径、base SHA、作者和受管 marker，复用已存在 PR。
  - fixture 测试先复现新功能缺失，再验证创建、幂等、更新与错误边界。
  - workflow Issue/bootstrap 与每日 UTC 18:00 驱动，读 token 扫描，App token 写入。candidate 不停止扫描。
  - 修改 focused CI 与合并 author/marker 识别、汇总多批次 PR、关闭 Issue 的 gate；运行 fixture、既有发布脚本与 actionlint，提交。
- [x] 将依赖、feature、汇总与 release PR 回写 Issue 正文。
  - 独立 Python 命令查询受管 Issue 与相关 PR，仅替换 marker 管理区；保留描述，幂等重试。
  - 同仓库 PR 事件与每日对账更新正文；忽略 App 自身 edited 事件以免递归。
  - fixture 覆盖分类、状态、去重、用户内容保留、错误版本/Issue 与关闭条件，提交。
- [x] 发布成功后更新 Issue 并关闭。
  - published 事件确认成功后同步 Release 链接与 published 标签，再关闭 Issue；失败不关闭。
  - 回归验证成功/失败与重试顺序，提交。
- [x] 同步 README、贡献指南、v0.3.5 说明、旧设计与计划；发布回归（48 个 CLI、20 个 Issue 测试和 Bash 契约）、actionlint、能力与形态门禁通过；只读审查指出的问题已修复。

提交记录：`48e323b`（官方 CLI 与 PR 发布）、`f99b6db`（Issue 同步与关闭）、`1315c12`（workflow 串联及文档）、`abd0173`（published 标签初始化）、`c57ba04`（基线推进和连续 API 中断恢复）。

## 验证边界

真实 GitHub 只读验证已识别 Issue #73 和固定分支 SHA，prepare 生成输入成功；未进行远端写入。Docker daemon 未运行，真实 CLI 扫描未验证。GitHub App 需要额外开启 `workflows: write`；修改尚未推送或合入 master。Issue 正文在写入前重读并保留人工区，GitHub REST 无 CAS，因此读取与写入之间仍有很小的并发窗口。

## 执行约束

无需 push、开 PR、合并或发布。保留最终 release PR 的人工审阅与合并。只纳入此任务文件；不得暴露 App 私钥/token。用户已选择自托管官方 CLI 并接受 PR 作者改为 Jimu GitHub App。
