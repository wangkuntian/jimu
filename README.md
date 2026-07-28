# Jimu Backend Framework

Go 语言通用后端基础框架 — 稳定底座 + 可组合模块 + 标准适配器 + 脚手架生成能力。

## 特性

- **模块化架构** — Clean Architecture 分层，业务逻辑依赖接口不依赖实现
- **统一认证** — JWT + Casbin RBAC 权限模型
- **统一响应** — 标准 `{code, message, data}` 格式 + 分页
- **配置灵活** — Viper + yaml + 环境变量覆盖，枚举值启动校验
- **结构化日志** — Zap + lumberjack 自动滚动
- **脚手架** — Cobra CLI 一键生成模块骨架
- **API 文档** — Swagger/OpenAPI 注释生成
- **健康检查** — `/health` 端点，DB + Redis 探活

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

复制并修改配置文件：

```bash
cp configs/app.yaml configs/app.local.yaml
# 编辑 configs/app.local.yaml 修改数据库、Redis 连接信息
```

### 运行

```bash
# 直接运行
make run

# 或编译后运行
make build
./bin/server
```

### CLI 工具

```bash
# 生成新模块骨架
make cli
./bin/jimu module create product
```

## 项目结构

```text
jimu/
├── cmd/
│   ├── server/main.go          # HTTP 服务入口
│   └── cli/main.go             # CLI 入口
├── configs/
│   └── app.yaml                # 默认配置
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
│   │   ├── db/                 # Gorm 连接
│   │   ├── redis/              # Redis 客户端
│   │   ├── logger/             # Zap 日志
│   │   ├── auth/               # JWT + Casbin
│   │   └── observability/      # 健康检查
│   ├── shared/                 # 跨模块通用能力
│   │   ├── errors/             # AppError + 错误码
│   │   ├── response/           # 统一响应格式
│   │   └── pagination/         # 分页
│   └── modules/                # 业务模块
│       ├── auth/               # 登录/注册/Token
│       ├── user/               # 用户管理
│       ├── role/               # 角色管理
│       └── permission/         # 权限管理
├── pkg/                        # 对外暴露的工具
├── tools/generator/            # 代码生成器
├── Makefile
└── Dockerfile
```

## API 示例

### 注册

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

### 创建用户

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"username": "newuser", "password": "pass1234"}'
```

### 健康检查

```bash
curl http://localhost:8080/health
```

## 配置说明

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `http.host` | 监听地址 | `0.0.0.0` |
| `http.port` | 监听端口 | `8080` |
| `http.mode` | Gin 模式 (`debug`/`release`/`test`) | `debug` |
| `db.*` | 数据库连接配置 | — |
| `redis.*` | Redis 连接配置 | — |
| `log.level` | 日志级别 (`debug`/`info`/`warn`/`error`) | `debug` |
| `log.format` | 日志格式 (`json`/`console`) | `console` |
| `log.output` | 输出目标 (`stdout` 或文件路径) | `stdout` |
| `log.max_size` | 单文件最大大小 (MB) | `100` |
| `log.max_backups` | 保留旧文件数 | `30` |
| `log.max_age` | 保留天数 | `7` |
| `log.compress` | 旧文件压缩 | `true` |
| `auth.jwt_secret` | JWT 密钥 | `change-me-in-production` |
| `auth.access_expire_min` | Access Token 有效期 (分钟) | `30` |
| `auth.refresh_expire_day` | Refresh Token 有效期 (天) | `7` |

环境变量覆盖：前缀 `JIMU`，层级分隔 `__`，例如 `JIMU__HTTP__PORT=9090`。

## 模块开发

### 创建新模块

```bash
./bin/jimu module create product
```

生成 `internal/modules/product/module.go` 骨架，然后按 Clean Architecture 补充：

1. `domain/` — 定义实体和仓储接口
2. `application/` — 实现业务逻辑
3. `infrastructure/` — 实现数据持久化
4. `interfaces/` — 实现 HTTP handler 和路由

### 注册模块

在 `cmd/server/main.go` 中添加：

```go
productModule := product.New(dbConn)
server := app.Bootstrap(userModule, authModule, roleModule, permModule, productModule)
```

## Docker 部署

```bash
docker build -t jimu:latest .
docker run -p 8080:8080 -v ./configs:/app/configs jimu:latest
```

## License

MIT
