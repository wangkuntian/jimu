# CLI 契约

来源：`cmd/cli/main.go`（Cobra）

## 命令表

| 命令 | 用途 |
|------|------|
| `jimu migrate up` | 执行迁移（Goose） |
| `jimu migrate down` | 回滚一个迁移 |
| `jimu migrate status` | 迁移状态 |
| `jimu migrate redo` | 重做最近迁移 |
| `jimu seed` | 插入管理员与基础权限（含 Casbin 策略同步） |
| `jimu module create [name]` | 脚手架生成完整模块骨架 |
| `jimu config check` | 校验配置文件合法性 |
| `jimu version` | 版本信息 |

## 迁移命名

`{seq}_create_{table}s.sql`（如 `001_create_users.sql`），字段与表 MUST 带中文 COMMENT。

## 种子数据

`seed` 一键写入：管理员账号 + 基础权限 + 角色绑定 + Casbin 策略（`SeedCasbinPolicies`），保证新环境开箱可用。
