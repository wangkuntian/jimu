# 跨仓协作工作流

> 本文定义 `jimu` 与 `jimu-web` 的跨仓协作方式。两个仓库保持独立版本化、独立 CI 和独立发布，不合并为 monorepo。

## 目标与边界

跨仓需求以一个可独立验收的纵向功能为单位，由同一个任务协调后端、契约和前端改动。任务可以同时读取两个仓库，但每个文件仍归属其所在仓库，不能把另一个仓库的文件复制过来替代真实改动。

- `jimu` 拥有 HTTP API、权限点、错误码、数据归属和后端行为。
- `jimu-web` 拥有页面、路由、交互、前端状态和 API 消费逻辑。
- `jimu/docs/openapi/swagger.yaml` 是 API 契约来源；`jimu-web/contracts/` 只保存固定后的副本和来源提交。
- `jimu-web/src/shared/api/generated.ts` 是生成文件，只供前端消费，不承载手写业务规则。

## 标准流程

1. **定义纵向需求。** 先写明用户结果、两端改动范围、API 契约和验收命令。契约未明确前，不并行猜前端字段或后端行为。
2. **后端先完成契约。** 在 `jimu` 实现 API、权限点和测试，并运行 `make swagger` 更新 OpenAPI。提交前运行 `make swagger-check` 和该改动相关的 Go 检查。
3. **固定后端提交。** 经用户明确授权后提交并推送 `jimu`，记录提交 SHA。`jimu-web` 的契约 pin 必须指向可获取的提交，不能直接固定后端工作区中的未提交文件。
4. **前端同步契约。** 在 `jimu-web` 运行 `scripts/sync-contract.sh <sha>`；脚本会更新固定契约、`backend-commit.txt` 并重新生成 TypeScript 类型。
5. **前端实现与验证。** 前端只依据生成的类型和契约实现页面、请求和交互，运行 `pnpm run check`；涉及真实流程时再运行 `pnpm run test:e2e`。
6. **独立交付。** 两端按各自分支和 CI 流程提交、评审与发布。跨仓任务完成的标准是契约 pin、后端行为和前端验证三者一致，而不是两端必须在同一个 commit 中。

```bash
# 后端：生成并校验契约
cd jimu
make swagger
make swagger-check

# 经明确授权，提交并推送后端后取得 SHA

# 前端：固定契约并验证
cd ../jimu-web
scripts/sync-contract.sh <jimu-commit-sha>
pnpm run check
```

## 并行规则

- 默认串行：先冻结契约，再实现前端。
- 只有契约已经明确且后端提交可获取时，才允许后端收尾和前端实现并行。
- 并行任务必须共享同一个需求标识、接口路径、权限点、错误码和后端提交 SHA。
- 两端使用各自的分支与 PR；除非用户明确要求，不创建跨仓提交，也不代替用户执行 commit 或 push。

## Codex 工作区建议

- 长期使用一个同时关联 `jimu` 和 `jimu-web` 的 Codex Local project，让单个任务拥有完整上下文。
- 工作区有未提交改动、或需要并行多个任务时，为对应仓库单独使用 worktree；不要覆盖其他任务的工作区。
- 任务开始时先检查两端 `git status --short`，结束时再次检查，只汇报本次任务相关文件。
- 跨仓任务模板见本文末尾；复制后补齐占位符，不要只写“前后端一起改”。

## 验证矩阵

| 仓库 | 最低验证 | 按改动追加 |
|---|---|---|
| `jimu` | `make swagger-check`、相关 Go 测试 | 能力或形态改动运行 `make check-capabilities`、`make profiles-check` |
| `jimu-web` | `scripts/sync-contract.sh <sha>`、`pnpm run check` | 真实流程运行 `pnpm run test:e2e`；交互变更进行浏览器验收 |
| 跨仓 | pin 中的 SHA 与后端改动提交一致 | 确认前端未手写修正生成类型或绕过契约 |

## 跨仓任务模板

```markdown
# 跨仓任务：<用户可感知的结果>

## 任务标识

- 需求/Issue：<链接或编号>
- 分支 slug：<小写短横线>
- 后端分支：<jimu 分支>
- 前端分支：<jimu-web 分支>

## 范围

- `jimu`：<API、权限、数据、测试、文档>
- `jimu-web`：<页面、路由、请求、状态、测试、文档>
- 不在范围：<明确排除项>

## 契约

- 方法与路径：<METHOD /api/v1/...>
- 请求：<字段和类型>
- 响应：<字段和类型>
- 权限点：<permission>
- 错误码：<code 与语义>
- 契约来源：<jimu commit SHA，完成后补>

## 执行顺序

1. 检查两端工作区，确认无关改动。
2. 在 `jimu` 实现 API、测试和 Swagger。
3. 经明确授权后提交并推送后端，取得 SHA。
4. 在 `jimu-web` 运行 `scripts/sync-contract.sh <SHA>`。
5. 按生成类型实现前端并补齐测试。
6. 分别运行两端验收，汇报实际结果。

## 验收

- 后端：<具体命令>
- 前端：<具体命令>
- 跨仓：`contracts/backend-commit.txt` 等于后端改动提交 SHA。
- 手工验收：<页面、账号、前置权限、预期结果>

## 约束

- 未经用户明确指令，不执行 commit 或 push。
- 不覆盖、回滚或格式化本次任务之外的改动。
- 前端不复制后端领域规则，不手写修改 `src/shared/api/generated.ts`。
- 后端契约未冻结时，不凭猜测实现前端。

## 交付说明

- 分别列出两个仓库改了什么、跑了什么验证、还剩什么风险。
- 未完成项必须说明阻塞原因，不能用“已完成”代替未验证结果。
```

## 相关入口

- 贡献与发布流程：[docs/CONTRIBUTING.md](../CONTRIBUTING.md)
- 前端仓库：[jimu-web](https://github.com/wangkuntian/jimu-web)
- 契约同步脚本：`jimu-web/scripts/sync-contract.sh`
- 契约 pin 校验：`jimu-web/scripts/check_contract_pin.sh`
