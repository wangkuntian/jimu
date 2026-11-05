# ADR 0006：Full-stack demo 的边界

- 状态：已接受
- 日期：2026-11-05

## 背景

路线 B 的进入条件尚未满足：没有真实需求、没有设计伙伴，也没有 generated full-stack 模板。当前 `jimu-web` 按 Jimu 全量 server 的契约开发；`auth-api` 是独立生成的代码库，DTO、分页、tenant ref、`/auth/me` 和 logout 语义都不兼容，直接接入需要数周契约适配。

## 决策

`v0.9.0` 只做一个 8 周实验性 full-stack demo，使用 `full` profile + 当前 `jimu-web`。该 demo 证明的是“Jimu 全量 server + Hub 的全栈体验”，不是“`jimu new` 生成的全栈项目”。生成器路径另做 CLI-only smoke：`jimu new --template=auth-api`、`make up`、`make verify`，不接前端。

8 周后按 gate 决定是否继续：demo 在干净机器可运行、至少 3 个外部试用或明确反馈、至少 1 个团队愿意进入下一步。没有这些证据就停止路线 B，不发布 certified full-stack 模板，也不把 demo 写成生产采用证据。

## 后果

- demo 依赖 `full` profile，体积和能力都大于 generated project；它是演示环境，不是产品交付物。
- generated backend 接入 `jimu-web` 的契约适配被推迟，只有 gate 通过后才进入下一阶段。
- Release Note、README 和需求文档必须区分 full-stack demo、generated project 和 certified template。
- 路线 B 可以在 gate 处停止，不产生必须长期维护的产品承诺。

## 备选方案

- **`auth-api` + 兼容层**：能证明生成器，但至少需要数周契约适配，会挤掉 demo 打磨和外部验证，未采用。
- **新建 fullstack certified 模板**：产品方向更大，单人六个月内无法可靠交付，未采用。
- **不做 demo**：没有可验证的全栈体验证据，未采用。
