# HTTP API 契约

来源：各模块 `interfaces/router.go` + swagger 注解。所有业务接口在 `/api/v1` 前缀下，返回统一响应格式（见 [response.md](response.md)）。

## 健康与观测（非业务）

| 路径 | 用途 |
|------|------|
| `GET /livez` | 存活检查 |
| `GET /readyz` | 就绪检查（探测 DB + Redis） |
| management server | 健康 / metrics / 可选 pprof（独立端口） |

## Auth 模块

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register` | 注册（可含验证码） |
| POST | `/api/v1/auth/login` | 登录（固定窗口限流） |
| POST | `/api/v1/auth/refresh` | 刷新令牌 |
| POST | `/api/v1/auth/logout` | 登出 |
| POST | `/api/v1/auth/logout-all` | 全设备登出 |
| GET  | `/api/v1/auth/oauth/{provider}/login` | 第三方登录入口（google/github/wechat） |

## Admin 模块

| 组 | 能力 |
|----|------|
| `/api/v1/admin/status` | 系统状态 |
| `/api/v1/admin/users` | 用户 CRUD、在线用户、强制下线 |
| `/api/v1/admin/apikeys` | API 密钥管理 |
| `/api/v1/admin/error-codes` | 错误码文档 |
| `/api/v1/admin/features` | 特性开关管理 |
| `/api/v1/admin/tasks` / `/jobs` / `/import` | 定时任务、作业、导入管理 |

## 中间件

| 中间件 | 挂载 |
|--------|------|
| JWT 认证 | 受保护路由 |
| Casbin RBAC | 按 `(resource, action)` 鉴权 |
| API Key | 服务间调用（`X-API-Key` 头，按需挂载） |
| 限流 | 全局令牌桶 + 登录固定窗口 + 用户滑动窗口 |
| 安全 | 体积/超时/可信代理/CORS/安全头 |
| CSRF / 签名 | 可选（`security.csrf_secret` / API 签名） |
| 审计 | 批量异步写 `audit_logs` |
| 追踪 | OTel span（开启时），日志注入 trace_id/span_id |

## 完整接口文档

Swagger UI：`internal/platform/http/swagger.go`（中文注解），文档源 `docs/openapi/swagger.yaml`。
