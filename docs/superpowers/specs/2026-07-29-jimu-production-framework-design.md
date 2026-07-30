# Jimu 生产级通用后端框架设计

> 日期：2026-07-29
> 状态：设计已确认，等待实现计划

## 1. 目标

将 Jimu 完善为面向 Docker Compose 和单机部署的生产级、可复用 Go 后端框架。框架同时支持企业内网中后台与公网用户系统，除 MariaDB 和 Redis 外不增加必需基础设施。

本次允许不兼容调整。继续保留 `/api/v1` 和 `{code,message,data}` 响应外形，但 HTTP 状态码、业务错误码、健康检查路径、JWT claims、配置项和 CLI 行为可以调整。

## 2. 范围

按四个可独立验证的阶段交付：

1. 运行时与安全基线。
2. 工程质量与 API 契约。
3. 可靠事件、后台任务与可观测性。
4. 配置、脚手架、容器加固与发布交付。

本设计明确不包含：

- Kubernetes 专属行为。
- 多租户。
- 插件系统。
- NATS、RabbitMQ、Kafka 或其他消息队列。
- 强制部署外部可观测性服务。

## 3. 总体架构

主 `server` 进程负责公共 API、管理 API、定时任务、审计写入和 Outbox Dispatcher。独立的 management HTTP server 在 Compose 私有网络中提供健康检查和诊断能力。

MariaDB 保存业务数据、RBAC 数据、审计记录和持久化 Outbox 事件。Redis 保存 refresh token 会话、登录与注册限流状态、缓存和临时后台任务状态。

所有长时间运行的组件遵循统一生命周期：使用 context 启动，停止接收新工作，在配置的期限内排空进行中工作，最后释放资源。应用启动过程显式返回错误，基础设施层不使用 `panic` 处理可预期错误。

关闭顺序为：

1. 将 readiness 标记为不可用，并停止接收新 HTTP 请求。
2. 等待进行中的 HTTP 请求结束。
3. 停止 Scheduler 和 Outbox 拉取。
4. 排空审计与后台任务队列。
5. 关闭 Redis 和数据库连接。
6. 刷新并关闭日志。

## 4. 运行时与安全基线

### 4.1 健康检查与管理端点

- `/livez` 只反映进程是否存活，不探测外部依赖。
- `/readyz` 使用有界超时探测 MariaDB 和 Redis。
- 必需依赖不可用时，`/readyz` 返回 HTTP 503。
- Prometheus metrics 和 pprof 只由 management server 提供。
- pprof 默认关闭，必须通过显式配置开启。
- 生产 Compose 不向宿主机发布 management server 端口。

### 4.2 生产配置校验

出现以下情况时，生产环境拒绝启动：

- JWT 密钥缺失、使用已知占位值或不满足最低强度要求。
- 数据库或管理员凭证仍使用开发默认值。
- 危险诊断端点已开启，但没有 management 网络隔离。
- 必需端口、超时或依赖地址不合法。

配置错误只指出无效配置键，不输出其值。

### 4.3 认证模式

公开注册由配置控制，默认关闭。管理员创建账号和 RBAC 在两种模式下始终可用。

公开注册开启时：

- 按 IP 和规范化账号名限制注册与登录频率。
- 认证失败统一使用对外凭证错误，避免账号枚举。
- 禁用或锁定账号不能获得 Token。

access token 和 refresh token 都包含 issuer、签发时间、过期时间、唯一 token ID、用户 ID 和明确的 token type。Redis 只保存 refresh token 状态的哈希。刷新成功后，提交的 refresh token 立即失效并签发一组新 Token；再次使用已失效 Token 时拒绝该会话。logout 撤销当前会话，用户级操作可以撤销该用户全部会话。

### 4.4 HTTP 边界

显式配置 Header、Read、ReadHeader、Write、Idle、Request 和 Shutdown Timeout。限制请求体大小，校验可信代理，并使用 allow-list CORS。公共 API 限流状态保存在 Redis 中，进程重启后仍然有效。

### 4.5 审计安全

审计记录不假设请求已经认证。匿名请求使用空用户字段，不得因类型断言触发 panic。密码、Authorization Header、access token、refresh token 和配置的敏感字段不得持久化。

审计写入使用有界进程内队列，并批量写入 MariaDB。队列溢出和写入失败必须增加指标并输出结构化错误日志。关闭时在应用 shutdown deadline 内排空队列。

## 5. 工程质量与 API 契约

### 5.1 测试分层

- domain/application 测试通过聚焦 fake 覆盖认证、用户状态、RBAC、refresh rotation、错误映射和服务行为。
- HTTP 测试通过 `httptest` 覆盖参数校验、状态码、响应外形、中间件顺序、认证和授权。
- 集成测试使用 Docker Compose 中的 MariaDB 和 Redis，覆盖迁移、repository、refresh session、审计批写与 Outbox。
- smoke test 覆盖配置校验、迁移、管理员初始化、服务启动、健康检查、登录、Token 刷新和受保护 API。

CI 执行格式检查、`go vet`、固定版本的 `golangci-lint`、带 race detector 的单元测试、集成测试、构建、Docker 镜像构建和 smoke test。覆盖率门槛优先应用于新增的安全敏感包，不以失真的全仓库高覆盖率为目标。

### 5.2 API 规则

- 公共 API 保持在 `/api/v1` 下。
- 响应继续使用 `{code,message,data}` 外形。
- HTTP 状态码表达协议结果，稳定业务错误码表达业务结果。
- 对外错误不包含 SQL、文件路径、堆栈或包装后的基础设施错误。
- 分页、排序和过滤使用统一契约，并限制最大页大小。
- Request ID 写入响应 Header，并贯穿日志、审计、错误响应和可用时的 metrics exemplar。
- CI 重新生成 OpenAPI；生成结果与仓库文件不一致时失败。

## 6. 可靠事件与后台任务

### 6.1 Transactional Outbox

业务变更与对应事件在同一个 MariaDB 事务内写入。Outbox 记录包含稳定 event ID、事件类型、版本、payload、状态、重试次数、下次执行时间、创建时间和处理元数据。

Dispatcher 领取待处理事件、调用已注册消费者并标记成功。失败时使用有界指数退避；超过最大重试次数后进入 dead 状态。CLI 支持查看 dead 事件，并显式重试选定事件。

消费者使用 event ID 作为幂等键。首版只运行一个 Dispatcher，但领取逻辑应避免未来出现第二实例时并发处理同一事件。

进程内 EventBus 仅用于崩溃时允许丢失的本地通知，不作为可靠集成设施。

### 6.2 Scheduler 与任务执行

模块通过现有 Module 契约注册固定周期任务。任务接收 context、超时、结构化日志字段和生命周期控制。必须跨重启保存或重试的工作通过 Outbox 或 Redis 表达，不使用无跟踪的 goroutine。

应用 bootstrap 必须为每个模块调用 HTTP、Job 和 Event 注册。注册失败时，在服务进入 ready 状态前终止启动。

## 7. 可观测性

management server 提供 Prometheus 格式指标：

- HTTP 请求数、状态和耗时。
- MariaDB 连接池状态与探测失败。
- Redis 操作与探测失败。
- 认证失败、Token 重用和限流决策。
- 审计队列深度、丢弃数量、批写失败和排空耗时。
- Outbox pending、retrying、dead 数量和处理耗时。
- Scheduler 执行次数、失败数和耗时。

OpenTelemetry tracing 默认关闭。开启后通过 OTLP 导出，不要求默认 Compose 增加服务。JSON 日志统一包含 request ID、可用时的 trace ID、可用时的 user ID、模块、业务错误码和操作名。

可选的 `observability` Compose profile 启动 Prometheus 和 Grafana。默认生产 profile 仍只包含 Jimu、MariaDB 和 Redis。

## 8. 配置与运维

配置仅使用 YAML 默认值与 `JIMU__...` 环境变量覆盖。生产 YAML 不依赖隐式 `${VAR}` 展开。`jimu config check` 加载并校验最终配置，只输出脱敏值。

数据库迁移继续使用 Goose。生产降级要求显式确认参数。Seed 只允许开发和测试环境执行。管理员初始化改为独立、幂等命令，密码来自受保护环境变量或交互输入，不提供默认管理员密码。

## 9. 模块生成器

`jimu module create <name>` 在写文件前校验模块名、仓库根目录和所有目标路径。任何目标已存在时，命令整体失败，不产生部分输出，也不覆盖用户文件。

默认生成可编译的 entity、repository 接口、service、Create/Get handler、router、migration 和聚焦测试。只有显式使用参数时才生成 List、Update 和 Delete。生成代码不得包含未实现 CRUD handler 或空 DTO。

Golden tests 在临时仓库运行生成器，格式化输出，并验证生成模块能够编译且测试通过。

## 10. 容器与发布交付

- Runtime image 固定基础镜像版本，并使用非 root 用户。
- 应用兼容只读根文件系统，仅通过声明的 volume 或 tmpfs 写入必要路径。
- MariaDB、Redis、management 端点和 Adminer 默认不公开；开发 profile 可以显式发布端口。
- CI 执行 `govulncheck`、Trivy 镜像扫描、SBOM 生成、依赖许可证报告和镜像 smoke test。
- 关键 Action 与工具固定版本，不跟随无边界的 `latest`。
- Release artifact 包含校验和与构建元数据。
- 通过 linker flags 注入 version、commit 和 build time，并由 `jimu version` 输出。

README 提供可直接执行的本地开发、生产 Compose、公开注册、内网中后台和可选 observability profile 流程。

## 11. 交付顺序

1. 修复审计 panic，建立生命周期和配置基础能力。
2. 增加 management 健康检查并保护诊断端点。
3. 实现 refresh token session、认证模式和 Redis 限流。
4. 增加 API 契约测试与核心集成测试。
5. 接通 Module 的 Job/Event 注册并实现 Outbox。
6. 增加 Prometheus metrics 与可选 OpenTelemetry tracing。
7. 用可编译、经过测试的输出替换不完整生成器。
8. 加固 Compose、CI、Release artifact 和运维文档。

每一步完成后，仓库都必须保持可构建和可独立验证。

## 12. 验收标准

在只安装 Docker Compose 和 Go 的干净主机上：

1. 配置校验能够拒绝文档列明的不安全生产值。
2. Compose 启动 Jimu、MariaDB 和 Redis，生产 profile 不公开私有服务端口。
3. 迁移和管理员初始化成功完成。
4. `/livez` 正常返回；依赖可用性变化时，`/readyz` 能在 200 与 503 之间正确变化。
5. 内网中后台模式与公开注册模式的 smoke test 都通过。
6. Refresh token 轮换、logout、重用拒绝和用户级全部撤销通过自动化测试。
7. 审计与 Outbox 在配置的 shutdown deadline 内完成排空。
8. 单元、HTTP、集成、race、lint、漏洞、构建、生成器、镜像和 smoke 检查通过。
9. 生成的 OpenAPI 与仓库提交文件一致。
10. 新生成模块无需手工修改即可格式化、编译并通过测试。
