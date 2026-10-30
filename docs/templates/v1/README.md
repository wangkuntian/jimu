# Jimu 模板契约 v0.1

> 状态：已冻结。契约版本为 `v0.1`，首个按本契约交付的产品版本为 Jimu `v0.5.0`。本文描述目标契约；CLI 和生成结果尚未完成迁移时，以 README 的“当前实现”说明为准。

机器可读事实源：

- [matrix.yaml](matrix.yaml)：模板、支持级别、兼容窗口、预检项和生命周期规则。
- [budgets.json](budgets.json)：v0.5.0 发布阻断预算及统一计量口径。

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
- 开箱即用的多租户产品。`auth-api` 的产品语义是单租户。

与普通 Go 框架或脚手架的差异：

| 对比对象 | Jimu 的差异 |
|---|---|
| Gin/Echo | 它们提供 HTTP 原语；Jimu 生成带项目约定、生命周期、错误响应、健康检查、测试和能力边界的完整工程。 |
| go-zero | Jimu 不以 DSL 为核心，不要求生成微服务全家桶，也不让业务代码依赖 Jimu 运行时。 |
| Kratos | Jimu 首要是普通 Go module 和 HTTP 服务，不把 RPC/SDK 体系作为默认起点。 |
| 整仓模板项目 | Jimu 只维护少量官方模板，每个模板都有硬预算和端到端验收。 |

## 生成物所有权

v0.5.0 锁定为“用户拥有的 fork”模型。生成完成后，项目内的代码、配置、迁移和部署资产归用户所有，用户可以自由修改；生成的工程不依赖常驻的 Jimu 运行时。

| 文件类别 | 归属 | 规则 |
|---|---|---|
| `.jimu/manifest.json` | Jimu 写入的机器元数据 | 记录模板、支持级别、framework commit、选择集和生成文件摘要；用户不应手工编辑。它是来源证明，不是持续管理权。 |
| `docs/profiles/generated-report.md` | 可再生成报告 | 允许被下一次生成或验收替换。 |
| 其余生成文件 | 用户代码的起始基线 | 生成后由用户维护；Jimu 不做隐式覆盖。 |
| 用户新增或修改的文件 | 用户 | 不在 manifest 生成文件清单中的文件永不自动覆盖。 |

v0.5.0 不提供自动升级，也不承诺框架安全修复自动进入已生成项目。推荐升级方式是用目标版本重新生成到新目录，人工 diff 后选择性迁移。

`jimu capability add` 在 v0.5.0 仍是专家实验能力，不是升级通道。`jimu upgrade` 计划在 v0.6.0 提供显式触发、三方 diff、仅修改未改动的生成文件、冲突不覆盖、tidy/build/test、备份和回滚。

## CLI 与版本语义

v0.5.0 的正式入口：

```bash
jimu new <dir>                         # 默认 basic-api
jimu new <dir> --template=basic-api
jimu new <dir> --template=auth-api
jimu new <dir> --with=user,access,auth # 专家模式，标记 experimental
```

规则：

- `<dir>` 必填；`--module` 继续显式推荐，无法可靠推导时拒绝生成。
- `--template` 只接受 `basic-api` 或 `auth-api`。
- `--template`、`--profile`、`--with` 三者互斥。
- 不传选择器时默认 `basic-api`。
- `--with` 允许生成，但 manifest 必须标记 `support: experimental`，并执行全部硬预检。
- `--profile=minimal` 在 v0.5.x 映射为 `auth-api` 并打印弃用警告，v0.6.0 移除；其他 profile 名称只供 Jimu 自身构建使用，不是对外模板。
- `--dry-run` 打印解析后的能力、驱动、迁移、外部服务、预检和预算结果，不写文件。
- 所有生成先写暂存目录；预检、tidy、build、test 和生成项目自检通过后才原子发布。失败不得留下半成品。

契约版本与产品版本分开管理：

- 本文件是模板契约 `v0.1`，目标产品版本为 Jimu `v0.5.0`；
- 模板契约的修订不自动代表 Jimu 主版本变化；
- `v0.5.0` 生成的 manifest 必须记录 `template`、`template_version`、`support_level`、`framework_version`、`framework_commit`、`dirty` 和生成文件 SHA-256；
- framework commit 必须通过 `git rev-parse --verify HEAD^{commit}` 解析，普通 clone 与 git worktree 都必须得到非空 commit；解析失败时拒绝生成；
- 开发态允许 `version=v0.5.0-dev`，但必须记录真实 commit 和 `dirty=true`。

## 支持模型

| 级别 | 范围 | 承诺 |
|---|---|---|
| 官方模板 | `basic-api`、`auth-api` | 固定契约、硬预算、端到端验收和发布阻断门禁。 |
| 专家选择 | 能通过硬预检的 `--with` | 允许生成并标记 `experimental`；不承诺组合稳定性、预算或升级。 |
| 内部 profile | `full`、`minimal`、`saas`、`enterprise`、`machine` | 仅用于 Jimu 自身构建和维护，不是 v0.5.0 对外模板矩阵。 |

本版本不新增“已认证专家组合”。只有写入 `matrix.yaml` 的选择才获得正式兼容承诺，`--with` 不能绕过支持边界。

## 生成前预检

预检发生在写目标目录之前，并在生成项目启动自检中复核关键运行时不变量。

1. **受保护中间件唯一性**
   - Descriptor 必须显式声明是否提供 `ProtectedHTTPMiddleware`。
   - 只要选择集中存在 `MountProtected` 路由，就必须恰好有一个有效提供者。
   - 零提供者或多提供者均拒绝；v0.5.0 不允许 `auth` 与 `apikey` 同时提供受保护链。
2. **迁移依赖与表所有权**
   - 迁移集等于声明能力迁移与 `MigrationSchemaDeps` 闭包的并集。
   - 每张表只能有一个所有者，同一张表只能出现一次 `CREATE`。
   - 迁移文件必须存在，能力内编号唯一，MySQL 的 Up/Down 必须成对。
   - `auth-api` 必须在空 MySQL 上完整执行迁移，不能依赖预置库。
3. **外部服务与启动不变量**
   - 能力必须声明 MySQL、Redis、broker 等运行时服务；生成器据此生成配置、Compose 和 `/readyz` 检查。
   - `basic-api` 的必需服务集合必须为空，启动路径不得创建 DB/Redis 客户端。
   - `auth-api` 必须同时声明 MySQL 与 Redis，缺少任一服务时 fail fast。
4. **硬拒绝，不提供绕过**
   - 未知能力或驱动、空能力名、重复能力；
   - `--template`、`--profile`、`--with` 冲突；
   - 受保护路由没有唯一提供者，或迁移 schema 依赖不完整；
   - 重复表所有权，或服务声明与配置/Compose 不一致；
   - framework commit 无法解析；
   - tidy、build、test 或生成项目自检失败。

## `basic-api`

适用场景：普通 HTTP API、BFF、Webhook、内部服务和需要完全自控基础设施的起点。

能力集合：

- Gin HTTP server、`/api/v1` 路由约定和示例路由；
- Viper 配置加载与严格校验；
- Zap 结构化日志、request id、访问日志和 panic recovery；
- 统一成功/错误响应与稳定错误码；
- `/livez`、`/readyz`；
- SIGINT/SIGTERM 优雅停机；
- 单元测试和最小 README、Makefile、Dockerfile。

目标目录：

```text
.
├── cmd/server/main.go
├── internal/app/{app.go,router.go}
├── internal/config/config.go
├── internal/api/v1/{routes.go,ping.go}
├── internal/platform/http/{server.go,middleware.go}
├── internal/platform/health/health.go
├── internal/platform/logging/logger.go
├── internal/shared/errorx/error.go
├── internal/shared/response/response.go
├── configs/app.yaml
├── README.md
├── Makefile
├── Dockerfile
├── .gitignore
├── .golangci.yml
├── go.mod
├── go.sum
└── .jimu/manifest.json
```

测试文件放在对应包内，不另建镜像目录。

依赖边界：Gin、Viper、Zap、validator 和少量日志轮转依赖；直接依赖最多 8 个。不得带入 GORM、Redis、Casbin、JWT、OTel、Prometheus、Kafka、S3 或云厂商 SDK。

配置只包含 `http`、`management`、`log`。默认 HTTP 端口为 `8080`，管理端口为 `9090`；敏感配置不从仓库默认值提供。

路由：

| 路由 | 说明 |
|---|---|
| `GET /api/v1/ping` | 示例业务路由，返回统一成功结构。 |
| `GET /livez` | 进程存活，不检查外部依赖。 |
| `GET /readyz` | 服务可接收请求；`basic-api` 不检查外部依赖。 |

404、405、绑定失败和 panic 都必须返回统一错误结构，不能泄露堆栈或内部路径。迁移、表、Compose、Kubernetes、Helm、OpenObserve 和 Swagger 资产均为 0。

首次启动：

```bash
jimu new demo
cd demo
go test ./...
go build ./...
make run
# GET http://127.0.0.1:8080/api/v1/ping
# GET http://127.0.0.1:9090/readyz
```

不启动任何容器，不设置 DB/Redis 环境变量。

## `auth-api`

适用场景：单部署、单租户的后台或业务 API，需要用户、登录会话、RBAC 和持久化。

能力集合：

- `basic-api` 的全部基础能力；
- 声明能力 `user`、`access`、`auth`；
- MySQL 持久化与 Redis 会话/限流；
- 迁移、结构性 seed 和管理员初始化；
- 项目自带 Compose；
- 项目自带 `cmd/cli migrate` 与 `seed`。

v0.5.0 不包含公开注册、MFA/passkey、OAuth、验证码、通知、字段级加密、审计、队列、对象存储、观测栈或多租户管理 API。

单租户语义：`auth-api` 的部署只有一个业务租户。为兼容现有 `user`/`access` schema，v0.5.0 保留一个不可操作的默认 tenant carrier：

- 保留 `tenants.id=1`、`users.tenant_id`、`roles.tenant_id` 和相关 join；
- manifest 标记为 `migration/domain-only`；
- 不生成 tenant 配置，不装配 tenant 模块，不注册 tenant 路由，不暴露租户选择；
- 所有管理 API 固定作用于默认租户。

目标目录：

```text
.
├── cmd/server/main.go
├── cmd/cli/main.go
├── internal/app/
├── internal/config/
├── internal/domain/models.go
├── internal/store/{db,migrate,redis,seed}.go
├── internal/auth/{jwt,middleware,password}.go
├── internal/api/v1/{routes,helpers,auth,users,access}.go
├── internal/migrations/migrate.go
├── internal/migrations/mysql/*.sql
├── internal/platform/
├── internal/shared/
├── configs/app.yaml
├── conf/rbac_model.conf
├── docker-compose.yml
├── scripts/{init.sh,verify.sh}
├── README.md
├── QUICKSTART.md
├── Makefile
├── Dockerfile
├── .gitignore
├── go.mod
├── go.sum
└── .jimu/manifest.json
```

业务路由固定为 24 条，不含 `/livez` 和 `/readyz`。v0.1 固定以下核心路由：

| 分组 | 路由 |
|---|---|
| 认证 | `POST /api/v1/auth/login`、`POST /api/v1/auth/refresh`、`POST /api/v1/auth/logout`、`GET /api/v1/auth/me`、`GET /api/v1/auth/login-history` |
| 用户 | `POST/GET /api/v1/users`、`GET/PUT/DELETE /api/v1/users/:id` |
| 角色 | `GET /api/v1/roles/options`、`POST/GET /api/v1/roles`、`GET/PUT/DELETE /api/v1/roles/:id`、`GET/POST /api/v1/roles/:id/permissions` |
| 权限 | `POST/GET /api/v1/permissions`、`GET/PUT/DELETE /api/v1/permissions/:id` |
| 用户角色 | `POST /api/v1/users/:id/roles` |

配置只包含 `http`、`management`、`log`、`db`（MySQL）、`redis`、`id`、`auth`。不得生成 `otel`、`grpc`、`queue`、`storage`、`tenant`、`mfa`、`oauth`、`captcha`、`email`、`notification` 等未启用段。

数据模型固定为 8 张表：`users`、`roles`、`permissions`、`role_permissions`、`user_roles`、`tenants`、`login_histories`、`password_histories`。固定 10 个 MySQL 迁移，依赖顺序为 tenant schema carrier、user、access、auth；不得复制 `tenant_plans` 等未使用表。

外部服务只包含 MySQL 8.4 LTS 与 Redis 7.x。Compose 只包含 `mysql`、`redis`、初始化 job 和 `app`，不包含观测、消息队列或其它可选栈。

首次启动：

```bash
jimu new demo --template=auth-api
cd demo
make up       # migrate + seed + start；返回时 /readyz 已就绪
make verify   # readyz + login + me + 一次受保护请求
make down
```

管理员密码由 `make up` 生成到被 gitignore 的本地 secret，只显示一次，不使用 `admin/123456` 等默认凭据。重复 `make up` 必须幂等，不得重复建表或产生第二个管理员。

## 预算与验收

发布阻断预算见 [budgets.json](budgets.json)。

| 指标 | SIRIUS-9 `minimal` 基线 | `basic-api` 上限 | `auth-api` 上限 |
|---|---:|---:|---:|
| 生成文件 | 431 | 40 | 160 |
| 直接依赖 | 48 | 8 | 20 |
| `cmd/server` Go 闭包文件 | 162 | 25 | 70 |
| Go 闭包代码行 | 17,217 | 2,200 | 9,000 |
| 外部服务 | MySQL + Redis | 0 | MySQL + Redis |
| 业务路由 | 31 | 1 | 24 |
| 表 | 9 | 0 | 8 |
| 迁移文件 | 11 | 0 | 10 |
| 非代码资产 | 45+ | ≤12 | ≤20 |
| 首次 `/readyz` | 需 DB/Redis | 热缓存 ≤10 秒；冷生成至就绪 ≤60 秒 | 热缓存 ≤2 分钟；冷镜像 ≤8 分钟 |

生成文件数排除 `.jimu/manifest.json` 和 `docs/profiles/generated-report.md`；依赖与闭包按生成项目的 `go list` / `go/packages` 口径统计。

自动门禁由 `make test-scaffold-matrix` 执行，并纳入 CI `Scaffold Gate`（`auth-api` 用例需要可用的 Docker Compose）：

- `TestOfficialTemplateBudgets` 直接读取 `budgets.json`，生成两个官方模板后实测文件数、直接依赖、Go 闭包、外部服务、路由、表、迁移和非代码资产，任一指标超限即失败。
- `TestBasicAPIStartsWithoutExternalServices` 不启动容器，先跑生成项目的 build/vet/test，再验证 `/livez`、`/readyz`、示例路由、统一 404/405 响应、SIGTERM 优雅退出和冷生成至就绪预算。
- `TestAuthAPIEmptyDatabaseEndToEnd` 先跑生成项目的 test/vet，再为每次运行创建独立 Compose 项目和宿主机端口，在全新数据卷上连续执行两次 `make up` / `make verify`，校验管理员密码只在首次输出、10 个迁移、8 张业务表、2 个用户，并验证缺失 MySQL 或 Redis 时快速失败和 `make down`。

`basic-api` 验收：

- `go test ./...`、`go vet ./...`、`go build ./...` 全部通过；
- 无 Compose，无 DB/Redis 连接尝试；
- `make run` 后 `/livez`、`/readyz` 和 `/api/v1/ping` 正常；
- 未知路由和方法返回统一错误结构；
- SIGTERM 后在配置的 shutdown timeout 内退出；
- 自动报告不超过全部预算。

`auth-api` 验收：

- 全新空 MySQL 上 10 个迁移全部成功；
- `make up` 一条命令完成 migrate、seed、start，`/readyz` 成功；
- 管理员登录返回 access/refresh token，`/auth/me` 可用；
- 无 token 访问受保护路由为 401，缺权限为 403；
- 重复启动和重复 seed 幂等；
- 故意缺 Redis 或 MySQL 时启动失败，错误中说明缺少哪个服务；
- 自动报告不超过全部预算。

预算超过即阻断发布。提高预算必须修改本契约和 `budgets.json`，不能在实现 issue 中顺手放宽。

## 实施顺序

1. 固化模板契约、支持矩阵和预算基线。
2. 为 Descriptor 补齐认证中间件提供者、迁移 schema 依赖和 runtime services，实现生成前预检。
3. 拆出零外部依赖的 HTTP 核心，实现 `basic-api`。
4. 实现 `--template`、默认 basic、legacy profile 映射和 experimental 标记。
5. 实现带隐藏 tenant carrier 的 `auth-api` 与一键启动。
6. 建立预算、basic smoke 和 auth 空库端到端门禁。
7. 修复 provenance，扩展 manifest。
8. 完成 v0.5.0 发布说明与破坏性变更说明。
9. 在 v0.6.0 启动 opt-in `jimu upgrade`。
