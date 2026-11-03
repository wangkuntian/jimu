# 运维

本文件覆盖本仓自带的本地栈、可观测性、备份和部署。生成项目优先使用它自己的 `README.md` / `QUICKSTART.md` 和 Makefile。

## 本地启动

```bash
make init
make up
```

`make init` 生成或校验 `secrets/`；`make up` 按需构建镜像、等待 MariaDB/Redis、执行迁移和 seed，再启动 `server`。管理员初始密码在 `secrets/admin_password.txt`，不要提交到版本库。

常用命令：

| 命令 | 作用 |
|---|---|
| `make status` | 查看核心栈状态 |
| `make logs` | 查看服务日志 |
| `make down` | 停止核心栈，保留数据卷 |
| `make restart` | 重启服务 |
| `make migrate-status` | 在运行中的 server 容器查看迁移状态 |
| `make adminer` | 启动 Adminer |

## 健康检查

- 管理端 `/livez`：进程存活。
- 管理端 `/readyz`：数据库/Redis 等已编入依赖的就绪检查。
- 业务端 `/api/v1/ping`：示例 HTTP 端点。

`basic-api` 不依赖外部服务；`auth-api` 的 `make verify` 会检查 readyz、登录、当前用户和受保护请求。

## 可观测性

```bash
make up-observability
```

OpenObserve 负责日志、指标、追踪和告警：

- UI/API：`http://127.0.0.1:5080`
- OTLP gRPC：`127.0.0.1:5081`
- 账号和密码在 `secrets/openobserve.env`

应用配置 `otel.enabled=true` 后，HTTP、GORM、Redis、队列和 Outbox 链路可贯通 trace；日志通过 `logger.WithContext(ctx)` 自动关联 `trace_id` / `span_id`。

Dashboard 和告警定义在 `deploy/openobserve/`，可按 `scripts/observability.sh` 和应用配置同步。生产环境应替换示例 webhook、账号和密码。

## 备份与恢复

```bash
make compose-db-backup
make compose-db-restore FILE=/backups/jimu_20250101_120000.sql.gz
```

脚本同时支持 MariaDB/MySQL 和 PostgreSQL，口令通过 `MYSQL_PWD` / `PGPASSWORD` 传递，不进入命令行参数。保留天数由 `RETENTION_DAYS` 控制，默认 7 天。恢复是覆盖式操作，非交互环境必须显式 `FORCE=1`。

## Docker 与 K8s

- Docker：`make init`、`make up`；已有数据卷时先轮换数据库密码，再 `make restart`。
- K8s：`deploy/k8s/` 提供 Deployment、Service、HPA、Ingress、PDB、NetworkPolicy 和迁移 initContainer。
- Helm：`deploy/helm/`；生产 Secret 必须在仓库外注入，不要把真实凭据提交到 Git。
- 多副本部署必须为每个实例设置唯一 `id.worker_id`，避免雪花 ID 冲突。

## 日志

使用结构化字段：`Debugw`、`Infow`、`Warnw`、`Errorw`；不要使用单参数 print 风格。字段名使用登记的 snake_case 词汇表，密码、token、验证码和未脱敏 PII 不得写入日志。CI 的 `make check-log-usage` 会检查调用形态。
