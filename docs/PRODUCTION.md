# 生产上线

本页是 `basic-api` 和 `auth-api` 的生产上线入口。模板级检查清单分别维护；上线时先完成对应清单，再按完整部署走查验证。

## 模板检查清单

- [`basic-api` 生产检查清单](production/basic-api.md)：无数据库/Redis 依赖的 HTTP API、BFF、Webhook 和内部服务。
- [`auth-api` 生产检查清单](production/auth-api.md)：单租户、MySQL、Redis、JWT 和 RBAC。

## 共同原则

- **Secret**：数据库密码、`JWT_SECRET` 和管理员初始密码通过 Secret 文件或环境变量注入，不写入 Git。
- **TLS**：生成模板不内置 TLS；必须在反向代理/负载均衡器终止 TLS，管理端口只对内网开放。
- **CORS / Header**：`http.allowed_origins` 必须使用明确来源，`*` 会在启动时被拒绝；安全 Header 已内置，由代理保留。
- **代理信任**：`http.trusted_proxies` 只填写实际代理的 IP/CIDR；直连部署留空。
- **限流**：生成模板不内置全局限流；必须在反向代理、网关或项目代码中配置。
- **迁移**：发布前备份数据库，由独立迁移任务执行；`jimu upgrade` 不自动迁移 schema。
- **可观测性**：模板提供 `/livez`、`/readyz` 和结构化日志；指标、trace 与告警由平台或项目自行接入。
- **停机**：部署平台的终止宽限期不得短于 `http.shutdown_timeout_sec`。

配置键见[配置](CONFIGURATION.md)，安全原则见[安全策略](SECURITY.md)，运维和备份见[运维](OPERATIONS.md)。

## 完整部署走查

以下是从生成项目到部署的最小完整路径。生产环境应替换为实际的镜像仓库、反向代理、Secret 管理和数据库/Redis 服务。

### basic-api

1. 生成并检查项目：

```bash
jimu new basic-api-demo --template=basic-api
cd basic-api-demo
jimu check .
jimu upgrade . --check
```

2. 修改 `configs/app.yaml` 的生产值：`http.mode=release`、明确的 `http.allowed_origins`、实际代理的 `http.trusted_proxies`、内网 `management.host` 和 `log.format=json`。

3. 构建镜像：

```bash
go test ./...
go build ./...
docker build -t registry.example.com/basic-api-demo:<version> .
```

4. 在反向代理后运行，只暴露业务端口；管理端口仅给探针：

```bash
docker run --rm \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:9090:9090 \
  registry.example.com/basic-api-demo:<version>
```

5. 验证：

```bash
curl -fsS http://127.0.0.1:8080/api/v1/ping
curl -fsS http://127.0.0.1:9090/livez
curl -fsS http://127.0.0.1:9090/readyz
curl -fsSI https://api.example.com/livez
```

6. 在反向代理/网关配置 TLS、限流、访问日志和超时，并确认客户端只能通过 HTTPS 访问。

### auth-api

1. 生成并先做本地空库验证：

```bash
jimu new auth-api-demo --template=auth-api
cd auth-api-demo
make up
make verify
make down
```

2. 准备独立 MySQL 8.4 和 Redis 7；为 `DB_PASSWORD_FILE`、`REDIS_PASSWORD_FILE`、`JWT_SECRET_FILE` 创建 Secret，并确认文件权限只允许运行用户读取。

3. 构建镜像并先执行迁移：

```bash
docker build -t registry.example.com/auth-api-demo:<version> .
docker run --rm \
  --env-file /run/config/auth-api.env \
  -v /run/secrets:/run/secrets:ro \
  registry.example.com/auth-api-demo:<version> \
  ./jimu migrate up
```

4. 首次部署使用一次性 `ADMIN_PASSWORD_FILE` 执行 seed，随后从应用运行环境移除该文件：

```bash
docker run --rm \
  --env-file /run/config/auth-api.env \
  -e ADMIN_PASSWORD_FILE=/run/secrets/admin_password \
  -v /run/secrets:/run/secrets:ro \
  registry.example.com/auth-api-demo:<version> \
  ./jimu seed
```

5. 启动应用；业务端口和 `9090` 管理端口都只对代理/探针开放：

```bash
docker run -d --name auth-api-demo \
  --env-file /run/config/auth-api.env \
  -v /run/secrets:/run/secrets:ro \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:9090:9090 \
  registry.example.com/auth-api-demo:<version>
```

6. 验证：

```bash
curl -fsS http://127.0.0.1:9090/readyz
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/api/v1/users
curl -fsSI https://admin-api.example.com/livez
```

7. 在反向代理/网关配置 TLS、登录/刷新限流、访问日志和超时；确认 `401`、`403`、`429` 行为符合预期。

8. 按[运维](OPERATIONS.md#备份与恢复)验证 MySQL 备份恢复和 Redis 持久化；发布新版本前先运行 `jimu upgrade . --check`，再按[升级](UPGRADING.md)执行。
