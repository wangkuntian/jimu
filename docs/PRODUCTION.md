# 生产检查入口

本页是 P0 的上线入口，不替代模板级检查表。**P1-02 的 `basic-api` / `auth-api` 三段式生产检查清单仍未完成**；这里只列出上线前必须确认的共同方向。

## 通用检查

- **Secret**：`JWT_SECRET`、数据库密码、`ENCRYPTION_KEY`、`AUDIT_HASH_SECRET` 等通过 Secret 文件或环境变量注入，不写入 Git。
- **TLS**：由反向代理终止 TLS，或启用 `http.tls`；不要把明文 HTTP 暴露到公网。
- **CORS / Headers**：明确 `http.allowed_origins`，启用安全 Headers，不要保留开发用 wildcard。
- **限流**：配置 `server.rate_limit_rate`、`server.rate_limit_burst`、`server.max_concurrency`，并根据流量压测调整。
- **迁移**：发布前备份数据库，按项目流程执行迁移；`jimu upgrade` 不自动迁移 schema。
- **备份与恢复**：验证最近一次备份可恢复，保留周期满足业务要求。
- **可观测性**：确认 `/livez`、`/readyz`、错误日志、trace 和 metrics 已接入，告警接收人可触达。
- **优雅停机**：确认部署平台发送 `SIGTERM` 后服务在 `shutdown_timeout_sec` 内完成退出。
- **部署**：核对镜像版本、资源限制、副本数、`id.worker_id` 唯一性和 Ingress/Service 配置。

## 模板入口

- `basic-api`：无数据库/Redis 依赖；重点检查 HTTP 安全、端口、日志和部署资源。
- `auth-api`：额外检查 MySQL/Redis、JWT secret、管理员密码、迁移、seed 和备份。

配置键见[配置](CONFIGURATION.md)，安全原则见[安全策略](SECURITY.md)，部署和观测见[运维](OPERATIONS.md)。模板级“必须配置 / 可选配置 / 上线前验证”清单仍由 P1-02 跟踪。
