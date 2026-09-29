# 运行时降级与排障

**何时读**：启动后某能力没生效或路由 404、日志出现 `capability degraded`、管理端 `/capabilities` 的 `degraded` 非空、形态起不来、或不确定该改配置还是改形态时。

**权威口径**：README「[能力开关（v0.3.0）](../../../README.md#能力开关v030)」、「[形态（profile）](../../../README.md#形态profile)」、[docs/releases/v0.3.0.md](../../../docs/releases/v0.3.0.md) 的「说明」段。

## 先分清三个时机

| 时机 | 手段 | 影响面 |
|---|---|---|
| 建项目 | `jimu new` / `jimu capability add` | 生成独立 module，`go mod tidy` 后 `go.mod` **真的变小** |
| 编译时 | 形态（profile），`PROFILE=<name> make build-server` | 决定哪些包与符号编进二进制；**`go.mod` 不变** |
| 运行时 | `capabilities.enabled` | 只决定能力是否生效，**不删表、不删数据** |

要「彻底不带某能力」用形态或脚手架，而不是靠配置开关。

## `capabilities.enabled` 语义

- 它是**完整的解析集**：除 catalog 硬依赖闭包外，**恒含七个非 catalog（`Ungated`）条目** —— `encryption` / `storage` / `notification` / `retention` / `apidocs` / `grpc` / `ws`。它们不受门控，只要该形态清单里有就会出现（缺了才是异常）
- `Requires`（硬依赖）：启用本能力会**补齐**依赖并参与拓扑序
- `SoftRequires`（软依赖）：目标缺失时**只降级**，不补齐、不参与拓扑序
- 降级判定是**声明层静态比对**（只读 `Descriptor`，不观测运行时装配）：组合根改为按启用集驱动之前可能**多报**

## 可观测点

```bash
# 启动日志
#   capabilities enabled   已装配模块集（含实际 Mount）
#   capabilities resolved  已解析启用集（含无 Module 实例的 outbox/search/breach）
#   capability degraded    warn，字段 name / missing —— 每个降级项一行

# 管理端口：只读、不鉴权
curl -s http://localhost:<mgmt-port>/capabilities
# → {"enabled":[…],"degraded":[{"capability":…,"missing":[…]}]}
```

## 受保护中间件：单提供者规则

- 启用集含 `auth` → 由 `auth` 提供受保护链（JWT + RBAC）
- 无 `auth` → 由 `apikey` 提供（`X-API-Key` + 框架唯一基线 scope `api:access` + 租户注入）
- 出现**两个非空链提供者** → 启动失败并列出冲突方（不再有「按 catalog 顺序取第一个」的隐式约定）；返回空链的提供者视为让位
- 两者都没有、却有受保护能力 → 拒绝启动。合法最小组合：`["auth"]`（闭包补齐 `user`/`access`）或 `["user","access","apikey"]`

## 形态启动冒烟

```bash
JIMU_PROFILES_SMOKE=1 make profiles-check     # 需 DB+Redis；逐形态启动并轮询管理端 /readyz
# 端口可用 JIMU_PROFILES_HTTP_PORT / JIMU_PROFILES_MGMT_PORT 覆盖
```

未设该变量时逐形态打印 `SKIP` —— **SKIP 不是通过**。

## `machine` 形态的例外

`machine` 刻意不含 `auth`，因此 `/api/v1/admin/apikeys`（位于 `middleware.AdminAuth()` 之后）无法自助签发。首把 API Key 走能力自带 CLI 带外签发：

```bash
PROFILE=machine make build-cli && ./bin/jimu-cli-machine apikey issue --name=first
```

## 常见红与定位

| 症状 | 定位 |
|---|---|
| 能力在 `catalog` 里但路由 404 | 该能力不在当前**形态清单**里（编译期就没进来）；用 `go run ./tools/profileoverlay -list` 与 `internal/profiles/<name>/assembly.go` 确认 |
| 管理端 `degraded` 列出某能力 | 它的 `SoftRequires` 目标没启用；要么补启用，要么接受降级行为 |
| `degraded` 里的东西看起来明明装配了 | 声明层静态比对的已知多报（组合根尚未按启用集驱动装配） |
| 启动报「多个受保护中间件提供者」 | 见上文单提供者规则：确认启用集里只有一个真实提供者 |
| `capabilities.enabled` 里写了非 catalog 名字 | `catalog.ValidateDeclarations()` 在 `catalog.Resolve` 前校验：必须是清单内能力名、不自引用、不与 `Requires` 重叠 |
