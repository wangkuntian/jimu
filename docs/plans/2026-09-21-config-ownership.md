# 能力可插拔 P2.1：配置归属下沉 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 落实设计 §8：能力配置的结构体、默认值与校验由**能力自身**声明，`internal/config` 只保留内核段；装配时按启用集解码+合并，**未启用能力的配置段既不出现也不校验**。对外 YAML 键**逐一不变**。

**Architecture:** `internal/config` 只留内核/基础设施段（http/db/redis/log/otel/management/id/server/security/ratelimit/cache/http_client/grpc/error_reporting/capabilities）。每个能力包新增 `config.go`：`ConfigKey`（点分路径，YAML 中位置不变）+ `Config` 结构体 + `ApplyDefaults()` + `Validate()`。组合根 `internal/app` 按启用集对每个能力段执行 `UnmarshalKey → ApplyDefaults → Validate`，并把强类型配置注入能力构造函数。

**Tech Stack:** Go 1.26 · viper（`UnmarshalKey` 支持点分路径）· testify

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md` §8（配置归属）、§10 P2（三层机制）、§11（配置键不变）

## Global Constraints

- **禁止自动提交**：commit 仅在用户明确指令后执行（AGENTS.md 最高优先级）
- **对外 YAML 键逐一不变**：`auth`/`auth.webauthn`/`auth.provisioning`/`oauth`/`storage`/`upload`/`queue`/`outbox`/`scheduler`/`audit`/`captcha`/`email`/`sms`/`notification`/`retention` 在 `configs/*.yaml` 中的位置与键名保持字节级不变
- **能力之间只经 `contract` 端口调用**；`internal/config` 不得 import `capabilities`
- 全绿门禁：`gofmt -l .`、`go build ./...`、`go vet ./...`、`golangci-lint run ./...`、`make check-log-usage`、`make test-cover`+`test-coverage-check`、`make test-race`、`make swagger-check`、`make bench-ci`、`make release-check COMPOSE_ENV=.env.example`
- 分支 `feature/config-ownership`，PR 目标 `release/v0.3.0`，feature→release 用 squash merge
- 每一步保持 `full` 全绿、可独立提交

## 裁定（执行前必读）

1. **内核保留段**（留在 `internal/config`）：`http`、`management`、`db`、`redis`、`ratelimit`、`log`、`server`、`id`、`cache`、`security`、`otel`、`error_reporting`、`http_client`、`grpc`、`capabilities`。
   理由：均为传输/基础设施/内核中间件机制，无归属能力（`grpc` 是内核传输，`cache` 是 Redis 缓存抽象，`http_client` 是共享出站客户端，`security`/`ratelimit` 是内核中间件配置）。

2. **下沉段与归属**（`ConfigKey` 为 YAML 点分路径，保持不变）：

   | YAML 键 | 归属能力 | 现结构体 |
   |---|---|---|
   | `auth`（不含 webauthn/provisioning） | `auth` | `AuthConfig` |
   | `auth.webauthn` | `passkey` | `WebAuthnConfig` |
   | `auth.provisioning` | `tenant` | `ProvisioningConfig` |
   | `oauth` | `oauth` | `OAuthConfig` |
   | `storage` | `storage` | `StorageConfig` |
   | `upload` | `uploadsec` | `UploadConfig` |
   | `queue` | `queue` | `QueueConfig` |
   | `outbox` | `outbox` | `OutboxConfig` |
   | `scheduler` | `queue` | `SchedulerConfig` |
   | `audit` | `audit` | `AuditConfig` |
   | `captcha` | `captcha` | `CaptchaConfig` |
   | `email`+`sms`+`notification` | `notification` | `EmailConfig`/`SMSConfig`/`NotificationConfig` |
   | `retention` | `retention` | `RetentionConfig` |

   跨能力嵌套段（`auth.webauthn` → passkey、`auth.provisioning` → tenant）用点分 `UnmarshalKey` 解码，**保持 YAML 布局不变**。

3. **`scheduler` 归 `queue`**：调度器实例（`kernel/scheduler`）由 queue 能力用于作业调度（`/admin/tasks*`、`job_history`），其配置随之归 queue。

4. **`Watch` 重载语义**：`config.Watch` 只重载并校验内核段；能力段变更需重启进程（与现有「结构类变更需重启」一致，文档注明）。

## Task 1: 机制（不改任何段，先建通道）

**Files:**
- Modify: `internal/config/config.go`（新增 `LoadRaw`/导出 viper 供组合根解码）
- Create: `internal/contract/config.go`（`Section` 解码的最小端口，避免能力直接依赖 viper）
- Create: `internal/app/capconfig.go`（按启用集解码+默认值+校验的聚合器）
- Test: `internal/app/capconfig_test.go`

**Interfaces:**
- Produces: `app.LoadCapabilityConfig[T any](sec contract.ConfigSection, key string, def func(*T)) (*T, error)`
- Produces: `contract.ConfigSection`（`UnmarshalKey(key string, raw any) error`）

- [ ] **Step 1: `config` 暴露原始段解码能力**

```go
// internal/config/config.go
// SectionDecoder 暴露按 YAML 键解码单个配置段的能力，供组合根解码能力配置段。
type SectionDecoder interface {
    UnmarshalKey(key string, rawVal any) error
}

// Load 返回值增加 viper 解码器（兼容旧调用：旧签名保留为 Load 的包装）
func LoadWithSections(env string) (*Config, SectionDecoder, error)
```

- [ ] **Step 2: `contract.ConfigSection` 端口**（避免 capability → viper）

- [ ] **Step 3: `app.capconfig.go` 聚合器**

```go
// LoadCapabilitySection 按 YAML 键解码能力配置段：解码 → 默认值 → 校验。
// 未启用的能力不调用本函数，因此其配置段既不出现也不校验（设计 §8）。
func LoadCapabilitySection[T any](dec contract.ConfigSection, key string, applyDefaults func(*T), validate func(T) error) (*T, error)
```

- [ ] **Step 4: 验证**（`gofmt -l .`、`go build ./...`、`go vet ./...`、`go test ./internal/app/ ./internal/config/ -count=1`）

- [ ] **Step 5: 提交**（用户授权后）`feat(config): add capability config section loading`

## Task 2: 试点 —— `retention`（最小段，验证机制端到端）

**Files:**
- Create: `internal/capabilities/retention/config.go`（`ConfigKey="retention"`、`Config`、`ApplyDefaults`、`Validate`）
- Modify: `internal/config/config.go`（删 `Retention` 字段）、`validate.go`（删 retention 校验）
- Modify: `internal/app/bootstrap.go`（改用 `retention.Config`）
- Test: `internal/capabilities/retention/config_test.go`

- [ ] **Step 1: retention 自有 config + 默认值 + 校验（迁移现有校验语义）**
- [ ] **Step 2: 组合根按启用集解码注入**
- [ ] **Step 3: 验证 + 提交** `refactor(retention): own its configuration section`

## Task 3–8: 逐段下沉（每段一个提交，保持全绿）

按依赖与风险从低到高：

- [ ] **Task 3** `audit` + `captcha`（简单段）
- [ ] **Task 4** `storage` + `upload`（storage 被 uploadsec 消费，需端口）
- [ ] **Task 5** `queue` + `outbox` + `scheduler`（三者有组合校验 `outbox.publisher=mq` 依赖 `queue.type`，跨能力校验需在组合根做）
- [ ] **Task 6** `notification`（email+sms+notification 三段同归）
- [ ] **Task 7** `oauth`
- [ ] **Task 8** `auth` + `auth.webauthn`(passkey) + `auth.provisioning`(tenant)（最复杂：嵌套段拆归属）

## Task 9: 收口 —— 内核段固化 + 文档

**Files:**
- Modify: `internal/config/config.go`（确保只剩内核段）、`validate.go`（只剩内核校验）
- Modify: `internal/app/container.go`、`cmd/server/main.go`
- Modify: `configs/app.yaml`（**仅注释说明归属，键不变**）、`README.md`（配置表标注归属能力）、`docs/design`（§10 P2.1 标记完成）、`docs/releases/v0.3.0.md`
- Test: `internal/config/validate_test.go`（断言未启用能力的配置段不校验）

- [ ] **Step 1: 加「未启用能力配置段不校验」的回归用例**

```go
// 关闭 retention 后，非法 retention 配置不得导致启动失败
caps := catalog.Resolve([]string{"user"})
require.NoError(t, app.ValidateCapabilityConfigs(cfg, caps))
```

- [ ] **Step 2: 全量回归**（含 `make release-check`、`make bench-ci`）
- [ ] **Step 3: 文档更新**
- [ ] **Step 4: 提交** `docs(config): record capability-owned configuration`

---

## Self-Review

**1. Spec 覆盖**：§8 三层（内核段保留 / 能力声明默认值与校验 / 装配合并、未启用不校验）逐项落实；§11「配置键不变」以点分 `ConfigKey` 保证；§10 P2 的「`capabilities.enabled` 配置合并与校验」由此完成。

**2. 占位符扫描**：无 TBD；每 Task 给出文件清单与验证命令。

**3. 风险**：① `auth.webauthn`/`auth.provisioning` 嵌套拆归属是本计划最易出错处 —— 必须用点分 `UnmarshalKey` 且 `configs/app.yaml` 字节不变（用 `git diff configs/` 断言）；② `outbox.publisher=mq` 校验依赖 `queue.type`，跨能力 → 必须在组合根（两者都启用时）校验，否则校验会漏；③ `Watch` 重载只覆盖内核段，能力段变更需重启，需在 README 注明；④ 51 处消费点主要在 `internal/app`，但 `capabilities/tenant/module.go`（读 `cfg.Auth`）、`capabilities/user/module.go`（读 `cfg.Cache`）、`capabilities/uploadsec/upload_handler.go`（读 storage 配置）三处需一并改为端口/注入，避免能力反向依赖 config 结构体布局。
