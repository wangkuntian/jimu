# minimal 生产上线检查清单

适用对象：由 `jimu new <dir> --template=minimal` 生成、需要不依赖数据库/Redis/消息队列的 HTTP API、BFF、Webhook 或内部服务。

生成项目自己的配置文件是 `configs/app.yaml`。生产值应通过 `configs/app.<env>.yaml` 或环境变量覆盖；不要修改 `.jimu/manifest.json`。

## 必须配置

| 项目 | 要求 | 对应位置 / 验证 |
|---|---|---|
| HTTP 模式 | 生产设置为 `release`，不要保留 Gin debug 输出 | `http.mode` |
| 监听地址 | 公网流量只进入反向代理；应用端口由代理转发 | `http.host`、`http.port` |
| TLS | 模板不内置 TLS；必须在反向代理/负载均衡器终止 TLS，禁止把明文 HTTP 暴露到公网 | 反向代理配置；见[完整部署走查](../PRODUCTION.md#完整部署走查) |
| CORS | 只允许明确的前端来源；`*` 会在启动时被拒绝 | `http.allowed_origins` |
| 可信代理 | 只信任实际反向代理的 IP/CIDR；直连部署设为空列表 | `http.trusted_proxies` |
| 管理端口 | `/livez`、`/readyz` 仅对内网/探针开放，不要暴露到公网 | `management.host`、`management.port` |
| 安全 Header | 模板已内置 `X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy`、`Permissions-Policy`、CSP 和 HSTS | `internal/platform/http/middleware/`；上线前用 `curl -i` 验证 |
| 请求体上限 | 按业务最大值设置，避免无界读取 | `http.max_body_bytes` |
| 超时 | 保留读取、写入、空闲和优雅停机预算，并与代理超时对齐 | `http.read_header_timeout_sec`、`http.read_timeout_sec`、`http.write_timeout_sec`、`http.idle_timeout_sec`、`http.shutdown_timeout_sec` |
| 限流 | 模板不内置限流；必须在反向代理、网关或项目代码中配置，至少覆盖单 IP 和突发流量 | 反向代理/网关；见[运维](../OPERATIONS.md) |
| 日志 | 生产使用结构化 JSON，输出到 stdout，禁止把敏感数据写入日志 | `log.format`、`log.level`、`log.output` |
| 优雅停机 | 容器编排平台的终止宽限期不得短于 `http.shutdown_timeout_sec` | 部署平台 `terminationGracePeriodSeconds` / `docker stop` |
| Secret | `minimal` 不需要数据库或 JWT Secret；不要把凭据写进 `configs/app.yaml` | 项目 Secret 清单 |

建议的生产配置片段：

```yaml
http:
  mode: release
  allowed_origins: ["https://app.example.com"]
  trusted_proxies: ["10.0.0.0/8"]
  shutdown_timeout_sec: 20

management:
  host: "127.0.0.1"

log:
  level: "info"
  format: "json"
  output: "stdout"
```

## 可选配置

| 项目 | 说明 | 对应位置 |
|---|---|---|
| 开发模式 | 只在本地设置 `http.mode: debug`，不要用于生产 | `http.mode` |
| pprof | 默认关闭；不要在生产管理端口开启 | `management.enable_pprof` |
| 日志轮转 | `minimal` 默认写 stdout；文件输出时才使用轮转参数 | `log.max_size`、`log.max_backups`、`log.max_age`、`log.compress` |
| 探针超时 | 根据管理端探针响应时间调整 | `management.probe_timeout_sec` |
| 可观测性 | 模板提供结构化日志和健康检查；指标/trace 需要由反向代理、平台或项目代码补充 | `docs/OPERATIONS.md` |
| 多副本 | 服务无本地状态，可水平扩展；不需要迁移或备份 | Docker/Kubernetes 部署配置 |
| 部署资源 | 根据压测设置 CPU/内存限制、副本数和 PDB | 部署平台配置 |

## 上线前验证

- [ ] `jimu check .` 返回合法，manifest 与组合一致。
- [ ] `jimu upgrade . --check` 返回 0；有冲突时按[升级](../UPGRADING.md)处理，不能带冲突上线。
- [ ] `go test ./...`、`go build ./...` 通过。
- [ ] `docker build -t minimal:<version> .` 通过，容器以非 root 用户运行。
- [ ] `GET /api/v1/ping`、`GET /livez`、`GET /readyz` 返回预期状态。
- [ ] 通过反向代理访问 `GET /livez`，确认 TLS、超时和转发头配置正确。
- [ ] CORS 只对 `http.allowed_origins` 中的来源返回 `Access-Control-Allow-Origin`，其他来源不返回。
- [ ] 响应包含安全 Header：

```bash
curl -si http://127.0.0.1:8080/api/v1/ping \
  | grep -iE '^(x-content-type-options|x-frame-options|referrer-policy|permissions-policy|content-security-policy|strict-transport-security):'
```

- [ ] 限流策略已验证，超限时返回预期的 `429` 或网关状态码。
- [ ] `SIGTERM` 后服务在 `http.shutdown_timeout_sec` 内退出，未出现半写入响应。
- [ ] 日志为 JSON、不包含密码、令牌、验证码或未脱敏 PII，告警接收人可触达。

完整的生成、构建、代理和验证顺序见[生产上线走查](../PRODUCTION.md#完整部署走查)。
