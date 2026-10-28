# 跨仓协作工作流

> 本文定义 `jimu` 与 `jimu-web` 的跨仓协作方式。两个仓库保持独立版本化、独立 CI 和独立发布，不合并为 monorepo。

## 目标与边界

跨仓需求以一个可独立验收的纵向功能为单位，由同一个任务协调后端、契约和前端改动。任务可以同时读取两个仓库，但每个文件仍归属其所在仓库，不能把另一个仓库的文件复制过来替代真实改动。

- `jimu` 拥有 HTTP API、权限点、错误码、数据归属和后端行为。
- `jimu-web` 拥有页面、路由、交互、前端状态和 API 消费逻辑。
- `jimu/docs/openapi/swagger.yaml` 是 API 契约来源；`jimu-web/contracts/` 只保存固定后的副本和来源提交。
- `jimu-web/src/shared/api/generated.ts` 是生成文件，只供前端消费，不承载手写业务规则。
- 契约冻结是唯一的串行点；契约提交可获取后，后端业务实现和前端开发并行推进。

## 标准流程

1. **定义纵向需求与契约。** 先写明用户结果、两端范围、方法路径、请求响应、字段类型、分页、可空性、权限点、错误码和验收命令。契约未明确前，不并行猜前端字段或后端行为。
2. **形成契约提交。** 在 `jimu` 实现可编译的 DTO、路由或 Handler 签名、权限声明和 Swagger 注解；业务实现可以使用占位逻辑。运行 `make swagger` 和 `make swagger-check`，确保契约可从提交稳定生成。
3. **冻结并发布契约。** 经用户明确授权后提交并推送 `jimu`，记录 `contract-sha`。`jimu-web` 的契约 pin 必须指向可获取的提交，不能直接固定后端工作区中的未提交文件。
4. **前后端并行开发。** 后端继续实现业务、数据库和测试；前端同步 `contract-sha` 后实现页面、请求、状态和测试。后端接口尚未可用时，前端使用 mock 完成聚焦测试。
5. **处理契约变更。** 契约冻结后不得静默改字段或错误语义。确需变更时重新生成 Swagger、形成新的契约提交，并让前端运行 `scripts/sync-contract.sh <new-contract-sha>` 后重新验证。
6. **集成与独立交付。** 后端可用后运行真实链路 E2E，并把前端 pin 收口到最终后端提交。两端按各自分支、CI 和 PR 流程交付；完成标准是契约来源、后端行为和前端验证三者一致。

```bash
# 契约阶段：后端生成并校验 Swagger
cd jimu
make swagger
make swagger-check

# 经明确授权，提交并推送契约后取得 contract-sha

# 并行阶段：前端固定契约并开始实现
cd ../jimu-web
scripts/sync-contract.sh <contract-sha>
pnpm run check

# 并行阶段：后端继续实现业务、数据库和测试

# 集成阶段：契约未变时把 pin 收口到最终后端提交
scripts/sync-contract.sh <final-jimu-sha>
```

## 并行规则

- 契约阶段串行，业务实现并行；前端等待的是 `contract-sha`，不是后端开发完成。
- 只有契约已经冻结且提交可获取时，才拆出后端和前端并行任务。
- 并行任务必须共享同一个需求标识、接口路径、权限点、错误码和 `contract-sha`。
- 两端使用各自的分支、worktree 和 PR；除非用户明确要求，不创建跨仓提交，也不代替用户执行 commit 或 push。
- 任何契约变更都要显式通知前端并重新同步；不能用口头约定替代新的契约提交。

## Codex 工作区建议

- 长期使用一个同时关联 `jimu` 和 `jimu-web` 的 Codex Local project；契约阶段和集成阶段在联合任务中完成。
- 契约冻结后，为后端和前端各开一个并行任务，并在对应仓库使用独立 worktree；两个任务共享 issue、slug 和 `contract-sha`。
- 任务开始时先检查两端 `git status --short`，结束时再次检查，只汇报本次任务相关文件。
- 跨仓任务模板见本文末尾；复制后补齐占位符，不要只写“前后端一起改”。

## 验证矩阵

| 仓库 | 最低验证 | 按改动追加 |
|---|---|---|
| `jimu` | `make swagger-check`、相关 Go 测试 | 能力或形态改动运行 `make check-capabilities`、`make profiles-check` |
| `jimu-web` | `scripts/sync-contract.sh <sha>`、`pnpm run check` | 真实流程运行 `pnpm run test:e2e`；交互变更进行浏览器验收 |
| 跨仓 | pin 中的 SHA 与最终后端契约来源一致 | 确认契约变更已回同步，前端未手写修正生成类型或绕过契约 |

## 跨仓任务模板

```markdown
# 跨仓任务：<用户可感知的结果>

## 任务标识

- 需求/Issue：<链接或编号>
- 分支 slug：<小写短横线>
- 后端分支：<jimu 分支>
- 后端 base：<目标 release 或其他明确基线>
- 前端分支：<jimu-web 分支>
- 前端 base：<目标 release 或其他明确基线>

## 范围

- `jimu`：<API、权限、数据、测试、文档>
- `jimu-web`：<页面、路由、请求、状态、测试、文档>
- 不在范围：<明确排除项>

## 契约

- 方法与路径：<METHOD /api/v1/...>
- 请求：<字段和类型>
- 响应：<字段和类型>
- 分页、可空性和时间格式：<约定>
- 权限点：<permission>
- 错误码：<code 与语义>
- 契约提交：<contract-sha，契约冻结后补>
- 契约变更规则：<变更时重新生成 Swagger、提交并通知前端>

## 执行顺序

1. 检查两端工作区，确认无关改动。
2. 在 `jimu` 实现可编译的 DTO、路由或 Handler 签名、权限声明和 Swagger，形成契约提交。
3. 经明确授权后提交并推送契约，取得 `contract-sha`。
4. 并行推进：后端继续实现业务与测试；前端运行 `scripts/sync-contract.sh <contract-sha>` 并实现页面、交互和测试。
5. 契约发生变化时生成新的契约提交，并让前端重新同步。
6. 后端可用后运行真实链路 E2E，并把前端 pin 收口到最终后端提交。
7. 分别运行两端验收，汇报实际结果。

## 验收

- 后端：<具体命令>
- 前端：<具体命令>
- 跨仓：`contracts/backend-commit.txt` 等于最终后端契约来源 SHA。
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
