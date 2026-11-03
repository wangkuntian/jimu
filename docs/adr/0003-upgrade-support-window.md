# 将升级支持窗口限定为 v0.5.0 起的认证模板

Jimu 只对 `v0.5.0` 及之后、使用 manifest schema v2/v3 的 `basic-api` / `auth-api` 项目承诺 `jimu upgrade` 兼容。`v0.4.x` 和 schema v1 项目继续采用“重新生成 + 人工迁移”；`--with` 与 `capability add` 保持 experimental，不获得组合级升级承诺。安全修复只保证当前 0.7.x 的最新 patch，旧 patch 不 backport，用户先升级到最新 patch。

## 考虑过的方案

- **承诺所有历史版本和 experimental 组合**：对用户最友好，但需要维护 schema v1、旧模板结构和任意能力组合的兼容矩阵，成本与安全责任远超当前维护能力。
- **只支持相邻版本逐级升级**：兼容面更小，但现有 v0.5 fixture 已能覆盖直接升级，逐级升级反而增加用户操作和版本管理成本。
- **只发布最新版本、不提供升级检查**：实现最简单，但用户无法判断本地修改和模板变化，违反“代码归用户所有、升级可检查”的产品定位。

## 影响

- `docs/templates/v1/matrix.yaml`、`docs/UPGRADING.md` 和 `docs/SECURITY.md` 必须使用同一支持窗口。
- 安全修复的 Release Note 必须给出受影响版本、修复版本和人工迁移部分。
- v0.4.x 项目不能把 `jimu upgrade` 当作迁移工具；需要重新生成并人工 diff。
- experimental 项目可以继续使用 `check` / `upgrade --check` 做诊断，但不获得兼容或安全修复承诺。
