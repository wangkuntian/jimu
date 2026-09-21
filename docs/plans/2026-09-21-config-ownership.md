# 能力可插拔 P2.1：配置归属下沉 实现计划

> 本文件是 P2 子阶段 P2.1 的详细计划；P2 的三层机制总纲见
> [2026-09-21-p2-three-layer-mechanism.md](2026-09-21-p2-three-layer-mechanism.md)。

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

2. **下沉段与归属**（侦察后修正：`storage`/`notification`/`retention`/`scheduler` **不是 catalog 能力**，其「不启用」由自身 `enabled` 配置驱动，不走启用集）:

   **A. catalog 能力段 —— 按启用集校验（未启用则既不出现也不校验）**

   | YAML 键 | 归属能力 | 现有校验 |
   |---|---|---|
   | `auth`（不含 webauthn/provisioning） | `auth` | issuer/过期/限流；jwt_secret 强度（prod） |
   | `auth.webauthn` | `passkey` | `validateWebAuthn` |
   | `auth.provisioning` | `tenant` | `validateProvisioning` + 依赖 `auth.public_registration` |
   | `oauth` | `oauth` | `validateOAuthProviders` |
   | `queue` | `queue` | `validQueueTypes` |
   | `outbox` | `outbox` | `validOutboxPublishers` + **跨能力**：`publisher=mq` 时 `queue.type` 必须受支持 |
   | `audit` | `audit` | queue/batch/flush 关系 |
   | `captcha` | `captcha` | `enabled` 时 `ttl_min>0` |
   | `upload` | `uploadsec` | 无校验（仅结构体下沉） |

   **B. 非 catalog 包段 —— 保持现有语义（不由启用集门控）**

   | YAML 键 | 归属包 | 门控方式 | 现有校验 |
   |---|---|---|---|
   | `retention` | `retention` | 自身 `enabled` | enabled 时 cron 必填、batch_size 非负 |
   | `scheduler` | `queue` | 无条件 | `validSchedulerStores` |
   | `storage` | `storage` | 无条件 | 无校验 |
   | `email`+`sms`+`notification` | `notification` | 无条件 | 无校验 |

   `scheduler` 归 `queue`：调度器实例由 queue 能力用于作业调度（`/admin/tasks*`、`job_history`）。


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

## Task 2: 试点 —— `captcha`（最小 catalog 能力段，验证启用集门控端到端）

**Files:**
- Create: `internal/capabilities/captcha/config.go`（`ConfigKey="captcha"`、`Config`、`Validate`、`Load`）
- Modify: `internal/config/config.go`（删 `Captcha` 字段）、`validate.go`（删 captcha 校验）
- Modify: `cmd/server/main.go`（按 `enabled["captcha"]` 解码注入）
- Test: `internal/capabilities/captcha/config_test.go`（含「未启用则不校验」回归）

- [ ] **Step 1: captcha 自有 config + 校验（迁移现有校验语义，不新增默认值）**
- [ ] **Step 2: 组合根按启用集解码：仅 `enabled["captcha"]` 时执行 `Load`**
- [ ] **Step 3: 验证 + 提交** `refactor(captcha): own its configuration section`

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

---

## 执行结果（部分完成，8/9 提交）

分支 `feature/config-ownership`，已完成机制与 8 个段：

| commit | 内容 |
|---|---|
| `fcad5ca` | 机制：`config.LoadWithSections` + `config.LoadSection`（泛型约束 `SectionConfig`） |
| `4fcc6fc` | `captcha`（catalog 能力，启用集门控试点） |
| `b5fed9a` | `audit`（含环境覆盖随段下沉、内层与 config 结构体解耦） |
| `661ac13` | `storage` + `upload`（Container 增加 `Sections`/`Enabled`；`configs/` 零改动） |
| `df1c708` | `queue` + `outbox` + `scheduler`（含跨能力校验移到组合根） |
| `8b7ef5d` | `email` + `sms` + `notification` |
| `28b069d` | `oauth` |
| `aa4cd6f` | `retention` |

`configs/*.yaml` 全程 `git diff` 为 0 行 —— 对外配置键不变（§11）得以保持。

### 机制修正（执行中发现）

1. **`LoadSection` 的方法值陷阱（已修）**：初版签名 `LoadSection(dec, key, out, applyDefaults func(), validate func() error)` 把校验钩子当函数值传参，Go 在传参时就把接收者按**零值副本**绑定，导致校验永远看零值、恒通过。改为泛型约束 `SectionConfig{ApplyDefaults(); Validate() error}`，在 `LoadSection` 内部对指针动态派发，并加回归用例。此后每个能力只需实现这两个方法（无默认值/校验者写空实现）。
2. **`Container` 成为能力配置的解码点**：`NewContainer(cfg, sections, enabled)`；`main` 的 `catalog.Resolve` 上移到建容器之前（容器需启用集决定哪些能力段加载）。`bootstrap` 经 `Container.OutboxPublisher` 拿到 outbox 决策，不再读 `cfg.Outbox`。
3. **段的环境覆盖随段下沉**：`config.GetEnvOrFile` 导出，`AUDIT_HASH_SECRET` 的覆盖移入 `audit.Config.ApplyDefaults`。

### 剩余：Task 8（`auth` 段）与 Task 9（收口）

> **修正（重读设计 §8 ¶2 后）**：设计明确把 `auth.webauthn.enabled` 与 `auth.provisioning.enabled` 列为
> **「保留为能力内配置」** 的开关，只有能力级开关改由 `capabilities.enabled` 表达。因此
> **不拆 `auth` 段**：`auth.Config` 拥有整个 `auth` 段（含嵌套 `webauthn`/`provisioning`），
> 对应 §6.1「一个能力一份 `Config()`」。装配期把子配置传出去：`passkey` 可收 `auth.Config`
> （`passkey.Requires` 含 auth，方向合法）；`tenant` 必须在 `main` 里构造自己的
> `ProvisioningConfig`（不得 import auth——auth 依赖 tenant，反向即越界，§9 门禁）。
> 原「点分键拆三段」的方案作废。

`auth` 段是唯一未下沉的段，因为它的 11 个字段被 **6 个包**消费且与内核 JWT 机制交织，拆分需先钉住两点：

- **JWT 参数归属**：`jwt_secret`/`jwt_previous_secret`/`issuer`/`access_expire_min`/`refresh_expire_day` → 归 `auth` 能力；`oauth`/`passkey`/`console` 都 `Requires auth`，导入 `auth.Config` 方向合法。唯一反向依赖是 `mfa`（`Requires user`，却用 `issuer`+`trusted_device_days`）→ 改为装配期传参（`mfa.New(db, mfa.Config{...}, users)`），不让 mfa 导入 auth 的类型。
- **`auth.webauthn` → `passkey`**（点分键，`passkey.Requires auth` ✓）、**`auth.provisioning` → `tenant`**（点分键 + tenant 自有的 `ProvisioningConfig` 类型，`tenant` 不 Requires auth，故**不得**导入 auth 类型）。
- **跨能力校验**：`provisioning.enabled` 要求 `auth.public_registration`，两者分属 tenant/auth → 移到组合根（与 outbox/queue 的处理一致）。
- **机制扩展**：`jwt_secret` 强度校验仅在 `APP_ENV=prod` 执行，而 `SectionConfig.Validate()` 不接收 env → 需给机制加可选钩子（`ValidateProd() error`，`LoadSection` 在 prod 下按类型断言调用）。

---

## 契约侧对齐（设计 §6.1）

`fc18f29` 起，配置声明改由能力契约承载（对齐设计原形）：

- `contract.ConfigSpec{Section, New}` + `contract.Descriptor.Configs`（一个能力可声明多段）。
- `app.LoadCapabilityConfigs(dec, caps, env)`：按**启用集**遍历各能力的声明段，执行
  解码 → `ApplyDefaults` → `Validate`，`env=prod` 时追加可选的 `ValidateProd`。
  段必须实现 `config.SectionConfig`，否则**立即报错**（避免校验被静默跳过）。
- `app.SectionOf[*T](cfgs, section)` 供组合根取回强类型配置。

**待做**：把 9 个段的 `Load(dec)` 换成 `Descriptor.Configs` 声明并让 `main`/`container`
从 `CapabilityConfigs` 取值；`storage`/`notification`/`retention` 无 `Descriptor`，
其段归属需在 P2.2 决定（见总纲）。
