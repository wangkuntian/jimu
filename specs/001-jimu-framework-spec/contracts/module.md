# Module 契约

来源：`internal/contract/module.go`

## 模块接口

所有业务模块 MUST 实现统一接口：

```go
type Module interface {
    Name() string
    RegisterHTTP(r Router)
    RegisterJobs(j JobRegistry)
    RegisterEvents(e EventBus)
}
```

- `Router`：GET/POST/PUT/DELETE/Group 抽象，路由统一注册在 `/api/v1` 前缀下
- `JobRegistry`：`AddFunc(spec string, cmd func()) error` — 注册 cron 任务
- `EventBus`：`Subscribe(event, handler)` / `Publish(event, payload)` — 事件订阅发布

## 分层结构

每个模块 MUST 遵循 4 层：

```text
internal/modules/{name}/
  domain/           # 实体、值对象、仓储接口
  application/      # 用例服务、DTO
  infrastructure/   # 数据库/缓存实现
  interfaces/       # HTTP handler + 路由注册
  module.go         # 实现 contract.Module
```

业务逻辑 MUST 依赖接口（domain 仓储接口），不依赖具体实现（infrastructure）。

## 组件生命周期

```go
type Component interface {
    Start(context.Context) error
    Stop(context.Context) error
}
```

组件通过 `ComponentProvider.Components()` 暴露，由应用容器按逆序优雅停机。

## 中间件挂载

```go
type HTTPMiddlewareProvider interface {
    HTTPMiddleware() []gin.HandlerFunc
}
type ProtectedHTTPMiddlewareProvider interface {
    ProtectedHTTPMiddleware() ([]gin.HandlerFunc, error)
}
```

全局中间件 / 受保护路由中间件分别挂载。
