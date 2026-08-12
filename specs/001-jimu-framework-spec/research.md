# Research: 能力审计与缺口决策

**Created**: 2026-08-12

对 spec 35 条 FR 逐条核对现有代码（internal/ 下真实实现，非 README 声称）。

## 审计矩阵

| FR | 状态 | 证据 |
|----|------|------|
| FR-001 统一响应 | implemented | `internal/shared/response/response.go` — body.code，HTTP 恒 200 |
| FR-002 多环境+枚举校验 | implemented | `internal/config/validate.go` 启动校验；`APP_ENV` 切换 |
| FR-003 文件注入敏感配置 | implemented | `internal/config/config.go` `getEnvOrFile` 支持 `*_FILE` |
| FR-004 结构化日志 | implemented | `internal/platform/logger/zap.go` + lumberjack 滚动 |
| FR-005 健康检查 | implemented | `internal/platform/observability/health.go` `/livez` `/readyz` |
| FR-006 优雅停机 | implemented | `internal/app/container.go` 逆序 Stop；HTTP `Shutdown(ctx)` |
| FR-007 分布式 ID | implemented | `internal/shared/id/id.go` snowflake + workerID 校验 |
| FR-008 注册登录 | implemented | `internal/modules/auth/interfaces/router.go`；凭据错误不枚举 |
| FR-009 令牌+刷新+登出 | implemented | auth router `/refresh` `/logout`；`internal/platform/auth/jwt.go` |
| FR-010 RBAC | implemented | `internal/platform/auth/casbin.go`；role/permission 模块 + 策略种子 |
| FR-011 API 密钥认证 | implemented | `internal/platform/auth/apikey.go` + middleware；按需挂载 |
| FR-012 OAuth 多提供商 | implemented | `internal/platform/oauth/{google,github,wechat}.go` 均真实 oauth2 |
| FR-013 图形验证码 | implemented | `internal/platform/captcha/captcha.go` RedisStore 一次性消费 |
| FR-014 限流 | implemented（措辞偏差） | 登录=固定窗口 `auth/limiter.go`；用户=滑动窗口 `ratelimit_user.go`；**全局=令牌桶 `ratelimit.go`** |
| FR-015 HTTP 安全边界 | implemented | 体积/超时/可信代理/CORS/安全头 |
| FR-016 CSRF+签名 | implemented | `middleware/signature.go` HMAC+nonce；`csrf.go` double-submit |
| FR-017 审计日志 | implemented | `modules/audit/application/worker.go` 批量写；不记敏感请求体 |
| FR-018 迁移+种子 | implemented | `cmd/cli/main.go` migrate/seed；goose |
| FR-019 事务封装 | implemented | `internal/platform/db/transaction.go` |
| FR-020 读写分离 | implemented | `config.go` ReadHosts/ReadPorts；`db/mysql.go` Replicas |
| FR-021 缓存+分布式锁 | implemented | `platform/cache/cache.go`；`platform/redis/lock.go` |
| FR-022 可插拔队列 | implemented | `platform/queue/factory.go` redis/kafka/rabbitmq 均有 Submit/Consume/Ack + 契约测试 |
| FR-023 事件总线 | implemented | `platform/event/bus.go` 同步+异步 |
| FR-024 Outbox | implemented | `platform/outbox/` 双发布器（event_bus/mq）均真实 |
| FR-025 定时任务 | implemented | `platform/scheduler/` MySQL store + 分布式锁去重 |
| FR-026 特性开关 | implemented | `platform/feature/feature.go` 灰度百分比+白名单 |
| FR-027 文件存储 | implemented | `platform/storage/factory.go` local/s3/oss/minio（OSS 复用 S3 协议） |
| **FR-028 通知** | **implemented** ✅（2026-08-12 修复） | email/websocket/webhook/log 真实；`sendAliyun` 已用阿里云 dysmsapi v5 SDK 实现（含契约测试），`sendTencent` 报"not configured" |
| FR-029 OTel+Prometheus | implemented | `platform/observability/tracing.go`；metrics 中间件 + deploy 配置 |
| FR-030 校验+i18n | implemented | `shared/validator/`；`shared/i18n/` |
| FR-031 管理能力 | implemented | `modules/admin/module.go` status/users/apikeys/error-codes/features/tasks/jobs/import 全接线 |
| FR-032 脚手架 | implemented | `jimu module create` → `tools/generator` |
| FR-033 Swagger | implemented | swag 注解 + swagger.yaml/json + UI |
| **FR-034 部署+门禁** | **implemented** ✅（2026-08-12 修复） | deploy/k8s 真实；`make release-check` 已含 govulncheck，ci.yml 增加 tag 触发使发布路径跑完整门禁 |
| FR-035 无多租户 | implemented（约束） | internal/ 无 tenant 字段或中间件 |

## Decisions

### D1: FR-034 — 发布门禁补漏洞扫描与覆盖率

**Decision**: `make release-check` 追加 `govulncheck`；`release.yml` 发布前运行完整门禁（含 govulncheck，Trivy 因需镜像构建保留在 CI 推送阶段并对齐 release 触发）。覆盖率门禁维持 CI 现有配置并确保 tag 路径触发。

**Rationale**: 宪法 V 明文要求发布经 `make release-check` 与 CI 门禁，含 govulncheck 依赖与 Trivy 镜像扫描。现状 tag 发布只跑 `make release-check`（=fmt/vet/test），漏洞扫描仅 PR/push 路径，发布路径违规。

**Alternatives considered**: 仅调整文档声称（"CI 已扫"）→ 宪法要求发布路径，不成立；release-check 全量挂 Trivy → Trivy 需构建镜像，本地慢且非依赖扫描，govulncheck 已覆盖依赖漏洞，Trivy 留 CI。

### D2: FR-028 — 实现短信渠道（阿里云 SDK）

**Decision**: 使用官方 SDK `github.com/alibabacloud-go/dysmsapi-20170525/v5` 实现 **阿里云** 短信适配器，填充 `sms.go` 的 `sendAliyun`；tencent 保持声明但改为明确报错"provider not configured"，并从 README 特性清单降级其声称，避免误导。由用户指定（2026-08-12）。

**Rationale**: 用户明确选择阿里云 SDK 优先实现；`sms.go` 现有 TODO 注释即指向 `aliyun-dysmsapi-sdk`。`SMSConfig`（api_key/api_secret/sign_name）与 `Message`（To=手机号、TemplateID、Data=模板变量）字段已齐备，无模型改动。SDK 用法：`dysmsapi/client` + Tea runtime，`SendSms` 接口。

**Dependency note（宪法 IV 偏差，经用户批准）**: 引入阿里云 SDK 及其 Tea 系传递依赖（client/tea/tea-utils 等）偏离"最少依赖"原则；此为构建期可选组件的依赖，仅启用 `provider=aliyun` 时编译进，不泄漏到其他组件。若后续改为原生 HTTP 调阿里云 REST 可去除此依赖（保留为可选重构方向）。

**Alternatives considered**: Twilio REST（stdlib http）→ 用户否决，明确要求阿里云 SDK；保留 "not implemented yet" 桩 → 一用即崩，违宪 VI；三家 SDK 全实现 → 无 tencent 需求，超范围。

### D3: FR-014 — 限流措辞对齐现实

**Decision**: 更新 spec.md 与 README：全局=令牌桶、登录=固定窗口、用户维度=滑动窗口。

**Rationale**: 现状实现是合理设计（整体限速用令牌桶、防爆破用固定/滑动窗口），改代码无用户价值；契约措辞应如实描述。

**Alternatives considered**: 把全局限流改成固定/滑动窗口 → 无行为收益，徒增改动。

## Open Items

- SMS 其他供应商（aliyun/tencent）是否在后续迭代补齐：取决于真实使用方需求，见 tasks 中的明确待定项。
- 覆盖率数值门槛在 CI 中的具体百分比：维持现有配置，不在此计划改动。
