# Jimu 模板契约 v0.1

> 状态：模板契约 `v0.1` 已冻结；`v0.5.0` 首次按本契约交付两个官方模板，`v0.6.0` 保持这两个模板及其预算不变，并交付显式升级与组合 schema。模板契约修订不自动代表 Jimu 主版本变化；当前发布状态见 [v0.6.0 发布说明](../../releases/v0.6.0.md)。

机器可读事实源：

- [matrix.yaml](matrix.yaml)：模板、支持级别、兼容窗口、预检项和生命周期规则。
- [budgets.json](budgets.json)：官方模板发布阻断预算及统一计量口径（自 v0.5.0 起未放宽，v0.6.0 沿用）。

## 定位

Jimu 是面向生产后端的可组合 Go 项目脚手架：从一个零外部依赖的最小 HTTP 服务开始，按需加入数据库、缓存、认证、RBAC、租户、队列和观测能力。

“通用”指核心足够小、能力可按明确模板组合，不表示默认装入全部能力，也不承诺任意能力组合都受支持。

Jimu 解决：

- 生成结构合理、能运行、能力边界清楚的普通 Go 工程；
- 固定配置、日志、统一错误、健康检查和优雅停机约定；
- 用少量官方模板交付经过端到端验收的能力组合；
- 在写目标目录前拒绝不自洽的能力、迁移和外部服务组合；
- 用文件数、依赖、Go 闭包、路由、表和首次启动时间阻止模板膨胀。

Jimu 不解决：

- 具体业务领域模型、管理前端、报表或行业应用；
- 任意能力的笛卡尔积兼容保证；
- 运行时插件热加载、控制面、应用托管或 Kubernetes Operator；
- 生成后的持续托管、自动同步或强制覆盖；
- 开箱即用的多租户 SaaS 产品。

与普通 Go 框架或脚手架的差异：

| 对比对象 | Jimu 的差异 |
|---|---|
| Gin/Echo | 它们提供 HTTP 原语；Jimu 生成带项目约定、生命周期、错误响应、健康检查、测试和能力边界的完整工程。 |
| go-zero | Jimu 不以 DSL 为核心，不要求生成微服务全家桶，也不让业务代码依赖 Jimu 运行时。 |
| Kratos | Jimu 首要是普通 Go module 和 HTTP 服务，不把 RPC/SDK 体系作为默认起点。 |
| 整仓模板项目 | Jimu 只维护少量官方模板，每个模板都有硬预算和端到端验收。 |

## 生成物所有权

自 v0.5.0 起锁定为“用户拥有的 fork”模型。生成完成后，项目内的代码、配置、迁移和部署资产归用户所有，用户可以自由修改；生成的工程不依赖常驻的 Jimu 运行时。

| 文件类别 | 归属 | 规则 |
|---|---|---|
| `.jimu/manifest.json` | Jimu 写入的机器元数据 | 记录模板、支持级别、framework commit、选择集和生成文件摘要；用户不应手工编辑。它是来源证明，不是持续管理权。 |
| `docs/profiles/generated-report.md` | 可再生成报告 | 允许被下一次生成或验收替换。 |
| 其余生成文件 | 用户代码的起始基线 | 生成后由用户维护；Jimu 不做隐式覆盖。 |
| 用户新增或修改的文件 | 用户 | 不在 manifest 生成文件清单中的文件永不自动覆盖。 |

v0.5.0 的兼容决策继续有效：

- manifest schema v1 不做只读兼容，当前生成器继续 fail-closed；
- v0.4.x 生成项目只支持“用目标版本重新生成到新目录 + 人工 diff + 选择性迁移”，不提供自动项目升级、manifest 重写或存量数据库模板迁移。

v0.5.0 未提供自动升级，也不承诺框架安全修复自动进入已生成项目；v0.6.0 起提供显式 `jimu upgrade`：默认事务以 manifest 基线、项目当前内容和当前框架目标产物做三方比较，`conflict` 在写盘前中止；备份后逐文件安装，随后执行 tidy/build/test，任一失败按原字节回滚；成功后才更新 manifest。`--dry-run` 与 `--check` 只读。升级只修改生成器管理的文件，用户自有文件、已删除的模板文件和 manifest 之外的路径都不触碰；存量数据库 schema 不在升级范围内。

`jimu capability add` 仍是专家实验能力，不是升级通道：它在 `--with` 生成的 experimental 项目和三个预设项目中追加能力，只拒绝已退役模板生成的项目；落盘复用 upgrade 的 manifest 事务，但不执行 tidy/build，也不提升支持级别。组合契约、解析顺序和 `validity`/`support` 的完整规则见[能力组合设计](../../design/capability-composition.md)。

## CLI 与版本语义

正式入口（v0.5.0 起；v0.6.0 保持同一选择器）：

```bash
jimu new <dir>                         # 默认 minimal
jimu new <dir> --template=minimal
jimu new <dir> --template=auth
jimu new <dir> --template=full
jimu new <dir> --with=user,access,auth # 专家模式，标记 experimental
jimu check <dir>                       # 只读检查 manifest 与组合一致性
jimu upgrade <dir> --dry-run           # 只读三方 diff
jimu upgrade <dir>                     # 显式升级事务
```

规则：

- `<dir>` 必填；`--module` 继续显式推荐，无法可靠推导时拒绝生成。
- `--template` 只接受 `minimal`、`auth` 或 `full`；`basic-api`/`auth-api` 已退役。
- `--template`、`--profile`、`--with` 三者互斥。
- 不传选择器时默认 `minimal`。
- `--with` 允许生成，但 manifest 必须标记 `support: experimental`，并执行全部硬预检。
- `--profile` 只用于拒绝框架内部形态；预设请用 `--template`。其他 profile 名称只供 Jimu 自身构建使用，不是对外预设。
- `--dry-run` 打印 requested/resolved/auto-added、驱动、迁移、外部服务、合法性/支持级别（status）、预检和预算结果，不写文件。
- 所有生成先写暂存目录；预检、tidy、build、test 和生成项目自检通过后才原子发布。失败不得留下半成品。
- `jimu check` 只读校验已有项目；`jimu upgrade` 不带 `--dry-run` / `--check` 时执行显式写入事务，冲突在写盘前中止，失败按原字节回滚。

契约版本与产品版本分开管理：

- 本文件是模板契约 `v0.1`，首个目标产品版本为 Jimu `v0.5.0`；v0.6.0 沿用同一模板集合、兼容窗口和预算；
- 模板契约的修订不自动代表 Jimu 主版本变化；
- `v0.5.0` 生成的 manifest 使用 schema v2，必须记录 `selection.template`、`selection.template_version`、`selection.support`、`framework.version`、`framework.commit`、`framework.dirty` 和 `generated_file_sha256`；schema v2 在 v0.6.0 只读用于升级；
- `v0.6.0` 生成的 manifest 只写 schema v3，在 provenance 之外增加 `composition` 块（契约版本、requested/resolved/auto-added、驱动、validity、support）；schema v1 继续 fail-closed；
- framework commit 必须通过 `git rev-parse --verify HEAD^{commit}` 解析，普通 clone 与 git worktree 都必须得到非空 commit；解析失败时拒绝生成；
- `generated_file_sha256` 必须覆盖 `generated_files` 的全部文件，值格式为 `sha256:<64 hex>`；manifest 和可选生成报告自身不进入该映射；
- 开发态允许 `version=<当前版本>-dev`，但必须记录真实 commit，且 `dirty` 必须为 `true`。

## 支持模型

| 级别 | 范围 | 承诺 |
|---|---|---|
| `certified` | `minimal`、`auth` 两个预设 | 固定契约、硬预算、端到端验收和发布阻断门禁；manifest 的 `composition.support` 为 `certified`。 |
| `validated` | `full` 预设 | 通过预算与构建门禁，但缺完整组合级 E2E；pairwise 构建矩阵不自动提升支持级别。 |
| `experimental` | 能通过硬预检的 `--with` 组合与 `capability add` 项目 | 允许生成/追加并标记 `experimental`；不承诺组合稳定性、预算、E2E 或升级兼容。 |
| 内部 profile | `full`、`minimal` | `full` 用于开发、测试和 demo；`minimal` 已冻结。`enterprise`、`machine`、`saas` 已退役。不是对外预设；manifest 的 `composition.support` 映射为 `experimental`。 |

`matrix.yaml` 的 `official` 档对应 `certified`，`experimental` 专家档对应 `experimental`。本版本不新增"已认证专家组合"：只有写入 `matrix.yaml` 的选择才获得正式兼容承诺，`--with` 不能绕过支持边界。`validity: valid|invalid` 与 `support` 正交；合法组合不自动获得 `validated` 或 `certified`。

## 升级支持窗口

- 支持从 `v0.5.0` 及之后生成、manifest schema v2/v3 的 `minimal` / `auth` 项目升级到当前 certified 预设。
- `basic-api` / `auth-api` 已在 v0.10.0 退役：由它们生成的项目执行 `jimu upgrade` 直接硬失败，需按新预设重新生成。
- `v0.4.x` 和 schema v1 项目不支持直接升级，使用目标版本重新生成并人工迁移。
- `--with` 与 `capability add` 保持 experimental；可以使用 `jimu check` / `upgrade --check` 诊断，但不承诺组合升级兼容。
- 安全修复只保证当前 0.7.x 的最新 patch，旧 patch 不 backport。

完整命令、冲突处理和 CI 配方见[升级](../../UPGRADING.md)；决策记录见 [ADR 0003](../../adr/0003-upgrade-support-window.md)。

## 生成前预检

预检发生在写目标目录之前，并在生成项目启动自检中复核关键运行时不变量。

1. **受保护中间件唯一性**
   - Descriptor 必须显式声明是否提供 `ProtectedHTTPMiddleware`。
   - 只要选择集中存在 `MountProtected` 路由，就必须恰好有一个有效提供者。
   - 零提供者或多提供者均拒绝；当前不允许 `auth` 与 `apikey` 同时提供受保护链。
2. **迁移依赖与表所有权**
   - 迁移集等于声明能力迁移与 `MigrationSchemaDeps` 闭包的并集。
   - 每张表只能有一个所有者，同一张表只能出现一次 `CREATE`。
   - 迁移文件必须存在，能力内编号唯一，MySQL 的 Up/Down 必须成对。
   - `auth` 预设必须在空 MySQL 上完整执行迁移，不能依赖预置库。
3. **外部服务与启动不变量**
   - 能力必须声明 MySQL、Redis、broker 等运行时服务；生成器据此裁剪配置段与 `/readyz` 检查。
   - `minimal` 的必需服务集合必须为空：启动路径不得创建 DB/Redis 客户端，配置不得含 `db:`/`redis:` 段。
   - `auth` 必须同时声明 MySQL 与 Redis，缺少任一服务时 fail fast。
4. **硬拒绝，不提供绕过**
   - 未知能力或驱动、空能力名、重复能力；
   - `--template`、`--profile`、`--with` 冲突；
   - 受保护路由没有唯一提供者，或迁移 schema 依赖不完整；
   - 重复表所有权，或服务声明与配置不一致；
   - framework commit 无法解析；
   - tidy、build、test 或生成项目自检失败。

## 三个预设

所有预设都从**当前主代码仓库**按 import 闭包裁剪生成，不维护独立的模板树。预设只是起点：
生成后可用 `jimu capability add` 继续组合能力。

### `minimal`

适用场景：不需要任何业务能力的 HTTP 服务、BFF、Webhook、内部服务。

- 0 个能力；内核只保留被 `cmd/server`/`cmd/cli` 闭包实际引用到的包。
- 内核提供 `/livez`、`/readyz`、`/api/v1/ping` 和 SIGINT/SIGTERM 优雅停机。
- 配置只含非 DB/Redis 段；不含 `internal/e2e`、`docker-compose.yml` 和 OpenAPI 资产。
- 设置或不设置 `ADMIN_PASSWORD` 都能启动：0 能力时 DB 为 nil，种子被护栏跳过。

### `auth`

适用场景：需要用户、登录会话、RBAC 和持久化的后台或业务 API。

- 能力集合 `user` + `access` + `auth`，编译闭包带 `tenant`。支持级别 `certified`。
- 生成 MySQL/Redis 客户端、11 个迁移与 `cmd/cli migrate`/`seed`。
- 不含公开注册、MFA/passkey、OAuth、验证码、通知、字段级加密、审计、队列、对象存储、观测栈或多租户管理 API。
- **不生成 `docker-compose.yml`**：数据库与 Redis 由使用者提供。

### `full`

适用场景：需要完整能力集的起点，或据此继续裁剪。

- 全部 23 个能力，支持级别 `validated`。
- 完整驱动集：`storage` 的 `s3`/`local`、`queue` 的 `kafka`/`rabbitmq`/`redis`、`dataops` 的 `excel`/`csv`。
  `--with` 的缺省只取每个能力的首个驱动，无法表达完整集，因此 `full` 从 `internal/profiles/full/assembly.go` 派生。
- 首发是 `validated` 而不是 `certified`：预算与构建门禁已就位，但缺完整的组合级 E2E。


## 预算与验收

发布阻断预算见 [budgets.json](budgets.json)。

| 指标 | `minimal` 上限 | `auth` 上限 | `full` 上限 |
|---|---:|---:|---:|
| 生成文件 | 300 | 420 | 800 |
| 直接依赖 | 44 | 44 | 64 |
| `cmd/server` Go 闭包文件 | 95 | 154 | 352 |
| Go 闭包代码行 | 11,650 | 17,400 | 39,200 |
| 外部服务 | 0 | MySQL + Redis | 4（MySQL、Redis、Kafka/RabbitMQ、对象存储） |
| 业务路由 | 0 | 31 | 94 |
| 表 | 0 | 9 | 23 |
| 迁移文件 | 0 | 11 | 26 |
| 非代码资产 | 9 | 9 | 10 |
| 首次 `/readyz` | 热缓存 ≤10 秒；冷生成至就绪 ≤60 秒 | 热缓存 ≤2 分钟；冷镜像 ≤8 分钟 | 热缓存 ≤3 分钟；冷镜像 ≤10 分钟 |

生成文件数包含 `.jimu/manifest.json`；依赖与闭包按生成项目的 `go list` / `go/packages` 口径统计。内核路由（`/livez`、`/readyz`、`/api/v1/ping`）不计入 `report.routes`，该字段只统计能力注册的业务路由。

v0.5.1 起 `Scaffold Gate` 分为 official 与 full 两档：official 是 `release/*` 和 `dependabot/*` → `master` PR 的必需检查，只跑需要 Docker 的 auth 空库 E2E；full 只在 `workflow_dispatch` 或夜间定时运行，覆盖全部 `--with`、内部 profile 和重型生成器用例。这样普通 PR 不必等待全矩阵，同时保留夜间/发布前的完整回归。

门禁按「需不需要 Docker」分工，同一条用例不在两个 workflow 各跑一遍：

- `CI (PR)` 的 race 分片跑 `TestPresetBudgets` 与 `TestMinimalPresetStartsWithoutDependencies`。它们不需要 Docker，所以每个 PR 都有覆盖，包括不触发 `CI (Scaffold)` 的 `feature/*`。
- `make test-scaffold-official` 只跑 `TestAuthPresetEmptyDatabaseEndToEnd`（需要 Docker daemon，race 分片里会 skip）。
- full 生成器矩阵由 `make test-scaffold-matrix` 在手动触发或夜间定时运行。v0.6.0 在 full 中新增固定 pairwise 组合矩阵。

各条用例的职责：

- `TestPresetBudgets` 直接读取 `budgets.json`，生成 minimal/auth/full 后实测文件数、直接依赖、Go 闭包、外部服务、路由、表、迁移和非代码资产，任一指标超限即失败。
- `TestMinimalPresetStartsWithoutDependencies` 不启动任何容器，构建并运行生成项目的 server 二进制，在「设置」与「不设置」`ADMIN_PASSWORD` 两种 env 下都验证 `/livez`、`/readyz` 和 `/api/v1/ping`。第二种 env 是空能力集种子的回归钉子：0 能力时 DB 为 nil，种子必须被跳过而不是 panic。
- `TestAuthPresetEmptyDatabaseEndToEnd` 用 `docker run` 起独立 MySQL 8.4 与 Redis 7 容器（不使用 Compose），在全新空库上迁移、seed，校验 `/readyz`、`/api/v1/ping` 和管理员登录返回 access token。
- `TestRepresentativeCompositionMatrix` / `TestCompositionMatrixBudget` / `TestCompositionNegativeMatrix` 运行固定、代表性的 pairwise 清单与冲突负例；组合项目只做 build/vet，不遍历全幂集，也不承诺组合级 E2E。
- `TestUpgradeFixtures` 现场生成 `auth` 预设作为 fixture（模板退役后无法再用旧模板渲染目标树），覆盖 untouched、用户修改生成文件冲突、用户自有文件保留、用户删除、模板删除、重复升级和验证失败回滚；`TestUpgradeRejectsRetiredTemplates` 钉住旧模板的硬失败路径。
- CI 的每个 full shard 把耗时写入 step summary；显式运行时测试必须报告 `PASS`，`SKIP` 视为门禁失败，避免 full 静默跳过。

`minimal` 验收：

- `go test ./...`、`go vet ./...`、`go build ./...` 全部通过；
- 无 Compose、无 DB/Redis 连接尝试，`configs/app.yaml` 不含 `db:`/`redis:` 段；
- 设置或不设置 `ADMIN_PASSWORD` 都能启动，`/livez`、`/readyz`、`/api/v1/ping` 正常；
- 自动报告不超过全部预算。

`auth` 验收：

- 全新空 MySQL 上 11 个迁移全部成功；
- `make migrate` + `make seed` 幂等，`/readyz` 成功；
- 管理员登录返回 access token，`/auth/me` 可用；
- 无 token 访问受保护路由为 401，缺权限为 403；
- 自动报告不超过全部预算。

`full` 验收：

- 23 个能力与完整驱动集（`s3`/`local`、`kafka`/`rabbitmq`/`redis`、`excel`/`csv`）齐全；
- `go build ./...` 通过，自动报告不超过全部预算；
- 支持级别是 `validated`，不是 `certified`。

预算超过即阻断发布。提高预算必须修改本契约和 `budgets.json`，不能在实现 issue 中顺手放宽。

## 实施顺序

1. 固化预设契约、支持矩阵和预算基线。
2. 为 Descriptor 补齐认证中间件提供者、迁移 schema 依赖和 runtime services，实现生成前预检。
3. 把种子模型从能力 domain 抽到内核，解除内核与能力的唯一耦合。
4. 实现 `--template=minimal`：0 能力 + 闭包裁剪内核 + 内核 `/api/v1/ping`。
5. 实现 `--template=auth`（`user`/`access`/`auth`，certified）与 `--template=full`（全部能力，validated）。
6. 建立三预设预算、minimal 无依赖 smoke 和 auth 空库端到端门禁。
7. 退役 `basic-api`/`auth-api` 模板，旧项目 upgrade 硬失败。
8. 完成 v0.10.0 发布说明与不兼容变更说明。
9. 在 v0.6.0 交付 opt-in `jimu upgrade`（已完成）。
