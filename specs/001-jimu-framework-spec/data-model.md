# Data Model: Jimu 框架持久化实体

**Created**: 2026-08-12

来源：`migrations/`（Goose，13 张表）。全部主键 `id BIGINT UNSIGNED` 由应用生成雪花 ID（非 AUTO_INCREMENT）；基础表含 `created_at`/`updated_at`/`deleted_at`（软删除）。

## 实体关系

```text
User ──< UserRole >── Role ──< RolePermission >── Permission
User ──< AuditLog
User ──< OAuthBinding (provider, subject) 唯一
APIKey  ── created_by ── User
ImportJob ── created_by ── User
OutboxEvent / ScheduledJob：独立，不关联业务实体
```

## 1. User（用户）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | 雪花 ID |
| username | VARCHAR(64) | UNIQUE | 登录名 |
| password | VARCHAR(255) | NOT NULL | bcrypt 哈希 |
| status | TINYINT | 默认 1 | 1=启用 0=禁用 |
| created_at / updated_at / deleted_at | TIMESTAMP | — | 软删除 |

## 2. Role（角色）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | |
| name | VARCHAR(64) | UNIQUE | 角色名 |
| description | VARCHAR(255) | 默认 '' | |
| status | TINYINT | 默认 1 | |

关联表：`user_roles`（用户-角色多对多）、`role_permissions`（角色-权限多对多）。

## 3. Permission（权限）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | |
| name | VARCHAR(128) | NOT NULL | 权限名 |
| resource | VARCHAR(64) | 联合唯一 | 资源路径（如 /api/v1/users） |
| action | VARCHAR(32) | 联合唯一 | 操作（GET/POST/PUT/DELETE） |
| resource + action | — | UNIQUE KEY | 唯一权限点 |

## 4. APIKey（API 密钥）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | |
| name | VARCHAR(64) | NOT NULL | Key 名 |
| key_prefix | VARCHAR(16) | NOT NULL | 识别前缀 |
| key_hash | VARCHAR(64) | INDEX | SHA-256 哈希（明文不落库） |
| scopes | TEXT | — | 权限范围 JSON |
| enabled | TINYINT(1) | 默认 1 | |
| expires_at / last_used | TIMESTAMP | NULL | 过期/最近使用 |
| use_count | BIGINT | 默认 0 | |
| created_by | BIGINT UNSIGNED | FK→User | 创建者 |

## 5. AuditLog（审计日志）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | |
| user_id / username | — | — | 操作者（匿名=0/''） |
| action / resource | VARCHAR | INDEX | 操作类型与资源 |
| detail | TEXT | — | JSON 详情 |
| ip / method / path / status | — | — | 请求上下文 |
| created_at | TIMESTAMP | INDEX | 操作时间（只追加，无更新/删除） |

## 6. OAuthBinding（第三方绑定）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | |
| user_id | BIGINT UNSIGNED | INDEX | 绑定到本地用户 |
| provider | VARCHAR(32) | 联合唯一 | google/github/wechat |
| subject | VARCHAR(128) | 联合唯一 | 提供商内唯一 ID |

`(provider, subject)` 全局唯一，防止重复绑定。

## 7. OutboxEvent（Outbox 事件）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | |
| aggregate_id | VARCHAR(128) | INDEX | 聚合根 ID |
| event_type | VARCHAR(128) | INDEX | 事件类型 |
| payload | JSON | NOT NULL | 事件数据 |
| metadata | JSON | NULL | |
| published_at | TIMESTAMP | INDEX | NULL=未发布（待投递） |
| retry_count | INT | 默认 0 | 重试次数 |

## 8. ScheduledJob（定时任务定义）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | VARCHAR(64) | PK | 任务 ID（业务键，非雪花） |
| name | VARCHAR(128) | NOT NULL | |
| cron | VARCHAR(64) | NOT NULL | cron 表达式 |
| enabled | TINYINT(1) | 默认 1 | |
| created_at / updated_at / deleted_at | DATETIME | — | |

## 9. ImportJob（导入任务）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | |
| type | VARCHAR(64) | NOT NULL | 导入类型（users 等） |
| filename | VARCHAR(255) | NOT NULL | |
| status | VARCHAR(16) | INDEX | pending/processing/completed/failed |
| total_rows / success_rows / error_rows | INT | 默认 0 | 进度 |
| errors | TEXT | — | JSON 错误详情 |
| created_by | BIGINT UNSIGNED | INDEX | |
| completed_at | TIMESTAMP | NULL | |

## 验证规则（映射 spec FR-030）

- username：唯一、长度 ≤64、用户名规则（`shared/validator`）
- password：bcrypt 哈希存储；强度校验规则（validator）
- 权限点：`(resource, action)` 全局唯一
- OAuth 绑定：`(provider, subject)` 全局唯一
