# ADR 0005：Solo maintainer 的合并规则

- 状态：已接受
- 日期：2026-11-05

## 背景

仓库贡献指南要求 PR 至少 1 人 review，但未来六个月的 Jimu 实际只有 1 名维护者。直接取消 PR 或放宽 master 保护会跳过 CI 门禁，风险不可接受；坚持第二人 review 又会让所有变更停在等待状态。

## 决策

保留 PR、分支保护和 required checks，不降低 CI 门禁。Solo maintainer 可以在所有 required checks 通过后自审并合并，但必须满足：

- 不直接 push master，不使用 force push。
- PR 描述记录本次改动、验证命令、未覆盖风险和回滚方式。
- 发布 tag 只在 release 候选 PR 合并后的 master tip 创建。
- 有第二 reviewer 可用时优先使用第二 reviewer；自审是例外，不是默认流程。

## 后果

- 单人维护不会被 review 流程卡死，但独立审查减少，风险记录成为合并证据的一部分。
- CI、Scaffold Gate 和发布检查仍是硬门槛；该 ADR 不改变任何门禁。
- 后续出现稳定协作者时，应恢复常规 review，并重新评估本 ADR。

## 备选方案

- **坚持第二人 review**：安全但会让六个月的变更长期阻塞，未采用。
- **放宽 master 保护或允许直推**：会绕过 CI 和发布门禁，未采用。
