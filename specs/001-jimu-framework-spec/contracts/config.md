# 配置契约

来源：`internal/config/config.go` + `configs/{app.yaml, app.prod.yaml}`

## 加载优先级

`环境变量 > app.{APP_ENV}.yaml > app.yaml`

## 枚举约束（非法值启动报错）

| 键 | 合法值 |
|----|--------|
| `http.mode` | `debug` / `release` / `test` |
| `log.level` | `debug` / `info` / `warn` / `error` |
| `log.format` | `json` / `console` |
| `queue.type` | `redis` / `kafka` / `rabbitmq` |
| `outbox.publisher` | `event_bus` / `mq` |
| `scheduler.store` | `memory` / `mysql` |

新增枚举值 MUST 在 `internal/config/config.go` 定义常量并加入 `valid*` 校验。

## 敏感值注入（Docker Secrets）

`_FILE` 后缀从文件读取：

```bash
DB_PASSWORD_FILE=/run/secrets/db_password
JWT_SECRET_FILE=/run/secrets/jwt_secret
```

实现：`getEnvOrFile`（`config.go:343-390`），支持 `DB_PASSWORD`/`JWT_SECRET`/`REDIS_PASSWORD` 等。

## 关键配置组

| 组 | 内容 |
|----|------|
| `db` | host/port/user/password/name + `read_hosts`/`read_ports`（读写分离） |
| `redis` | 地址、密码（缓存/会话/限流/锁共用） |
| `auth` | JWT 密钥/有效期、APIKey 中间件、限流参数 |
| `oauth.providers` | `google`/`github`/`wechat` 各含 client_id/secret/redirect_url/enabled |
| `captcha` | enabled 开关 |
| `queue` | type + 各中间件连接 |
| `outbox` | publisher 切换 |
| `scheduler` | store 切换 |
| `id` | `worker_id`（多实例唯一，雪花 ID） |
| `otel` | enabled + OTLP 端点 |
| `security` | csrf_secret、可信代理等 |
