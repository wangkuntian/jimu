# Quickstart: 框架能力端到端验证

**Created**: 2026-08-12

本指南验证 spec 中声明的能力真实可用。契约细节见 [contracts/](contracts/)，数据模型见 [data-model.md](data-model.md)。

## 前置条件

- Go 1.26+、MariaDB 10.5+、Redis 6+
- Docker（本地依赖容器）

## 环境搭建

```bash
# 1. 配置
cp .env.example .env
# 编辑数据库/Redis 连接

# 2. 启动依赖
docker compose up -d mariadb redis

# 3. 迁移 + 种子（admin 账号与基础权限）
go run ./cmd/cli migrate up
go run ./cmd/cli seed

# 4. 启动服务
go run ./cmd/server
```

## 验证场景

### V1 认证闭环（FR-008/009/010/011）

```bash
# 登录 → 拿 token
curl -s -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<seed密码>"}'

# 用 token 访问受保护接口（RBAC）
curl -s localhost:8080/api/v1/admin/status -H "Authorization: Bearer $TOKEN"
# 预期：{"code":0,...} 状态恒 200

# 未认证被拒
curl -s -o /dev/null -w '%{http_code}' localhost:8080/api/v1/admin/status
# 预期：200（body.code 非 0，未认证业务错误）

# 刷新 / 登出
curl -s -X POST localhost:8080/api/v1/auth/refresh -H "Authorization: Bearer $TOKEN"
curl -s -X POST localhost:8080/api/v1/auth/logout -H "Authorization: Bearer $TOKEN"

# API Key（服务间）
# 管理端生成 key 后，X-API-Key 头访问
```

### V2 运维与观测（FR-005/006/029/031）

```bash
curl -s localhost:8080/livez   # 200
curl -s localhost:8080/readyz  # DB+Redis 可达时 200

# 优雅停机：Ctrl+C / SIGTERM，观察日志确认在途请求完成后退出

# 指标（management server）
curl -s localhost:9090/metrics | grep jimu_http

# 追踪：开启 otel.enabled 后，访问日志含 trace_id/span_id
```

### V3 模块扩展（FR-032/033/契约）

```bash
go run ./cmd/cli module create orders
# 生成骨架 → 实现 contract.Module（见 contracts/module.md）
# 启动后新路由自动注册在 /api/v1，返回统一格式
# Swagger：http://localhost:8080/swagger/index.html 可见新接口
```

### V4 可选能力（FR-012/013/022/024/025/026/027）

| 能力 | 验证 |
|------|------|
| OAuth | 配置 `oauth.providers.google.enabled=true` → 走第三方登录，绑定本地用户 |
| 验证码 | `captcha.enabled=true` → 登录返回验证码，错误一次性校验失败 |
| 队列 | `queue.type=redis/kafka/rabbitmq` 切换 → 任务投递/消费，调用方代码不变 |
| Outbox | `outbox.publisher=event_bus/mq` → 事务内发事件，跨服务可投递 |
| 定时任务 | `scheduler.store=mysql` → 持久化任务，多实例不重复执行 |
| 特性开关 | admin 调整灰度百分比/白名单 → 目标用户按规则命中 |
| 文件存储 | storage 配置切换 local/s3/oss/minio → 上传/下载/PresignedURL |

### V5 安全与限流（FR-014/015/016/017）

```bash
# 登录错误凭据 5 次 → 触发固定窗口限流，返回限流业务错误
# 超大请求体 → 414/413 边界拒绝
# 审计：数据库 audit_logs 有批量写入，匿名请求无敏感详情
# CSRF：enable security.csrf_secret 后，无 CSRF token 的变更请求被拒
```

## 质量门禁

```bash
make release-check   # fmt/vet/test
golangci-lint run    # 静态检查
# CI：govulncheck + Trivy + 覆盖率 + race（发布路径须含 govulncheck，见 plan D1）
```

## 预期结果

- 全部 V1–V5 场景通过 → spec 能力契约成立
- 任一场景失败 → 对照 [research.md](research.md) 审计矩阵判断是契约高估还是实现缺陷

## 验证结果备注（2026-08-12 实现阶段）

- **V1 认证闭环**：部分验证 ✅。compose 栈运行时 login 对错误凭据返回 `{"code":1006,"message":"用户名或密码错误","request_id":...}`（统一响应 + 认证路径正常）；正确登录需 `ADMIN_PASSWORD` 种子账号，本环境未配置，未做 token 全流程。
- **V2 运维观测**：已验证 ✅。management server :9090 的 `/livez`、`/readyz`、`/metrics` 均返回 200；优雅停机、链路追踪未在本次验证。
- **V3 模块扩展**：已验证 ✅。`go test ./tools/generator/` 全过（含 TestGeneratedModuleCompiles），脚手架生成模块骨架编译通过。
- **V4 可选能力**：SMS 契约测试 ✅（`go test ./internal/platform/notification/` 3 用例，httptest mock server 断言手机号/签名/模板/变量与错误路径）；**真实发送未验证** —— 需阿里云 AccessKey（AccessKey ID/Secret 非本环境可用凭据），列为待验证项。
- **V5 安全与限流**：未完整跑。限流/CSRF/审计需特定配置与流量，未在本阶段逐项触发。
- **质量门禁**：`make release-check` 通过（fmt-check + vet + test + govulncheck，0 漏洞）；`golangci-lint run` 报 7 处**预存在**告警（cache.go SA4004、csrf.go unused、redis_queue.go ZRangeByScore deprecated、s3.go 端点解析 deprecated + pathStyle unused），非本次任务引入，未在本次修复。
