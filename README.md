# Jimu Backend Framework

Go 语言通用后端基础框架 — 稳定底座 + 可组合模块 + 标准适配器 + 脚手架生成能力。

## 特性

- **模块化架构** — Clean Architecture 分层，业务逻辑依赖接口不依赖实现
- **统一认证** — typed JWT + Redis refresh session + Casbin RBAC v3 权限模型；API Key 认证（服务/机器间调用，`X-API-Key` 头 + `auth.APIKeyAuthMiddleware` + `auth.RequireScope` scope 校验，复用 `api_keys` 表并按 `tenant_id` 归属租户，认证后自动注入租户上下文；能力标签与 Scope 约定见 [API Key 与 Scope](#api-key-与-scope)）
- **租户体系** — 单归属多租户：`tenants` 表 + 租户 CRUD API（`/api/v1/tenants`），`users`/`roles`/`audit_logs`/`api_keys` 携带 `tenant_id` 做行级隔离，任务队列（`jobs`/`job_history`/`dead_letters`）与导入任务（`import_jobs`）同样归属租户；租户身份写入 JWT claim（`tid`）经中间件注入请求上下文，不接受客户端 header 传入；存量数据迁移时归入默认租户（`code=default`），角色名唯一性为租户内唯一，用户名/邮箱保持全局唯一（登录无需传租户标识）；归属关系为**租户 1:N 用户、用户单归属且不可跨租户**（见 [归属模型](#归属模型)）
- **开通式注册** — 可选的 SaaS 语义（`auth.provisioning.enabled`）：注册即单事务开通新租户，注册者成为 owner，按可配置的角色模板自动初始化租户角色与全局权限绑定（模板模式，全部可配置：开关/owner 角色/角色与权限模板）；未启用时注册用户归默认租户
- **密码重置** — 邮箱验证码自助重置（`POST /api/v1/auth/forgot-password` + `reset-password`），6 位数字码 Redis 一次性存储，防用户枚举，重置后强制登出全部会话
- **敏感信息脱敏** — `platform/mask` 提供手机号/邮箱/身份证/银行卡/姓名/IP 等脱敏函数与按字段名判定（`RedactByKey`/`Map`）；日志链路（文件、stdout、OTLP 导出）统一接入，凭证类字段整体替换为 `***`、PII 部分保留，避免明文落盘
- **敏感字段加密** — AES-256-GCM 字段级加密 + HMAC-SHA256 盲索引（email/phone，`security.encryption_key` 配置后启用；未配置时明文模式，功能不受影响）
- **OAuth 登录** — Google/GitHub/微信第三方登录，`oauth.providers` 配置开关
- **图形验证码** — 登录/注册验证码，Redis 存储一次性校验，`captcha.enabled` 配置开关
- **统一响应** — 标准 `{code, message, data}` 格式 + 分页
- **多环境配置** — Viper + yaml + 环境变量覆盖，枚举值启动校验
- **结构化日志** — Zap + lumberjack 自动滚动
- **数据库迁移** — Goose 迁移 CLI (up/down/status/redo)
- **数据初始化** — Seed 命令一键插入管理员和基础权限（含 Casbin 策略同步）
- **限流保护** — 全局令牌桶（IP）+ Redis 登录/注册固定窗口 + 用户/租户/API Key 维度滑动窗口（租户维度全局挂载、平台级视角跳过；API Key 维度按路由挂载且以 Key ID 计数，不落明文）；并发上限负载保护（`server.max_concurrency`，超限可短排队后返回 `1010`/503，避免过载雪崩）
- **HTTP 安全边界** — 请求体大小、超时、可信代理、CORS、安全 Headers；**IP 白名单**（`security.ip_allowlist` 全局 + `security.admin_ip_allowlist` 管理端，CIDR/单 IP，启动校验，客户端 IP 依赖可信代理配置）；CSRF 防护（配置 `security.csrf_secret` 启用，Bearer 请求自动跳过）；API 签名验证中间件（可选，服务间调用按需挂载）
- **缓存抽象** — Cache-Aside 模式，GetOrSet 自动回填；两级防击穿（进程内 singleflight 合并同 key 并发回源 + Redis 分布式锁跨实例互斥，锁异常时直接回源兜底）
- **自定义校验** — 手机号、密码强度、身份证、用户名等常用规则
- **国际化** — 按 `Accept-Language` 返回中文/英文错误与校验消息
- **事件总线** — 内存实现，支持同步/异步发布订阅
- **多队列支持** — Redis/Kafka/RabbitMQ 统一队列接口，`queue.type` 切换；三者均为 at-least-once：Redis（BLMove 原子消费 + 可见性超时重入队 + 延迟队列）、RabbitMQ（autoAck=false + requeue + 断连重投）、Kafka（FetchMessage 不自动提交 + Ack 显式 CommitMessages，崩溃重启重投未提交区间）。消费幂等：已成功/死信任务重复投递时 Ack 跳过，避免业务副作用重复执行（outbox 事件无状态机，不做去重）。失败任务按指数退避延迟重投（Redis 延迟队列），耗尽重试入死信表（`dead_letters`，可经管理 API 查询与标记解决）。任务归属提交者租户（`jobs.tenant_id`），消费时恢复该租户到执行上下文，管理端任务/死信接口按租户隔离
- **事务封装** — 统一的事务管理 helper
- **审计日志** — 有界队列批量写入，匿名请求安全处理；**防篡改哈希链**：每条审计按租户写入 `prev_hash`/`entry_hash`（`audit.hash_secret` 配置时用 HMAC-SHA256，否则 SHA-256），写入时锁定链头行保证多实例全序；`GET /api/v1/audits/verify` 可按范围重算校验，检测内容篡改、链接断裂与链尾截断
- **管理端点** — 独立 management server 暴露健康检查、metrics 和可选 pprof
- **管理 API** — 系统状态、在线用户、强制下线、错误码文档
- **脚手架** — Cobra CLI 一键生成完整模块骨架
- **API 文档** — Swagger UI 交互式文档（中文注释）
- **健康检查** — `/livez` 与 `/readyz`，readiness 有界探测 DB + Redis
- **优雅停机** — 显式 Application 生命周期，反向停止组件
- **分布式锁** — Redis 实现的分布式锁（防并发、选主）
- **文件存储** — 本地/S3/OSS/MinIO 统一接口
- **上传安全** — 文件大小限制 + magic-byte 嗅探覆盖可伪造的 Content-Type 头 + MIME 白名单；可选 ClamAV 病毒扫描（`upload.clamav.enabled`，stdlib 实现 INSTREAM 协议，落库前同步扫描，fail-closed：不干净或扫描不可达均拒绝落库）
- **数据导入/导出** — CSV/Excel 模板解析、校验与导入/导出（`internal/platform/importer` / `internal/platform/exporter`）；通用 importer 保留 `Importer.Import`，通过可选逐行 `RowSink` 注入持久化，未配置时明确报错，业务应用负责事务落库，导出结果可被导入器回读验证；管理端用户导入按操作者所在租户归属（无租户上下文时归默认租户），不产生未归属数据
- **历史数据保留** — `platform/db` 保留服务按表分批硬删除过期历史数据（`audit_logs`/`jobs`/`job_history`/`dead_letters`/`outbox_events`/`import_jobs`），挂在定时任务上（`retention.enabled`，默认关闭）；只清理终态记录（已发布事件、已处理死信、已结束任务），指标 `jimu_retention_deleted_total`
- **全文检索** — `platform/search` 统一接口（`Index`/`Delete`/`Search`）+ 公共索引表 `search_documents`：MySQL 走 FULLTEXT（`MATCH ... AGAINST`），PostgreSQL 走 `tsvector` 表达式 GIN 索引；按 `tenant_id` 隔离，`(tenant_id, doc_type, doc_id)` 唯一保证幂等覆盖。中文分词需数据库侧扩展（MySQL ngram / PG zhparser）
- **通知系统** — 邮件/短信(SMS)/WebSocket/Webhook 抽象；短信支持阿里云（dysmsapi SDK，`sms.enabled` 配置开关）；Webhook 回调载荷支持 HMAC-SHA256 签名（`notification.webhook.sign_secret`，附加 `X-Jimu-Timestamp`/`X-Jimu-Signature` 头，防重放）
- **统一出站 HTTP client** — 封装 timeout + retry/backoff（仅网络错误与 5xx）+ 熔断（连续失败自动开启，冷却后探测恢复）+ 按目标 host 独立限流（令牌桶）+ OTel `traceparent` 注入（`internal/platform/httpclient`），OAuth 提供商与 Webhook 共用
- **依赖熔断** — 统一熔断器 `internal/platform/breaker`（连续失败阈值 + 冷却后半开探测）接入 Redis（命令/连接级 hook）与 DB（语句级 `ConnPool`），依赖不可用时快速失败而非每请求等超时；只把连接/网络类错误计为失败（Redis 未命中与业务错误、DB 慢查询超时都不触发）；指标 `jimu_breaker_open` / `jimu_breaker_rejected_total` / `jimu_breaker_trip_total`。DB 在启用读写分离时自动跳过（dbresolver 管理独立连接池，已日志提示）
- **Outbox 模式** — 事件发布与数据库事务一致性保证，支持 MQ 跨服务发布（`outbox.publisher` 切换；`mq` 模式下通过 WorkerPool 消费事件，`event_bus` 模式通过 `outbox:*` 桥接器注入事件总线）
- **定时任务** — Cron 调度器（robfig/cron），支持 MySQL 持久化（`scheduler.store=mysql`）与多实例分布式锁协调，启动时通过 `RestoreFromStore` 恢复持久化任务（内置任务去重）
- **Feature Flag** — 运行时特性开关（灰度百分比、白名单）
- **OpenTelemetry 可观测性（OpenObserve）** — 统一 OTLP gRPC 输出：分布式追踪（HTTP/Gin + Gorm 查询 + Redis 命令全链路 span，队列/Outbox 异步边界透传 `traceparent`/`tracestate`）、Prometheus 指标转 OTLP 推送、结构化日志异步推送（`otel.enabled` 开启）
- **Prometheus 指标** — DB 连接池 + 运行时 + HTTP 请求指标（`jimu_http_*`）+ 队列执行/死信（`jimu_queue_*`）+ Outbox 发布（`jimu_outbox_*`）+ 出站熔断（`jimu_httpclient_*`）+ 定时任务执行（`jimu_scheduler_*`，成功/失败计数 + 耗时分布）；Management `/metrics` 暴露 Prometheus 格式，`otel.metrics_enabled` 时定期转 OTLP 推送 OpenObserve
- **gRPC server** — 与 HTTP 双栈并存，内置健康检查（`grpc_health_v1`）与反射（grpcurl 可探），可选启用（`grpc.enabled`，默认端口 9091）；业务示例 `UserInfoService` 演示 proto 定义 → `make proto` 生成 → 服务实现 → 注册全流程，业务模块经 `RegisterService` 接入
- **PostgreSQL 支持** — `db.driver=postgres`（或 `DB_DRIVER=postgres`）切换，迁移文件独立于 `migrations/postgres/`，与 MySQL 语法（`BIGINT UNSIGNED`/`ENGINE=InnoDB`/`ON UPDATE`）完全隔离；连接、迁移、seed、JWT/RBAC 全链路已用真实 PG 17 验证
- **分布式 ID** — 雪花 ID 生成器（`internal/shared/id`），所有数据库主键由应用生成，`id.worker_id` 配置多实例唯一编号
- **Docker 支持** — Dockerfile + docker-compose 一键起服务
- **Docker Secrets** — 敏感配置通过文件注入（`_FILE` 后缀）
- **K8s 部署** — Deployment/Service/HPA/Ingress manifests
- **CI/CD** — GitHub Actions + Dependabot 自动化 + 测试覆盖率门禁
- **安全扫描** — govulncheck 依赖漏洞扫描（纳入 `make release-check` 与 CI）、Trivy 镜像扫描、SBOM 生成与镜像 smoke test
- **静态检查** — golangci-lint + pre-commit 钩子（fmt / vet / golangci-lint）
- **追踪关联** — 访问日志自动注入 trace_id / span_id，关联 OpenTelemetry 追踪
- **Redis 高可用** — `redis.mode` 支持 `single` / `sentinel` / `cluster` 三种部署模式（默认 single 行为不变）：哨兵模式通过 `master_name` + `sentinel_addrs` 自动故障转移，集群模式通过 `cluster_addrs` 连接分片；统一 `redis.Client` 接口，框架内 session/缓存/队列/限流/分布式锁全复用
- **TOTP 二次验证** — RFC 6238 自研实现（`internal/shared/totp`，无外部依赖），用户可自助绑定/启用/关闭：`POST /auth/mfa/setup` 生成密钥与 otpauth URI（二维码绑定）、`/auth/mfa/enable` 首次验证码确认、`/auth/mfa/disable` 校验后关闭；启用后登录必须携带 `totp_code`（缺失 `2006`，错误 `2007`），密钥 AES-GCM 字段级加密落库
- **统一 gRPC 客户端** — 出站调用封装（`internal/platform/grpc` `Client`）：连接管理 + 调用超时 + 指数退避重试（仅 Unavailable/ResourceExhausted 幂等安全码）+ panic 恢复拦截器 + Prometheus 指标（`jimu_grpc_client_*`），支持 TLS/insecure，与 HTTP client 对齐的框架风格
- **登录历史** — 每次登录尝试落库 `login_histories`（成功/失败/锁定 + 原因 + IP + User-Agent，账号不存在也记录用户名），`GET /api/v1/auth/login-history` 供用户自助排查异常登录；写入失败只记日志，不影响登录主流程
- **错误追踪上报** — `internal/platform/reporter`：结构化错误日志（含 trace_id/span_id），HTTP `Recovery` 中间件 panic 自动上报；日志链路接入 OpenObserve 后错误自动汇聚，配合 OpenObserve 告警覆盖错误监控场景（`error_reporting.enabled` 开关）

## 技术栈

| 类别 | 选型 |
|------|------|
| HTTP 框架 | Gin |
| ORM | Gorm |
| 数据库 | MariaDB / MySQL / PostgreSQL（`db.driver` 切换） |
| 缓存 | Redis |
| 配置 | Viper |
| 日志 | Zap + lumberjack |
| 鉴权 | JWT + Casbin v3 |
| 迁移 | Goose |
| CLI | Cobra |
| 校验 | go-playground/validator |
| API 文档 | swaggo/swag |
| 追踪 | OpenTelemetry |
| 指标 | Prometheus client_golang |
| 调度 | robfig/cron |

> 注：`go.mod` 中的 `github.com/ClickHouse/clickhouse-go/v2` 与 `gorm.io/driver/clickhouse` 为 `gorm.io/plugin/opentelemetry`（Gorm 追踪插件）的传递依赖，框架本身未直接使用 ClickHouse。

## 快速开始

### 前置条件

- Go 1.26+
- MariaDB 10.5+
- Redis 6+

### 安装

```bash
git clone https://github.com/your-org/jimu.git
cd jimu
go mod download
```

### 配置

```bash
cp .env.example .env
# 编辑 .env 修改数据库、Redis、OpenObserve、OTEL 推送等连接信息
```

`.env.example` 覆盖全部可调环境变量（镜像/端口/数据卷、OpenObserve 账号与端口、OTEL 开关与端点）：
- **docker compose** 自动读取项目根 `.env` 做变量插值（也可用 `--env-file` 指定）
- **make** 通过 `include .env + export` 将变量注入子进程（deploy 脚本、compose 命令）
- docker-compose.yml 内的 `${VAR:-默认}` 均可在 `.env` 覆盖，缺省用 yml 内默认值

### 方式一：本地运行

```bash
# 1. 启动依赖
docker compose up -d mariadb redis

# 2. 运行迁移
make migrate

# 3. 初始化数据（管理员密码通过环境变量提供）
ADMIN_PASSWORD=admin123 make seed

# seed 写入：默认租户（code=default）、基础权限（含租户管理）、超级管理员角色、admin 用户

# 4. 启动服务
make run
```

### 方式二：Docker Compose 一键启动

```bash
# 1. 构建镜像
make docker-build

# 2. 创建密码文件
mkdir -p secrets
echo "your-root-password" > secrets/db_root_password.txt
echo "your-db-password" > secrets/db_password.txt
openssl rand -hex 32 > secrets/jwt_secret.txt

# 3. 启动全部服务
make compose-up

# 4. 运行迁移
docker compose run --rm server ./jimu migrate up

# 5. 初始化数据
docker compose run --rm -e ADMIN_PASSWORD=admin123 server ./jimu seed
```

`secrets/*.txt` 仅在 MariaDB 数据卷首次初始化时创建数据库账号。已存在数据卷时，改写 Secret 文件不会自动修改库内密码；请先轮换数据库账号密码，再重启 `server`，避免因凭据不一致反复重启。

服务启动后访问：
- API: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html （非 release 模式）
- Management: `http://127.0.0.1:9090/livez`、`/readyz`、`/metrics`
- Adminer: `docker compose --profile dev up -d adminer` 后访问 http://127.0.0.1:8081

### 可观测性（可选）

OpenObserve 监控栈（日志/指标/追踪/告警/仪表盘，替代原 Prometheus + Grafana + Loki + Promtail + AlertManager 五件套）随 `make compose-up` 按环境变量开启：

```bash
make compose-up                       # OTEL_ENABLED 默认开启；OTEL_ENABLED=false make compose-up 关闭
```

- OpenObserve UI/API: http://127.0.0.1:5080 （默认账号 admin@jimu.local / Admin@12345，可用 `ZO_OBSERVE_ROOT_USER_EMAIL` / `ZO_OBSERVE_ROOT_USER_PASSWORD` 覆盖）
- OTLP gRPC: `127.0.0.1:5081`（tracing / metrics / logs 统一入口）
- 默认 dashboard **Jimu Overview**：开启时自动创建（幂等），含 17 面板：stat 卡片（错误日志/日志总量/DB 连接池/Goroutines）、时间序列（DB/运行时/日志/HTTP/熔断/MySQL/Redis）、最近错误日志表格
- 官方数据库 dashboard **MySQL Metrics Monitoring** / **Redis Metrics Dashboard**：随启动自动创建（数据来自 OTel Collector 采集的 `mysql_*` / `redis_*` 指标流）；可选的 **PostgreSQL**（opentelemetry-contrib 采集）见下文

**Dashboard 配置与同步（面板进 git）**：dashboard 定义保存在 `deploy/openobserve/dashboards/*.json`（v8 结构，含面板查询与布局），启动时按此文件创建/重建；MySQL/Redis 官方模板取自 [openobserve/dashboards](https://github.com/openobserve/dashboards) 并归一为 v8 结构与 192 列网格布局入库（官方原始文件为旧版小网格，直接导入面板会缩成一条）：

```bash
./deploy/openobserve/sync-dashboard.sh                # 应用 dashboards/*.json（幂等：同名重建）
./deploy/openobserve/sync-dashboard.sh --export 名称  # 线上 dashboard 导出为 json（UI 微调后同步回 git）
```

JSON 中维护面板查询（`queries.fields` 流与轴映射、`type` 渲染类型）与布局（`layout` 网格坐标）；在 UI 手工调整后可用 `--export` 拉回并提交（运行时元数据自动剥离）。

**默认告警（可选）**：内置 8 条告警规则（`deploy/openobserve/alerts/jimu_*.json`，覆盖错误日志激增、HTTP 5xx 占比、DB 连接池饱和、队列死信、Outbox 发布失败、出站熔断、MySQL 线程数、Redis key 淘汰），经 OpenObserve v2 告警 API 同步（幂等：存在则更新，否则创建）。创建告警必须绑定已存在的通知目的地，因此同步依赖 `ZO_ALERT_WEBHOOK_URL`：

```bash
# .env 设置后 make compose-up 随监控栈启动自动同步；webhook 指向企业微信/钉钉/飞书/Slack 等网关均可
ZO_ALERT_WEBHOOK_URL=https://example.com/webhook
./deploy/openobserve/sync-alerts.sh                   # 手工同步（含通知模板 jimu_alert_http + 目的地 jimu_webhook）
```

- `ZO_ALERT_WEBHOOK_URL` 留空时跳过同步（不影响其它功能），告警也可在 UI 内手工配置
- 告警创建要求目标 stream 已有数据入库（按 stream schema 校验）；首次启动数据未就绪的规则会自动重试，超限跳过，稍后重跑补齐
- 目的地固定 `jimu_webhook`（POST JSON，载荷含 `alert_name` / `stream` / `level` / `fired_count` 等），阈值为保守默认值，在 `alerts/*.json` 或 UI 中按业务调整

**数据库集成（MySQL/Redis 指标）**：`make compose-up`（OTEL_ENABLED 默认开启）会同时启动 OTel Collector（`otel-collector` 服务，配置 `deploy/otel-collector.yaml`），采集 MySQL（performance_schema 指标：连接池/缓冲池/锁/慢查询相关）与 Redis（客户端/内存/命令吞吐）指标，经 OTLP/gRPC 推送到 OpenObserve（约 45 个 `mysql_*` / `redis_*` 指标流）：

```bash
docker compose --profile observability up -d otel-collector   # 单独启动采集
```

- 采集凭据：compose 内 MySQL 用 root 密码（secret `db_root_password`）；k8s/helm 生产环境默认用应用用户（`jimu`），需为其授予 `PROCESS, REPLICATION CLIENT` 权限以读取状态变量
- PostgreSQL（可选）：`deploy/otel-collector.yaml` 预留 `postgres` receiver，与 compose 的 `PG_ENDPOINT` / `PG_USER` / `PG_PASSWORD` 注释配置成对启用；启用后同步可选模板 `deploy/openobserve/dashboards-optional/postgresql-metrics.json`（官方 PostgreSQL (opentelemetry-contrib) 版，35 面板）：
  ```bash
  DASH_DIR=deploy/openobserve/dashboards-optional ./deploy/openobserve/sync-dashboard.sh
  ```
- 面板：dashboard 已预置 MySQL 线程数 / Redis 客户端连接 / Redis 内存 / Redis 指令吞吐 4 个时间序列面板（数据来自 collector）；独立官方模板提供 MySQL 6 面板（页/行/缓冲池操作等）与 Redis 7 面板（内存/吞吐/CPU 等）

让应用接入 OpenObserve（`otel.enabled` 开启，tracing + metrics + logs 均经 OTLP gRPC 推送）：

```bash
make compose-up   # OTEL_ENABLED 默认开启，server 自动指向 openobserve:5081 推送
```

应用侧：`/metrics` 端点保留 Prometheus 格式供外部工具抓取；指标另按 `metrics_interval_sec` 周期转 OTLP 推送（按指标名分流成独立 stream）；结构化日志异步推送（缓冲满丢弃，不影响主链路）；日志与追踪分别写入 `jimu_logs` / `jimu_traces` stream（可用 `otel.logs_stream_name` / `otel.traces_stream_name` 调整，空值回落 OpenObserve 默认流 `default`）。内置默认告警随监控栈同步（见上文「默认告警」），其余告警在 OpenObserve UI 内配置（VQL 告警规则 + 通知渠道）。

**日志输出规范**（保证 OpenObserve 中可检索、可聚合、可与 trace 关联）：

- **结构化调用**：必须使用 `Debugw / Infow / Warnw / Errorw`（msg + k/v 字段）。单参数 print 风格会把 k/v 拼进消息导致字段丢失（CI 的 `make check-log-usage` 强制检查）。
- **字段命名**：统一小写 snake_case，且 **key 来自标准词汇表**（`user_id`、`event_type`、`duration`…，见 `tools/logcheck/main.go` 及上文「开发规范 · 日志调用规范」；`make check-log-usage` 静态检查：防粘连、禁动态 key、未登记 key 告警、禁直接塞 struct/map/slice）。
- **类型保留**：数值/布尔按原类型上报（可范围查询与聚合），时长字段为纳秒数值，不会退化为字符串。
- **trace 关联**：请求上下文内用 `logger.WithContext(ctx)` 记录日志，自动携带 `trace_id`/`span_id` 并挂到对应 trace，OpenObserve 日志与链路可互跳。
- **敏感信息**（密码、token、验证码、手机号原文等）禁止入日志，PII 需脱敏（如 `138****1234`）。

停止：`make compose-down`（统一入口，保留数据卷）。

## CLI 工具

```bash
# 编译 CLI
make cli

# 模块管理
./bin/jimu module create product    # 生成完整模块骨架

# 数据库迁移
./bin/jimu migrate up               # 执行所有迁移
./bin/jimu migrate down             # 回滚最后一次迁移
./bin/jimu migrate status           # 查看迁移状态
./bin/jimu migrate redo             # 重做最后一次迁移

# 数据初始化
./bin/jimu seed                     # 插入初始数据（含 Casbin 策略同步）
```

## 项目结构

```text
jimu/
├── cmd/
│   ├── server/main.go          # HTTP 服务入口
│   └── cli/main.go             # CLI 入口
├── configs/
│   ├── app.yaml                # 默认配置（开发环境）
│   └── app.prod.yaml           # 生产环境配置
├── conf/
│   └── rbac_model.conf         # Casbin RBAC 模型
├── deploy/
│   ├── k8s/                     # Kubernetes manifests
│   │   ├── configmap.yaml
│   │   ├── secret.yaml
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── hpa.yaml
│   │   └── ingress.yaml
│   │   ├── openobserve.yaml     # OpenObserve 部署（Deployment/Service/PVC）
│   │   └── otel-collector.yaml  # 数据库指标采集（MySQL/Redis → OpenObserve）
│   ├── otel-collector.yaml      # OTel Collector 配置（receiver mysql/redis）
│   ├── openobserve/             # OpenObserve 初始化脚本
│   │   ├── init-dashboard.sh    # 兼容入口（等同 sync-dashboard.sh）
│   │   ├── sync-dashboard.sh    # dashboard 同步：apply（dashboards/*.json → 线上，幂等）/ --export（线上 → git）
│   │   ├── sync-alerts.sh       # 默认告警同步（模板/目的地/规则 → 线上，幂等；依赖 ZO_ALERT_WEBHOOK_URL）
│   │   ├── dashboards/          # dashboard 定义 JSON（面板查询与布局，git 管理）
│   │   │   ├── jimu-overview.json      # Jimu Overview 17 面板
│   │   │   ├── mysql-metrics.json      # 官方 MySQL Metrics Monitoring 6 面板
│   │   │   └── redis-metrics.json      # 官方 Redis Metrics Dashboard 7 面板
│   │   ├── dashboards-optional/ # 可选 dashboard（按需用 DASH_DIR 同步）
│   │   │   └── postgresql-metrics.json # 官方 PostgreSQL (opentelemetry-contrib) 35 面板
│   │   └── alerts/              # 默认告警规则 JSON（模板/目的地/jimu_*.json，git 管理）
│   └── helm/                    # Helm Chart（含 openobserve / otel-collector 配置）
├── docs/                         # 文档
│   ├── openapi/                  # Swagger 生成的 API 文档
│   ├── releases/                 # 版本 changelog / GitHub Release body（每版本一个文件）
│   ├── CONTRIBUTING.md           # 贡献指南（分支/PR/发布/集成测试手册）
│   └── SECURITY.md               # 安全政策（漏洞报告流程）
├── migrations/
│   ├── mysql/                  # MySQL 迁移脚本（按功能合并）
│   └── postgres/               # PostgreSQL 迁移脚本（按功能合并）
├── secrets/                    # Docker Secrets（gitignore）
├── internal/
│   ├── app/
│   │   ├── bootstrap.go        # 应用启动装配
│   │   ├── container.go        # 依赖容器
│   │   └── application.go      # Application 生命周期
│   ├── config/                 # 配置加载 + 校验
│   ├── contract/               # Module 接口定义
│   ├── platform/               # 基础设施
│   │   ├── http/               # HTTP Server + 中间件
│   │   ├── db/                 # Gorm 连接 + 迁移 + Seed + 事务
│   │   ├── redis/              # Redis 客户端 + 分布式锁
│   │   ├── cache/              # 缓存抽象层
│   │   ├── logger/             # Zap 日志
│   │   ├── auth/               # JWT + Casbin + Session + API Key
│   │   ├── tenant/             # 租户上下文注入 + 编码校验/归一化
│   │   ├── oauth/              # OAuth 第三方登录 Provider
│   │   ├── captcha/            # 图形验证码（生成 + Redis 存储 + 校验）
│   │   ├── encryption/         # AES-GCM 字段级加密 + HMAC 盲索引
│   │   ├── event/              # 事件总线
│   │   ├── queue/              # 多队列抽象（Redis/Kafka/RabbitMQ）+ 死信
│   │   ├── outbox/             # Outbox 模式
│   │   ├── scheduler/          # Cron 调度器
│   │   ├── observability/      # 健康检查 + Metrics + Tracing + OTLP 推送（OpenObserve）
│   │   ├── reporter/           # 错误上报（结构化错误日志，接入 OpenObserve）
│   │   ├── httpclient/         # 统一出站 HTTP 客户端（超时/重试/熔断/限流）
│   │   ├── grpc/               # gRPC server + 统一出站 Client（健康/反射/超时/重试/指标）
│   │   ├── ws/                 # WebSocket（Hub + 会话/频道管理）
│   │   ├── storage/            # 文件存储抽象（本地/S3/OSS/MinIO）
│   │   ├── importer/           # 数据导入（CSV/Excel 模板解析与校验）
│   │   ├── exporter/           # 数据导出（CSV/Excel）
│   │   ├── notification/       # 通知系统（邮件/短信/WebSocket/Webhook）
│   │   └── feature/            # Feature Flag
│   ├── shared/                 # 跨模块通用能力
│   │   ├── errors/             # AppError + 错误码
│   │   ├── response/           # 统一响应格式
│   │   ├── pagination/         # 分页
│   │   ├── validator/          # 自定义校验规则
│   │   ├── i18n/               # 国际化翻译
│   │   ├── id/                 # 雪花 ID 生成器
│   │   ├── totp/               # RFC 6238 TOTP（二次验证）
│   │   └── testutil/           # 测试工具
│   └── modules/                # 业务模块
│       ├── auth/               # 登录/注册/Token
│       ├── oauth/              # 第三方登录绑定
│       ├── user/               # 用户管理
│       ├── role/               # 角色管理
│       ├── permission/         # 权限管理
│       ├── tenant/             # 租户管理
│       ├── audit/              # 审计日志
│       └── admin/              # 系统管理
├── tools/
│   ├── generator/                # 代码生成器
│   └── logcheck/                 # 日志调用规范静态检查（make check-log-usage）
├── .github/                    # GitHub Actions + Dependabot
├── Makefile
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

## API 示例

### 获取验证码

启用 `captcha.enabled` 后，登录/注册需先获取验证码。返回 `captcha_id` 与 base64 图片：

```bash
curl http://localhost:8080/api/v1/captcha
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "captcha_id": "hQq...",
    "captcha_image": "data:image/png;base64,iVBOR..."
  }
}
```

### 登录

启用验证码时需携带 `captcha_id` 与 `captcha_code`：

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "secret123", "captcha_id": "hQq...", "captcha_code": "1234"}'
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "expires_in": 1800
  }
}
```

### OAuth 登录

```bash
# 跳转第三方授权页（浏览器访问，返回 302）
curl -L "http://localhost:8080/api/v1/oauth/google/login?state=<state>"
```

回调接口（第三方授权后重定向）：

```bash
# 第三方授权后回调，签发 JWT（浏览器访问）
GET /api/v1/oauth/google/callback?code=<code>&state=<state>
```

响应（与登录相同）：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "expires_in": 3600
  }
}
```

### 刷新 Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

### 创建用户

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: <uuid>" \
  -d '{"username": "newuser", "password": "pass1234", "email": "newuser@example.com", "phone": "13800138000"}'
```

`email`/`phone` 可选，入库前 AES-256-GCM 加密（配置 `security.encryption_key` 时），重复校验走盲索引。

### 忘记密码（发送验证码）

```bash
curl -X POST http://localhost:8080/api/v1/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

邮箱不存在时同样返回成功（防枚举）。验证码 15 分钟内有效。

### 重置密码

```bash
curl -X POST http://localhost:8080/api/v1/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "code": "123456", "new_password": "newpass123"}'
```

验证码一次性，成功后强制登出该用户全部会话。

### TOTP 二次验证

用户启用 TOTP 后，登录必须附带 `totp_code`；未启用则与普通登录一致，无需该字段。

```bash
# 1. 生成绑定密钥（返回 secret 与 otpauth URI，供二维码/认证器绑定）
curl -X POST http://localhost:8080/api/v1/auth/mfa/setup \
  -H "Authorization: Bearer <access_token>"

# 2. 用认证器的首次验证码确认启用
curl -X POST http://localhost:8080/api/v1/auth/mfa/enable \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"code":"123456"}'

# 3. 启用后的登录：携带 totp_code（缺失返回 2006，错误返回 2007）
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123","totp_code":"123456"}'

# 4. 关闭二次验证（校验当前验证码）
curl -X POST http://localhost:8080/api/v1/auth/mfa/disable \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"code":"123456"}'
```

### 开通式注册（可选）

启用 `auth.provisioning.enabled`（要求 `public_registration: true`）后，注册即开通新租户：单事务创建租户 + owner 用户，并按角色模板自动绑定全局权限。请求携带 `tenant_name`（必填，`tenant_code` 可选，不传自动生成）：

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"founder1","password":"secret123","tenant_name":"Acme Inc","tenant_code":"acme"}'
```

响应包含 `user`（owner）与 `tenant`（新租户）。owner 登录后自动获得模板中 `owner_role` 指定的角色（缺省为模板第一个角色）。`tenant_code` 统一转小写存储，不传自动生成。模板中引用的权限需先由 `jimu seed` 写入全局权限表，缺失条目跳过。未启用开通式时，注册保持普通语义（用户归默认租户，忽略租户字段）。

相关错误码：`2002` 用户名/邮箱已存在、`5002` 租户编码已存在、`5004` 租户编码格式无效。

### 租户管理

租户 CRUD 挂载在受保护路由下，需具备对应权限（seed 已内置 `/api/v1/tenants` 全套策略）。租户上下文来自 JWT `tid` claim：用户/角色/审计日志的查询与创建按当前租户自动隔离，无需传请求头。

#### 归属模型

- **租户 1:N 用户**：一个租户可有多个用户，一个用户只属于一个租户。结构上是 `users.tenant_id` 单列，没有用户-租户关联表。
- **身份全局唯一**：`username` 与邮箱/手机号盲索引（`email_hash`/`phone_hash`）为全局唯一索引，同一身份不能存在于多个租户；租户归属创建后不可修改（`UpdateUserRequest` 仅支持 `status`），也没有切换租户接口。
- **不支持一人多租**：跨租户协作请使用平台级视角（上下文无租户）或按租户归属的 API Key，而不是让同一账号加入多个租户。
- **管理端用户接口按租户隔离**：`/api/v1/admin/users` 的列表、详情、更新、禁用、分配角色均限定在当前租户（跨租户按不存在处理），分配角色时角色名只在用户所属租户内解析。
- **任务与导入任务按租户隔离**：`/api/v1/admin/jobs`、`/api/v1/admin/jobs/dead-letters` 的列表/详情/重试/标记解决与 `/api/v1/admin/users/import/:id` 均限定在当前租户；任务消费时会把任务归属租户注入执行上下文（outbox 事件无任务行，租户未知，按平台级处理）。
- **`tenant_id=0` 表示未归属**：迁移前的存量数据或绕过服务层直接写库会产生该状态，登录时会归一到默认租户（`id=1`、`code=default`）。

```bash
# 创建租户（编码全局唯一，仅限字母/数字/短横线/下划线，统一转小写存储）
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"code": "acme", "name": "Acme Inc"}'

# 租户列表（分页）
curl "http://localhost:8080/api/v1/tenants?page=1&page_size=20" \
  -H "Authorization: Bearer <access_token>"

# 租户详情
curl http://localhost:8080/api/v1/tenants/<tenant_id> \
  -H "Authorization: Bearer <access_token>"

# 更新租户（编码不可修改；status 0-禁用 1-启用）
curl -X PUT http://localhost:8080/api/v1/tenants/<tenant_id> \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Acme Inc", "status": 1}'

# 删除租户（软删除；默认租户 code=default 受保护，返回 5003）
curl -X DELETE http://localhost:8080/api/v1/tenants/<tenant_id> \
  -H "Authorization: Bearer <access_token>"
```

相关错误码：`5001` 租户不存在、`5002` 租户编码已存在、`5003` 默认租户受保护、`5004` 租户编码格式无效。

### API Key 与 Scope

API Key 用于服务/机器间调用，请求携带 `X-API-Key` 头，复用 `api_keys` 表。认证中间件为 `auth.APIKeyAuthMiddleware`，框架不默认挂载，业务模块按需加到路由组上；scope 校验用 `auth.RequireScope` 挂在认证之后：

```go
// verifier 由容器提供：container.APIKeyVerifier
api := r.Group("/api/v1/integration", auth.APIKeyAuthMiddleware(verifier))
// 只读接口要求 user:read
api.GET("/users", auth.RequireScope("user:read"), userHandler.List)
// 写接口要求 user:write
api.POST("/users", auth.RequireScope("user:write"), userHandler.Create)
```

`RequireScope` 未通过时返回 `401`（缺少/未认证 API Key）或 `403`（scope 不足）。

#### Scope 命名约定

`scopes` 是 API Key 的能力标签，落库为 `api_keys.scopes`（JSON 数组），命名格式 `资源:动作`，统一小写：

| Scope | 含义 | 典型用途 |
|-------|------|----------|
| `user:read` | 读取用户列表与详情 | 外部系统同步用户 |
| `user:write` | 创建、更新、删除用户 | 自动化开通与回收账号 |
| `job:submit` | 提交异步任务 | 触发批量导入等后台作业 |
| `audit:read` | 读取审计日志 | 合规系统拉取操作记录 |
| `*` | 全部能力 | 内部服务全权 Key |

约定：

- **空 `scopes` 表示拒绝一切**：`APIKey.HasScope` 对空列表恒返回 false；只有显式包含 `*` 才代表全权，不要依赖"不填即全权"的隐式行为。
- Scope 清单由业务方定义，框架不内置强制集合；`HasScope` 已提供通配匹配（`s == scope || s == "*"`），`RequireScope` 按同一语义校验。
- **认证与授权分离**：`APIKeyAuthMiddleware` 只校验 Key 有效性（格式、存在、启用、未过期）并注入 Key；是否需要某个 scope 由路由上的 `RequireScope` 决定，未挂载即不校验 scope。
- **API Key 维度限流**：`middleware.APIKeyRateLimitMiddleware(rdb, limit, window)` 挂在认证之后，按 Key ID 计数（不落明文），未携带 Key 的请求跳过该维度；租户维度由 `ratelimit.tenant.*` 全局启用。配额（按天/按月上限）用同一中间件配长窗口即可（例如 `window=24h`）。
- **API Key 归属租户**：`api_keys.tenant_id` 在创建时取自创建者所在租户（上下文无租户时归默认租户，见 `platform/tenant.DefaultTenantID`）；认证通过后中间件把该租户注入请求上下文，业务层用 `tenant.FromContext(ctx)` 读取即可完成行级隔离。租户只来自 Key 自身，**不接受客户端 header/query 传入**。未归属（`tenant_id=0`）的存量 Key 按平台级视角处理。

#### 管理 API

```bash
# 创建（明文 Key 仅返回一次；scopes 省略即拒绝一切）
curl -X POST http://localhost:8080/api/v1/admin/apikeys \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-bot", "scopes": ["user:read"], "expires_in": 90}'

# 列表 / 详情 / 撤销
curl http://localhost:8080/api/v1/admin/apikeys -H "Authorization: Bearer <access_token>"
curl http://localhost:8080/api/v1/admin/apikeys/<id> -H "Authorization: Bearer <access_token>"
curl -X DELETE http://localhost:8080/api/v1/admin/apikeys/<id> -H "Authorization: Bearer <access_token>"
```

`expires_in` 单位为天，缺省或 0 表示不过期。列表按当前租户过滤，详情/撤销跨租户不可见（返回 404）；平台级视角（上下文无租户）不过滤。

### 获取系统状态

管理端点统一挂载 JWT + Casbin RBAC 认证，需携带管理员 `access_token`（无有效策略默认拒绝，返回 403）：

```bash
curl http://localhost:8080/api/v1/admin/monitoring/status \
  -H "Authorization: Bearer <access_token>"
```

### 查看认证限流状态

只读端点，不消费令牌地查看某 `scope` + `key` 的当前计数与剩余窗口（如登录爆破防护）：

```bash
curl "http://localhost:8080/api/v1/admin/ratelimit/auth?scope=login&key=ip:1.2.3.4" \
  -H "Authorization: Bearer <access_token>"
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "scope": "login",
    "key": "ip:1.2.3.4",
    "count": 3,
    "ttl_ms": 58000,
    "reset_at": 1724047200,
    "redis_key": "jimu:auth:limit:login:<sha256>"
  }
}
```

`key` 不存在时 `count`/`ttl_ms` 均为 0。`redis_key` 中存 sha256 摘要，不泄露原始 key 明文。

### 健康检查

```bash
curl http://127.0.0.1:9090/livez
curl http://127.0.0.1:9090/readyz
```

### Metrics

```bash
curl http://127.0.0.1:9090/metrics
```

## 配置说明

### 多环境配置

通过 `APP_ENV` 环境变量切换：

| 环境 | 配置文件 | 说明 |
|------|----------|------|
| 开发 | `app.yaml` | 默认，日志输出到 stdout |
| 生产 | `app.prod.yaml` | JSON 日志、文件滚动、release 模式 |

优先级：`环境变量 > app.{env}.yaml > app.yaml`

### 环境变量

敏感配置支持 `_FILE` 后缀从文件读取（Docker Secrets 兼容）：

```bash
# 直接环境变量
DB_HOST=mariadb
DB_PASSWORD=secret

# 或从文件读取（推荐生产环境）
DB_PASSWORD_FILE=/run/secrets/db_password
JWT_SECRET_FILE=/run/secrets/jwt_secret
ENCRYPTION_KEY_FILE=/run/secrets/encryption_key
```

`ENCRYPTION_KEY` 不注入时字段加密退化为明文模式（功能可用，email/phone 明文落库）。

### 配置项

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `http.host` | 监听地址 | `0.0.0.0` |
| `http.port` | 监听端口 | `8080` |
| `http.mode` | Gin 模式 (`debug`/`release`/`test`) | `debug` |
| `http.max_body_bytes` | 请求体大小上限 | `1MB`（开发）/ `10MB`（生产） |
| `http.allowed_origins` | CORS 允许的来源列表 | `*` |
| `http.trusted_proxies` | 可信代理 CIDR（影响 ClientIP 判定） | `127.0.0.1` |
| `http.tls.enabled` | 是否启用 TLS（反向代理终止时保持 `false`） | `false` |
| `http.tls.cert_file` / `http.tls.key_file` | TLS 证书 / 私钥路径 | — |
| `db.driver` | 数据库驱动：`mysql` / `postgres` / `mariadb`（环境变量 `DB_DRIVER` 覆盖） | `mysql` |
| `db.host` | 数据库地址 | `127.0.0.1` |
| `db.port` | 数据库端口（MySQL 默认 3306，PostgreSQL 默认 5432） | `3306` |
| `db.user` | 数据库用户名 | `jimu` |
| `db.password` | 数据库密码（通过环境变量覆盖） | — |
| `db.max_open` | 最大连接数 | `25`（开发）/ `100`（生产） |
| `db.max_idle` | 最大空闲连接数 | `10`（开发）/ `20`（生产） |
| `db.conn_max_lifetime_sec` | 连接最大存活时间（秒） | `3600` |
| `db.read_hosts` / `db.read_ports` | 只读副本地址 / 端口（读写分离） | — |
| `db.breaker.enabled` / `db.breaker.max_failures` / `db.breaker.reset_timeout_sec` | DB 语句级熔断开关 / 连续失败阈值 / 冷却秒数（读写分离启用时自动跳过） | `true` / `5` / `10` |
| `redis.mode` | Redis 部署模式：`single` / `sentinel` / `cluster` | `single` |
| `redis.addr` | Redis 地址（单机模式） | `127.0.0.1:6379` |
| `redis.password` | Redis 密码（通过环境变量覆盖） | — |
| `redis.db` | Redis 数据库编号（cluster 模式不支持） | `0` |
| `redis.master_name` | 哨兵模式 master 名称（`mode=sentinel` 必填） | — |
| `redis.sentinel_addrs` | 哨兵模式节点地址列表（`mode=sentinel` 必填） | — |
| `redis.sentinel_password` | 哨兵节点密码（可选） | — |
| `redis.cluster_addrs` | 集群模式节点地址列表（`mode=cluster` 必填） | — |
| `redis.pool_size` | Redis 连接池大小 | `10`（开发）/ `50`（生产） |
| `redis.min_idle_conns` | 最小空闲连接数 | `2`（开发）/ `10`（生产） |
| `redis.breaker.enabled` / `redis.breaker.max_failures` / `redis.breaker.reset_timeout_sec` | Redis 熔断开关 / 连续失败阈值 / 冷却秒数 | `true` / `5` / `10` |
| `log.level` | 日志级别 | `debug`（开发）/ `info`（生产） |
| `log.format` | 日志格式 | `console`（开发）/ `json`（生产） |
| `auth.jwt_secret` | JWT 签名密钥（生产必须 `JWT_SECRET` 环境变量注入） | — |
| `auth.access_expire_min` | Access Token 有效期 (分钟) | `60`（开发）/ `15`（生产） |
| `auth.refresh_expire_day` | Refresh Token 有效期 (天) | `30`（开发）/ `7`（生产） |
| `auth.reset_code_ttl_min` | 密码重置验证码有效期 (分钟) | `15` |
| `auth.provisioning.enabled` | 开通式注册：注册即开通新租户（要求 `auth.public_registration: true`） | `false` |
| `auth.provisioning.owner_role` | owner 绑定的模板角色名；缺省为模板第一个角色 | — |
| `auth.provisioning.roles[]` | 开通时初始化的角色模板（`name`/`description`/`permissions[]{resource,action}`）；permissions 引用全局权限表（`jimu seed` 写入），缺失条目跳过 | — |
| `auth.public_registration` | 是否开放 `/auth/register` 公开注册端点 | `true`（开发）/ `false`（生产） |
| `server.timeout_sec` | 请求超时（秒），0 不限；超时且未产出响应时返回 `1008`/504 | `30` |
| `server.rate_limit_rate` / `server.rate_limit_burst` | 全局限流速率（每秒）/ 桶容量 | `100` / `200` |
| `server.max_concurrency` / `server.concurrency_wait_ms` | 并发处理上限 / 超限排队等待上限（毫秒，0=立即拒绝）；超限返回 `1010`/503，0 表示不限制 | `512` / `200` |
| `ratelimit.tenant.enabled` / `ratelimit.tenant.limit` / `ratelimit.tenant.window_sec` | 租户维度限流开关 / 窗口内请求上限 / 窗口秒数（平台级视角 `tid=0` 跳过，Redis 异常 fail-open） | `false` / `6000` / `60` |
| `retention.enabled` / `retention.cron` / `retention.batch_size` | 历史数据保留任务开关 / 调度表达式 / 每批删除行数（默认关闭） | `false` / `30 3 * * *` / `500` |
| `retention.audit_log_days` / `retention.job_days` / `retention.job_history_days` | 审计日志 / 已终态任务 / 任务执行历史保留天数（0=不清理） | `180` / `7` / `30` |
| `retention.dead_letter_days` / `retention.outbox_event_days` / `retention.import_job_days` | 已处理死信 / 已发布 outbox 事件 / 已结束导入任务保留天数（0=不清理） | `30` / `7` / `90` |
| `security.ip_allowlist` / `security.admin_ip_allowlist` | 全局 / 管理端 IP 白名单（CIDR 或单个 IP，可多项）；为空表示不限制，非法值启动报错 | `[]` |
| `audit.hash_secret` | 审计链 HMAC 密钥（建议经 `AUDIT_HASH_SECRET` 或 Secret 文件注入）；为空时退化为 SHA-256，篡改者可重算整条链 | — |
| `id.worker_id` | 雪花 ID worker 编号（0-1023）；多实例部署时每个副本需唯一，避免 ID 冲突 | `0` |
| `storage.type` | 存储类型 (`local`/`s3`/`oss`/`minio`)。`oss` 复用 S3 协议（path style + endpoint），无需阿里云 SDK；`minio` 需 `path_style: true` | `local` |
| `upload.clamav.enabled` | 是否启用文件上传 ClamAV 病毒扫描；`false` 时上传不扫描 | `false` |
| `upload.clamav.address` | clamd 监听地址（如 `127.0.0.1:3310`） | `127.0.0.1:3310` |
| `upload.clamav.timeout_sec` | 单次扫描超时（秒），0 用默认 10 | `10` |
| `queue.type` | 队列类型 (`redis`/`kafka`/`rabbitmq`)，切 Kafka/RabbitMQ 时需保证 broker 可用，否则启动失败 | `redis` |
| `outbox.publisher` | Outbox 发布器类型 (`event_bus`/`mq`)。`mq` 支持 `queue.type=kafka/rabbitmq/redis` | `event_bus` |
| `scheduler.store` | 任务定义存储类型 (`memory`/`mysql`)；`mysql` 需迁移表 `scheduled_jobs`（迁移 003） | `memory` |
| `oauth.providers.{name}.enabled` | 是否启用某 OAuth 提供商 (`google`/`github`/`wechat`) | `false` |
| `oauth.providers.{name}.client_id` | 提供商应用 Client ID | — |
| `oauth.providers.{name}.client_secret` | 提供商应用 Client Secret（生产建议环境变量注入） | — |
| `oauth.providers.{name}.redirect_url` | 授权回调地址 | — |
| `captcha.enabled` | 是否启用登录/注册验证码 | `false`（开发与生产，前端验证码流程就绪后再开启） |
| `captcha.ttl_min` | 验证码有效期 (分钟) | `5` |
| `email.enabled` | 是否启用真实 SMTP 发送；`false` 时邮件通知回退日志渠道 | `false` |
| `email.host` | SMTP 服务器地址（如 `smtp.example.com`） | — |
| `email.port` | SMTP 端口（25/465/587） | `587` |
| `email.username` | SMTP 认证用户名 | — |
| `email.password` | SMTP 认证密码（生产建议 `EMAIL_PASSWORD` 环境变量注入） | — |
| `email.from` | 发件人地址（如 `noreply@example.com`） | — |
| `sms.enabled` | 是否启用真实短信发送；`false` 时短信通知回退日志渠道 | `false` |
| `sms.provider` | 短信服务商（当前支持 `aliyun`） | — |
| `sms.api_key` | 阿里云 AccessKey ID（生产建议 `SMS_API_KEY` 环境变量注入） | — |
| `sms.api_secret` | 阿里云 AccessKey Secret（生产建议 `SMS_API_SECRET` 环境变量注入） | — |
| `sms.sign_name` | 短信签名 | — |
| `security.encryption_key` | 字段级加密密钥（≥32 字符，AES-256-GCM）；生产必须 `ENCRYPTION_KEY` 环境变量注入，否则 email/phone 明文存储 | — |
| `security.csrf_secret` | CSRF 密钥；非空时启用 CSRF 中间件（Bearer 认证请求自动跳过） | — |
| `security.content_type_options` / `frame_options` / `xss_protection` | HTTP 安全响应头（`X-Content-Type-Options` 等，留空用默认值） | 见 `DefaultSecurityConfig` |
| `security.strict_transport` | `Strict-Transport-Security` 头 | `max-age=31536000; includeSubDomains` |
| `security.content_security_policy` | `Content-Security-Policy` 头 | `default-src 'self'` |
| `cache.prefix` | 缓存 key 前缀 | `jimu` |
| `audit.queue_size` / `batch_size` / `flush_interval_ms` | 审计日志队列容量 / 批量写入条数 / 刷新间隔（ms） | `1024` / `100` / `500` |
| `otel.enabled` | 是否启用 OpenTelemetry 可观测性（tracing + metrics + logs 统一 OTLP gRPC 推送 OpenObserve） | `false`（开发）/ `true`（生产） |
| `otel.endpoint` | OTLP gRPC 端点（OpenObserve，compose 内为 `openobserve:5081`）；环境变量 `OTEL_ENDPOINT` 可覆盖 | `localhost:4317` |
| `otel.service_name` / `otel.service_version` | 服务标识（resource 属性） | `jimu` / `dev` |
| `otel.sample_rate` | 追踪采样率 0-1，1.0 全量 | `1.0` |
| `otel.metrics_enabled` | 指标推送：Prometheus 指标转 OTLP 推送到 OpenObserve（`/metrics` 端点仍保留） | `true` |
| `otel.logs_enabled` | 日志推送：zap 结构化日志异步转 OTLP logs | `true` |
| `otel.metrics_interval_sec` | 指标推送间隔（秒），0 用默认 | `15` |
| `otel.auth_email` / `otel.auth_password` | OpenObserve 账号凭据（OTLP/gRPC Basic Auth；生产可用 `OTEL_AUTH_EMAIL` / `OTEL_AUTH_PASSWORD`（或 `OTEL_AUTH_PASSWORD_FILE`）环境变量注入） | 与 OpenObserve 账号一致 |
| `otel.org_id` | OpenObserve 组织（gRPC `organization` header；`OTEL_ORG_ID` 可覆盖） | `default` |
| `otel.logs_stream_name` / `otel.traces_stream_name` | OpenObserve 日志 / 追踪 stream 名（gRPC `stream-name` header；`OTEL_LOGS_STREAM_NAME` / `OTEL_TRACES_STREAM_NAME` 可覆盖；空则落 OpenObserve 默认流 `default`，非法字符会被 OpenObserve 规范为 `_`） | `jimu_logs` / `jimu_traces` |
| `http_client.timeout_sec` | 出站 HTTP 单次请求超时（秒），0 用默认 | `10` |
| `http_client.max_retries` | 出站 HTTP 失败重试次数（仅网络错误与 5xx），0 用默认 | `2` |
| `http_client.retry_interval_ms` | 出站 HTTP 重试基础间隔（毫秒，指数退避），0 用默认 | `200` |
| `http_client.max_failures` | 出站 HTTP 熔断阈值：连续失败次数达此值开启熔断，0 用默认 | `5` |
| `http_client.reset_timeout_ms` | 出站 HTTP 熔断冷却时长（毫秒），到期后放行探测，0 用默认 | `30000` |
| `http_client.rate_limit_rate` | 出站 HTTP 每秒请求数（按目标 host 独立限流），0 不限流 | `0` |
| `http_client.rate_limit_burst` | 出站 HTTP 令牌桶容量，0 用 rate（桶=平均速率） | `0` |
| `notification.webhook.sign_secret` | Webhook 回调载荷签名密钥（HMAC-SHA256，`X-Jimu-Signature`）；空则不签名 | — |
| `grpc.enabled` | 是否启用 gRPC server（与 HTTP 双栈并存，默认关闭） | `false` |
| `grpc.host` / `grpc.port` | gRPC 监听地址 / 端口 | `0.0.0.0` / `9091` |
| `grpc` 业务服务 | 示例 `UserInfoService`（`internal/platform/grpc/userinfo_service.go`，proto 在 `proto/jimu/v1/userinfo.proto`，`make proto` 重新生成）；业务模块仿照 `RegisterUserInfoService` 经 `RegisterService` 接入 | — |
| `error_reporting.enabled` | 是否启用错误上报（结构化错误日志输出，含 trace_id；未启用时零开销） | `false`（开发）/ `true`（生产） |
| gRPC 出站客户端 | 统一封装 `internal/platform/grpc` `Client`（`NewClient`）：超时/重试/恢复/指标 `jimu_grpc_client_*`，业务经 `Conn()` 走生成的强类型客户端 | — |

### 静态加密（Data at Rest）

静态加密分两层，本框架只负责**字段级**，**全库透明加密**委托给数据库/存储层：

**字段级加密（框架内置，`security.encryption_key`）**

- AES-256-GCM 字段级加密 + HMAC-SHA256 盲索引，实现在 `internal/platform/encryption` + `internal/platform/db/encryption.go`（Gorm hook）。
- 带结构体 tag `encryption:"true"` 的字段写入时加密、读取时解密；带 `blind:"<source>"` 的字段用对应明文计算确定性盲索引，支撑唯一约束与精确等值查询。
- 当前覆盖 `users.email` / `users.phone`（见 `internal/modules/user/domain/user.go`），密文落库、`email_hash`/`phone_hash` 盲索引支撑重复校验。
- 密钥经 `ENCRYPTION_KEY` 环境变量或 `ENCRYPTION_KEY_FILE`（Docker Secrets）注入；**未注入时退化为明文模式**（功能不受影响，email/phone 明文落库）。
- 密码字段 `users.password` 始终存 bcrypt 哈希，不参与字段级加密——不可逆，无需可解密。

**全库静态加密（框架范围外，委托外部）**

框架不内置也不接管数据库全库透明加密，由部署侧在数据库/存储层落实：

- **MariaDB/MySQL**：`file_key_management` 插件 + 密钥文件，或使用 Percona/云厂商的透明数据加密（TDE）。
- **PostgreSQL**：`pgcrypto`（应用侧，与字段级加密重叠）或云 RDS 的磁盘加密。
- **磁盘层**：LUKS / 云盘加密（EBS、托管磁盘加密）作为兜底，对数据库实现无关。

字段级加密面向「即便拿到数据库快照也无法直接读取 email/phone」的场景；全库加密面向「物理介质丢失」场景，二者正交、可叠加。

## 开发规范

### 模块结构

每个业务模块必须遵循 Clean Architecture 分层：

```text
internal/modules/{name}/
  domain/           # 实体、值对象、仓储接口
  application/      # 用例服务、DTO
  infrastructure/   # 数据库/缓存实现
  interfaces/       # HTTP handler + 路由注册
  module.go         # 实现 contract.Module 接口
```

- 业务逻辑必须依赖接口，不依赖具体实现
- 所有模块实现 `contract.Module` 接口（`Name` / `RegisterHTTP` / `RegisterJobs` / `RegisterEvents`）
- HTTP 路由统一注册在 `/api/v1` 前缀下

### 错误码

定义在 `internal/shared/errors/errors.go`，按模块分段分配，后续模块依次向后分配：

- `1xxx` — 通用错误
- `2xxx` — 用户/认证模块
- `3xxx` — OAuth 模块
- `4xxx` — 验证码模块
- `5xxx` — 租户模块

新增错误码需同步加入 `HTTPStatus` 映射与 `AllErrorCodes` 文档列表。

### 配置

- 新增配置项必须在 `internal/config/config.go` 定义常量并加入校验；枚举值非法时启动报错
- 敏感值支持 `_FILE` 后缀从文件读取（Docker Secrets 兼容）

### 数据库

- Gorm + Goose 迁移，命名 `{seq}_create_{table}s.sql`，迁移文件需为每个字段和表添加中文 COMMENT
- 基础表包含 `id`、`created_at`、`updated_at`、`deleted_at`；主键由应用生成雪花 ID（gorm hook），建表不使用 `AUTO_INCREMENT`
- 支持读写分离（`read_hosts`、`read_ports` 配置，MySQL/MariaDB 与 PostgreSQL 均支持，从库按 `RandomPolicy` 轮询）；**注意从库存在复制延迟**：写后立即读可能读到旧数据，强一致读请走主库（框架未做写后粘主，需要强一致的查询请在业务层显式指定主库或加读己之写补偿）
- 时间统一 **UTC**：连接固定 `loc=UTC` + MySQL 会话 `time_zone='+00:00'`（PostgreSQL `TimeZone=UTC`），驱动与服务器时区必须一致，否则 `TIMESTAMP` 列与 `DEFAULT CURRENT_TIMESTAMP` 会相差一个时区偏移；API 以 RFC3339（带 `Z`）返回，展示时区由前端/SDK 转换
- 金额/精度：框架**不提供**decimal 抽象（避免引入依赖与过早抽象）；金额用 `DECIMAL(m,n)` 列存储、Go 侧用 `string` 或最小货币单位 `int64` 传输，**禁止用 float 表示金额**
- 全文检索：`platform/search.New(db)` 按方言返回实现；索引写入需在业务写事务提交后调用（或经 outbox 异步补索引），避免主数据与索引不一致
- 并发写控制：`platform/db` 提供 `LockRow`（事务内 `SELECT ... FOR UPDATE` 锁定单行，SQLite 自动降级）与 `SaveOptimistic`（`version` 列乐观锁，冲突返回 `db.ErrConcurrentUpdate`，调用方映射 409）；`users`/`roles`/`tenants` 已带 `version` 列，角色/租户更新走乐观锁，角色权限替换与用户角色分配在事务内先锁目标行

### 日志调用规范

- **必须使用结构化方法** `Debugw/Infow/Warnw/Errorw`（msg + k/v 字段）；禁止 `Debug/Info/Warn/Error(...)` 传多个参数——sugared logger 会把参数 `fmt.Sprint` 拼进消息导致字段丢失。`make check-log-usage` 强制检查（CI 已接入）
- 消息用静态动词短语，变量一律进字段；key 小写 snake_case，必须来自 `tools/logcheck/main.go` 内置词汇表（`_id` / `_ms` 等后缀自动放行），禁止动态 key
- 错误写 `"error", err` 而非 `err.Error()`；数值给数值类型；不记录密码、token、验证码等敏感原文，PII 脱敏
- 级别：`debug` 排查细节 | `info` 业务事件 | `warn` 可恢复异常 | `error` 不可恢复（必须带 `error` 字段）

### 质量门禁

所有改动必须通过 `make fmt`、`make vet`、`make lint`、`make test`；贡献流程与本地集成测试见 [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md)。

## Makefile 命令

| 命令 | 说明 |
|------|------|
| `make run` | 运行服务（开发模式） |
| `make dev` | 开发模式：fmt + vet + 构建 + 运行 |
| `make build` | 编译 server + cli |
| `make migrate` | 执行迁移 |
| `make migrate-down` | 回滚最后一次迁移 |
| `make migrate-status` | 查看迁移状态 |
| `make seed` | 插入初始数据 |
| `make backup` | 备份数据库（`scripts/backup.sh`，mariadb-dump/mysqldump + gzip + 保留 7 天） |
| `make restore` | 从备份恢复数据库（`scripts/restore.sh`，需 `BACKUP_FILE=...`，`FORCE=1` 跳过确认） |
| `make test-backup-restore` | 备份/恢复往返测试（`scripts/test_backup_restore.sh`，需运行中 mariadb 容器） |
| `make test` | 运行测试 |
| `make test-coverage` | 测试 + 覆盖率报告 |
| `make bench` | 运行性能基准测试（ID 生成 / 登录 / Webhook 发送） |
| `make bench-ci` | 性能回归门禁（绝对阈值模式，CI 用：`scripts/bench_ci.sh --absolute`） |
| `make proto` | 重新生成 gRPC 代码（需 protoc + protoc-gen-go + protoc-gen-go-grpc） |
| `make loadtest` | 本地 HTTP 压测（需 hey：`go install github.com/rakyll/hey@latest`） |
| `make vet` | 静态分析 |
| `make fmt` | 格式化代码 |
| `make fmt-check` | 检查代码格式 |
| `make lint` | golangci-lint |
| `make swagger` | 生成 API 文档 |
| `make cli` | 编译 CLI |
| `make docker-build` | 构建 Docker 镜像 |
| `make docker-run` | 直接运行 Docker 容器 |
| `make compose-up` | 启动所有容器 |
| `make compose-down` | 停止所有容器 |
| `make compose-restart` | 重启所有容器 |
| `make compose-logs` | 查看应用日志 |
| `make compose-migrate` | Compose 环境执行迁移 |
| `make compose-seed` | Compose 环境插入初始数据（需 `.env` 提供 `ADMIN_PASSWORD`，或 `make compose-seed ADMIN_PASSWORD=xxx`） |
| `make compose-up`（`OTEL_ENABLED=false` 关闭） | 统一开关（默认开）：启动 OpenObserve 监控栈（含数据库指标 Collector、默认 dashboard 与内置告警同步）并让 server 推送遥测 |
| `make compose-check` | 使用临时项目、Secret 和数据卷进行 Compose 运行时/API 验证，不影响本地服务 |
| `make release-check` | 发布前检查（fmt-check + vet + test + govulncheck + Compose 运行时/API 验证） |
| `make ci` | 本地 CI 检查（无外部依赖：fmt-check + vet + lint + test + 覆盖率 + race + swagger + smoke + build + govulncheck，完整 CI 见 `.github/workflows/ci.yml`） |
| `make clean` | 清理构建产物 |
| `make hooks` | 安装 git 钩子（pre-commit 框架 + `githooks/commit-msg` 提交消息全英文检查） |

## Docker 部署

```bash
# 1. 构建镜像
make docker-build

# 2. 创建密码文件
mkdir -p secrets
echo "your-root-password" > secrets/db_root_password.txt
echo "your-db-password" > secrets/db_password.txt
openssl rand -hex 32 > secrets/jwt_secret.txt

# 3. 启动全部服务
make compose-up

# 4. 运行迁移和初始化
docker compose run --rm server ./jimu migrate up
docker compose run --rm -e ADMIN_PASSWORD=admin123 server ./jimu seed
```

已有数据卷中的 MariaDB 账号密码不会因修改 `secrets/*.txt` 自动轮换。先完成数据库账号轮换，再执行 `make compose-restart`。

## K8s 部署

`deploy/k8s/` 提供完整 manifests（Deployment/Service/HPA/Ingress/PDB/NetworkPolicy）：

```bash
# 1. 替换镜像仓库地址（deployment.yaml 中 registry.example.com/jimu:1.0.0）
# 2. 创建 TLS 证书 secret（ingress 引用）
kubectl create secret tls jimu-tls --cert=tls.crt --key=tls.key -n jimu

# 3. 应用全部资源
kubectl apply -f deploy/k8s/
```

- 数据库迁移由 initContainer 在主容器启动前自动执行（`jimu migrate up`），迁移失败则 Pod 不就绪、不滚动
- 敏感配置通过 `jimu-secrets` Secret 注入（`change-me-*` 占位值部署前必须替换）
- 多副本（`replicas > 1`）需为每个 Pod 配置唯一 `id.worker_id`，避免雪花 ID 冲突（如用 StatefulSet 序号或环境变量注入）
- Ingress 终止 TLS，应用侧 `http.tls.enabled` 保持 `false`

## License

MIT
