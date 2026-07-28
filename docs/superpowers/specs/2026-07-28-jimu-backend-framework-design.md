# Jimu Backend Framework — 设计文档

> 创建日期：2026-07-28
> 状态：已确认
> 参考：`docs/sepc.md`

---

## 1. 项目定位

Jimu 是一个用 Go 语言开发的通用后端基础框架，目标是为中后台业务提供稳定底座 + 可组合模块 + 标准适配器 + 脚手架生成能力。

**核心原则**：
- 业务模块化
- 依赖接口化
- 基础设施平台化
- 模块注册标准化
- 错误响应统一化
- 权限认证底座化

---

## 2. 首个版本范围

**实现范围**：底座 + 认证模块 + RBAC 权限

**明确不做**：
- 多租户（不预留 tenant_id）
- 消息队列
- 插件化
- 完整的代码生成器（只做 module create）

---

## 3. 技术选型

| 类别 | 选型 | 说明 |
|------|------|------|
| HTTP 框架 | Gin | 高性能、生态成熟 |
| ORM | Gorm | 开发效率高，适合中后台 |
| 数据库 | MariaDB | MySQL 协议兼容 |
| 配置 | Viper | 支持 yaml + 环境变量 |
| 日志 | Zap | 高性能结构化日志 |
| 缓存 | Redis | 会话/Token 存储 |
| 鉴权 | JWT + Casbin | 认证 + RBAC |
| 迁移 | Goose | SQL 迁移脚本 |
| 校验 | go-playground/validator | 请求参数校验 |
| API 文档 | swaggo/swag | 注释生成 Swagger |
| CLI | Cobra | 脚手架命令 |
| 链路追踪 | OpenTelemetry | 可观测性预留 |

---

## 4. 项目结构

```text
jimu/
├── cmd/
│   ├── server/main.go          # HTTP 服务入口
│   └── cli/main.go             # cobra CLI 入口
├── configs/
│   └── app.yaml                # 默认配置
├── migrations/                 # Goose 迁移脚本
│   ├── 001_create_users.sql
│   ├── 002_create_roles.sql
│   ├── 003_create_permissions.sql
│   └── 004_create_user_roles.sql
├── internal/
│   ├── app/
│   │   └── bootstrap.go        # 应用启动、依赖装配
│   ├── config/
│   │   └── config.go           # 配置结构 + Load
│   ├── contract/
│   │   └── module.go           # Module 接口定义
│   ├── platform/
│   │   ├── http/
│   │   │   ├── server.go       # HTTP Server 封装
│   │   │   └── middleware/     # RequestID/Logger/Recovery/CORS/RateLimit
│   │   ├── db/
│   │   │   └── mysql.go        # Gorm 连接
│   │   ├── redis/
│   │   │   └── redis.go        # Redis 客户端
│   │   ├── logger/
│   │   │   └── zap.go          # Zap 封装
│   │   ├── auth/
│   │   │   ├── jwt.go          # JWT 生成/解析
│   │   │   └── casbin.go       # Casbin 权限引擎
│   │   └── observability/
│   │       └── health.go       # Health check
│   ├── shared/
│   │   ├── errors/             # AppError + 错误码
│   │   ├── response/           # 统一响应格式
│   │   ├── pagination/         # 分页请求/响应
│   │   └── validator/          # 请求校验封装
│   └── modules/
│       ├── auth/               # 登录/注册/Token
│       │   ├── application/
│       │   ├── domain/
│       │   ├── infrastructure/
│       │   └── interfaces/
│       ├── user/               # 用户 CRUD
│       ├── role/               # 角色管理
│       └── permission/         # 权限管理
├── pkg/
│   └── middleware/             # 可复用中间件（对外暴露）
├── go.mod
├── Makefile
└── Dockerfile
```

---

## 5. 核心契约

### 5.1 Module 接口

```go
// internal/contract/module.go
package contract

import "github.com/gin-gonic/gin"

// Router 抽象路由注册器
type Router interface {
    GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
    POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
    PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
    DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
    Group(relativePath string, handlers ...gin.HandlerFunc) Router
}

// JobRegistry 定时任务注册器
type JobRegistry interface {
    AddFunc(spec string, cmd func()) error
}

// EventBus 事件总线
type EventBus interface {
    Subscribe(event string, handler func(payload interface{}))
    Publish(event string, payload interface{})
}

// Module 统一模块接口
type Module interface {
    Name() string
    RegisterHTTP(r Router)
    RegisterJobs(j JobRegistry)
    RegisterEvents(e EventBus)
}
```

### 5.2 启动装配

```go
// internal/app/bootstrap.go
package app

func Bootstrap(modules ...contract.Module) *platformhttp.Server {
    // 1. 加载配置
    cfg := config.Load()

    // 2. 初始化基础设施
    db := platformdb.New(cfg.DB)
    rdb := platformredis.New(cfg.Redis)
    log := platformlogger.New(cfg.Log)

    // 3. 构建容器
    container := NewContainer(cfg, db, rdb, log)

    // 4. 创建路由根节点
    router := platformhttp.NewRouter()

    // 5. 注册模块
    for _, m := range modules {
        m.RegisterHTTP(router)
        log.Info("module registered", "name", m.Name())
    }

    // 6. 返回 HTTP Server
    return platformhttp.NewServer(cfg.HTTP, router)
}
```

### 5.3 main.go 调用

```go
// cmd/server/main.go
func main() {
    server := app.Bootstrap(
        authmodule.New(container),
        usermodule.New(container),
        rolemodule.New(container),
        permmodule.New(container),
    )
    server.Run()
}
```

---

## 6. 认证与权限

### 6.1 认证流程（JWT）

```
客户端 → POST /api/v1/auth/login {username, password}
  → AuthService.Login()
    → UserRepository.FindByUsername()  [查用户]
    → bcrypt.CompareHashAndPassword()  [校验密码]
    → JWT.Generate()                   [生成 access_token + refresh_token]
  → 返回 {access_token, refresh_token, expires_in}
```

### 6.2 中间件链

```
Request → RequestID → Logger → Recovery → CORS → AuthMiddleware → CasbinMiddleware → Handler
```

- `AuthMiddleware`：解析 JWT → 提取 userID/roles → 注入 context
- `CasbinMiddleware`：根据路由 + 方法 + 用户角色做权限校验

### 6.3 RBAC 模型（Casbin）

```ini
# conf/rbac_model.conf
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
```

### 6.4 基础表设计

| 表名 | 说明 |
|------|------|
| `users` | 用户（id, username, password, status, created_at, updated_at, deleted_at） |
| `roles` | 角色（id, name, description, status） |
| `permissions` | 权限（id, name, resource, action） |
| `user_roles` | 用户-角色关联 |
| `role_permissions` | 角色-权限关联 |

权限粒度示例：`user:list`, `user:create`, `user:update`, `user:delete`（resource:action 格式存 Casbin policy）

---

## 7. 统一响应与错误处理

### 7.1 响应格式

```go
// internal/shared/response/response.go
type Body struct {
    Code    int         `json:"code"`    // 0 = 成功, 非0 = 业务错误码
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

type Paginated struct {
    Code     int         `json:"code"`
    Message  string      `json:"message"`
    Data     interface{} `json:"data"`
    Total    int64       `json:"total"`
    Page     int         `json:"page"`
    PageSize int         `json:"page_size"`
}
```

### 7.2 错误码

| Code | 标识 | 说明 |
|------|------|------|
| 0 | OK | 成功 |
| 1001 | INVALID_PARAM | 参数校验失败 |
| 1002 | UNAUTHORIZED | 未登录或 Token 无效 |
| 1003 | FORBIDDEN | 无权限 |
| 1004 | NOT_FOUND | 资源不存在 |
| 1005 | INTERNAL_ERROR | 服务器内部错误 |
| 2001 | USER_NOT_FOUND | 用户不存在 |
| 2002 | USER_ALREADY_EXISTS | 用户已存在 |
| 2003 | INVALID_PASSWORD | 密码错误 |
| 2004 | ROLE_NOT_FOUND | 角色不存在 |

### 7.3 AppError

```go
type AppError struct {
    Code    int
    Message string
    Cause   error  // 原始错误，不暴露给客户端
}

func (e *AppError) Error() string { ... }
func (e *AppError) Unwrap() error { return e.Cause }
```

### 7.4 错误处理中间件

- 业务代码返回 `AppError`
- 中间件统一捕获并转换为 `response.Body`
- HTTP 状态码始终 200，业务码在 body.code 中体现
- 内部错误记录日志，但不暴露细节给客户端

### 7.5 分页请求

```go
type Pagination struct {
    Page     int `form:"page" binding:"min=1"`
    PageSize int `form:"page_size" binding:"min=1,max=100"`
}
```

---

## 8. CLI 脚手架

```go
// cmd/cli/main.go
var moduleCmd = &cobra.Command{
    Use:   "module create [name]",
    Short: "Create a new module skeleton",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        name := args[0]
        return generator.GenerateModule(name)
    },
}
```

生成内容：
```text
internal/modules/{name}/
  ├── application/
  │   └── service.go
  ├── domain/
  │   ├── entity.go
  │   └── repository.go
  ├── infrastructure/
  │   └── mysql_repository.go
  └── interfaces/
      ├── handler.go
      └── router.go
```

---

## 9. 配置管理

```go
type Config struct {
    HTTP platformhttp.Config `mapstructure:"http"`
    DB   platformdb.Config   `mapstructure:"db"`
    Redis platformredis.Config `mapstructure:"redis"`
    Log  platformlogger.Config `mapstructure:"log"`
    Auth AuthConfig           `mapstructure:"auth"`
}

type AuthConfig struct {
    JWTSecret        string `mapstructure:"jwt_secret"`
    AccessExpireMin  int    `mapstructure:"access_expire_min"`
    RefreshExpireDay int    `mapstructure:"refresh_expire_day"`
}
```

加载优先级：`configs/app.yaml` 默认值 → 环境变量 `JIMU__SECTION__KEY` 覆盖。

---

## 10. 模块内部结构（以 user 为例）

```text
internal/modules/user/
  domain/
    user.go             # 实体、值对象、领域规则
    repository.go       # 仓储接口
  application/
    service.go          # 用例服务
    dto.go              # 输入输出模型
  infrastructure/
    mysql_repository.go # MySQL 实现
  interfaces/
    http_handler.go     # HTTP 接口
    router.go           # 路由注册
```

业务服务依赖接口，不依赖具体实现：

```go
type UserService struct {
    repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService {
    return &UserService{repo: repo}
}
```

---

## 11. 落地顺序

1. 搭建基础项目结构 + go.mod 依赖
2. 配置（Viper）+ 日志（Zap）
3. HTTP Server（Gin）+ 中间件
4. 数据库连接（Gorm + MariaDB）+ 迁移（Goose）
5. 统一错误（AppError）+ 统一响应
6. 用户模块（domain → application → infrastructure → interfaces）
7. 认证模块（JWT + 登录/注册）
8. RBAC 权限（Casbin + 角色/权限管理）
9. Module 接口 + Bootstrap 装配
10. swaggo/swag API 文档
11. cobra CLI 基础脚手架
12. Makefile + Dockerfile

---

## 12. 不做的事（明确排除）

- 多租户支持
- 消息队列集成
- 插件化（Go plugin）
- 完整的代码生成器（CRUD 生成不做）
- gRPC 服务
- 管理后台前端
