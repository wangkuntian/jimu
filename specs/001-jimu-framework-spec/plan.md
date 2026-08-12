# Implementation Plan: Jimu 后端框架能力规格

**Branch**: `001-jimu-framework-spec` | **Date**: 2026-08-12 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-jimu-framework-spec/spec.md`

## Summary

从现有代码反推框架能力契约（35 条 FR）。代码审计（research.md）显示 32/35 已真实实现；存在 **2 处规格与实现不一致** 需收敛：

1. **FR-028 通知（SMS 桩）**：`internal/platform/notification/sms.go` 声明 aliyun/tencent/twilio 三家，但 `sendAliyun`/`sendTencent` 直接返回 "not implemented yet"，SMS 渠道一用即失败。违反宪法 VI（能力需可运行示例）。
2. **FR-034 发布门禁**：`make release-check` 仅 `fmt-check vet test`（Makefile:227），不含 govulncheck/Trivy；`release.yml` 打 tag 发布只跑 `make release-check`。违反宪法 V（发布 MUST 含 govulncheck 依赖与 Trivy 镜像扫描）。

次要：FR-014 全局限流实为令牌桶（spec 措辞为"固定+滑动窗口"，登录=固定、用户=滑动为真），收敛 spec 措辞即可。

计划动作：修复 2 处缺口 + 生成框架契约文档（data-model / contracts / quickstart）。

## Technical Context

**Language/Version**: Go 1.26+

**Primary Dependencies**: Gin（HTTP）、Gorm + gorm.io/driver/mysql（ORM）、Viper（配置）、Zap + lumberjack（日志）、Casbin v3（RBAC）、Goose（迁移）、Cobra（CLI）、go-playground/validator（校验）、swaggo/swag（文档）、OpenTelemetry otlptracegrpc（追踪）、Prometheus client_golang（指标）、robfig/cron（调度）、golang.org/x/time/rate（令牌桶）；队列三中间件客户端（redis/kafka/rabbitmq）、对象存储客户端（s3/oss/minio 经 S3 协议）

**Storage**: MariaDB（MySQL 协议）+ Redis；主键雪花 ID（应用生成，非 AUTO_INCREMENT）

**Testing**: stdlib `testing`；单测 sqlite/in-memory；集成测试真实 MySQL，经 `testutil.SkipUnlessMysql` 自动跳过；httptest 用于中间件；门禁 `make release-check` + golangci-lint + pre-commit（fmt/vet/lint）+ CI（govulncheck/Trivy/覆盖率/race）

**Target Platform**: Linux server；Docker + Kubernetes（deploy/k8s）

**Project Type**: 通用后端库（library）+ server + CLI 脚手架

**Performance Goals**: 未定义量化目标（spec Assumptions 明确不虚构数值）。设计取向：多实例水平扩展（雪花 ID worker_id、分布式锁、调度去重）

**Constraints**: 单租户（无多租户）；业务无关（宪法 I）；最少第三方依赖（宪法 IV）；公共 API 向后兼容（宪法 II）；HTTP 状态码恒 200、错误走 body.code

**Scale/Scope**: 多实例部署；13 张迁移表、9 个业务模块、4 层架构（domain/application/infrastructure/interfaces）

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| 门禁 | 结果 | 说明 |
|------|------|------|
| I. 业务无关 | ✅ | 所有改动均为框架能力，不引入业务逻辑 |
| II. API 稳定性 | ✅ | 新增均为可选适配器/工具，向后兼容；无破坏性变更 |
| III. 可组合模块/标准适配器 | ✅ | SMS 走既有 notification 渠道接口；门禁改动不影响模块装配 |
| IV. 简单优先/最少依赖 | ⚠️ 见缺口 | SMS 实现经用户指定用阿里云 SDK（`dysmsapi-20170525/v5`），属批准的最小必要依赖偏差，仅 provider=aliyun 时编译进；govulncheck/Trivy 是工具非运行时依赖 |
| V. 验证与质量 | ❌ **违规** | 发布路径缺 govulncheck/Trivy，宪法明确要求；本计划修复 |
| VI. 文档与示例 | ❌ **违规** | SMS 能力声称存在但运行时报错；本计划实现或移除该声称 |

Phase 0 前：门禁 V/VI 违规确认为待修复项而非违背宪法的理由，Phase 0/1 无其他阻塞。

## Project Structure

### Documentation (this feature)

```text
specs/001-jimu-framework-spec/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output — 能力审计 + 决策
├── data-model.md        # Phase 1 output — 9 实体数据模型
├── quickstart.md        # Phase 1 output — 端到端验证指南
├── contracts/           # Phase 1 output — 公共契约（模块/响应/配置/CLI/API）
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

本特性不改代码目录结构（框架结构已定型），改动为外科手术式局部修改：

```text
internal/platform/notification/
├── sms.go               # [FR-028] 实现 sendAliyun（dysmsapi-20170525/v5 SDK，见 research.md D2）
└── sms_test.go          # [FR-028] 契约测试（client Endpoint 指向 mock server）

go.mod                   # [FR-028] 新增 github.com/alibabacloud-go/dysmsapi-20170525/v5

Makefile                 # [FR-034] release-check 追加 govulncheck（+可选 trivy）
.github/workflows/
├── ci.yml               # [FR-034] 校验/对齐门禁
└── release.yml          # [FR-034] 发布前跑完整门禁

README.md                # 同步 SMS 能力表述与限流策略描述
spec.md                  # FR-014 限流措辞对齐现实
```

**Structure Decision**: 不新增源代码目录；复用既有分层与平台包。文档产物放 `specs/001-jimu-framework-spec/`。改动仅触及 4 类文件（notification 包、Makefile、CI、文档）。

## Complexity Tracking

> Constitution Check 违规项的复杂度证明

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| SMS 渠道实现（FR-028） | 宪法 VI：声称的能力 MUST 可运行；README/spec 已声明 SMS；用户指定用阿里云 SDK | 保留"not implemented yet"桩 → 用户一用即崩，违背宪法；从文档删除 SMS 声称 → 用户明确要求实现阿里云 SDK，删属不执行决策；引入 SDK 属批准的最小必要依赖偏差（仅 aliyun provider 编译进，不泄漏） |
| 发布门禁补 govulncheck（FR-034） | 宪法 V：发布 MUST 含 govulncheck 与 Trivy | 仅 CI 扫（现状）→ tag 发布路径绕过，宪法字面违规 |

> 说明：govulncheck 与 Trivy 为构建期工具，非运行时依赖，不违反宪法 IV 依赖最小化。
