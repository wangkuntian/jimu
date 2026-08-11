# Jimu Backend Framework

Go 语言通用后端基础框架 — 稳定底座 + 可组合模块 + 标准适配器 + 脚手架生成能力。

## 特性

- **模块化架构** — Clean Architecture 分层，业务逻辑依赖接口不依赖实现
- **统一认证** — typed JWT + Redis refresh session + Casbin RBAC v3 权限模型
- **统一响应** — 标准 `{code, message, data}` 格式 + 分页
- **多环境配置** — Viper + yaml + 环境变量覆盖，枚举值启动校验
- **结构化日志** — Zap + lumberjack 自动滚动
- **数据库迁移** — Goose 迁移 CLI (up/down/status/redo)
- **数据初始化** — Seed 命令一键插入管理员和基础权限（含 Casbin 策略同步）
- **限流保护** — 全局令牌桶 + Redis 登录/注册固定窗口限流 + 用户维度滑动窗口限流
- **HTTP 安全边界** — 请求体大小、超时、可信代理、CORS、CSRF 防护、API 签名验证、安全 Headers
- **缓存抽象** — Cache-Aside 模式，GetOrSet 自动回填
- **自定义校验** — 手机号、密码强度、身份证、用户名等常用规则
- **事件总线** — 内存实现，支持同步/异步发布订阅
- **多队列支持** — Redis/Kafka/RabbitMQ 统一队列接口，`queue.type` 切换（Kafka/RabbitMQ 当前 at-most-once）
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
- **Outbox 模式** — 事件发布与数据库事务一致性保证，支持 MQ 跨服务发布（`outbox.publisher` 切换）
- **定时任务** — Cron 调度器（robfig/cron）
- **Feature Flag** — 运行时特性开关（灰度百分比、白名单）
- **多租户** — 租户中间件 + 行级数据隔离
- **OpenTelemetry** — 分布式追踪（OTLP gRPC）
- **Prometheus 指标** — DB 连接池 + 运行时指标
- **Docker 支持** — Dockerfile + docker-compose 一键起服务
- **Docker Secrets** — 敏感配置通过文件注入（`_FILE` 后缀）
- **K8s 部署** — Deployment/Service/HPA/Ingress manifests
- **CI/CD** — GitHub Actions + Dependabot 自动化 + 测试覆盖率门禁
- **静态检查** — golangci-lint + pre-commit 钩子（fmt / vet / lint）
- **追踪关联** — 访问日志自动注入 trace_id / span_id，关联 OpenTelemetry 追踪

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
# 1. 创建密码文件
mkdir -p secrets
echo "your-root-password" > secrets/db_root_password.txt
echo "your-db-password" > secrets/db_password.txt
openssl rand -hex 32 > secrets/jwt_secret.txt

# 2. 启动全部服务
make compose-up

# 3. 运行迁移
docker compose run --rm server ./jimu migrate up

# 4. 初始化数据
docker compose run --rm -e ADMIN_PASSWORD=admin123 server ./jimu seed
```

服务启动后访问：
- API: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html （非 release 模式）
- Management: `http://127.0.0.1:9090/livez`、`/readyz`、`/metrics`
- Adminer: `docker compose --profile dev up -d adminer` 后访问 http://127.0.0.1:8081

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
├── deploy/k8s/                 # Kubernetes manifests
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── hpa.yaml
│   └── ingress.yaml
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
│   │   ├── event/              # 事件总线
│   │   ├── observability/      # 健康检查 + Metrics + Tracing
│   │   ├── storage/            # 文件存储抽象
│   │   ├── notification/       # 通知系统
│   │   ├── outbox/             # Outbox 模式
│   │   ├── scheduler/          # Cron 调度器
│   │   ├── feature/            # Feature Flag
│   │   └── tenant/             # 多租户支持
│   ├── shared/                 # 跨模块通用能力
│   │   ├── errors/             # AppError + 错误码
│   │   ├── response/           # 统一响应格式
│   │   ├── pagination/         # 分页
│   │   ├── validator/          # 自定义校验规则
│   │   └── testutil/           # 测试工具
│   └── modules/                # 业务模块
│       ├── auth/               # 登录/注册/Token
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

### 登录

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "secret123"}'
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

```bash
curl http://localhost:8080/api/v1/admin/status
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
| `db.host` | 数据库地址 | `127.0.0.1` |
| `db.port` | 数据库端口 | `3306` |
| `db.user` | 数据库用户名 | `jimu` |
| `db.password` | 数据库密码（通过环境变量覆盖） | — |
| `db.max_open` | 最大连接数 | `25`（开发）/ `100`（生产） |
| `redis.addr` | Redis 地址 | `127.0.0.1:6379` |
| `redis.pool_size` | Redis 连接池大小 | `10`（开发）/ `50`（生产） |
| `log.level` | 日志级别 | `debug`（开发）/ `info`（生产） |
| `log.format` | 日志格式 | `console`（开发）/ `json`（生产） |
| `auth.access_expire_min` | Access Token 有效期 (分钟) | `60`（开发）/ `15`（生产） |
| `auth.refresh_expire_day` | Refresh Token 有效期 (天) | `30`（开发）/ `7`（生产） |
| `storage.type` | 存储类型 (`local`/`s3`/`oss`/`minio`) | `local` |
| `queue.type` | 队列类型 (`redis`/`kafka`/`rabbitmq`) | `redis` |
| `outbox.publisher` | Outbox 发布器类型 (`event_bus`/`mq`) | `event_bus` |
| `otel.enabled` | 是否启用 OpenTelemetry | `false`（开发）/ `true`（生产） |

## Makefile 命令

| 命令 | 说明 |
|------|------|
| `make run` | 运行服务 |
| `make build` | 编译 server + cli |
| `make test` | 运行测试 |
| `make test-coverage` | 测试 + 覆盖率报告 |
| `make vet` | 静态分析 |
| `make fmt` | 格式化代码 |
| `make lint` | golangci-lint |
| `make migrate` | 执行迁移 |
| `make seed` | 插入初始数据 |
| `make swagger` | 生成 API 文档 |
| `make cli` | 编译 CLI |
| `make docker-build` | 构建 Docker 镜像 |
| `make compose-up` | 启动所有容器 |
| `make compose-down` | 停止所有容器 |
| `make compose-logs` | 查看应用日志 |
| `make compose-migrate` | Compose 环境执行迁移 |
| `make compose-seed` | Compose 环境插入初始数据 |
| `make release-check` | 发布前检查 |
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

## License

MIT
