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
| `http` | host/port/mode、请求超时与请求体限制、可信代理、跨域、TLS |
| `management` | 运维面 host/port、pprof、探活超时（livez/readyz/metrics） |
| `db` | host/port/user/password/name + `read_hosts`/`read_ports`（读写分离） |
| `redis` | 地址、密码（缓存/会话/限流/锁共用） |
| `log` | level/format/output + lumberjack 滚动（max_size/max_backups/max_age/compress） |
| `auth` | JWT 密钥/有效期、APIKey 中间件、限流参数 |
| `server` | timeout_sec、rate_limit_rate/rate_limit_burst（全局限流） |
| `id` | `worker_id`（多实例唯一，雪花 ID） |
| `cache` | prefix（缓存 key 前缀，多实例隔离） |
| `audit` | queue_size/batch_size/flush_interval_ms（批量落库） |
| `storage` | type（local/s3/oss/minio）+ 对象存储连接参数 |
| `security` | 安全响应头 + csrf_secret |
| `queue` | type + 各中间件连接 |
| `outbox` | publisher 切换 |
| `scheduler` | store 切换 |
| `oauth.providers` | `google`/`github`/`wechat` 各含 client_id/secret/redirect_url/enabled |
| `captcha` | enabled 开关 |
| `email` | SMTP host/port/username/from（enabled=false 回退日志渠道） |
| `sms` | provider（aliyun）/api_key/api_secret/sign_name |
| `otel` | enabled + OTLP 端点 |
| `http_client` | 统一出站 HTTP：timeout_sec/max_retries/retry_interval_ms/max_failures/reset_timeout_ms（仅网络错误与 5xx 重试，指数退避，熔断 + traceparent 注入） |
