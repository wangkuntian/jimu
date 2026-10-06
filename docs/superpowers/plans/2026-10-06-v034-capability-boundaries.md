# v0.3.4 Capability Boundaries Implementation Plan

> **执行方式：** 在当前会话 inline 按 task 顺序执行；每项完成后运行该项验证并勾选。未明确要求时不创建 commit。

**目标：** 移除 `feature` 与独立 `retention` capability，把配置和清理行为归还给对应数据所有者，并让能力清单、配置文档与报告一致。

**架构：** `feature` 从能力系统彻底删除。分批删除机制作为 `internal/shared/dbpurge` 的通用数据库工具保留，能力自身持有表规则、配置、开关和定时任务；WebAuthn、租户开通和可信设备配置分别由 `passkey`、`tenant`、`mfa` 声明。

**技术栈：** Go、GORM、Cron scheduler、YAML/Mapstructure、能力 `Descriptor`、Profiles 与 generator manifest。

**设计：** [v0.3.4-capability-boundaries-design.md](../specs/2026-10-06-v0.3.4-capability-boundaries-design.md)

## 全局约束

- 能力之间只能经 `internal/contract` 端口交互，能力代码不得跨能力 import。
- 表清理只允许由声明该表 `Owns` 的能力执行。
- 每个所有者的清理任务默认关闭，天数为 0 时跳过相应表。
- 改源码时同步更新 README 与 `docs/releases/v0.3.4.md`。
- 清单变化后运行 `make check-capabilities`、`make profiles-check` 与 `make compose-report-check`。

---

### Task 1：移除无消费者的 feature 能力

**文件：**
- 修改：`internal/capabilities/catalog/catalog.go`、`internal/capabilities/catalog/catalog_test.go`
- 修改：`internal/profiles/full/assembly.go`、`internal/profiles/full/assembly_test.go`、`internal/profiles/full/routes_golden_test.go`
- 修改：`internal/e2e/admin_routes_parity_test.go`
- 修改：`internal/capabilities/console/application/config_service.go`、`internal/capabilities/console/application/config_service_test.go`
- 删除：`internal/capabilities/feature/`
- 验证并修改：`tools/generator/frameworkmanifest/actions.go` 及 generator fixture 中描述能力闭包的字段

- [x] 从 catalog 与 full profile 删除注册，更新 catalog/profile/golden 断言。
- [x] 删除 `feature` 路由和其实现；从 e2e 管理端路由清单移除两条 `/admin/features` 路由。
- [x] 删除 console 动态配置白名单中无消费者的 `feature_flags` 项及对应测试断言。
- [x] 检查 generator 没有 `feature` 硬编码；将 `retention` 配置清理留到 Task 8 的完整移除中。
- [x] 运行聚焦验证：`go test ./internal/capabilities/catalog ./internal/capabilities/console/application ./internal/profiles/full ./internal/e2e -count=1`。

### Task 2：抽取无所有权的分批删除机制并归还 audit 清理

**文件：**
- 新建：`internal/shared/dbpurge/dbpurge.go`、`internal/shared/dbpurge/dbpurge_test.go`
- 修改：`internal/capabilities/audit/config.go`、`internal/capabilities/audit/module.go`、`internal/capabilities/audit/wire.go`
- 修改：`configs/app.yaml`、`configs/app.prod.yaml`

- [x] 将原实现中跨方言的分批硬删除机制迁入 `internal/shared/dbpurge`；工具只接收调用方提供的模型、时间列、条件、截止时间和批大小，不内置表清单或任务。
- [x] 在 audit 配置中增加默认关闭的保留配置，保留现有 `audit_log_days=180`、`batch_size=500` 与 `cron=30 3 * * *` 数值。
- [x] 只在 `audit` 清理开关开启且 DB 存在时注册 audit 自有任务；清理规则只针对 `audit_logs`。
- [x] 添加批量删除 SQL 行为测试、配置解码/校验测试与任务构造开关测试。
- [x] 运行 `go test ./internal/shared/dbpurge ./internal/capabilities/retention ./internal/capabilities/audit ./internal/profiles/full -count=1` 与 `git diff --check`。

### Task 3：归还 queue 自有历史表清理

**文件：**
- 修改：`internal/capabilities/queue/factory.go`、`internal/capabilities/queue/config.go`、`internal/capabilities/queue/migrations.go`、`internal/capabilities/queue/wire.go`
- 修改：`configs/app.yaml`、`configs/app.prod.yaml`
- 新建或更新：`internal/capabilities/queue/retention_test.go` 与 `internal/capabilities/queue/config_test.go`

- [x] 把 `job_days=7`、`job_history_days=30`、`dead_letter_days=30` 移入 `queue.retention`，保留默认关闭、批大小 500 和每日 03:30 默认调度。
- [x] 在 queue 装配中只注册 queue 自有清理任务，规则限定为终态 `jobs`、`job_history` 和已解决 `dead_letters`。
- [x] 用测试覆盖每类状态条件、0 天跳过、开关关闭不注册，以及配置校验。
- [x] 运行 `go test ./internal/capabilities/queue ./internal/profiles/full -count=1` 与 `git diff --check`。

### Task 4：归还 outbox 自有历史事件清理

**文件：**
- 修改：`internal/capabilities/outbox/config.go`、`internal/capabilities/outbox/migrations.go`、`internal/capabilities/outbox/wire.go`
- 修改：`configs/app.yaml`、`configs/app.prod.yaml`
- 新建或更新：`internal/capabilities/outbox/retention_test.go`、`internal/capabilities/outbox/config_test.go`

- [x] 把 `outbox_event_days=7` 移入 `outbox.retention`，保留默认关闭、批大小 500 和默认调度。
- [x] 由 outbox 自行注册清理任务，只删除 `published_at` 非空且早于截止时间的 `outbox_events`。
- [x] 测试未发布事件保留、过期已发布事件清理、0 天跳过和关闭时不注册任务。
- [x] 运行 `go test ./internal/capabilities/outbox ./internal/profiles/full -count=1` 与 `git diff --check`。

### Task 5：归还 dataops 自有导入记录清理

**文件：**
- 新建：`internal/capabilities/dataops/config.go`、`internal/capabilities/dataops/config_test.go`、`internal/capabilities/dataops/retention_test.go`
- 修改：`internal/capabilities/dataops/migrations.go`、`internal/capabilities/dataops/wire.go`
- 修改：`configs/app.yaml`、`configs/app.prod.yaml`

- [x] 新增 `dataops` 自有配置段及默认关闭的 `retention` 子段，采用原 `import_job_days=90`。
- [x] 只在开关开启且 DB 可用时注册 dataops 清理任务，只清理 `completed`/`failed` 的旧 `import_jobs`。
- [x] 测试活动导入任务不删除、终态任务按期限删除、0 天跳过和任务开关。
- [x] 运行 `go test ./internal/capabilities/dataops ./internal/profiles/full -count=1` 与 `git diff --check`。

### Task 6：归还 mfa 可信设备配置和清理

**文件：**
- 修改：`internal/capabilities/mfa/config.go`、`internal/capabilities/mfa/module.go`、`internal/capabilities/mfa/wire.go`
- 修改：`internal/capabilities/auth/config.go`、`internal/capabilities/auth/config_test.go`、`internal/capabilities/auth/wire.go`
- 修改：`internal/contract/ports.go` 及 auth config port 使用处
- 修改：`configs/app.yaml`、`configs/app.prod.yaml`
- 新建或更新：`internal/capabilities/mfa/config_test.go`、`internal/capabilities/mfa/retention_test.go`

- [x] 将设备有效期 `trusted_device_days=30` 从 auth 移入 mfa 配置；将失效设备清理期 `expired_device_days=7` 移入 `mfa.retention`，文档分别解释两个期限。
- [x] 在 mfa Descriptor 声明自己的配置段；由 mfa 负责可信设备失效记录清理，只操作 `trusted_devices`。
- [x] 移除 `contract.AuthConfig` 中不再属于 auth 的设备时长字段及相应赋值；JWT 配置仍由 auth/内核机制提供。
- [x] 测试设备签发期限、清理期限、0 天行为、默认关闭及配置解码。
- [x] 运行 `go test ./internal/capabilities/catalog ./internal/capabilities/mfa ./internal/capabilities/auth ./internal/contract ./internal/profiles/full -count=1` 与 `git diff --check`。

### Task 7：把 WebAuthn 与 provisioning 配置归还 passkey 和 tenant

**文件：**
- 修改：`internal/capabilities/auth/config.go`、`internal/capabilities/auth/config_test.go`、`internal/capabilities/auth/module.go`、`internal/capabilities/auth/wire.go`
- 修改：`internal/capabilities/passkey/module.go`、`internal/capabilities/passkey/wire.go`、`internal/capabilities/passkey/module_test.go`
- 修改：`internal/capabilities/tenant/config.go`、`internal/capabilities/tenant/module.go`、`internal/capabilities/tenant/wire.go`、`internal/capabilities/tenant/wire_test.go`
- 修改：`internal/capabilities/tenant/application/provisioning.go`、`internal/contract/ports.go` 及其调用处
- 修改：`configs/app.yaml`、`configs/app.prod.yaml`

- [ ] 新增 `passkey` 配置段并由 passkey Descriptor 声明；把 WebAuthn 启用开关、RP 参数和 session TTL 从 auth 段迁入 passkey。
- [ ] 新增 tenant 自有 provisioning 配置并由 tenant Descriptor 声明；把 provisioning 开关和模板从 auth 段迁入 tenant。
- [ ] `auth` 保留 `public_registration`，Wire 通过 `tenant.provisioner` 能力端口判断是否启用 provisioning；保留“provisioning 需要公开注册”的组合根校验。
- [ ] 收窄 `contract.AuthConfig`，移除 WebAuthn 与 provisioning 配置视图；passkey 从自身配置读取 WebAuthn 参数，只从 auth 消费登录收尾端口和 JWT/限流需要的认证信息。
- [ ] 更新配置默认值与生产校验，覆盖启用配置缺少必填值、未启用配置不触发校验、租户模板校验和公开注册约束。
- [ ] 运行 `go test ./internal/capabilities/auth ./internal/capabilities/passkey ./internal/capabilities/tenant ./internal/assembly -count=1`。

### Task 8：删除 retention capability 并同步清单、报告和文档

**文件：**
- 修改：`internal/profiles/*/assembly.go`、`internal/profiles/*/assembly_test.go`、`scripts/check_profiles.sh`
- 修改：`tools/generator/frameworkmanifest/actions.go`、`tools/generator/frameworkmanifest/testdata/*.json` 与相关测试
- 修改：`docs/profiles/compose-report.md`、`README.md`
- 删除：`internal/capabilities/retention/`
- 新建：`docs/releases/v0.3.4.md`

- [ ] 从所有 profile 删除 `retention` Ungated 项，删除 retention 包和配置段，移除 generator 的 retention 特判。
- [ ] 更新能力总数、profile 依赖闭包、路由 golden、配置段归属、CLI `--with` 能力列表和脚手架 manifest fixture。
- [ ] 用脚手架导出 manifest/报告的现有测试路径刷新 JSON golden，确认 `storage` 仍保留在六个 Ungated 能力中。
- [ ] 更新 README 配置表与能力目录树，新增 v0.3.4 release note 的变更与验证章节。
- [ ] 运行 `go test ./... -count=1`、`make check-capabilities`、`make profiles-check`、`make compose-report-check`、`make release-check COMPOSE_ENV=.env.example`、`git diff --check`。
