# 系统角色与租户开通角色设计

## 背景

当前结构性 seed 只创建一个默认租户下的“超级管理员”角色，并为它绑定全部权限。这个角色同时承担：

- 平台超管 bootstrap
- RBAC 授权证明
- 租户开通后的管理入口

首版需要把平台治理角色与租户内置角色分开，并为新开通租户自动创建基础角色。角色不能在后续 seed 中继续依赖可变名称识别，否则重命名、新增同名角色或误改权限都可能扩大授权范围。

## 目标

- 首版内置四个系统角色：
  - `platform_super_admin`：平台超级管理员
  - `platform_auditor`：平台审计员
  - `tenant_admin`：租户管理员
  - `tenant_member`：租户成员
- 系统角色使用稳定编码识别，并声明平台或租户范围。
- 新租户开通时自动创建 `tenant_admin` 与 `tenant_member`，owner 默认绑定 `tenant_admin`。
- 平台管理员可读取全部租户角色；租户管理员可读取当前租户的启用角色选项，用于用户角色分配。
- 租户身份不能读取、分配或管理任何 `scope=platform` 的角色及其用户。
- 系统角色禁止通过业务 API 改名、停用或替换权限。
- 至少保留一个有效平台超级管理员。
- 开发阶段不提供旧数据兼容；数据库按新系统角色模型重新初始化。

## 非目标

- 首版不开放租户管理员自建、改名或删除角色。
- 不处理旧角色 adoption、存量租户角色回填或历史权限迁移。
- 不实现角色继承、临时授权、审批流或权限转授上限。
- 不允许配置自定义 provisioning role code；首版只有平台内置角色拥有稳定 code。
- 不定义业务角色，例如运营、财务或客服；这些应由业务能力落地后再创建。
- 不修改前端代码；只提供并锁定后端契约。

## 角色矩阵

| Code | Scope | 名称 | 默认权限 |
|---|---|---|---|
| `platform_super_admin` | `platform` | 平台超级管理员 | 当前启用能力声明的全部权限 |
| `platform_auditor` | `platform` | 平台审计员 | 租户、用户、角色、权限目录与审计日志只读权限 |
| `tenant_admin` | `tenant` | 租户管理员 | 当前租户用户管理、用户角色分配、角色选项读取、审计读取 |
| `tenant_member` | `tenant` | 租户成员 | 无管理权限 |

权限集合按 `(resource, action)` 声明。对应能力未编入当前 profile 时，种子跳过该权限，不阻塞启动。

### 平台审计员

默认权限：

```text
GET /api/v1/tenants
GET /api/v1/users
GET /api/v1/users/*
GET /api/v1/roles
GET /api/v1/roles/*
GET /api/v1/roles/*/permissions
GET /api/v1/permissions
GET /api/v1/audits
GET /api/v1/audits/*
GET /api/v1/audits/export
```

### 租户管理员

默认权限：

```text
GET    /api/v1/users
POST   /api/v1/users
GET    /api/v1/users/*
PUT    /api/v1/users/*
DELETE /api/v1/users/*
POST   /api/v1/users/*/roles
GET    /api/v1/roles/options
GET    /api/v1/audits
GET    /api/v1/audits/*
```

租户管理员不具备 `GET /api/v1/tenants`，因此不会被判定为平台范围。

### 租户成员

首版不授予管理端权限。该角色作为新成员的稳定默认归属；后续业务能力可以定义自己的成员权限。

## 数据模型

`roles` 增加三列：

```text
code       VARCHAR(64) NOT NULL DEFAULT ''
scope      VARCHAR(16) NOT NULL DEFAULT 'tenant'
is_system  BOOLEAN/TINYINT NOT NULL DEFAULT false/0
```

约束与语义：

- 自定义角色的 `code` 为空。
- 系统角色的 `code` 在租户内唯一；接口不提供写入 `code`。
- `scope` 只能是 `platform` 或 `tenant`。
- `tenant_admin`、`tenant_member` 在每个非默认租户内各一份。
- `platform_super_admin`、`platform_auditor` 固定归属默认租户，以 `scope=platform` 表达平台语义。
- 不引入 `tenant_id=0` 的平台角色，避免破坏 `user_roles` 的租户归属规则。

数据库增加 `code` 与 `scope` 的普通索引，不增加 `(tenant_id, code)` 唯一约束，因为自定义角色统一为空编码。系统角色的唯一性由 seed 和开通事务保证。

Casbin subject 使用 `role:<id>`，不使用角色名称。角色名称只用于展示；这样做避免不同租户的同名角色在同一个 Casbin enforcer 中合并策略。

## 迁移

access 能力新增下一号迁移，同时提供 MySQL 与 PostgreSQL：

- 增加 `code`、`scope`、`is_system`
- 增加 `idx_roles_code`
- 增加 `idx_roles_scope`
- Down 按相反顺序删除

结构性 seed 的顺序：

1. 先按 `tenant_id=default AND code=platform_super_admin` 查找。
2. 不存在时创建稳定角色。
3. 对 `platform_super_admin` 按当前权限集合精确替换权限。
4. 确保默认 `admin` 用户与 `platform_super_admin` 的 `user_roles` 关联存在。
5. 创建并精确收敛 `platform_auditor`。

seed 不按名称给自定义角色授予全部权限。

系统角色权限采用目标集合整体替换而不是只增不减，保证重复 seed 能移除误加或过期的权限。

## 租户开通

租户开通事务始终创建两个内置角色：

- `tenant_admin`
- `tenant_member`

`tenant.provisioning.roles` 只用于追加自定义角色，不再要求手写基础角色。自定义角色始终 `code=''`，不支持配置自定义 code。

`owner_role` 继续兼容现有配置：

- 先匹配内置 `tenant_admin` code
- 再按自定义角色 name 匹配
- 留空时使用 `tenant_admin`
- 不允许指向 `platform_super_admin`、`platform_auditor` 或 `tenant_member`
- 如果指定了不存在的租户角色，配置校验失败

自定义角色模板名称不能与内置名称冲突。当前默认配置中的空 roles 在 provisioning 开启时应合法。

租户 scope 角色不能持有平台专属权限（租户、套餐、权限目录、角色管理、平台 admin 等）；provisioning 配置和角色授权接口都必须在写入前拒绝。

开通事务仍保持全有或全无：角色、权限、用户、租户任一写入失败时整体回滚。

所有租户创建路径都必须创建内置租户角色：

- `POST /api/v1/tenants`
- 开通式注册 provisioner

默认租户是平台租户，只承载两个平台系统角色，不创建 `tenant_admin`/`tenant_member`。非平台身份不得管理默认租户用户；这样平台管理账号不会与租户管理员处于同一可管理范围。

## API 契约

### RoleResponse

增加：

```json
{
  "code": "tenant_admin",
  "scope": "tenant",
  "is_system": true
}
```

自定义角色返回空 code、`scope=tenant`、`is_system=false`。

### 角色选项

新增：

```text
GET /api/v1/roles/options
GET /api/v1/roles/options?tenant_id=<id>
```

用途：租户管理员给用户分配角色；平台管理员为指定租户加载可分配角色。

返回：

```json
{
  "code": 0,
  "message": "ok",
  "data": [
    {
      "id": "字符串",
      "code": "tenant_admin",
      "name": "租户管理员",
      "scope": "tenant",
      "is_system": true
    }
  ]
}
```

规则：

- 只返回启用且未删除的角色。
- 租户身份只能读取当前 JWT 租户。
- 租户身份返回结果必须排除 `scope=platform` 的角色。
- 平台身份可显式指定 tenant_id；省略时使用当前默认租户。
- 排序固定为 `is_system desc, id asc`，保证输出稳定。
- 增加权限点 `GET /api/v1/roles/options`。

### 系统角色保护

`PUT /roles/{id}`、`DELETE /roles/{id}`、`POST /roles/{id}/permissions` 遇到 `is_system=true` 返回 `403`。

停止角色使用 `PUT /roles/{id}` 并传 `status=0`，可通过 `status=1` 恢复。`DELETE /roles/{id}` 是永久删除：只允许未分配给用户的自定义角色，事务内删除 `role_permissions` 和角色行；存在 `user_roles` 引用时返回 `409`，避免产生悬空授权。

角色列表、详情、选项均返回 code/scope/is_system，前端无需再按名称猜测系统角色。

租户身份即使伪造角色 ID 或用户 ID：

- 不能通过用户创建或角色替换授予平台角色
- 不能识别或管理默认租户用户，平台角色用户只能由平台身份管理

## 最后平台超管保护

以下操作在目标用户是最后一个有效 `platform_super_admin` 时返回 `409`：

- 替换该用户全部角色且新集合不包含 `platform_super_admin`
- 停用该用户
- 删除该用户

“有效”表示：

- 用户 `status=1`
- 用户未软删除
- 角色 `status=1`
- 角色未软删除
- `user_roles` 关联存在

并发保护通过事务内锁定 `platform_super_admin` 角色行实现，避免两个请求同时移除最后两个超管。

实现必须先执行 `SELECT ... FOR UPDATE` 锁定默认租户的 platform superadmin 角色，再进行任何 active/other 计数，避免 MySQL REPEATABLE READ 下旧快照绕过保护。

`UserRoleAssigner` 端口增加停用前检查能力，由 user 能力消费；access 能力继续拥有 `user_roles` 和平台角色事实。

seed 完成角色和 admin 用户创建后必须验证至少存在一个有效平台超管；否则返回错误并阻止初始化成功。不得静默重启已停用或已软删除的管理员账号。

## 审计读取范围

- 平台身份读取全部租户审计。
- 租户管理员只读取当前租户审计。
- 审计详情必须执行与列表相同的租户可见性校验；跨租户详情按不存在处理。
- 审计导出沿用相同范围。

## 配置

`tenant.provisioning.roles`：

- 允许为空
- 自定义角色不持久化 code
- 自定义 name 不能与四个内置名称重复

`tenant.provisioning.owner_role`：

- 空值默认 `tenant_admin`
- 可匹配 `tenant_admin` 或自定义租户角色名称
- 不能匹配平台角色或 `tenant_member`

## 错误码

- `400`：tenant_id、owner_role、自定义 role name 或权限声明无效
- `403`：修改系统角色、租户越权读取角色选项
- `404`：角色或租户不存在
- `409`：移除最后一个有效平台超管

## 兼容性

- 开发阶段不提供旧数据库兼容或存量角色 adoption。
- 自定义角色始终 `code=''`、`scope='tenant'`、`is_system=false`。
- 新安装无需在 YAML 中重复声明四个系统角色。

## 验收

- 全新数据库 seed 后在默认租户存在两个平台系统角色；任何新租户创建后存在两个租户系统角色。
- `POST /tenants` 和开通式注册都会在各自事务内创建两个内置租户角色。
- 新租户开通后 owner 自动获得 `tenant_admin`，同时存在 `tenant_member`。
- 租户管理员能读取本租户角色选项，不能跨租户。
- 租户身份不能读取或授予平台角色，也不能管理平台角色用户。
- 平台审计员可跨租户读取审计；租户管理员只能读取本租户审计和详情。
- 系统角色不能被业务 API 修改。
- 最后一个有效平台超管不能被移除、停用或删除。
- seed 无法确认有效平台超管时 fail closed。
- OpenAPI、路由 golden、生成器 golden、编译面报告和权限矩阵同步。
