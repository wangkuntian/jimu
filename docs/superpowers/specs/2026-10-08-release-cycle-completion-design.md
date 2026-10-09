# Release Issue 自动化补全设计

用户确认的目标：使用 Release Issue 模板启动版本周期；自动创建 release 与固定依赖汇总分支；立即及每日扫描依赖；所有相关 PR 回写 Issue；最终 release PR 合并后打 tag、发布 Release、更新并关闭 Issue。取消固定收集期与静默期。

## 扫描与合并

- 使用官方 Dependabot CLI v1.94.0，扫描 Go、GitHub Actions、Docker 三类依赖。
- Bootstrap 完成后立即扫描，每日上海时间 02:00（UTC 18:00）重试。只接受 OPEN、受管、未 published/blocked 的 collecting/candidate Issue；关闭后停止扫描。candidate 阶段继续接受新依赖批次。
- CLI 用只读 token 计算更新；输出交给独立的 App 写入步骤。更新 PR 的作者为 Jimu Release Automation App，head 使用 `dependabot-cli/<version>/<ecosystem>/<dependency-key>`，base 为 `dependabot-updates`。PR 含受管 marker，与 Issue 关联。
- 同一依赖集合复用 PR；最新分支提交通过 focused CI 与当前 head 校验后 squash 合入固定分支。历史托管 Dependabot PR 继续兼容。
- 固定分支有差异时创建汇总 PR；无依赖差异不创建空 PR。合入 release 后可以创建下一依赖批次汇总 PR，并复用同一个最终 release PR。
- 托管版本更新设 `open-pull-requests-limit: 0`，保留安全更新与 master 的 Scaffold Matrix 成功 skip。

## Issue 与发布

- `.github/ISSUE_TEMPLATE/release_cycle.md` 提供标题、版本、范围与关联 PR 内容。
- 只在自动化 marker 区域维护分支、各类 PR 及状态，不覆盖用户描述。PR 创建、同步、关闭与合并事件，以及每日对账均刷新该区域。
- 自动更新正文不再次 bootstrap 或扫描，避免 Issue edited 事件递归；人类编辑/重开仍可恢复。
- 最终 release PR 人工合并后沿用现有 tag 与 master tip 校验。成功发布事件更新 Issue 正文与 published 标签，再关闭 Issue；失败保留 blocked 状态和 Issue。

## 验证与边界

使用官方 CLI 的真实输入/输出 schema；本地 fixture 验证 PR 幂等、路径与作者边界、Issue 正文保留及关闭顺序；actionlint、Bash/Python 检查、发布脚本回归验证 workflow 串联。尚未部署的 GitHub 扫描、App 权限与构建发布不宣称已完成端到端验证。
