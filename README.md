# Jimu Backend Framework

Go 语言通用后端基础框架 — 稳定底座 + 可组合模块 + 标准适配器 + 脚手架生成能力。

## 特性

- **模块化架构** — Clean Architecture 分层，业务逻辑依赖接口不依赖实现
- **统一认证** — typed JWT + Redis refresh session + Casbin RBAC v3 权限模型；API Key 认证（服务/机器间调用，`X-API-Key` 头 + `auth.APIKeyAuthMiddleware`，复用 `api_keys` 表）
- **OAuth 登录** — Google/GitHub/微信第三方登录，`oauth.providers` 配置开关
- **图形验证码** — 登录/注册验证码，Redis 存储一次性校验，`captcha.enabled` 配置开关
- **统一响应** — 标准 `{code, message, data}` 格式 + 分页
- **多环境配置** — Viper + yaml + 环境变量覆盖，枚举值启动校验
- **结构化日志** — Zap + lumberjack 自动滚动
- **数据库迁移** — Goose 迁移 CLI (up/down/status/redo)
- **数据初始化** — Seed 命令一键插入管理员和基础权限（含 Casbin 策略同步）
- **限流保护** — 全局令牌桶 + Redis 登录/注册固定窗口限流 + 用户维度滑动窗口限流
- **HTTP 安全边界** — 请求体大小、超时、可信代理、CORS、安全 Headers；CSRF 防护（配置 `security.csrf_secret` 启用，Bearer 请求自动跳过）；API 签名验证中间件（可选，服务间调用按需挂载）
- **缓存抽象** — Cache-Aside 模式，GetOrSet 自动回填
- **自定义校验** — 手机号、密码强度、身份证、用户名等常用规则
- **国际化** — 按 `Accept-Language` 返回中文/英文错误与校验消息
- **事件总线** — 内存实现，支持同步/异步发布订阅
- **多队列支持** — Redis/Kafka/RabbitMQ 统一队列接口，`queue.type` 切换；Redis 为 at-least-once（BLMove 原子消费 + 可见性超时重入队 + 延迟队列），Kafka/RabbitMQ 当前 at-most-once
- **事务封装** — 统一的事务管理 helper
- **审计日志** — 有界队列批量写入，匿名请求安全处理
- **管理端点** — 独立 management server 暴露健康检查、metrics 和可选 pprof
- **管理 API** — 系统状态、在线用户、强制下线、错误码文档
- **脚手架** — Cobra CLI 一键生成完整模块骨架
- **API 文档** — Swagger UI 交互式文档（中文注释）
- **健康检查** — `/livez` 与 `/readyz`，readiness 有界探测 DB + Redis
- **优雅停机** — 显式 Application 生命周期，反向停止组件
- **分布式锁** — Redis 实现的分布式锁（防并发、选主）
- **文件存储** — 本地/S3/OSS/MinIO 统一接口
- **通知系统** — 邮件/SMS/WebSocket/Webhook 抽象
- **Outbox 模式** — 事件发布与数据库事务一致性保证，支持 MQ 跨服务发布（`outbox.publisher` 切换；`mq` 模式下通过 WorkerPool 消费事件，`event_bus` 模式通过 `outbox:*` 桥接器注入事件总线）
- **定时任务** — Cron 调度器（robfig/cron），支持 MySQL 持久化（`scheduler.store=mysql`）与多实例分布式锁协调，启动时通过 `RestoreFromStore` 恢复持久化任务（内置任务去重）
- **Feature Flag** — 运行时特性开关（灰度百分比、白名单）
- **OpenTelemetry** — 分布式追踪（OTLP gRPC），HTTP/Gin + Gorm 查询 + Redis 命令全链路 span（`otel.enabled` 开启）
- **Prometheus 指标** — DB 连接池 + 运行时 + HTTP 请求指标（`jimu_http_*`）
- **分布式 ID** — 雪花 ID 生成器（`internal/shared/id`），所有数据库主键由应用生成，`id.worker_id` 配置多实例唯一编号
- **Docker 支持** — Dockerfile + docker-compose 一键起服务
- **Docker Secrets** — 敏感配置通过文件注入（`_FILE` 后缀）
- **K8s 部署** — Deployment/Service/HPA/Ingress manifests
- **CI/CD** — GitHub Actions + Dependabot 自动化 + 测试覆盖率门禁
- **安全扫描** — CI 执行 govulncheck 依赖漏洞扫描、Trivy 镜像扫描、SBOM 生成与镜像 smoke test
- **静态检查** — golangci-lint + pre-commit 钩子（fmt / vet / lint）
- **追踪关联** — 访问日志自动注入 trace_id / span_id，关联 OpenTelemetry 追踪

## 非目标

以下能力明确不做，新增需求不得引入相关抽象：

- **租户隔离** — 本项目定位单租户/单组织部署，用户体系全局唯一。数据表不预留 `tenant_id`、`tenant` 字段或租户中间件。若产品出现多租户需求，需重新设计数据模型，而非在现有表上打补丁。

## 技术栈

| 类别 | 选型 |
|------|------|
| HTTP 框架 | Gin |
| ORM | Gorm |
| 数据库 | MariaDB (MySQL 协议) |
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
# 编辑 .env 修改数据库、Redis 连接信息
```

### 方式一：本地运行

```bash
# 1. 启动依赖
docker compose up -d mariadb redis

# 2. 运行迁移
make migrate

# 3. 初始化数据（管理员密码通过环境变量提供）
ADMIN_PASSWORD=admin123 make seed

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

服务启动后访问：
- API: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html （非 release 模式）
- Management: `http://127.0.0.1:9090/livez`、`/readyz`、`/metrics`
- Adminer: `docker compose --profile dev up -d adminer` 后访问 http://127.0.0.1:8081

### 可观测性（可选）

启动 Prometheus + Grafana 监控栈（抓取 Management `/metrics` 端点）：

```bash
make compose-observability   # 或 docker compose --profile observability up -d
```

- Prometheus: http://127.0.0.1:9093
- Grafana: http://127.0.0.1:3000 （默认账号 admin / admin，用 GRAFANA_ADMIN_PASSWORD 覆盖）

停止：`make compose-observability-down`。

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
│   ├── prometheus.yml           # Prometheus 抓取配置（observability profile）
│   └── grafana/
│       ├── dashboard.json       # Jimu 监控面板
│       └── provisioning/        # Grafana 数据源 + 面板自动加载
├── docs/openapi/               # Swagger 生成的 API 文档
├── migrations/                 # Goose 迁移脚本
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
│   │   ├── auth/               # JWT + Casbin + Session
│   │   ├── oauth/              # OAuth 第三方登录 Provider
│   │   ├── captcha/            # 图形验证码（生成 + Redis 存储 + 校验）
│   │   ├── event/              # 事件总线
│   │   ├── observability/      # 健康检查 + Metrics + Tracing
│   │   ├── storage/            # 文件存储抽象
│   │   ├── notification/       # 通知系统
│   │   ├── outbox/             # Outbox 模式
│   │   ├── scheduler/          # Cron 调度器
│   │   ├── feature/            # Feature Flag
│   ├── shared/                 # 跨模块通用能力
│   │   ├── errors/             # AppError + 错误码
│   │   ├── response/           # 统一响应格式
│   │   ├── pagination/         # 分页
│   │   ├── validator/          # 自定义校验规则
│   │   ├── i18n/               # 国际化翻译
│   │   ├── id/                 # 雪花 ID 生成器
│   │   └── testutil/           # 测试工具
│   └── modules/                # 业务模块
│       ├── auth/               # 登录/注册/Token
│       ├── oauth/              # 第三方登录绑定
│       ├── user/               # 用户管理
│       ├── role/               # 角色管理
│       ├── permission/         # 权限管理
│       ├── audit/              # 审计日志
│       └── admin/              # 系统管理
├── tools/generator/            # 代码生成器
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
  -d '{"username": "newuser", "password": "pass1234"}'
```

### 获取系统状态

管理端点统一挂载 JWT + Casbin RBAC 认证，需携带管理员 `access_token`（无有效策略默认拒绝，返回 403）：

```bash
curl http://localhost:8080/api/v1/admin/monitoring/status \
  -H "Authorization: Bearer <access_token>"
```

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
```

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
| `db.host` | 数据库地址 | `127.0.0.1` |
| `db.port` | 数据库端口 | `3306` |
| `db.user` | 数据库用户名 | `jimu` |
| `db.password` | 数据库密码（通过环境变量覆盖） | — |
| `db.max_open` | 最大连接数 | `25`（开发）/ `100`（生产） |
| `db.max_idle` | 最大空闲连接数 | `10`（开发）/ `20`（生产） |
| `db.conn_max_lifetime_sec` | 连接最大存活时间（秒） | `3600` |
| `db.read_hosts` / `db.read_ports` | 只读副本地址 / 端口（读写分离） | — |
| `redis.addr` | Redis 地址 | `127.0.0.1:6379` |
| `redis.password` | Redis 密码（通过环境变量覆盖） | — |
| `redis.db` | Redis 数据库编号 | `0` |
| `redis.pool_size` | Redis 连接池大小 | `10`（开发）/ `50`（生产） |
| `redis.min_idle_conns` | 最小空闲连接数 | `2`（开发）/ `10`（生产） |
| `log.level` | 日志级别 | `debug`（开发）/ `info`（生产） |
| `log.format` | 日志格式 | `console`（开发）/ `json`（生产） |
| `auth.jwt_secret` | JWT 签名密钥（生产必须 `JWT_SECRET` 环境变量注入） | — |
| `auth.access_expire_min` | Access Token 有效期 (分钟) | `60`（开发）/ `15`（生产） |
| `auth.refresh_expire_day` | Refresh Token 有效期 (天) | `30`（开发）/ `7`（生产） |
| `server.timeout_sec` | 请求超时（秒），0 不限 | `30` |
| `server.rate_limit_rate` / `server.rate_limit_burst` | 全局限流速率（每秒）/ 桶容量 | `100` / `200` |
| `id.worker_id` | 雪花 ID worker 编号（0-1023）；多实例部署时每个副本需唯一，避免 ID 冲突 | `0` |
| `storage.type` | 存储类型 (`local`/`s3`/`oss`/`minio`)。`oss` 复用 S3 协议（path style + endpoint），无需阿里云 SDK；`minio` 需 `path_style: true` | `local` |
| `queue.type` | 队列类型 (`redis`/`kafka`/`rabbitmq`)，切 Kafka/RabbitMQ 时需保证 broker 可用，否则启动失败 | `redis` |
| `outbox.publisher` | Outbox 发布器类型 (`event_bus`/`mq`)。`mq` 支持 `queue.type=kafka/rabbitmq/redis` | `event_bus` |
| `scheduler.store` | 任务定义存储类型 (`memory`/`mysql`)；`mysql` 需迁移表 `scheduled_jobs`（迁移 014） | `memory` |
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
| `security.csrf_secret` | CSRF 密钥；非空时启用 CSRF 中间件（Bearer 认证请求自动跳过） | — |
| `security.content_type_options` / `frame_options` / `xss_protection` | HTTP 安全响应头（`X-Content-Type-Options` 等，留空用默认值） | 见 `DefaultSecurityConfig` |
| `security.strict_transport` | `Strict-Transport-Security` 头 | `max-age=31536000; includeSubDomains` |
| `security.content_security_policy` | `Content-Security-Policy` 头 | `default-src 'self'` |
| `cache.prefix` | 缓存 key 前缀 | `jimu` |
| `audit.queue_size` / `batch_size` / `flush_interval_ms` | 审计日志队列容量 / 批量写入条数 / 刷新间隔（ms） | `1024` / `100` / `500` |
| `otel.enabled` | 是否启用 OpenTelemetry | `false`（开发）/ `true`（生产） |

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
| `make test` | 运行测试 |
| `make test-coverage` | 测试 + 覆盖率报告 |
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
| `make compose-seed` | Compose 环境插入初始数据 |
| `make compose-observability` | 启动监控栈（Prometheus + Grafana） |
| `make compose-observability-down` | 停止监控栈 |
| `make release-check` | 发布前检查 |
| `make clean` | 清理构建产物 |
| `make hooks` | 安装 pre-commit 钩子 |

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
