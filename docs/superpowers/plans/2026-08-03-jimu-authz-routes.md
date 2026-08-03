# Jimu Authz Routes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让业务路由在运行时真正受 JWT 认证、角色加载和 Casbin 权限检查保护。

**Architecture:** Auth 模块提供业务路由保护中间件，Bootstrap 先收集中间件再注册模块路由。角色加载通过 Gorm 从 `user_roles` 和 `roles` 查询角色名，并写入 Gin context 的 `roles`。

**Tech Stack:** Go 1.26.5、Gin、Gorm、Casbin、现有 `internal/shared/errors` 与 `response`。

## Global Constraints

- 不新增依赖。
- 不新增外部服务。
- 不改变 token 格式。
- 不改变现有 API 路径。
- Auth 公开路由保持公开。
- 业务模块 User、Role、Permission、Audit 必须受保护。

---

### Task 1: Authz Middleware

**Files:**
- Create: `internal/platform/auth/roles.go`
- Create: `internal/platform/auth/roles_test.go`
- Modify: `internal/platform/auth/permission_middleware.go`
- Create: `internal/modules/auth/interfaces/protected.go`
- Create: `internal/modules/auth/interfaces/protected_test.go`

**Interfaces:**
- Produces: `type RoleLoader interface { RolesForUser(ctx context.Context, userID uint64) ([]string, error) }`
- Produces: `func RoleLoaderMiddleware(loader platformauth.RoleLoader) gin.HandlerFunc`
- Produces: `func ProtectedMiddleware(jwtUtil *platformauth.JWT, loader platformauth.RoleLoader, enforcer *casbin.Enforcer) []gin.HandlerFunc`

- [ ] **Step 1: Write failing middleware tests**

Run: `GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/modules/auth/interfaces ./internal/platform/auth -run 'TestProtected|TestPermission'`

Expected: FAIL because `ProtectedMiddleware` and `RoleLoaderMiddleware` do not exist.

- [ ] **Step 2: Implement role loader and protected middleware**

Implement only role lookup, context population, and existing permission middleware wiring.

- [ ] **Step 3: Verify middleware tests pass**

Run: `GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/modules/auth/interfaces ./internal/platform/auth`

Expected: PASS.

### Task 2: Wire Business Routes

**Files:**
- Modify: `internal/modules/auth/module.go`
- Modify: `internal/app/bootstrap.go`
- Create: `internal/app/authz_bootstrap_test.go`

**Interfaces:**
- Consumes: `ProtectedMiddleware(jwtUtil, loader, enforcer)`
- Produces:业务模块路由注册前已挂载保护中间件。

- [ ] **Step 1: Write failing bootstrap test**

Test `GET /api/v1/users` without token returns `401` when app bootstraps auth + user modules.

Run: `GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/app -run TestBusinessRoutesRequireAuth`

Expected: FAIL, currently returns non-401.

- [ ] **Step 2: Implement module middleware wiring**

Auth module creates role loader and enforcer. Bootstrap applies protected middleware before registering non-auth business modules.

- [ ] **Step 3: Verify bootstrap test passes**

Run: `GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/app -run TestBusinessRoutesRequireAuth`

Expected: PASS.

### Task 3: Final Verification

**Files:**
- Modify: `docs/superpowers/plans/2026-08-03-jimu-authz-routes.md`

- [ ] **Step 1: Run full checks**

Run:

```bash
gofmt -w internal/platform/auth internal/modules/auth internal/app
GOCACHE=/private/tmp/jimu-go-build-cache go test ./...
GOCACHE=/private/tmp/jimu-go-build-cache go build ./cmd/server ./cmd/cli
git diff --check
```

Expected: all commands exit 0.

- [ ] **Step 2: Commit**

Run:

```bash
git add docs/superpowers/specs/2026-08-03-jimu-authz-routes-design.md docs/superpowers/plans/2026-08-03-jimu-authz-routes.md internal/platform/auth internal/modules/auth internal/app
git commit -m "feat: protect business routes with authz"
```
