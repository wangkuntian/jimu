# ADR 0007：Profile 是内部构建形态，模板才是产品

- 状态：已接受
- 日期：2026-11-05

> 实施状态：`v0.8.1` 已删除 `enterprise`、`machine`、`saas` 的 assembly、registry、golden 和测试；`full`/`minimal` 保留，`minimal` 冻结。

## 背景

Jimu 现有 `full`、`minimal`、`saas`、`enterprise`、`machine` 五个内部 profile。它们只改变服务二进制的编译期组成，不改变 `go.mod`，也不是 `jimu new` 的产物。发布物只有 `full` 和 `minimal`；`enterprise` 和 `machine` 没有实际使用场景，`saas` 也没有明确用户。`minimal` 与 `auth-api` 看起来相似，但前者是同一套框架代码的裁剪，后者是独立生成的契约，不能互相替代。

## 决策

- Profile 只作为 Jimu 自身服务二进制的内部构建形态，不是对外产品契约。
- 对外产品面只保留 template：`basic-api`、`auth-api`，以及未来经过验证的 full-stack template。
- 退役 `enterprise`、`machine`、`saas`；冻结 `minimal`；保留 `full` 作为开发、测试和 demo 的全量形态。
- 目标状态是 Release 只发 CLI + template bundle，server 二进制降级为内部产物。过渡期暂时保留 `full`/`minimal` 发布，8 周 gate 后再决定是否停发 server 二进制。
- 不新增 profile。未来的 full-stack 交付如果成立，以 template 形式实现，不以 profile 形式实现。

## 后果

- 退役会触及 `internal/profiles`、registry、路由 golden、`scripts/check_profiles.sh`、compose-report 和设计文档；必须作为独立代码变更验证。
- `minimal` 冻结后不再新增能力，只保持构建和闭包门禁通过。
- `full` 仍是 `go test ./...`、IDE、Swagger 和 demo 的默认形态。
- 发布矩阵在过渡期不变；停发 server 二进制需要单独的 Release Note 和发布流程验证。

## 备选方案

- **五个 profile 全部保留**：维护成本最高，且没有对应使用场景，未采用。
- **立即停发所有 server 二进制**：发布面变化过大，未采用；先退役零使用 profile，再按 gate 决定。
- **把 `saas` 提升为 full-stack demo 后端**：会把内部 profile 变成事实产品契约，且仍缺 mfa，未采用。
