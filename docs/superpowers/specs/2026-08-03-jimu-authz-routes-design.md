# Jimu 业务路由认证授权设计

## 目标

让 `users`、`roles`、`permissions`、`audits` 业务路由在运行时真正受 JWT 认证和 Casbin 授权保护，保持 Auth 公开/保护边界不变。

## 范围

- `POST /auth/login`、`POST /auth/register`、`POST /auth/refresh` 保持公开。
- `POST /auth/logout`、`POST /auth/logout-all` 保持 JWT 认证保护。
- User、Role、Permission、Audit 模块路由统一要求 JWT。
- 通过 `user_roles` 和 `roles` 查询当前用户角色，写入 Gin context 的 `roles`。
- 使用现有 `PermissionMiddleware` 调用 Casbin 判断 `path + method`。
- 不新增依赖、不新增服务、不改变 token 格式、不改变 API 路径。

## 设计

新增一个轻量角色加载仓储，负责按 user ID 查询角色名。Auth 模块构造时创建 Casbin enforcer 和角色加载器，并把认证、角色加载、权限检查作为中间件提供给业务模块复用。

业务模块的 `RegisterHTTP` 继续只负责路由注册；认证授权由 `Bootstrap` 在注册模块路由前统一挂载模块提供的 HTTP middleware。Auth 模块自身不把授权中间件挂到 `/auth` 公开路由上，避免破坏登录、注册和刷新。

## 错误行为

- 无 `Authorization` header：`401`。
- token 无效：`401`。
- 无角色：`403`。
- 角色无权限：`403`。
- 角色查询失败：脱敏 `500`。

## 验证

- Router 测试覆盖业务路由未带 token 返回 `401`。
- Authz middleware 测试覆盖角色加载成功、无角色、查询错误。
- 权限测试覆盖无权限 `403` 和有权限放行。
- 全量运行 `go test ./...` 与 `go build ./cmd/server ./cmd/cli`。
