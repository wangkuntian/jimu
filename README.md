# Jimu Backend Framework

Go 语言通用后端基础框架 — 稳定底座 + 可组合模块 + 标准适配器 + 脚手架生成能力。

## 特性

- **模块化架构** — Clean Architecture 分层，业务逻辑依赖接口不依赖实现
- **统一认证** — typed JWT + Redis refresh session + Casbin RBAC 权限模型
- **统一响应** — 标准 `{code, message, data}` 格式 + 分页
- **多环境配置** — Viper + yaml + 环境变量覆盖，枚举值启动校验
- **结构化日志** — Zap + lumberjack 自动滚动
- **数据库迁移** — Goose 迁移 CLI (up/down/status/redo)
- **数据初始化** — Seed 命令一键插入管理员和基础权限
- **限流保护** — 全局令牌桶 + Redis 登录/注册固定窗口限流
- **HTTP 安全边界** — 请求体大小、超时、可信代理、allow-list CORS
- **缓存抽象** — Cache-Aside 模式，GetOrSet 自动回填
- **自定义校验** — 手机号、密码强度、身份证、用户名等常用规则
- **事件总线** — 内存实现，支持同步/异步发布订阅
- **事务封装** — 统一的事务管理 helper
- **审计日志** — 有界队列批量写入，匿名请求安全处理
- **管理端点** — 独立 management server 暴露健康检查、metrics 和可选 pprof
- **脚手架** — Cobra CLI 一键生成完整模块骨架
- **API 文档** — Swagger UI 交互式文档
- **健康检查** — `/livez` 与 `/readyz`，readiness 有界探测 DB + Redis
- **优雅停机** — 显式 Application 生命周期，反向停止组件
- **Docker 支持** — Dockerfile + docker-compose 一键起服务
- **CI/CD** — GitHub Actions + Dependabot 自动化

## 技术栈

| 类别 | 选型 |
|------|------|
| HTTP 框架 | Gin |
| ORM | Gorm |
| 数据库 | MariaDB (MySQL 协议) |
| 缓存 | Redis |
| 配置 | Viper |
| 日志 | Zap + lumberjack |
| 鉴权 | JWT + Casbin |
| 迁移 | Goose |
| CLI | Cobra |
| 校验 | go-playground/validator |
| API 文档 | swaggo/swag |

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
cp configs/app.yaml configs/app.local.yaml
# 编辑 configs/app.local.yaml 修改数据库、Redis 连接信息
```

### 方式一：本地运行

```bash
# 1. 启动依赖
docker-compose up -d mariadb redis

# 2. 运行迁移
make migrate

# 3. 初始化数据（管理员密码通过受保护环境变量或交互输入提供）
make seed

# 4. 启动服务
make run
```

### 方式二：Docker Compose 一键启动

```bash
cp .env.example .env
# 编辑 .env 修改配置
docker-compose up -d
```

服务启动后访问：
- API: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html （非 release 模式）
- Management: `http://127.0.0.1:9090/livez`、`/readyz`、`/metrics`（仅绑定宿主机 loopback）
- Adminer: `docker compose --profile dev up -d adminer` 后访问 http://127.0.0.1:8081

公共 API 不暴露 `/debug/pprof/` 或 metrics。pprof 默认关闭，只能通过 management server 显式开启。

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
./bin/jimu seed                     # 插入初始数据
```

## 项目结构

```text
jimu/
├── cmd/
│   ├── server/main.go          # HTTP 服务入口
│   └── cli/main.go             # CLI 入口
├── configs/
│   ├── app.yaml                # 默认配置（开发环境）
│   ├── app.prod.yaml           # 生产环境配置
│   └── app.test.yaml           # 测试环境配置
├── conf/
│   └── rbac_model.conf         # Casbin RBAC 模型
├── migrations/                 # Goose 迁移脚本
├── internal/
│   ├── app/
│   │   ├── bootstrap.go        # 应用启动装配
│   │   └── container.go        # 依赖容器
│   ├── config/                 # 配置加载 + 校验
│   ├── contract/               # Module 接口定义
│   ├── platform/               # 基础设施
│   │   ├── http/               # HTTP Server + 中间件
│   │   ├── db/                 # Gorm 连接 + 迁移 + Seed + 事务
│   │   ├── redis/              # Redis 客户端
│   │   ├── cache/              # 缓存抽象层
│   │   ├── logger/             # Zap 日志
│   │   ├── auth/               # JWT + Casbin
│   │   ├── event/              # 事件总线
│   │   └── observability/      # 健康检查 + Metrics
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
│       └── audit/              # 审计日志
├── pkg/                        # 对外暴露的工具
├── tools/generator/            # 代码生成器
├── .github/                    # GitHub Actions + Dependabot
├── Makefile
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

## API 示例

### 注册（可选）

公开注册默认关闭。需要公网用户自助注册时设置：

```bash
JIMU__AUTH__PUBLIC_REGISTRATION=true
```

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "secret123"}'
```

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

Refresh token 存在 Redis session 中，刷新成功后旧 refresh token 立即失效。

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

### 退出登录

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <access_token>"

curl -X POST http://localhost:8080/api/v1/auth/logout-all \
  -H "Authorization: Bearer <access_token>"
```

### 创建用户

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"username": "newuser", "password": "pass1234"}'
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

## API 契约


公共 API 固定在 `/api/v1` 下。响应使用 `{code,message,data}` 外形；错误响应可省略 `data`，分页响应额外包含 `total`、`page`、`page_size`。`X-Request-ID` 会写入响应 Header，统一响应体在存在 request ID 时也包含 `request_id`。

HTTP status 表达协议结果，业务 `code` 表达稳定业务结果。内部错误对外固定为 `{"code":1005,"message":"internal error"}`，不返回 SQL、文件路径、堆栈或底层基础设施错误。

列表接口统一支持以下查询参数：

| 参数 | 默认值 | 约束 |
|------|--------|------|
| `page` | `1` | 最小 `1` |
| `page_size` | `20` | 最小 `1`，最大 `100` |
| `sort` | `id` | 必须在 handler allow-list 内 |
| `order` | `desc` | 仅 `asc` 或 `desc` |
| `filter` | 空 | 自动 trim 空白 |

## API 契约检查

```bash
work/test_runtime_security.sh
work/smoke_api_contract.sh
make swagger
make swagger
git diff -- docs/openapi
```

CI 会在干净工作区执行 OpenAPI 生成并检查 `docs/openapi` 是否与仓库一致。本地未提交阶段性改动时，以上命令用于确认重复生成没有额外漂移。

## 配置说明

### 多环境配置

通过 `JIMU_ENV` 环境变量切换：

| 环境 | 配置文件 | 说明 |
|------|----------|------|
| 开发 | `app.yaml` | 默认，日志输出到 stdout |
| 测试 | `app.test.yaml` | 独立数据库 `jimu_test` |
| 生产 | `app.prod.yaml` | JSON 日志、文件滚动、release 模式 |

优先级：`环境变量 > app.{env}.yaml > app.yaml`

### 配置项

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `http.host` | 监听地址 | `0.0.0.0` |
| `http.port` | 监听端口 | `8080` |
| `http.mode` | Gin 模式 (`debug`/`release`/`test`) | `debug` |
| `http.read_header_timeout_sec` | Header 读取超时 | `5` |
| `http.read_timeout_sec` | 请求读取超时 | `15` |
| `http.write_timeout_sec` | 响应写入超时 | `30` |
| `http.idle_timeout_sec` | Keep-alive 空闲超时 | `60` |
| `http.shutdown_timeout_sec` | 关闭等待超时 | `30` |
| `http.max_body_bytes` | 最大请求体字节数 | `1048576` |
| `http.trusted_proxies` | 可信代理列表 | `["127.0.0.1"]` |
| `http.allowed_origins` | CORS allow-list | `["http://localhost:3000"]` |
| `management.host` | Management server 监听地址 | `127.0.0.1` |
| `management.port` | Management server 端口 | `9090` |
| `management.enable_pprof` | 是否开启 pprof | `false` |
| `management.probe_timeout_sec` | readiness 依赖探测超时 | `2` |
| `db.*` | 数据库连接配置 | — |
| `redis.*` | Redis 连接配置 | — |
| `log.level` | 日志级别 (`debug`/`info`/`warn`/`error`) | `debug` |
| `log.format` | 日志格式 (`json`/`console`) | `console` |
| `log.output` | 输出目标 (`stdout` 或文件路径) | `stdout` |
| `log.max_size` | 单文件最大大小 (MB) | `100` |
| `log.max_backups` | 保留旧文件数 | `30` |
| `log.max_age` | 保留天数 | `7` |
| `log.compress` | 旧文件压缩 | `true` |
| `auth.jwt_secret` | JWT 密钥，生产必须用 `JIMU__AUTH__JWT_SECRET` 覆盖 | `change-me-in-production` |
| `auth.issuer` | JWT issuer | `jimu` |
| `auth.access_expire_min` | Access Token 有效期 (分钟) | `30` |
| `auth.refresh_expire_day` | Refresh Token 有效期 (天) | `7` |
| `auth.public_registration` | 是否开放公网注册 | `false` |
| `auth.login_rate_limit` | 登录窗口内允许次数 | `10` |
| `auth.login_rate_window_sec` | 登录限流窗口秒数 | `60` |
| `auth.register_rate_limit` | 注册窗口内允许次数 | `5` |
| `auth.register_rate_window_sec` | 注册限流窗口秒数 | `300` |
| `server.timeout_sec` | 请求超时秒数 | `30` |
| `server.rate_limit_rate` | 全局限流速率 | `100` |
| `server.rate_limit_burst` | 限流桶容量 | `200` |
| `cache.prefix` | 缓存 key 前缀 | `jimu` |
| `audit.queue_size` | 审计队列容量 | `256` |
| `audit.batch_size` | 审计批写大小 | `50` |
| `audit.flush_interval_ms` | 审计批写间隔毫秒 | `500` |

### 环境变量

前缀 `JIMU`，层级分隔 `__`，例如 `JIMU__HTTP__PORT=9090`。

生产配置不依赖 YAML `${VAR}` 插值。敏感值通过环境变量覆盖，例如：

```bash
JIMU__DB__PASSWORD=replace-with-strong-password
JIMU__AUTH__JWT_SECRET=replace-with-at-least-32-characters
JIMU__AUTH__PUBLIC_REGISTRATION=false
JIMU__HTTP__ALLOWED_ORIGINS=https://admin.example.com
```

生产启动会拒绝弱 JWT secret、空/默认 DB 密码、wildcard CORS、非法端口和非法超时配置。错误信息只包含配置键，不输出敏感值。

## 模块开发

### 创建新模块

```bash
./bin/jimu module create product
```

生成完整 Clean Architecture 骨架：

```
internal/modules/product/
  module.go              # 模块注册
  domain/
    entity.go            # 实体（含基础字段 + gorm tag）
    repository.go        # 仓储接口（CRUD）
  application/
    service.go           # 用例服务（CRUD 框架）
    dto.go               # 请求/响应 DTO
  infrastructure/
    mysql_repository.go  # Gorm 实现
  interfaces/
    handler.go           # HTTP handler（CRUD 端点）
    router.go            # 路由注册（RESTful）
```

### 注册模块

在 `cmd/server/main.go` 中添加：

```go
productModule := product.New(dbConn)
application, err := app.Bootstrap(container, userModule, authModule, roleModule, permModule, auditModule, productModule)
```

## Makefile 命令

| 命令 | 说明 |
|------|------|
| `make run` | 运行服务 |
| `make run ENV=prod` | 指定环境运行 |
| `make build` | 编译 server + cli |
| `make test` | 运行测试 |
| `make test-coverage` | 测试 + 覆盖率报告 |
| `make vet` | 静态分析 |
| `make fmt` | 格式化代码 |
| `make lint` | golangci-lint |
| `make migrate` | 执行迁移 |
| `make migrate-down` | 回滚迁移 |
| `make migrate-status` | 查看迁移状态 |
| `make seed` | 插入初始数据 |
| `make swagger` | 生成 API 文档 |
| `make cli` | 编译 CLI |
| `make docker` | 构建镜像 |
| `make docker-up` | 启动所有容器 |
| `make docker-down` | 停止所有容器 |
| `make docker-logs` | 查看应用日志 |
| `make release-check` | 发布前检查 |

## Docker 部署

```bash
# 构建镜像
docker build -t jimu:latest .

# 运行（需外部 DB + Redis）
docker run -p 8080:8080 \
  -p 127.0.0.1:9090:9090 \
  -e JIMU_ENV=prod \
  -e JIMU__DB__HOST=host.docker.internal \
  -e JIMU__DB__USER=jimu \
  -e JIMU__DB__PASSWORD=replace-with-strong-password \
  -e JIMU__REDIS__ADDR=host.docker.internal:6379 \
  -e JIMU__AUTH__JWT_SECRET=replace-with-at-least-32-characters \
  jimu:latest

# 或使用 docker-compose 一键启动全部
cp .env.example .env
# 编辑 .env：替换 DB_ROOT_PASSWORD、DB_PASSWORD、JIMU__DB__PASSWORD、JIMU__AUTH__JWT_SECRET
docker-compose up -d
```

## License

MIT
