# 能力目录

Jimu 交付的是**独立 Go 项目**，不是运行时插件系统。生成项目只包含所选模板或 experimental 组合真正需要的代码、迁移、配置段和资产；未选择的代码不会进入项目。

## 官方模板

| 模板 | 适用场景 | 主要能力 |
|---|---|---|
| `basic-api` | 零外部依赖的 HTTP API、BFF、轻量服务 | HTTP 路由、配置加载、结构化日志、统一错误、健康检查、优雅停机、Docker 启动 |
| `auth-api` | 单租户后台 API、需要登录和权限的业务服务 | 用户、JWT access/refresh、RBAC、MySQL、Redis、迁移、seed、管理员初始化、Compose、`make verify` |

模板的预算、兼容窗口和发布阻断规则见[模板契约 v0.1](templates/v1/README.md)；生成流程与 manifest 规则见[形态与项目生成](design/profiles-and-project-generation.md)。

## experimental 组合

`jimu new --with=<cap>[:<driver>]` 和 `jimu capability add <name>` 可以组合框架已有能力。该模式标记为 `support: experimental`，不承诺任意组合的兼容性。

| 领域 | 能力 |
|---|---|
| 身份与租户 | `user`、`access`、`auth`、`tenant`、`apikey`、`oauth`、`mfa`、`passkey` |
| 数据与异步 | `queue`、`outbox`、`scheduler`、`dataops`、`search` |
| 安全与通信 | `encryption`、`captcha`、`uploadsec`、`notification`、`ws`、`grpc` |
| 运维与观测 | `audit`、`console`、`apidocs`、`storage` |

驱动和资产会随能力选择、语言运行时服务和 profile 一起裁剪。具体实现以能力 `Descriptor`、`jimu check` 输出和生成项目 `.jimu/manifest.json` 为准。

## 检查与升级

- `jimu check <dir>`：只读检查 manifest、组合依赖、驱动、迁移、配置段和生成计划。
- `jimu upgrade <dir> --dry-run`：输出 `untouched` / `modified` / `conflict` / `new` / `deleted` 分类。
- `jimu upgrade <dir> --check`：有冲突时退出非 0，适合用户 CI。
- `jimu upgrade <dir>`：先备份、再逐文件安装，失败按原字节回滚。

详细的升级策略和 CI 示例见[升级](UPGRADING.md)。

## API 文档

框架自身启动后可访问 Swagger UI：

```text
http://localhost:8080/swagger/index.html
```

生成项目只包含其模板需要的 API；`auth-api` 的端点、迁移和验证方式见生成项目自己的 `README.md` 和 `QUICKSTART.md`。
