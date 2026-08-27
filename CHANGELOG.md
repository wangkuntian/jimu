# Changelog

所有 notable 变更都将记录在此文件。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增

- **Redis 高可用** — `redis.mode` 支持 `single`/`sentinel`/`cluster`，哨兵自动故障转移、集群连接分片；统一客户端接口，session/缓存/队列/限流/分布式锁跨模式复用。
- **TOTP 二次验证** — RFC 6238 自研实现；`/auth/mfa/setup|enable|disable` 自助管理，登录可携带 `totp_code`，密钥 AES-GCM 加密落库。
- **统一 gRPC 客户端** — 出站调用封装：连接管理 + 超时 + 指数退避重试 + 恢复拦截器 + Prometheus 指标。
- **错误追踪上报** — `error_reporting` 配置接入 Sentry，HTTP panic 自动上报（含 trace_id/span_id），优雅停机 Flush。

### 修复

- 修复 Compose 持久化数据卷与 Secret 变更后数据库账号密码不一致时的排障与隔离验证流程。
- 保留通用 `Importer.Import` 接口，改为通过逐行持久化回调导入；未配置回调时明确报错，避免伪报成功。
- 让发布门禁覆盖隔离 Compose 运行时与 API 契约验证。
- 运行时镜像 `apk upgrade` 升级 OpenSSL，修复 Trivy 扫描出的 CVE-2026-14456（HIGH）。

## [v0.1.0] - 2026-08-21

首个正式版本：Go 通用后端框架稳定底座 + 可组合模块 + 标准适配器 + 脚手架生成。

### 新增

- **模块化架构** — Clean Architecture 分层，业务逻辑依赖接口不依赖实现；`contract.Module` 统一注册
- **统一认证** — typed JWT + Redis refresh session + Casbin RBAC v3；API Key 认证（`X-API-Key`）；自助密码重置（邮箱验证码，防枚举，重置后强制登出全部会话）
- **敏感字段加密** — AES-256-GCM 字段级加密 + HMAC-SHA256 盲索引（email/phone）
- **OAuth 登录** — Google/GitHub/微信第三方登录
- **图形验证码** — 登录/注册验证码，Redis 一次性校验
- **统一响应** — 标准 `{code, message, data}` + 分页
- **多环境配置** — Viper + yaml + 环境变量覆盖，枚举值启动校验；`_FILE` 后缀文件注入敏感值（Docker Secrets 兼容）
- **结构化日志** — Zap + lumberjack 自动滚动
- **数据库** — Gorm + MySQL/MariaDB/PostgreSQL（`db.driver` 切换），Goose 迁移，雪花 ID 主键，读写分离
- **多队列支持** — Redis/Kafka/RabbitMQ 统一接口，三者均 at-least-once；消费幂等去重（已成功/死信任务重复投递 Ack 跳过）；失败按指数退避延迟重投；耗尽重试入死信表（管理 API 可查询与标记解决）
- **事件总线 + Outbox** — 内存事件总线同步/异步发布订阅；Outbox 保证事件发布与 DB 事务一致性
- **定时任务** — robfig/cron，MySQL 持久化 + 多实例分布式锁协调
- **文件存储** — 本地/S3/OSS/MinIO 统一接口
- **上传安全** — 大小限制 + magic-byte 嗅探覆盖可伪造 Content-Type + MIME 白名单；可选 ClamAV 病毒扫描（stdlib INSTREAM 协议，落库前同步扫描，fail-closed）
- **通知系统** — 邮件/短信(阿里云)/WebSocket/Webhook 抽象；Webhook 载荷 HMAC-SHA256 签名
- **统一出站 HTTP client** — timeout + retry/backoff + 熔断 + 按目标 host 限流 + OTel traceparent 注入
- **可观测性** — OpenTelemetry 分布式追踪（HTTP/Gin + Gorm + Redis 全链路）；Prometheus 指标（DB 连接池/运行时/HTTP/队列/Outbox/熔断/调度）；访问日志注入 trace_id/span_id
- **gRPC server** — 与 HTTP 双栈并存，内置健康检查 + 反射
- **Feature Flag** — 运行时特性开关（灰度百分比 + 白名单）
- **HTTP 安全边界** — 请求体大小/超时/可信代理/CORS/安全头；CSRF 防护；API 签名验证中间件
- **限流** — 全局令牌桶 + 登录/注册固定窗口 + 用户维度滑动窗口；管理端只读端点查看限流计数
- **审计日志** — 有界队列批量写入，匿名请求安全处理
- **缓存** — Cache-Aside + GetOrSet 自动回填 + 防雪崩
- **分布式锁** — Redis 实现
- **数据导入/导出** — CSV/Excel 模板导入与导出，格式互通可回读
- **管理 API** — 系统状态、在线用户、强制下线、错误码文档、特性开关、任务、死信
- **脚手架** — Cobra CLI 一键生成完整模块骨架
- **API 文档** — Swagger UI（中文注解）
- **健康检查** — `/livez` + `/readyz`（DB + Redis 探测）
- **优雅停机** — 显式 Application 生命周期，反向停止组件
- **国际化** — `Accept-Language` 中文/英文错误与校验消息
- **自定义校验** — 手机号、密码强度、身份证、用户名规则
- **事务封装** — 统一事务管理 helper
- **Docker / K8s / CI/CD** — Dockerfile + docker-compose + K8s manifests（Deployment/Service/HPA/Ingress）+ Helm chart；GitHub Actions + Dependabot；govulncheck + Trivy + SBOM + 镜像 smoke test；golangci-lint + pre-commit；覆盖率门禁 ≥ 70%

### 变更

- 配置系统标准化，统一枚举值启动校验（http.mode / log.level / log.format）
- Casbin 升级至 v3，RBAC 模型适配
- Makefile 命令命名统一（docker / compose 分组）
- Docker 构建使用七牛云模块代理加速
- 移除多租户抽象，明确单租户 RBAC 定位

### 修复

- Swagger UI 不可访问问题
- Viper 不再加载 `.env`，避免与 docker-compose `env_file` 冲突
- 移除硬编码管理员密码，改为环境变量注入
