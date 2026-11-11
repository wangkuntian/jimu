# auth 生产上线检查清单

适用对象：由 `jimu new <dir> --profile=auth` 生成、需要用户、JWT、RBAC、MySQL 和 Redis 的单租户后台 API。

`auth` 预设携带 `user`/`access`/`auth` 及其 `tenant` 编译闭包。生产环境必须使用独立 MySQL 和 Redis；预设不生成 Compose，本地与生产都通过 `DB_*`/`REDIS_*` 环境变量指定连接。

需要单实例或无 Redis 运行时，可用 `jimu new <dir> --with=auth:memory` 生成内存驱动版本。
内存驱动只在本进程内保存会话、限流和重置码，重启即丢失，多实例之间不共享；生产多实例仍必须使用 Redis 驱动。

## 必须配置

| 项目 | 要求 | 对应位置 / 验证 |
|---|---|---|
| 环境与模式 | 生产设置 `APP_ENV=prod`，并设置 `http.mode: release` | 环境变量 `APP_ENV`、`http.mode` |
| JWT Secret | 必须注入至少 32 字节随机值；空值会启动失败 | `JWT_SECRET` 或 `JWT_SECRET_FILE` |
| 数据库 | 使用独立 MySQL 8.4；密码通过 Secret 文件注入 | `db.host`、`db.port`、`db.user`、`db.database`、`DB_PASSWORD` / `DB_PASSWORD_FILE` |
| 数据库连接池 | 按实例数和数据库上限设置，避免连接耗尽 | `db.max_open`、`db.max_idle` |
| Redis | `auth` 预设用 Redis 保存 refresh session；必须使用持久化/高可用实例 | `redis.addr`、`redis.db`、`REDIS_PASSWORD` / `REDIS_PASSWORD_FILE` |
| 管理员初始化 | 首次迁移后执行 seed；`ADMIN_PASSWORD_FILE` 只用于 seed，不要在应用运行时长期暴露 | 生成项目 `./jimu seed`、`ADMIN_PASSWORD_FILE` |
| 迁移 | 上线前先备份，再由独立迁移任务执行；`jimu upgrade` 不会自动改数据库 schema | 生成项目 `./jimu migrate up`；见[升级](../UPGRADING.md#数据库-schema) |
| TLS | 模板不内置 TLS；必须在反向代理/负载均衡器终止 TLS | 反向代理配置；见[完整部署走查](../PRODUCTION.md#完整部署走查) |
| CORS | 只允许明确的前端来源；`*` 会在启动时被拒绝 | `http.allowed_origins` |
| 可信代理 | 只信任实际反向代理的 IP/CIDR；直连部署设为空列表 | `http.trusted_proxies` |
| 管理端口 | `/livez`、`/readyz` 仅对内网/探针开放 | `management.host`、`management.port` |
| 安全 Header | 模板已内置安全 Header，并由反向代理保留/补充 | `internal/platform/http/middleware/` |
| 请求体上限 | 登录、用户和角色接口都不应无界读取 | `http.max_body_bytes` |
| 限流 | 模板不内置限流；必须在反向代理/网关至少限制登录、刷新和公开接口 | 反向代理/网关；见[运维](../OPERATIONS.md) |
| 日志 | JSON 输出到 stdout；不得记录密码、token、验证码或未脱敏 PII | `log.format`、`log.level`、`log.output` |
| 优雅停机 | 终止宽限期不得短于 `http.shutdown_timeout_sec` | 部署平台 `terminationGracePeriodSeconds` / `docker stop` |
| 备份与恢复 | 备份 MySQL 和 Redis；上线前至少验证一次 MySQL 恢复 | 项目备份流程；见[运维](../OPERATIONS.md#备份与恢复) |

建议的生产配置片段：

```yaml
http:
  mode: release
  allowed_origins: ["https://admin.example.com"]
  trusted_proxies: ["10.0.0.0/8"]
  shutdown_timeout_sec: 20

management:
  host: "127.0.0.1"

log:
  level: "info"
  format: "json"
  output: "stdout"

auth:
  access_expire_min: 30
  refresh_expire_day: 7
```

数据库密码、JWT Secret 和管理员初始密码必须来自 Secret 注入，不要写入 YAML 或 Git。

## 可选配置

| 项目 | 说明 | 对应位置 |
|---|---|---|
| pprof | 默认关闭；不要在生产管理端口开启 | `management.enable_pprof` |
| Redis DB | 与其他 Redis 使用方隔离时设置独立 DB | `redis.db` |
| Token 生命周期 | 按组织安全策略缩短 access/refresh 有效期 | `auth.access_expire_min`、`auth.refresh_expire_day` |
| JWT Issuer | 多环境需要区分 token 来源时设置唯一值 | `auth.issuer` |
| 日志轮转 | 默认写 stdout；文件输出时才使用轮转参数 | `log.max_size`、`log.max_backups`、`log.max_age`、`log.compress` |
| 可观测性 | 模板提供结构化日志和健康检查；指标/trace 需要由反向代理、平台或项目代码补充 | `docs/OPERATIONS.md` |
| 部署资源 | 根据登录/刷新流量设置 CPU/内存限制、副本数和 PDB | 部署平台配置 |

当前生成模板不内置登录失败锁定、全局限流、指标端点和 OpenTelemetry；不要把框架自身的能力误写成 `auth` 预设的默认承诺。需要时在项目代码或网关注入。

## 上线前验证

- [ ] `jimu check .` 返回合法，manifest 与组合一致。
- [ ] `jimu upgrade . --check` 返回 0；有冲突时按[升级](../UPGRADING.md)处理。
- [ ] `go test ./...`、`go build ./...` 通过。
- [ ] `docker build -t auth:<version> .` 通过，容器以非 root 用户运行。
- [ ] 在全新 MySQL/Redis 上执行 `./jimu migrate up` 和 `./jimu seed`，重复执行保持幂等。
- [ ] `GET /livez`、`GET /readyz` 返回预期状态；MySQL/Redis 不可用时启动快速失败。
- [ ] 不带 token 访问受保护路由返回 `401`；无权限 token 返回 `403`。
- [ ] 登录、刷新、登出、`/auth/me` 和用户/角色/权限主要路径通过。
- [ ] 短密码创建/更新被拒绝；管理员初始密码只在首次 seed 使用。
- [ ] CORS 只对 `http.allowed_origins` 中的来源返回 `Access-Control-Allow-Origin`。
- [ ] 响应包含安全 Header：

```bash
curl -si http://127.0.0.1:8080/api/v1/auth/me \
  | grep -iE '^(x-content-type-options|x-frame-options|referrer-policy|permissions-policy|content-security-policy|strict-transport-security):'
```

- [ ] 限流策略覆盖登录/刷新和公开路由，超限返回预期状态码。
- [ ] MySQL 备份可恢复；Redis 持久化/高可用满足 refresh session 的业务要求。
- [ ] `SIGTERM` 后服务在 `http.shutdown_timeout_sec` 内退出，迁移任务与旧实例的发布顺序明确。
- [ ] 日志为 JSON、不包含密码、令牌、验证码或未脱敏 PII，告警接收人可触达。

完整的本地验证和生产发布顺序见[生产上线走查](../PRODUCTION.md#完整部署走查)。
