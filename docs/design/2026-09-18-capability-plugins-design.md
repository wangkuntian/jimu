# 能力可插拔架构设计（v0.3.0）

> 状态：待评审 · 2026-09-18 · 集成分支 `release/v0.3.0`
> 本文取代本次设计早期的草稿（那份草稿基于"8 个模块 + 仅运行时开关"的旧假设，未入库，已废弃）。

## 1. 目标与非目标

把 jimu 从"一个功能齐全的后端服务"改造成**可自由组合的能力集合**：按形态挑选能力，新项目只带走选中的部分，从代码、依赖、迁移到二进制一起变小。

| 能力 | 验收 |
|---|---|
| **可组合** | 5 个目标形态都能从同一份仓库生成/构建出来，各自只含需要的能力 |
| **可关** | 运行时关闭某能力 → 它的路由、任务、事件、迁移、权限点、种子全部不生效；启用闭包缺失时启动报错而非静默降级 |
| **可删** | 删除任一能力目录 → 依赖它的形态必须编译失败（依赖校验生效），不依赖它的形态必须仍能 build 与启动 |
| **可量化** | 每个形态产出 `compose-report`：代码行数 / 文件数 / `go.mod` 直接依赖数 / 二进制大小 / 路由数 / 迁移数 / 表数 |

**新项目与 jimu 的关系**：模板复制。新项目用脚手架生成，是**独立仓库**，保持 `internal/` 不变。jimu 后续修复不会自动流入已生成项目（靠重新生成或 cherry-pick），这是已接受的取舍。

**非目标**：对外 HTTP API 变更、配置键重命名、数据库 schema 破坏性变更（存量实例数据必须保留）、运行时热插拔、`.so` 或进程外插件、把 jimu 拆成多个 Go module 供 import（`internal/` 不可被外部 import，也因此排除了库依赖模式）。

## 2. 目标形态

| 形态 | 场景 | 组成 | 关键验收（示例） |
|---|---|---|---|
| ① `minimal` | 内部微服务 / 新项目起点 | kernel + user + auth + access + notify（日志渠道兜底，零配置）；apikey 按需 | 不含租户/审计/控制台；生成项目 `go.mod` 直接依赖显著少于 `full` |
| ② `saas` | 面向外部客户的多租户产品 | ① + tenancy + 开通式注册 + audit + 真实邮件渠道 | `/tenants*`、`/tenant-plans*`、`/audits*` 存在；含租户维度限流 |
| ③ `enterprise` | 公司内部系统，单租户 | ① + console + audit + sso + storage + dataops | 单租户（`tid=0` 平台级视角）；无 `/tenants*` |
| ④ `machine` | 无界面、服务间调用 | kernel + user（仅归属主体）+ apikey + access + grpc | **无任何登录/注册/会话端点**；靠 `X-API-Key` + scope |
| ⑤ `full` | 全功能基准 | 全部能力 | 现有全部测试、`release-check`、`compose-check` 的回归基准，不允许退化 |

每个形态的验收数字由 §6.2 的脚手架报告产生并进 CI 归档对比 —— "体积更小"必须可回归验证，而不是感觉。

## 3. 能力清单

### 3.1 内核 `internal/kernel/`（不可勾选）

> 路径注记：本文写于 P1-B1 之前，正文中的 `platform/x` 指该包**搬迁前**的位置；搬迁后**原平台层**的内核机制在 `internal/kernel/x`、能力实现在 `internal/capabilities/…`。§3.5.3/§3.6 的表格另有就地说明。

`config`（内核配置段 + 能力段合并）、`log`（Zap + 脱敏 + logcheck 规范）、`db`（连接池/方言/迁移执行器/事务/并发控制）、`cache`（Redis 抽象 + Cache-Aside + singleflight）、`httpx`（HTTP server + 内核中间件 + management server，见 §3.5）、`obs`（健康检查/Prometheus/OTel 追踪与导出/错误上报 reporter）、`contract`（`Capability` 契约 + 端口 + 注册表）、`app`（装配 + 生命周期）、`shared`（错误码/i18n/校验器/统一响应/分页）、`breaker`（统一熔断器，HTTP/Redis/DB/gRPC 共用）、`event`（事件总线，能力 `RegisterEvents` 的载体）、`tlsconf`（TLS/mTLS 配置构建，HTTP 与 gRPC 共用）、`mask`（敏感信息脱敏，日志链路强制）、`httpclient`（统一出站 HTTP：超时/重试/熔断/限流/traceparent 注入）、`id`（雪花 ID 生成器，与现 `platform/db/snowflake.go` 合并去重）。

内核不含任何业务表（迁移执行器自建的版本表除外）。

### 3.2 横切能力

| 能力 | 拥有表 | 依赖 | 说明 |
|---|---|---|---|
| `user` | `users` | — | 身份主体，全形态必带；同时提供自助面与管理面用例（见 §5.3） |
| `access` | `roles` `permissions` `role_permissions` `user_roles` | user | RBAC + Casbin + 鉴权中间件 + 角色分配 |
| `tenancy` | `tenants` `tenant_plans` | — | 租户实体/套餐/配额/用量 + 开通式注册（上下文机制在 kernel/tenant） |
| `audit` | `audit_logs` `audit_chain_head` | tenancy（可空） | 审计写入/哈希链/校验/导出；作为可选中间件提供者 |
| `apikey` | `api_keys` | user，tenancy（可空） | 机器凭证 + scope + Key 维度限流 |
| `notify` | — | — | email/sms/webhook/ws 派发；无渠道时日志降级（已有实现） |
| `outbox` | `outbox_events` | — | 事务性事件 + MQ 桥接 |
| `jobs` | `jobs` `job_history` `dead_letters` `scheduled_jobs` | outbox / queue | 队列 worker + 定时任务 + 死信 + 管理面 |

### 3.3 领域能力

| 能力 | 拥有表 | 依赖 | 说明 |
|---|---|---|---|
| `auth` | `login_histories` `password_histories` | user, notify | 登录/会话/JWT/注册/改密/密码策略/登录历史 |
| `sso` | `user_oauth_bindings` | user | 第三方登录 + 企业 OIDC |
| `mfa` | `user_mfa`（新建，见 §4 冲突 1） | auth | TOTP + 可信设备（跳过 MFA） |
| `passkey` | `webauthn_credentials` | auth, user | WebAuthn / 通行密钥无密码登录 |
| `captcha` | — | — | 图形验证码（Redis 一次性校验） |
| `breach` | — | — | HIBP 泄露口令检查（k-匿名） |
| `console` | — | 只读多方 | 平台级视图：系统状态/在线用户/配置/监控/限流/错误码 |
| `dataops` | `import_jobs` | jobs, user | 导入导出任务与模板 |

### 3.4 技术能力

| 能力 | 说明 |
|---|---|
| `storage` | 文件存储，驱动级可插拔（见 §3.7） |
| `uploadsec` | 上传安全：大小限制 + magic-byte 嗅探 + MIME 白名单 + 可选 ClamAV 扫描；含上传路由（现散落在 `platform/http/upload_handler.go` 与 `clamav.go`） |
| `search` | 全文检索（`search_documents`）+ CJK LIKE/ILIKE 回退 |
| `grpc` | gRPC 双栈 + 统一出站客户端 |
| `queue` | 队列抽象（Redis/Kafka/RabbitMQ），驱动级可插拔（见 §3.7） |
| `featureflag` | 运行时灰度开关 |
| `encryption` | 字段级 AES-GCM + 盲索引；经能力声明注册全局 gorm hook |
| `retention` | 历史数据保留（现散落在 `platform/db/retention.go` 与 `cleanup.go`） |
| `ws` | WebSocket Hub + 会话/频道，由 `notify` 与 `console` 消费 |
| `apidocs` | Swagger UI + `docs/openapi/` 生成物 + swaggo 注解；`minimal`/`machine` 默认不带（省掉 `swaggo/*` 与 `go-openapi/swag` 共 10 个 module） |

### 3.5 基础功能与中间件

中间件**不作为独立能力**（避免清单从 26 项膨胀到 40+，且多数中间件只有几十行，"省 40 行换一套依赖机制"不划算）。按归属分三类：

**3.5.1 内核中间件（固定，任何形态都要）**

| 中间件 | 作用 |
|---|---|
| `RequestID` / `Logger` | 请求 ID、访问日志（注入 trace_id/span_id） |
| `Recovery` | panic 恢复 + 错误上报，进程安全边界 |
| `Timeout` | 请求超时；handler 未产出响应时补 `1008`/504 |
| `Metrics` | HTTP 指标（`jimu_http_*`） |
| `Locale` | `Accept-Language` → i18n 上下文 |
| `ValidateJSON` / `ValidateQuery` | 请求结构校验（handler 依赖） |

**3.5.2 配置开关控制的中间件（归属明确，不做独立能力）**

| 中间件 | 归属 | 说明 |
|---|---|---|
| `UserRateLimitMiddleware` | 随 `user` | 用户维度滑动窗口 |
| `TenantRateLimitMiddleware` | 随 `tenancy` | 租户维度；平台级视角跳过 |
| `APIKeyRateLimitMiddleware` | 随 `apikey` | Key 维度，按 Key ID 计数不落明文 |
| `AdminAuth` + `AdminIPAllowlist` | 随 `console` | 管理端准入，与 `console` 同生共死 |
| `Security` / `SecurityHeadersFromConfig` | 内核 `httpx`，配置开关 | HSTS/CSP/X-Frame-Options/Referrer-Policy 等；浏览器场景才需要 |
| `CORSMiddleware` | 内核 `httpx`，配置开关 | 有前端才需要 |
| `CSRF` | 内核 `httpx`，配置开关 | Cookie 场景；Bearer 请求自动跳过，`machine` 形态无意义 |
| `Signature` / `SignRequest` | 内核 `httpx`，配置开关 | 服务间调用签名 |
| `IPAllowlist` | 内核 `httpx`，配置开关 | 全局 IP 白名单（CIDR/单 IP） |
| `ConcurrencyLimit` | 内核 `httpx`，配置开关 | 并发上限 + 短排队/拒绝（`1010`/503） |
| `GzipCompression` | 内核 `httpx`，配置开关 | 响应压缩 |
| `GlobalRateLimit` | 内核 `httpx`，配置开关 | IP 令牌桶全局限流 |
| `IdempotencyMiddleware` + `body_recorder` | 内核 `httpx`，配置开关 | 请求幂等（依赖 Redis） |

> P1.4 归位：user/tenancy/apikey 三个维度限流均已随能力落位——user 维度在 `kernel/http/middleware/ratelimit_user.go`（P1.3），tenancy/apikey 维度在 `capabilities/apikey/middleware/ratelimit_dimension.go`（tenancy 能力未落位前暂驻 apikey 能力包）；管理端准入（`admin_auth.go`）暂留 `kernel/http/middleware`，随 P1.7 `console` 能力迁出。

**3.5.3 现有 `shared/` 与 `platform/` 包的归位**

`platform/auth`、`platform/tenant`、`platform/db`、`platform/http` 四个混装包的内部拆分见 §3.6；下表是其外围包的归位：

> 注：下表各路径是**改造前的现状路径**（`platform/…`）。P1-B1 已按归属把 `internal/platform/` 迁到 `internal/kernel/…` 与 `internal/capabilities/…`；表格保留原路径以便与当时的评审记录对照。

| 包 | 行数 | 归属 |
|---|---|---|
| `shared/errors` | 199 | 内核（错误码 + 多语言映射） |
| `shared/i18n` | 197 | 内核 |
| `shared/response` | 170 | 内核（统一响应格式） |
| `shared/id` | 99 | 内核（雪花 ID）；与 `platform/db/snowflake.go` **合并去重**（现为两处实现） |
| `shared/pagination` | 67 | 内核 |
| `shared/validator` | 70 | 内核（自定义校验规则） |
| `shared/testutil` | 472 | 开发期内核（testdb + 迁移锁，不进生产二进制） |
| `shared/totp` | 128 | **已迁入 `mfa` 能力内部**（`internal/capabilities/mfa/totp`，P1.6） |
| `platform/importer` / `platform/exporter` | 494 / 147 | **`dataops` 能力内部** |
| `platform/breaker`、`event`、`tlsconf`、`mask`、`httpclient`、`ws` | 167 / 73 / 57 / 201 / 195 / 965 | 见 §3.1（前五个内核）与 §3.4（`ws` 为可选能力包） |

### 3.6 平台包归位（包内拆分）

现有若干"混装包"横跨多个能力，必须按下表拆开 —— 否则能力边界只是目录改名：

> 注：下表"现包"列是**改造前的现状路径**（`platform/…`）。P1-B1 已按整包归属把 `internal/platform/` 迁到 `internal/kernel/…` 与 `internal/capabilities/…`（`platform/tenant` → `kernel/tenant`，见该行）；包内拆分仍按本表执行。

| 现包 | 行数 | 内部组成 | 归位 |
|---|---|---|---|
| `platform/auth` | 905 | `jwt.go` `session.go` `middleware.go` `limiter.go` `lockout.go` | `auth` |
| | | `casbin.go` `permission_middleware.go` `roles.go` | `access` |
| | | `apikey.go` `apikey_middleware.go` | `apikey` |
| `platform/tenant` | 67 | 上下文注入 + 编码校验/normalize | `kernel/tenant`（上下文机制；`tenancy` 能力负责租户实体/套餐/配额/开通式注册）；上下文无租户时为 `tid=0` 平台级视角，`tenancy` 关闭不影响其他能力读取租户上下文 |
| `platform/db` | 1408 | `migrate.go` `mysql.go` `postgres.go` `transaction.go` `concurrency.go` `gorm_logger.go` | 内核 |
| | | `snowflake.go`（与 `shared/id` 重复实现） | 内核，**两处合并去重** |
| | | `breaker.go` | 内核 `breaker` |
| | | `encryption.go` | `encryption` |
| | | `cleanup.go` + `retention.go` | `retention` |
| | | `seed.go`（含 `basePermissions()`） | 内核只留默认租户 + 超管角色 + Casbin 模型加载；权限点与业务种子改由能力声明（§6.1 的 `Permissions`）<br>P1.5 归位：种子整体迁至 `internal/app/seed.go`（kernel 不再 import capabilities），权限点聚合自启用集各能力 `Descriptor.Permissions`；默认租户/套餐/超管角色等结构性种子暂不做能力门控（随 P1 profile 落地） |
| `platform/http` | 2254 | `server.go` `middleware/`（18 个）`management.go` | 内核 `httpx` |
| | | `swagger.go` | `apidocs` |
| | | `upload_handler.go` `clamav.go` | `uploadsec` |
| 顶层 `conf/rbac_model.conf` | — | Casbin 模型文件 | `access` |
| `proto/jimu/v1/userinfo.proto` + `platform/grpc/userinfo_service.go` | — | 示例服务，且反向依赖 `internal/modules/user/domain`（搬迁前路径；现为 `internal/capabilities/user/domain`，§4 的 platform→module 违规） | 移出平台层，作为 `grpc` 能力的可选示例或 `examples/` |
| `container.go` 里的 `new_dashboard` / `beta_features` | — | 演示性 Feature Flag | 随能力或配置声明，不进内核容器 |

> P1.4 拆分结果：auth 机制（JWT/session/限流/lockout/AuthMiddleware + `apikey_context.go` 上下文助手）留 `kernel/auth`；access → `kernel/access`；apikey → `capabilities/apikey`（`APIKey` 模型与仓储接口在 `apikey/domain/`（仓储实现暂留 admin/infrastructure，P1.7 迁移），维度限流在 `apikey/middleware/`）。

### 3.7 驱动级可插拔

以下能力的第三方驱动现在**恒在二进制里**，是"体积更小"最直接的着力点：

| 能力 | 驱动包 | 现在恒进二进制的东西 | 拆分收益 |
|---|---|---|---|
| `storage` | `storage/local`、`storage/s3` | AWS SDK v2（5 个 service module + 4 个 internal） | 只用本地存储时整套 AWS SDK 不进依赖与二进制 |
| `queue` | `queue/redis`、`queue/kafka`、`queue/rabbitmq` | `kafka-go` v0.4.51、`amqp091-go` v1.13.0 | 只用 Redis 队列时 Kafka/RabbitMQ 客户端不进产物 |
| `dataops` | `dataops/csv`、`dataops/excel` | `xuri/excelize/v2`（**同时是 `GO-2026-6452` govulncheck 豁免的唯一来源**） | 只用 CSV 时 excelize 不进产物，豁免项可从 `scripts/govulncheck.sh` 删除 |

机制：驱动独立成包，能力核心只定义接口与工厂；由 `catalog`（或 profile 入口）**显式 import** 选中的驱动包完成注册 —— 与"显式清单、不用隐式 `init()` 自注册"的既定选择一致（`init()` 注册发生在被显式 import 的包内，不违背该原则）。

### 3.8 非代码资产模块化

代码之外，以下资产也必须随能力裁剪，否则"体积更小"只体现在 `.go` 文件上：

| 资产 | 现状 | 模块化方式 |
|---|---|---|
| `deploy/openobserve/`（dashboards 3 个 + alerts 8 条 + 同步脚本） | 与全部 46 个 deploy 文件一起存在 | 随 `obs` 能力携带；未选中 `obs` 时不出现 |
| `deploy/k8s/backup-cronjob.yaml`、`deploy/backup/Dockerfile`、Helm `backup.*` | — | 内核运维资产，全形态携带 |
| `deploy/helm/values.yaml` | 一份全量 values | 按选中能力渲染，只保留相关段落 |
| `configs/app.yaml` | 27 段全量 | 按选中能力渲染（§8） |
| `docs/openapi/` | 生成物随仓库 | 随 `apidocs` 能力；未选中时 `make swagger` 与 CI 的 `git diff --exit-code docs/openapi` 检查一并去掉 |
| CLI 子命令（`cmd/cli`） | 只有内核命令 + 脚手架 | 能力声明自己的子命令（如 `jobs` → `jimu jobs list`）；生成项目的 CLI 按能力裁剪 |
| `internal/e2e` 契约测试 | 整体契约测试 | 按能力拆分，与 §9 门禁配套 |

## 4. 边界规则与现状违反

**规则**：① 一张表只属于一个能力 ② 其他能力不得给别人的表加列，需要附加数据就建自己的从表 ③ 跨能力只经 `contract` 端口读，写只在所有者 ④ 依赖必须单向无环 ⑤ 能力不得 import 其他能力的内部包。

用规则核对现状，暴露 4 处违反，构成本次重构的硬性任务：

| # | 违反 | 现状 | 处理 |
|---|---|---|---|
| 1 | `users` 被 4 个迁移改 | `001` 建表、`004` 加 TOTP 列、`005` 加 `tenant_id`、`008` 加 `version` | `users` 归 `user`；TOTP 密钥迁到 `mfa` 的 `user_mfa` 从表；`tenant_id`/`version` 列保留（`tenancy`/并发控制在 §11 说明） |
| 2 | `user_roles` 与 `users` 同批迁移 | `001_core.sql` | 归 `access`（角色分配是 RBAC 用例） |
| 3 | `audit_logs` 被别人加列 | `005` 加 `tenant_id`、`010` 加哈希链 | 两处改造移入 `audit` 自己的迁移 |
| 4 | `api_keys` 被别人加列 | `006` 加 `tenant_id` | 移入 `apikey` 自己的迁移 |

另外两类依赖方向违规（16 处模块间 import / 8 文件 / 8 组模块对；5 处 `platform → module` 反向依赖：`platform/db/seed.go`、`platform/auth/apikey.go`、`platform/grpc/userinfo_service.go`）在 §5 与 §10 的 P1/P2 中消除。

## 5. 重点能力拆分

> 注：本节各表中的路径是**改造前的现状路径**（搬迁前位于 `internal/modules/` 下，表内多为相对写法）。P1-A 已把业务模块搬入 `internal/capabilities/…`，后续 P1 子计划继续拆分；表格保留原路径以便与当时的评审记录对照。

### 5.1 `auth`（2914 行）→ 6 个能力

| 现有位置 | 行数 | 去向 | 依据 |
|---|---|---|---|
| `service.go` Login / LoginWithTOTP / finishLogin / Refresh / Logout / LogoutAll / issueTokenPair / recordFailure | ~350 | `auth` | 会话与凭证本体 |
| `service.go` Register / checkRegistrationAvailable / ForgotPassword / ResetPassword / generateResetCode | ~180 | `auth` | 注册与改密是 auth 用例（邮件经 `notify` 端口） |
| `service.go` checkPasswordReuse / recordPasswordHistory（+ 迁移 012） | ~80 | `auth` | 密码策略属改密用例 |
| `service.go` SetupTOTP / EnableTOTP / DisableTOTP | ~75 | `mfa` | TOTP 是独立认证方式 |
| `service.go` checkBreachedPassword | ~20 | `breach` | 外部服务依赖，独立可关 |
| `application/trusted_device.go` + infra + domain + 迁移 013 | ~235 | `mfa` | "跳过 MFA"的能力 |
| `application/webauthn.go` + `interfaces/webauthn_handler.go` + infra + domain + 迁移 015 | ~800 | `passkey` | 无密码登录，与密码路径互斥 |
| `application/provisioning.go` | 195 | `tenancy` | 开通式注册是租户开户用例 |
| `application/login_history.go` + infra + 迁移 011 | ~120 | `auth` | 登录审计属登录用例 |
| `interfaces/handler.go` verifyCaptcha + `RegisterCaptchaRoute` + `platform/captcha` | ~170 | `captcha` | 已有独立 `/api/v1/captcha` 路由 |
| `interfaces/handler.go` allow / writeAuthRateLimitHeaders | ~30 | `auth` | 认证保护 |
| `NewAuthService(... deps ...interface{})` 与 `WithIssuer`/`WithPasswordHistory`/`WithTrustedDeviceTTL`/`WithWebAuthnSessionTTL` | — | 改为 `auth.Deps` 显式结构体 | 现在靠无类型变参 + type-assert 传依赖 |

### 5.2 `admin`（2430 行）→ 拆掉"路由命名空间"

`admin/module.go:101` 把所有东西挂在同一个 `/api/v1/admin` 组下，但其中装的是 7 个不同能力的用例：

| 现有文件 | 行数 | 去向 |
|---|---|---|
| `interfaces/users.go` + `application/users_service.go` | 403 | 合并进 `user`（§5.3） |
| `interfaces/jobs.go` + `application/tasks_service.go` + 3 个 job repository | ~430 | `jobs` |
| `interfaces/apikeys.go` + `application/apikeys_service.go` + `apikey_repository.go` | ~290 | `apikey` |
| `interfaces/import.go` + `application/import_service.go` + `import_jobs` | ~280 | `dataops` |
| `interfaces/audit.go` | ~60 | `audit` |
| `interfaces/features.go` | 59 | `featureflag` |
| `interfaces/config.go` + `application/config_service.go` | 138 | `console` |
| `interfaces/monitoring.go` + `application/monitoring_service.go` | ~150 | `console` |
| `interfaces/ws.go` | 87 | `console`（经 ws hub 端口） |
| `interfaces/ratelimit.go` | 77 | `console` |
| `interfaces/middleware.go`（IP 白名单） | 54 | `console` / kernel |

结果：`console` 从 2430 行缩到约 500 行，只保留真正无归属的平台级视图。**对外 URL 全部不变**，变的是代码归属与依赖方向。

### 5.3 `user` 双写合并

`/api/v1/users`（`user` 能力，254 行）与 `/api/v1/admin/users`（`admin` 能力，258 行）是两份独立实现、同一张表，隔离与配额逻辑各写一遍（`user/application/service.go:48 tenantAllowed` vs `admin/application/users_service.go:62 WithQuota`）。合并后由 `user` 能力同时提供自助面与管理面用例，共享同一 repository 与不变量（租户隔离 / 配额 / 缓存失效）。这顺带消除"两处实现漂移导致越权或配额失效"的真实隐患。

### 5.4 其余结论

- `tenancy` = 租户 CRUD + 套餐/配额/用量 + 开通式注册（从 `auth` 迁入；上下文机制在 `kernel/tenant`）；它不拥有 `tenant_id` 列本身，那些列由各表所有者维护
- `audit` 自包含（service/worker/export/chain/middleware），其 middleware 是全局写者 → 设计为"可选中间件提供者"，其他能力无需感知其存在
- `access` = role + permission + Casbin + 鉴权中间件 + `user_roles` 分配（从 `admin` 迁入）

拆分后除 `tenancy`（~1300 行，含原 `tenant` 模块的租户 CRUD + 套餐配额 + 开通式注册）外，没有任何能力超过 1000 行：`auth` ~900 / `passkey` ~800 / `console` ~500 / `jobs` ~430 / `apikey` ~290 / `dataops` ~280 / `mfa` ~250。

## 6. 三层机制

### 6.1 能力自描述（唯一元数据来源）

每个能力目录下一个 `capability.go`：

```go
var Capability = contract.Capability{
    Name:         "auth",
    Tags:         []string{"domain"},
    Requires:     []string{"user", "notify"},
    SoftRequires: []string{"captcha", "breach"},   // 缺失时降级
    Owns:         []string{"login_histories", "password_histories"},
    Config:       func() any { return DefaultConfig() },
    Migrations:   contract.Migrations{Mysql: "migrations/mysql", Postgres: "migrations/postgres"},
    Permissions:  []contract.Permission{{Resource: "/api/v1/auth/*", Action: "POST"}},
    Mount:        contract.MountPublic,
}
```

模块注册沿用已确认的 **`catalog` 显式清单**（不用 `init()` 自注册）：`internal/capabilities/catalog/catalog.go` 是全仓库唯一列出能力的地方，删能力 = 删目录 + 删一行。

### 6.2 层① 脚手架（建项目时）

- `jimu new <project> --profile=saas` 或 `--with=auth,apikey,console`
- 解析 `Requires` 闭包 → 只复制选中的能力目录、迁移、配置段、对应文档
- 生成专属 `catalog.go`（只含选中能力）与 `configs/app.yaml`（只含选中能力配置段）
- 生成后跑 `go mod tidy`，让 `go.mod` 收缩到真实依赖
- 生成 `compose-report.md`：代码行数 / 文件数 / `go.mod` 直接依赖数 / 二进制大小 / 路由数 / 迁移数 / 表数
- 复用现有 `tools/generator`：`jimu module create` 升级为 `jimu new` + `jimu capability add <name>`

### 6.3 层② 构建（编译时）

仓库内提供多个 profile 入口包，每个是独立 main 包，只 import 该形态需要的能力：

```text
profiles/
├── full/       main.go   → go build ./profiles/full
├── minimal/    main.go
├── saas/       main.go
├── enterprise/ main.go
└── machine/    main.go
```

用 **profile 入口包而不是 build tag**：裁剪由 import 图天然决定，不引入新的 tag 空间（现有 `sqlite`、`integration` 等 tag 已在用），IDE 与 CI 不需要组合矩阵。

层②能减小**二进制、启动路由数、迁移数、表数**，但**不减小仓库代码量与 `go.mod`**（Go 的依赖裁剪作用于整个 module）—— 后两者由层①负责。

### 6.4 层③ 运行时

`capabilities.enabled` 配置：启动时计算 `Requires` 闭包并**自动补齐**（只写 `["oauth"]` 会连带启用 `auth`/`user`/`role`/`tenant`），仅当出现未知能力名或清单内无法满足的依赖时才明确报错；关闭的能力不挂路由、不注册任务与事件、不执行迁移、不 seed 权限点；启动日志与管理端点输出最终启用清单。

三层语义为包含关系：**`enabled ⊆ profile 选中集 ⊆ 仓库目录`**，各有独立验收，不互相掩盖。

## 7. 迁移归属与存量桥接

- 目录：`capabilities/<name>/migrations/{mysql,postgres}/001_*.sql`，能力内自行编号
- 版本表：`goose_db_version_<capability>`，互不干扰；删能力 = 删它的表 + 迁移 + 版本记录
- 执行顺序：kernel 基线 → 按 `Requires` 拓扑序执行各能力
- 表归属（由现有 15 个迁移的实际内容确定）：`users`→`user`；`roles`/`permissions`/`role_permissions`/`user_roles`→`access`；`tenants`/`tenant_plans`→`tenancy`；`audit_logs`/`audit_chain_head`→`audit`；`api_keys`→`apikey`；`jobs`/`job_history`/`dead_letters`/`scheduled_jobs`→`jobs`；`import_jobs`→`dataops`；`search_documents`→`search`；`login_histories`/`password_histories`→`auth`；`trusted_devices`→`mfa`；`webauthn_credentials`→`passkey`；`user_oauth_bindings`→`sso`；`outbox_events`→`outbox`
- **存量实例桥接**：现有实例的 `goose_db_version` 记录的是全局 001–015。迁移搬迁后新增 `jimu migrate adopt-capabilities`：读取现状 → 为各能力版本表预置"已应用到对应版本"的基线 → 之后只跑新迁移。全新生成的项目无此包袱
- P1.5 归位注记：目录落位 `internal/capabilities/<name>/migrations/{mysql,postgres}/`；**已按原全局编号落位**——各能力迁移沿用全局 001–015 中的原编号（如 auth 011–015、queue 001/007），能力内编号只对新生成的能力从 001 起；adopt 基线按原编号对齐

## 8. 配置归属

`kernel/config` 只保留内核段（http / db / redis / log / otel / management / id / server / security / ratelimit 内核部分）。能力配置由 `capability.go` 的 `Config()` 声明默认值与校验，装配时合并；未选中或已关闭的能力，其配置段既不出现也不校验。这是 695 行中心配置（`auth` 23 行、`oauth` 22 行、`storage` 12 行、`queue` 13 行等）的解法。

现有 12 个分散开关（`oauth.providers.*.enabled`、`captcha.enabled`、`email.enabled`、`sms.enabled`、`grpc.enabled`、`upload.clamav.enabled`、TLS、`breaker.enabled`、`retention.enabled`、`ratelimit.enabled`、`auth.webauthn.enabled`、`auth.provisioning.enabled`）保留为能力内配置，但**能力级开关统一由 `capabilities.enabled` 表达**，避免两套机制语义打架。

## 9. 门禁与验收

| 门禁 | 检查内容 |
|---|---|
| `make check-capabilities` | `capability.go` 声明与实际一致：`Owns` 的表只出现在所有者迁移里、`Requires`/`SoftRequires` 与实际 import 一致、无未声明的跨能力 import、无 `capabilities/A → capabilities/B/internal` 越界、无 `kernel → capabilities` 反向依赖、中间件归属正确（`user`/`tenancy`/`apikey` 维度限流随对应能力，管理端准入随 `console`，`shared/totp` 已迁入 `mfa`） |
| `scripts/check-profiles.sh` | 每个 profile 都能 `go build`、启动、健康检查通过；输出该形态的路由数与迁移数 |
| 驱动与资产检查 | profile 与生成项目只 import 已声明的驱动包（§3.7）；未选中能力对应的部署资产、CLI 子命令、`docs/openapi` 生成物不出现（§3.8） |
| `scripts/check-pluggable.sh` | 删除任一能力目录后：依赖它的 profile **必须失败**（依赖校验生效），不依赖它的 profile **必须仍能 build** |
| `jimu new --report` | 生成项目的代码行数 / 文件数 / 直接依赖数 / 二进制大小进 CI 归档对比，防止"最小形态"悄悄变胖 |
| 回归基准 | `full` 形态必须保持现有全部测试通过、覆盖率 ≥70%、`make release-check COMPOSE_ENV=.env.example` 通过、`make compose-check` 通过 |

## 10. 分期实施

| 阶段 | 内容 | 完成判据 |
|---|---|---|
| **P0 契约与内核归位** | 定义 `contract.Capability` 与端口；建立 `internal/kernel/`；`internal/capabilities/` 下按现有 8 模块原样落位（先不改内部）；`catalog` 显式清单；运行时 `capabilities.enabled` + 依赖闭包校验；去掉 `bootstrap.go` 的 `auth`/`oauth` 字符串特判与"第一个中间件提供者"约定 | `full` 行为与 master 完全一致（测试全绿）；关闭 `oauth` 后其路由/迁移/权限点消失 |
| **P1 边界重划** | `auth` → 6 个能力（§5.1）；`admin` 拆散到各能力（§5.2）；`user` 双写合并（§5.3）；**平台混装包归位**（§3.6：拆 `platform/auth`、`platform/db`、`platform/http`，`platform/tenant` → `kernel/tenant`（上下文机制；租户实体/套餐/配额/开通式注册归 `tenancy` 能力），`conf/rbac_model.conf` → `access`，示例服务移出平台层）；表所有权与迁移搬迁 + `adopt-capabilities`；`platform → module` 反向依赖消除；中间件归位（§3.5.2）（执行拆分为子阶段：P1.1 命名空间搬迁 → P1.2 平台包归位 → P1.3 内核混装包拆分 → P1.4 auth 拆分与端口（已完成：grpc userinfo 经 `contract.UserinfoSource` 端口消费、apikey 模型迁入 `capabilities/apikey/domain`，`kernel/db/seed.go` 的 kernel→capabilities import 留待 P1.5）→ P1.5 种子/迁移归属（已完成：迁移落位 `capabilities/<name>/migrations/{mysql,postgres}/` 并经 embed 进二进制，运行器按能力走独立版本表 + adopt 基线登记，种子迁至 `internal/app`、权限点由 Descriptor 声明，kernel→capabilities import 归零；迁移沿用原全局编号）→ P1.6 auth 六能力（已完成：auth/mfa/passkey 三能力落位 + breach/captcha catalogize + 开通式注册迁入 tenant；TOTP 由 `users.totp_*` 迁到 mfa 自有 `user_mfa` 表（迁移 016，密文原样搬迁、可回滚）；五条 contract 端口 MFAVerifier/LoginFinalizer/TenantProvisioner/BreachChecker/CaptchaVerifier 消除跨能力 import）→ P1.7 admin 拆散与 user 合并（已完成：`role`+`permission` 合并为 `access`（四表所有者 + `contract.UserRoleAssigner`）；`/api/v1/admin/*` 路由按用例归还 user/queue/apikey/dataops/audit/feature/uploadsec，平台级视图与管理端准入中间件归新能力 `console`（声明 `/api/v1/admin/*` 通配权限点）；`/admin/users` 并入 `user`，管理面与自助面共用同一 repository/配额与租户可见性；`admin` 能力目录删除；对外 URL 不变，e2e 以路由对齐用例钉住）；P1.1–P1.3 已合入 release/v0.3.0；P1.4–P1.7 已完成） | 16 处模块间 import 归零；迁移按能力归属并各有版本表；存量实例可平滑 adopt；`platform/` 下不再有跨能力的混装包 | 16 处模块间 import 归零；迁移按能力归属并各有版本表；存量实例可平滑 adopt；`platform/` 下不再有跨能力的混装包 |
| **P2 三层机制** | `capabilities.enabled` 配置合并与校验；`profiles/{full,minimal,saas,enterprise,machine}` 入口包；**驱动级可插拔**（§3.7：`storage/{local,s3}`、`queue/{redis,kafka,rabbitmq}`、`dataops/{csv,excel}`）；**非代码资产模块化**（§3.8：deploy 资产、Helm values、CLI 子命令、契约测试）；`jimu new` / `capability add` 脚手架；`compose-report` | 5 个 profile 均能构建并启动；`minimal` 的报告数字显著低于 `full`；只用本地存储/Redis 队列/CSV 时对应重型依赖不出现 |
| **P3 门禁与文档** | `check-capabilities` / `check-profiles` / `check-pluggable`；生成器模板同步新形态；README / CONTRIBUTING / AGENTS.md 更新（能力清单、形态、新增能力流程） | 四道门禁在 CI 生效 |
| **P4 v0.3.0 收尾** | 版本日志补验证结果；`release-check`；`release/v0.3.0` → `master` 合并；打 tag 发布 | GitHub Release 发布成功 |

P0 完成后即可供其他 feature 分支并行开发，P1–P3 逐步收敛。

> **P2 进展（P2.1 运行时配置归属已完成）**：§8 的配置归属已落地 —— 能力配置段由能力在
> `Descriptor.Configs`（`contract.ConfigSpec{Section, New}`）声明，段实例实现
> `ApplyDefaults()`/`Validate()`，生产加严走可选 `ValidateProd()`；组合根
> `app.LoadCapabilityConfigs` 按启用集统一执行「解码 → 默认值 → 校验」，**未启用能力的配置段
> 既不出现也不校验**。`internal/config` 只保留内核段，对外 YAML 键逐一不变。`auth` 段按 §8 ¶2
> 整体归属 `auth` 能力（含 `auth.webauthn`/`auth.provisioning`），不拆段；`mfa`/`tenant` 不 import
> `auth`，其 JWT/开通模板参数由组合根装配期传递。非 catalog 包（`storage`/`notification`/
> `retention`）的段暂由组合根显式加载，**是否 catalogize 是 P2.2 的决定点**（P2.2 裁定推迟到
> P2.5/P2.6，见下条）。子阶段拆分与执行
> 记录见 [`docs/plans/2026-09-21-p2-three-layer-mechanism.md`](../plans/2026-09-21-p2-three-layer-mechanism.md)
> 与 [`docs/plans/2026-09-21-config-ownership.md`](../plans/2026-09-21-config-ownership.md)。
>
> **P2 进展（P2.2 能力自描述契约 + P2.3 层③ 运行时已完成）**：§6.1 的自描述契约落地 ——
> `contract.Descriptor` 新增 `SoftRequires`（可选依赖：目标缺失只降级，**不自动补齐、不参与拓扑序**）
> 与 `Owns`（本能力迁移 `CREATE` 的表），`Descriptor` 由此成为能力元数据的唯一来源（启用闭包、
> 配置段加载、权限点种子、路由挂载与门禁都只读它）。`catalog.ValidateDeclarations()` 在每次
> `catalog.Resolve` 前校验声明自洽（必须是清单内能力名、不自引用、不与 `Requires` 重叠、不重复），
> `catalog.Degraded(caps)` 计算已解析启用集的降级项；启动时对每个降级项打 `capability degraded`
> warn（字段 `name`/`names`），管理端口新增只读不鉴权的 `GET /capabilities`，返回
> `{"enabled":[…],"degraded":[{"capability":…,"missing":[…]}]}`（`HealthRouter` 因此加了可变参数
> `extra ...func(*http.ServeMux)`，内核包不必 import 能力）。18 个能力逐个补齐声明（`Owns`：13 个
> 有表、5 个无表），真实软依赖为
> `user`→`access`/`tenant`、`access`→`tenant`、`mfa`→`auth`、`auth`→`captcha`/`breach`、
> `apikey`→`tenant`、`outbox`→`queue`。降级清单是**声明层**的静态比对（只读 `Descriptor`，不观测
> 运行时装配）：组合根当前仍无条件注入多数依赖，在其改为按启用集驱动（P1 显式 `Deps`）之前可能
> 多报。§9 门禁落地第一块 `make check-capabilities`（`tools/checkcapabilities`）：
> 校验 `Owns` ↔ mysql 迁移「单表唯一归属、无孤儿表、无未声明建表」（PostgreSQL 迁移表名与 mysql
> 一致，暂以 mysql 为准），其余三道门禁留 P2.8。
>
> **P2.2/P2.3 裁定与推迟**：`Tags` 在出现真实消费方之前**不加**（避免纸面字段）；
> `storage`/`notification`/`retention`/`ws`/`grpc`/`apidocs`/`encryption` 本轮保持**非 catalog**、
> 由组合根显式装配（能力清单仍为 18 项，`configs/*.yaml` 零改动），是否 catalogize 连同 profile
> 入口包一起在 **P2.5/P2.6** 定夺（与 P2.1 的裁定 2B 一致）。执行记录见
> [`docs/plans/2026-09-22-p2-contract-and-runtime.md`](../plans/2026-09-22-p2-contract-and-runtime.md)。

## 11. 风险与取舍

- **破坏内部 API**：`Deps` 结构体、构造函数、包路径、目录都会变。对外 HTTP API 与配置键不变，内部一次性重构，v0.x 允许破坏
- **`adopt-capabilities` 是存量实例的唯一门槛**：迁移搬迁必须先把版本基线登记做对，否则会重跑建表语句。实现时以真实 MariaDB / PostgreSQL 实例验证"adopt 后不重跑、后续新迁移正常应用"
- **`tenant_id` 列与 `version` 列保留**：`tenancy` 关闭时列仍在（单租户模式 `tid=0` 平台级视角）。要得到"干净的单租户 schema"需要双份迁移，本版本不做（已在 §1 非目标中排除）
- **`mfa` 与 `user` 的表拆分需要数据迁移**（TOTP 密钥从 `users.totp_secret` 迁到 `user_mfa`）：迁移需幂等且可回滚
- **层②的边界**：profile 入口不减小 `go.mod`，文档必须说清楚，避免"以为换个 profile 依赖就少了"的误解
- **能力间 `SoftRequires` 降级路径要逐条测试**：`auth` 缺 `captcha`/`breach`、`notify` 缺真实渠道等，否则"可勾选"只是纸面能力
- **P1 是工作量大头**（`auth` 2914 + `admin` 2430 行重排，外加 `platform/auth`/`platform/db`/`platform/http` 三个混装包拆分）：建议 P0 尽早合并，P1 按能力逐个 PR，每个 PR 都保持 `full` 全绿
- **驱动级拆分把"编译期可见"换成"显式 import"**：驱动注册从同包 `switch` 变为"显式 import + 注册"，漏 import 会在**运行时**报"未知驱动"而不是编译失败 → 用两道保险：`check-capabilities` 静态校验"能力声明的驱动集合 = profile 实际 import 的驱动包"，启动时校验配置里的驱动已在编译期注册并给出明确错误
- **`minimal` 的"省"要防止被内核吃掉**：`httpclient`/`breaker`/`event`/`crypto` 类基础库会随内核恒在，若内核继续膨胀，"最小形态"的报告数字就降不下来 → 报告进 CI 归档对比是最直接的刹车

## 12. 未决事项

- 能力目录内是否统一强制 `domain/application/infrastructure/interfaces` 四层：`passkey`(~800) 与 `console`(~500) 是否需要，待 P1 落位时按实际规模决定
- `profiles/*` 的默认形态与命名是否需要暴露给使用者（`jimu new` 的 `--profile` 取值），P2 定
- 驱动级拆分是否需要更细（`storage` 现有 local/s3 两个驱动，OSS 与 MinIO 未实现；`queue` 是否需要把死信存储也纳入驱动抽象）
- 是否引入 `examples/` 目录承接移出平台层的示例服务（`userinfo` gRPC 示例）
