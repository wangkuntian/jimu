# Changelog

所有 notable 变更都将记录在此文件。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

> 注：本仓库尚未发布 tag 版本。当前所有提交均在 `master` 上，视为 pre-1.0 开发阶段。

## [Unreleased]

### 新增

- **安全中间件** — CSRF 防护、安全响应头、API 请求签名验证
- **分布式能力** — Redis 分布式锁、用户级滑动窗口限流
- **存储与通知** — 文件存储抽象（local / S3 / OSS / MinIO）、通知分发器（Email / SMS / Webhook / WebSocket）
- **生产就绪** — 管理端 API、特性开关、K8s 部署清单
- **可靠传递** — Outbox 发件箱模式、数据库读写分离
- **API 契约** — OpenAPI 版本协商中间件、统一错误码文档
- **连接韧性** — DB / Redis 连接重试与连接池配置
- **可观测性** — OpenTelemetry 分布式追踪接入
- **请求校验** — JSON / Query 自动绑定校验中间件，中文友好错误翻译
- **工程化** — golangci-lint 配置、CONTRIBUTING 指南、CHANGELOG
- **限流可视化** — 管理端只读端点 `GET /api/v1/admin/ratelimit/auth`，不消费令牌查看认证限流计数与剩余窗口
- **Seed 幂等** — 种子命令按唯一键 upsert，可安全重复执行
- **文档** — ClickHouse 依赖说明、静态加密（字段级 + 全库边界）章节、CHANGELOG CI gate

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

---

## 提交历史（pre-tag）

以下为首次 tag 前的关键里程碑（按时间倒序）：

| 提交 | 说明 |
|------|------|
| `c9ed3a4` | docs: 更新 README 与 CLAUDE.md 反映项目现状 |
| `cb064ca` | feat: 集成全部组件并升级 Casbin v3 |
| `2351e52` | feat: WebSocket、管理端认证、错误码文档 |
| `cf9c251` | feat: 管理端 API、多租户、特性开关、K8s |
| `11b70bf` | feat: Outbox、读写分离、API 版本、错误码 |
| `0543c3a` | feat: 存储抽象、通知系统 |
| `bddbd65` | feat: 分布式锁、用户级限流 |
| `4129c47` | feat: CSRF、安全头、API 签名中间件 |
| `2487541` | feat: OpenTelemetry 追踪、请求校验中间件 |
| `2fc2b08` | feat: DB/Redis 连接韧性（重试 + 连接池） |
| `6992ce9` | feat: 业务路由接入 authz 保护 |
| `947d0cc` | feat: 生成完整 CRUD 模块 |
| `8c1f702` | feat: 运行时安全与 API 契约基线 |
| `075b321` | feat: 应用启动装配（bootstrap / container） |
| `4a541b4` | feat: RBAC 角色权限模块 + Casbin |
| `62431ef` | feat: JWT 认证（登录 / 注册 / 刷新） |
| `af9560b` | feat: 用户模块（完整 Clean Architecture） |
| `0e57b64` | feat: 基础表迁移 |
| `6554b3d` | feat: Gin HTTP 服务 + 中间件链 |
| `c39ac79` | feat: errors / response / pagination 共享包 |
| `b23d987` | feat: Gorm MySQL + Redis 客户端封装 |
| `010407d` | feat: Zap 结构化日志 |
| `5650581` | feat: Viper 配置加载 + 环境变量覆盖 |
| `64bccaf` | chore: 初始化项目依赖与配置 |
