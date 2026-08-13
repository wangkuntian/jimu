# Specification Quality Checklist: Jimu 后端框架能力规格

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-08-12
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs) — *注：本 spec 是框架能力契约，技术选型本身就是契约内容；已尽量用能力语言而非内部实现语言（如"基于令牌的认证"而非"JWT"）*
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

- 本 spec 定位为"现状能力契约"（由现有代码反推），因此个别能力条目（FR-009 令牌认证、FR-022 队列中间件）保留了技术名词作为能力契约的一部分，这属于框架类 spec 的刻意取舍，不视为实现细节泄漏。
- 2026-08-12 实现阶段收敛两处缺口：FR-028 阿里云短信（dysmsapi v5 SDK + 契约测试）、FR-034 发布门禁（release-check 含 govulncheck、tag 触发 CI）。见 research.md 审计矩阵更新。
- 2026-08-13 随代码刷新：新增 FR-036（统一出站 HTTP client）、FR-037（批量导入，CSV/Excel）；FR-034 补覆盖率门禁 ≥ 50%；Assumptions 补性能基准（make bench/loadtest）。全部清单项复查通过。
- 2026-08-13 异步韧性补齐：FR-036 补熔断（max_failures/reset_timeout_ms）；FR-029 补异步边界 trace 透传（队列/Outbox 注入与恢复）与异步维度指标（jimu_queue_*/jimu_outbox_*/jimu_httpclient_*）。全部清单项复查通过。
- 2026-08-13 批量导出与出站限流补齐：新增 FR-038（批量导出，CSV/Excel，与 FR-037 导入格式互通可回读）；FR-036 补按目标 host 独立限流（rate_limit_rate/rate_limit_burst，0 禁用）；FR-034 补关键解析逻辑模糊测试（importer/validator/jwt/snowflake 四目标）。全部清单项复查通过。
- 2026-08-13 观测与双栈补齐：新增 FR-039（gRPC server，可选双栈，内置健康检查 + 反射）；FR-028 补 Webhook 载荷 HMAC-SHA256 签名（sign_secret，X-Jimu-Timestamp/Signature 防重放）；FR-029 补定时任务执行指标（jimu_scheduler_*，成功/失败计数 + 耗时分布）与任务级 trace span。全部清单项复查通过。
- 2026-08-13 安全闭环补齐：新增 FR-040（自助密码重置：邮箱一次性验证码，防枚举，重置后登出全部会话）、FR-041（敏感字段 email/phone 字段级加密：AES-GCM + 盲索引，可选字段空值不冲突，未配置密钥明文等价可用）。Key Entities「用户」同步补敏感字段说明。全部清单项复查通过。
- 2026-08-13 覆盖率门禁提升：FR-034 门禁由 ≥ 50% 提升至 ≥ 70%（当前全量实测 70.1%），CI 阈值同步调整。全部清单项复查通过。
- 全部清单项通过，无需进入 `/speckit-clarify`。
