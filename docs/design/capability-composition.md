# 能力组合设计

## 背景与范围

Jimu 的脚手架只从固定 catalog 选择能力，不接入第三方能力或插件源。组合阶段要把用户请求的任意子集解析成一份自洽的装配事实，并在写入目标目录前拒绝未知能力、缺失依赖、冲突、端口、驱动、迁移、运行时服务和中间件问题。

本文定义固定 catalog 子集的组合契约、解析顺序、诊断、状态和持久化规则。具体能力清单、驱动、迁移路径和错误文案由代码与门禁派生，不在本文复制完整清单。

非目标：运行时动态加载能力、跨框架版本的插件协议、固定 catalog 的全幂集兼容承诺，以及生成项目的存量数据库迁移。

## 设计目标与非目标

设计目标：

- 任意请求子集都得到确定性结果，重复解析不改变顺序、驱动或状态。
- 同一套组合校验（resolver + preflight）服务 `jimu new`、`jimu check`、`jimu upgrade` 和 `jimu capability add`。
- 非法组合在创建暂存目录或写入目标之前失败，并给出可定位、可修复的诊断。
- 结构合法性与官方支持级别分开表达，不用“能生成”暗示“已认证”。

非目标：让所有合法子集都获得官方 E2E、预算或升级承诺；把组合器做成运行时控制面。

## 设计方案

### 契约声明

能力根包在 [`contract.Descriptor`](../../internal/contract/capability.go) 中声明组合所需事实：

| 字段 | 组合含义 |
|---|---|
| `Requires` | 硬依赖；解析时补齐闭包。 |
| `Conflicts` | 不能同时启用的能力名；声明本身必须非空、不自引用、不重复且目标已知。 |
| `Provides` / `Consumes` | contract 端口的提供与消费关系；`Consumes.Required` 表示缺失提供方即非法。 |
| `Drivers` / `DriverSpecs` | 能力支持的驱动，以及选中驱动后额外需要的运行时服务。 |
| `MigrationSchemaDeps` | 本能力迁移依赖的其他能力 schema。 |
| `Owns` | 本能力拥有的表；每张表只能有一个所有者。 |
| `RuntimeServices` | 启动所需外部服务；生成器据此推导配置、Compose 与诊断。 |
| `ProtectedHTTPMiddleware` | 受保护路由链的 primary/fallback 提供方式。 |

声明级不变量由 `make check-capabilities` 校验：未知/自引用/重复冲突、全仓重复提供方、每个形态内必需消费者缺失，以及非法驱动声明。组合级规则由 resolver 和生成前 preflight 执行。

### 状态模型

组合状态使用两个正交维度，定义在 [`internal/contract/composition.go`](../../internal/contract/composition.go)：

| 维度 | 取值 | 含义 |
|---|---|---|
| `validity` | `valid` / `invalid` | 是否通过静态组合规则；只表示结构合法性。 |
| `support` | `certified` / `validated` / `experimental` | 官方支持程度；不因组合合法自动获得。 |

支持级别的当前含义：

| 级别 | 适用范围 | 承诺 |
|---|---|---|
| `certified` | `basic-api`、`auth-api` 两个官方模板预设 | 固定契约、硬预算、端到端验收和发布阻断门禁。 |
| `validated` | 固定 catalog 中通过完整发布门禁的子集（保留级别） | 用于未来推广；v0.6.0 没有选择器产出该级别，pairwise 构建矩阵不自动提升支持级别。 |
| `experimental` | 所有 `--with` 生成项目与 `capability add` 项目 | 允许生成/追加并通过组合预检，但不承诺组合稳定性、预算、E2E 或升级兼容。 |

非法组合仍可携带支持级别用于报告；合法组合也不一定是 `certified`。内部 profile 只服务 Jimu 自身构建，其 manifest 的 `selection.support` 保持 `internal`，`composition.support` 映射为 `experimental`。

### 解析顺序

resolver（[`internal/capability/compose.go`](../../internal/capability/compose.go)）按固定阶段解析：

1. 输入校验：空能力名、未知能力、重复请求、未知驱动键。
2. 硬依赖闭包：新建项目时补齐并记录 `auto_added`；检查已有项目时只报告缺失依赖，不静默补入。
3. 冲突：只检查已选中的能力对。
4. `Provides` / `Consumes`：重复提供方和必需消费者缺失。
5. 驱动：校验选择、补默认驱动（仅新建路径），并把驱动级运行时服务并入结果。
6. 迁移 schema 依赖：检查覆盖并按规则携带迁移能力。
7. 表与迁移：表所有权唯一、迁移文件唯一、拥有表必须有迁移。
8. 运行时服务/配置：preflight 把声明服务映射到配置段，并检查 manifest 的合并事实。
9. 中间件：受保护路由必须恰好有一个 primary，或没有 primary 时恰好有一个 fallback。
10. plan：调用方校验 manifest 能生成确定性执行计划。

前七个阶段由 `capability.Compose` 完成，运行时配置和中间件阶段由 [`tools/generator/preflight`](../../tools/generator/preflight/preflight.go) 追加，plan 阶段由生成器执行；`jimu check` / `jimu upgrade` 会把 plan 失败转成诊断，`jimu new` 直接失败。任一阶段可返回部分结果，便于一次报告多条诊断。

### 诊断

每条诊断包含 `code`、`capability`、`path`、`message` 和 `fix`，并按阶段、能力、路径、错误码、消息稳定排序。稳定错误码当前包括：

| 阶段 | 错误码 |
|---|---|
| 输入 | `unknown_capability`、`duplicate_capability` |
| 依赖闭包 | `missing_dependency` |
| 冲突 | `conflict` |
| 端口 | `duplicate_provider`、`missing_provider` |
| 驱动 | `unknown_driver`、`missing_driver` |
| 迁移 schema | `missing_migration_schema_dependency` |
| 表/迁移 | `duplicate_table_owner`、`missing_migration`、`duplicate_migration` |
| 运行时 | `unknown_runtime_service`、`missing_runtime_service_config` |
| 中间件 | `middleware_provider_missing`、`middleware_conflict` |
| plan | `invalid_plan` |

错误码的完整定义以 [`internal/capability/diagnostic.go`](../../internal/capability/diagnostic.go) 为准；新增错误码必须同时有阶段、稳定排序和表驱动测试。

### 持久化

生成器只写 schema v3 manifest。`composition` 块记录组合契约版本、`requested`、`resolved`、`auto_added`、驱动、`validity` 和 `support`，是升级和诊断的事实来源。

manifest 校验保证：

- `composition.resolved` 与 `selection.capabilities` 的集合一致。
- `requested` 和 `auto_added` 都是 `resolved` 的子集，且两者不重叠。
- `drivers` 的键必须属于 `resolved`。
- `contract_version`、`validity` 和 `support` 必须是已定义值。

生成时只写 `validity: valid`；非法组合不会产生目标目录。schema v2 manifest 只读：加载时先按 v2 校验 digest，再确定性补全 `composition`（`selection.capabilities` 即 requested/resolved，无 auto-added，驱动取自 `selection.drivers`，`validity` 推导为 `valid`）并在内存中换成 v3。writer 拒绝 v2；schema v1 继续 fail-closed。需要判断当前组合是否仍然合法时，应重新运行 resolver，而不是依赖 v2 推导值。

`selection.support` 保留历史的 `official` / `experimental` / `internal` 取值；组合状态以 `composition.support` 的 `certified` / `validated` / `experimental` 为准。`jimu check` 和 `--dry-run` 读取后者。

### 入口行为

| 入口 | 组合输入 | 校验 | 写行为 |
|---|---|---|---|
| `jimu new` | `--template` 或 `--with` | `--with` 走 resolver，模板使用 certified 预设；两者都执行 preflight + 生成项目 tidy/build/自检 | 通过后才发布目标目录；`--with` 标记 `experimental`。 |
| `jimu check` | 已有 `.jimu/manifest.json` | 执行组合 preflight/plan，并以同一 upgrade planner 检查受管文件、目标版本和 conflict | fast 只读；受管文件漂移/升级/conflict 为 warning，非法组合退出非 0；`--deep` 只在临时副本运行配置、build、test。 |
| `jimu upgrade` | manifest 的 template/profile/`composition.requested` | 模板按预设重建，profile / `--with` 重新解析；执行 preflight 和 plan | `--dry-run` / `--check` 只读；默认事务备份、安装、验证并在失败时回滚。 |
| `jimu capability add` | 已有 `--with` 项目的声明集 + 新能力 | 严格依赖闭包（不自动补依赖）、resolver、preflight | 复用 upgrade 的 manifest 事务；不执行 tidy/build；支持级别保持 `experimental`。 |

`jimu capability add` 拒绝官方模板项目；官方模板需要扩展时用 `--with` 重新生成 experimental 组合。`jimu check` 是框架仓命令，不复制进生成项目。

### 升级语义

`jimu upgrade` 是显式 opt-in，不做隐式重生成。规划阶段以 manifest 的 `generated_file_sha256` 为基线，把 manifest 记录的选择集按当前框架渲染到系统临时目录，再与项目当前内容做三方比较：

- `untouched`：不动作。
- `modified`：安装目标产物。
- `conflict`：本地已偏离基线（修改、删除或路径被占用）；绝不覆盖，事务在写盘前中止。
- `new`：安装目标新增文件。
- `deleted`：保留本地副本，不自动删除，只从后续 manifest 基线移除。

默认事务先备份所有将改变的文件（含 manifest 和可能被 tidy 改写的 `go.mod`/`go.sum`），逐文件安装；受支持配置按 manifest 的 `merges.strategy=sections` 以项目现有段值为底合并；随后执行 `go mod tidy`、`go build ./...` 和 `go test ./...`。任一安装、验证或 manifest 写入失败都按原字节恢复。成功后才写回新 manifest 的最终文件哈希与组合块；文件和 manifest 都没变化时不重复验证，重复执行幂等。

发布门禁只固定覆盖 v0.5.0 的 `basic-api` 与 `auth-api` 快照。`--with` 组合是 experimental，升级不获得与官方模板相同的兼容承诺；已有数据库 schema 不在升级范围内。

### 生命周期诊断

`jimu check` 与 `jimu upgrade --check` 读取同一 upgrade plan。`check --json` 使用 schema version 1 输出 summary、诊断、受管文件、upgrade 和 deep 结果；该 JSON 不含时间戳并稳定排序。`upgrade --check --json --diff` 使用 schema `v1` 输出同一计划的条目，manifest baseline 只提供哈希，不伪造三方正文。

受管文件出现 `modified`、`missing` 或 baseline 缺失时，`check` 输出 warning 和修复建议；非法 manifest、非法组合、无法形成 upgrade plan 或 deep 失败输出 error 并退出非 0。deep 在私有临时副本中串行执行配置校验、`go build ./...` 和 `go test ./...`，不调用 Make、业务脚本或项目内写入流程。

本地实验口径和 v0.8.0 基线见 [EXPERIMENTS](../EXPERIMENTS.md) 与 [experiments/v0.8.0](../experiments/v0.8.0.md)。实验只做本地显式采集，不上传数据，不作为 CI、生产或真实项目采用证据。

## 关键决策

- **写前失败。** 组合、预检或 plan 非空时不得创建目标目录或修改项目。
- **同一套规则。** `new`、`check`、`upgrade`、`capability add` 复用同一 resolver 和诊断，不按入口维护第二份组合逻辑。
- **合法与支持分离。** `validity` 只回答结构是否自洽，`support` 只回答官方承诺程度；二者不能互相推导。
- **新建补齐、存量严格。** `jimu new` 可以记录自动补齐的硬依赖；检查已有项目时缺失依赖只报告，避免掩盖 manifest 已分叉的事实。
- **manifest 是事实来源。** 组合结果持久化到 schema v3，升级和诊断读取该块；schema v2 只读用于迁移。
- **用户文件优先。** 冲突绝不覆盖，删除不自动同步；升级事务只触碰生成器管理的路径。
- **实验组合不冒充认证。** `--with` 和 `capability add` 保持 `experimental`；固定 pairwise 矩阵是回归网，不是支持级别提升。

## 约束与不变量

- 任何生成、检查或升级路径都不得绕过 resolver/preflight。
- 诊断顺序必须稳定；同一输入重复解析必须得到相同 `requested`、`resolved`、`auto_added`、驱动和服务集合。
- 组合契约变更时同步更新 `internal/contract`、resolver、preflight、manifest schema 校验、生成器入口和对应测试。
- 新增能力或驱动时，至少补一个真实组合交互；不要通过扩大全幂集来替代代表性用例。
- 官方模板的整文件配置被用户修改时按 `conflict` 处理；只有 manifest 声明段合并的项目才保留段值。
- schema v1 继续 fail-closed；不得为兼容 v1 放宽当前校验。

## 实现映射

- 契约：[`internal/contract/composition.go`](../../internal/contract/composition.go)、[`contract.Descriptor`](../../internal/contract/capability.go)
- resolver 与诊断：[`internal/capability/compose.go`](../../internal/capability/compose.go)、[`internal/capability/diagnostic.go`](../../internal/capability/diagnostic.go)
- 声明门禁：[`tools/checkcapabilities/composition.go`](../../tools/checkcapabilities/composition.go)
- 写前预检：[`tools/generator/preflight/preflight.go`](../../tools/generator/preflight/preflight.go)
- manifest：[`tools/generator/manifest/model.go`](../../tools/generator/manifest/model.go)、[`validate.go`](../../tools/generator/manifest/validate.go)、[`decode.go`](../../tools/generator/manifest/decode.go)
- 生成与升级：[`tools/generator/frameworkmanifest/export.go`](../../tools/generator/frameworkmanifest/export.go)、[`tools/generator/upgrade.go`](../../tools/generator/upgrade.go)、[`tools/generator/workspace/upgrade.go`](../../tools/generator/workspace/upgrade.go)、[`tools/generator/add.go`](../../tools/generator/add.go)
- CLI：[`cmd/cli/check.go`](../../cmd/cli/check.go)、[`cmd/cli/upgrade.go`](../../cmd/cli/upgrade.go)、[`cmd/cli/capability.go`](../../cmd/cli/capability.go)
- 测试：`go test ./internal/capability/... ./tools/generator/...`、`make test-scaffold-composition`、`make test-scaffold-upgrade`

## 相关设计

- [能力架构](capability-architecture.md)：能力声明、装配与运行时筛选。
- [形态与项目生成](profiles-and-project-generation.md)：profile、生成流程和升级事务。
- [模板契约 v0.1](../templates/v1/README.md)：官方模板、预算和支持边界。
