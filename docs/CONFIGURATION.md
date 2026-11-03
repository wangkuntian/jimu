# 配置

框架使用 Viper 加载 YAML，并允许环境变量覆盖。生成项目拥有自己的配置和依赖，不继承框架未选择的段。

## 加载顺序

优先级从高到低：

```text
环境变量 > configs/app.<APP_ENV>.yaml > configs/app.yaml
```

`APP_ENV` 为空或 `dev` 时使用 `configs/app.yaml`。敏感值支持 `_FILE` 后缀从文件读取，适用于 Docker Secrets：

```bash
DB_PASSWORD_FILE=/run/secrets/db_password
JWT_SECRET_FILE=/run/secrets/jwt_secret
ENCRYPTION_KEY_FILE=/run/secrets/encryption_key
```

正常启动不读取项目根 `.env`；Compose 使用 `--env-file /dev/null`。

## 核心配置

事实源和完整默认值见 [`configs/app.yaml`](../configs/app.yaml)、[`configs/app.prod.yaml`](../configs/app.prod.yaml) 以及 [`internal/config`](../internal/config)。配置段归属和加载规则见[配置与数据生命周期](design/configuration-and-data-lifecycle.md)。

| 段 | 代表键 | 说明 |
|---|---|---|
| HTTP | `http.host`、`http.port`、`http.mode`、`http.max_body_bytes` | 监听地址、Gin 模式、请求体上限；TLS 在 `http.tls` |
| Management | `management.host`、`management.port` | `/livez`、`/readyz`、`/metrics`、`/capabilities` |
| Log | `log.level`、`log.format`、`log.output` | `console` / `json`，生产建议 `json` + `info` |
| Database | `db.driver`、`db.host`、`db.port`、`db.user`、`db.password` | MySQL / MariaDB / PostgreSQL；只读副本在 `db.read_hosts` |
| Redis | `redis.mode`、`redis.addr`、`redis.password`、`redis.db` | `single` / `sentinel` / `cluster` |
| Auth | `auth.jwt_secret`、`auth.access_expire_min`、`auth.refresh_expire_day` | JWT 密钥和会话有效期；生产必须通过 Secret 注入 |
| Tenant | `tenant.provisioning.enabled`、`tenant.provisioning.roles` | 可选开通式注册和租户角色模板 |
| Security | `security.encryption_key`、`security.csrf_secret`、`security.ip_allowlist` | 字段加密、CSRF、IP 白名单和安全 Headers |
| Queue | `queue.type`、`queue.retry.*` | `redis` / `kafka` / `rabbitmq` 队列 |
| Storage | `storage.type` | `local` / `s3` / `oss` / `minio`；驱动必须编入当前 profile |
| Observability | `otel.enabled`、`otel.endpoint`、`otel.metrics_enabled` | OpenTelemetry OTLP tracing / metrics / logs |
| Capabilities | `capabilities.enabled` | 留空表示启用当前 profile 的全部受门控能力 |

## 模板差异

- `basic-api`：只需要 HTTP、Management、Log；不读取数据库或 Redis 配置，启动不依赖外部服务。
- `auth-api`：额外需要 `db`、`redis`、`auth`；`make up` 会生成本地 `secrets/admin_password.txt` 和 `secrets/jwt_secret.txt`，并完成迁移与 seed。

生成项目的配置键和默认值以其仓库内 `configs/app.yaml` 为准；不要从框架全量配置复制未选择的段。

## 安全配置

生产上线前至少检查：

- `auth.jwt_secret` 通过 `JWT_SECRET` 或 Secret 文件注入，不使用默认值。
- 设置明确的 `http.allowed_origins`，不要保留 wildcard。
- 启用 `http.tls` 或由反向代理终止 TLS，并配置 `http.trusted_proxies`。
- 根据流量设置 `server.rate_limit_rate`、`server.rate_limit_burst` 和 `server.max_concurrency`。
- 配置 `security.ip_allowlist` / `security.admin_ip_allowlist`（如适用）。
- 可选字段加密必须使用恰好 32 字节的 `security.encryption_key`。

更完整的生产入口见[生产检查](PRODUCTION.md)和[安全策略](SECURITY.md)。
