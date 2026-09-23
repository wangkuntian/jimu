# 能力可插拔 P2：三层机制 实现计划（总纲）

**Goal:** 落实设计 §6「三层机制」——让能力在**建项目时（层①）**、**编译时（层②）**、**运行时（层③）**三个时机都可组合，并把 4 项验收判据做到可测：5 个 profile 均能构建启动、`minimal` 报告数字显著低于 `full`、只用本地存储/Redis 队列/CSV 时对应重型依赖不出现、四道门禁在 CI 生效。

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md` §6（三层机制）、§6.1（能力自描述）、§3.7（驱动级可插拔）、§3.8（非代码资产）、§8（配置归属）、§9（门禁）、§10 P2/P3

> **范围澄清（本轮修正）**：P2 不只是「配置下沉」。`contract.Capability` 的声明形态（§6.1 的 `Name`/`Tags`/`Requires`/`SoftRequires`/`Owns`/`Config`/`Migrations`/`Permissions`/`Mount`）是 P2 的**核心产物**，因为层①的目录复制、层②的 profile 校验、层③的门禁都以它为唯一元数据来源。此前把它当作「P3 再说」是错的。

## 子阶段与依赖

```
P2.1 运行时配置归属（§8）              ← 已完成
P2.2 能力自描述契约（§6.1）             ← 已完成（依赖 P2.1 的 Config 建模；产出 SoftRequires/Owns，Tags 推迟）
P2.3 层③ 运行时：capabilities.enabled   ← 已完成（依赖 P2.2；软依赖降级报告 + 管理端点 /capabilities）
P2.4 层② 构建：profiles 入口包          ← 已完成（依赖 P2.2；5 个 profile + compose-report）
P2.5 层② 驱动级可插拔（§3.7）           ← 已完成（依赖 P2.4；驱动独立成包 + 两层声明 + 门禁）
P2.5b 单一入口与构建期形态参数化         ← 已完成（依赖 P2.5；唯一入口 cmd/server + 选点包 + overlay）
P2.6 层② 非代码资产模块化（§3.8）        ← 已完成（依赖 P2.5b；资产归属门禁 + 本仓条件化）
P2.7 层① 脚手架：jimu new / capability add ← 依赖 P2.2 + P2.4（生成专属 catalog 与 app.yaml）
P2.8 门禁（§9）：四道 check-*            ← 贯穿 P2.2–P2.7，最后在 CI 生效
```

每个子阶段独立 PR、保持 `full` 全绿。feature→release 用 squash merge。

## P2.1 运行时配置归属（已完成）

详见 `docs/plans/2026-09-21-config-ownership.md`（含执行记录与完成记录）。

- **已完成**：机制（`contract.ConfigSpec`/`Descriptor.Configs` + `app.LoadCapabilityConfigs`，prod 加严走可选 `ValidateProd`）；14 个段全部下沉 —— catalog 能力段 `auth`/`captcha`/`audit`/`oauth`/`queue`+`scheduler`/`outbox`/`uploadsec` 经 `Descriptor.Configs` 按启用集加载，非 catalog 包段 `storage`/`notification`（`email`+`sms`+`notification`）/`retention` 由组合根显式加载；`configs/*.yaml` 全程零改动。
- **已完成（auth 段）**：按 §8 ¶2 **不拆段** —— `auth.Config` 拥有整个 `auth` 段（含嵌套 `webauthn`/`provisioning`）。`passkey`/`oauth` 收 `auth.Config`（`Requires` 含 auth）；`tenant` 由 `main` 构造自有的 `ProvisioningConfig`（auth 依赖 tenant，反向 import 越界）；`mfa` 改装配期传参（`Requires user`，不 import auth）。`provisioning.enabled`→`public_registration` 跨字段校验留在组合根；`jwt_secret` 的 prod 加严走 `ValidateProd` 钩子（`APP_ENV=prod` 时由 `LoadCapabilityConfigs` 按类型断言调用，已有装配级回归用例覆盖）。
- **③ 裁定（B）**：`storage`/`notification`/`retention` 保持非 catalog、由组合根显式加载（与本文档 `config-ownership.md` 的裁定 2B 一致）；它们是否 catalogize 交由 **P2.2** 按 §6.1 逐一定夺。
- **④ 收口**：已补「未启用能力的配置段既不出现也不校验」的装配级回归用例（`internal/app/capconfig_test.go`，用真实 descriptor + 非法 YAML）；README 配置归属与新增能力流程、设计 §10、release note 已更新。

## P2.2 能力自描述契约（§6.1）（已完成）

执行记录见 `docs/plans/2026-09-22-p2-contract-and-runtime.md`。

- **已完成**：`contract.Descriptor` 新增 `SoftRequires` / `Owns`（`Config` 在 P2.1 建立）。`Descriptor` 是能力元数据的唯一来源：启用闭包、配置段加载、权限点种子、路由挂载与能力门禁都只读它。**不改名为 `Capability`、`Migrations` 保持 `fs.FS`**（embed 进二进制的形态不动，改名/换形态没有收益）。
- **已完成**：18 个能力逐个补声明 —— `Owns` 按 §7 表归属、逐条对齐迁移里 `CREATE TABLE` 的实际表名（13 个能力有表、5 个无表：`console`/`captcha`/`feature`/`uploadsec`/`breach`）。有表能力：`user`→`users`；`access`→`roles`/`permissions`/`role_permissions`/`user_roles`；`tenant`→`tenants`/`tenant_plans`；`audit`→`audit_logs`/`audit_chain_head`；`apikey`→`api_keys`；`queue`→`jobs`/`job_history`/`dead_letters`/`scheduled_jobs`；`dataops`→`import_jobs`；`search`→`search_documents`；`auth`→`login_histories`/`password_histories`；`mfa`→`user_mfa`/`trusted_devices`；`passkey`→`webauthn_credentials`；`oauth`→`user_oauth_bindings`；`outbox`→`outbox_events`。`SoftRequires` 取实际软依赖：`user`→`access`/`tenant`、`access`→`tenant`、`mfa`→`auth`、`auth`→`captcha`/`breach`、`apikey`→`tenant`、`outbox`→`queue`。
- **已完成**：`catalog.ValidateDeclarations()` 在 `catalog.Resolve` 顶部校验声明自洽（清单内能力名、不自引用、不与 `Requires` 重叠、不重复）；`catalog.Degraded(caps)` 计算降级项。启用闭包算法**不变**（仍只用 `Requires`），`SoftRequires` 只产出降级报告 —— 即「硬依赖闭包不变 + 软依赖降级报告」。
- **③ 裁定（推迟）**：`Tags` 在出现真实消费方之前**不加**；`storage`/`notification`/`retention`/`ws`/`grpc`/`apidocs`/`encryption` 本轮**保持非 catalog**、由组合根显式装配（能力清单仍为 18 项，`configs/*.yaml` 零改动），是否 catalogize 连同 profile 入口一起在 **P2.5/P2.6** 定夺（沿用 P2.1 裁定 2B）。
- **④ 门禁第一块**：`make check-capabilities` → `tools/checkcapabilities` 校验 `Owns` ↔ mysql 迁移「单表唯一归属、无孤儿表、无未声明建表」；PostgreSQL 迁移表名与 mysql 一致，暂以 mysql 为准；正则只认 `CREATE TABLE`，`ALTER TABLE ... ADD` 不参与归属（符合「`Owns` 只认 CREATE」）。**不接入 CI / `make ci` / `make release-check`**，完整四道门禁归 P2.8。

## P2.3 层③ 运行时（已完成）

- **已完成**：`capabilities.enabled` 与 `SoftRequires` 降级路径 —— 启用闭包只补硬依赖；软依赖缺失时本能力降级运行：启动打 `capability degraded` warn（log 字段 `name`/`missing`），管理端口 `GET /capabilities` 输出 `{"enabled":[…],"degraded":[{"capability":…,"missing":[…]}]}`。`HealthRouter` 改可变参数 `extra ...func(*http.ServeMux)`（调用点 `bootstrap.go` 与 `management_test.go`，向后兼容），使内核包不 import 能力包。
- **已完成**：未启用能力不挂路由/不注册任务事件/不启动后台组件（P0 已建）。
- **已完成**：未启用能力的配置段「既不出现也不校验」的装配级回归用例（P2.1 的 `internal/app/capconfig_test.go`）。

## P2.4 层② 构建：profiles 入口包（已完成）

- `profiles/{full,minimal,saas,enterprise,machine}` 入口包，各自能力清单 + `main`（原计划写「catalog 子集」，实际按裁定 6 落地为：catalog 仍是全量 18 项，形态清单显式列出选中能力，避免引入第二份清单）。
- `compose-report`：代码行数 / 文件数 / `go.mod` 直接依赖数 / 二进制大小 / 路由数 / 迁移数 / 表数，进 CI 归档对比。
- **验收**：5 个 profile 均能构建启动；`minimal` 报告数字显著低于 `full`。
- 层②边界要在文档写清：profile 入口**不减小 `go.mod`**，避免「以为换 profile 依赖就少了」的误解（§11）。

- **已完成（装配接缝）**：`internal/capability`（描述符解析叶子包：`Resolve`/`ValidateDeclarations`/`Degraded`，只 import `contract`）与 `internal/assembly`（`Assembly`/`Capability`/`Context`/`Run`/`ValidatePortFlow`/`ProbeAssembly`）+ 24 个能力 `wire.go` 自装配；`internal/app` 收敛为内核容器 + 生命周期：不 import `catalog`、任何能力**根包**或 `internal/assembly`（唯一例外是 `access/domain`、`tenant/domain`、`user/domain` 三个能力 **domain 叶子包**，结构性种子所需，见下方「记录偏差」），`cmd/server` 降为 `full` 的薄包装（保留 swagger 注解，Dockerfile/Makefile/compose/`swag init -g`/CI 不变）。
- **已完成（5 个形态）**：`internal/profiles/{full,minimal,saas,enterprise,machine}` 声明能力清单与结构性种子，`profiles/<name>/main.go` 只调用 `assembly.Run`；非 catalog 条目在清单里显式标 `Ungated`（不受 `capabilities.enabled` 门控）；能力清单仍是 18 项、`configs/*.yaml` 零改动。`auth.Requires` 放宽 —— `tenant`/`mfa` 降为 `SoftRequires`（行为变更，见 release note），无 `auth` 的 `machine` 由 `apikey.ProtectedHTTPMiddleware` 承担受保护路由。
- **已完成（门禁与报告）**：`make profiles-check`（构建 + **golden 依赖闭包裁剪门禁**：逐形态能力根包集合逐值锁定，`JIMU_PROFILES_SMOKE=1` 时额外启动并轮询管理端 `/readyz`）与 `make compose-report`（`tools/composereport` → `docs/profiles/compose-report.md`，不连库、不启动监听）。**实测**：二进制 full 122.6 MB / minimal 85.8 MB（−30.0%）/ saas 86.1 MB / enterprise 99.5 MB / machine 84.4 MB；路由 99 / 32 / 48 / 55 / 28；表 23 / 7 / 11 / 11 / 6；迁移 25 / 7 / 13 / 13 / 7；本仓闭包代码行 33995 / 17804 / 20450 / 23241 / 18080。五个形态的 `go.mod` 直接依赖数**完全相同**（各 64 个）—— §11「层②不减小 `go.mod`」被实测钉死。
- **已知限制（转 P2.6/P2.7）**：`machine` 可启动，但 `/api/v1/admin/apikeys` 需要它刻意排除的 JWT 链，首把 API Key 必须带外签发（CLI 归 §3.8）；迁移与结构种子仍按 catalog 全量执行，profile 驱动的迁移裁剪归 P2.6/P2.8。
- **编译期残留（留待共享类型迁移）**：`user`/`auth` 直接 import `outbox`/`queue`/`notification`（`console` import `ws`）的具体类型，因此 `minimal`/`saas` 闭包多出 `outbox`/`queue`、`machine` 多出 `notification`/`outbox`/`queue`、`enterprise` 多出 `outbox`/`queue`/`ws`；装配期一个都不构造，已由 golden 闭包门禁冻结，消除需把 `*outbox.Outbox`/`notification.Message`/`outbox.Event` 迁到 `contract`/内核。
- **记录偏差（`internal/app` 的能力 domain 叶子包）**：`internal/app` 不 import `catalog`、任何能力根包或 `internal/assembly`，但仍 import `access/domain`、`tenant/domain`、`user/domain` 三个能力 **domain 叶子包**（`internal/app/seed.go` 的结构性种子需要 `Tenant`/`Plan`/`Role`/`Permission`/`User` 等实体类型）；`go list -deps ./internal/app` 实测 jimu 侧能力项仅此三个。这是 P2.4 收尾裁定记录的偏差，本阶段不搬（属独立重构）：把这些类型移到 `contract`/内核后可消除该偏差，并进一步缩小每个形态的二进制。

## P2.5 层② 驱动级可插拔（§3.7）（已完成）

执行记录见 `docs/plans/2026-09-22-p2.5-driver-pluggability.md`。

- **已完成（机制）**：`storage/{local,s3}`、`queue/{redis,kafka,rabbitmq}`、`dataops/{csv,excel}` 各为独立驱动包，包内 `init()` 调用能力核心的 `Register`；核心只留接口 + 注册表（`New`/`Get` 查表，未注册即 **fail-closed**、不静默回退；`storage` 空 `type` 仍按 `local`，`queue.Wire` 启动即校验配置类型已编译）。
- **已完成（两层声明）**：`contract.Descriptor.Drivers` = 能力声明的**可用集**（驱动包名：storage `[local s3]`、queue `[redis kafka rabbitmq]`、dataops `[csv excel]`；`s3` 包覆盖 `s3`/`oss`/`minio` 三个配置取值）+ `assembly.Capability.Drivers` = 形态选中的**子集**（装配期强制 ⊆ 可用集），形态入口在 `internal/profiles/<name>/drivers.go` blank import 落实。
- **已完成（两道保险）**：`make check-capabilities` 由 1 项扩到 **6 项**（1 项既有：`Owns` ↔ 迁移归属；5 项驱动：可用集 ↔ 目录存在 / 核心包生产闭包零驱动且零重型依赖 / 形态选中 == 形态**生产** import 闭包（集合比较）/ 驱动归属（驱动包只被 `internal/profiles/*` import）/ 形态生产代码与入口包只 import 已声明驱动）；`make compose-report` 新增「重型依赖」列。两者**仍不接入** `make ci`/`release-check`（P2.8 收口）。
- **实测闭包计数**（当时对形态入口包逐个 `go list -deps` 计数）：`full` = aws-sdk-go-v2 67 / excelize 1 / kafka-go 49 / amqp091-go 1；`enterprise` = 0 / 0 / 0 / 0（收敛前 67 / 1 / 0 / 0，kafka/amqp 在队列驱动拆包时归零）；`minimal`/`saas`/`machine` = 0 / 0 / 0 / 0。形态→驱动矩阵（终态）：`full` = local+s3 / redis+kafka+rabbitmq / csv+excel；`enterprise` = local /（无 queue 能力）/ csv；`minimal`/`saas`/`machine` = 无。
- **行为变更（仅 `enterprise`）**：该形态下 `storage.type: s3|oss|minio` 与 xlsx 导入/导出改为 fail-closed 报错（`storage driver "s3" is not compiled into this build (compiled: local)` / `import format "xlsx" is not compiled into this build (compiled: csv)`）；`full` 行为不变；`configs/app.yaml` 默认 `storage.type: local`、`queue.type: redis`，默认路径不受影响。
- **豁免复核**：`GO-2026-6452`（excelize）豁免保留 —— 拆包不解除可达性（`govulncheck ./...` 扫整个 module，`full` 仍 import Excel 驱动），复核条件改为「excelize 发布 v2.11.1（或含 `rows.go` 负索引防护的正式版本）后移除」。
- **已知限制**：`dataops/exporter` 目前无生产消费方（`/api/v1/users/export.csv` 由 `user` 能力自带 handler 实现），`dataops/excel` 仍注册导出方向供将来端点接入；`outbox`/`queue` **核心包**的类型残留仍在轻形态闭包里（驱动已退出），消除需共享类型迁移（独立改动）。

## P2.5b 单一入口与构建期形态参数化（已完成）

执行记录见 `docs/plans/2026-09-23-p2.5b-unified-entry.md`。

- **已完成（唯一入口）**：删除 5 个 `profiles/<name>/main.go`，层②入口收敛为唯一 `cmd/server` —— `cmd/server/main.go` 只 import `internal/assembly` 与选点包 `internal/profiles/active`（提交态默认 `full`，`go build ./cmd/server`、`go test ./...`、IDE、`make swagger` 默认都是 full）。
- **已完成（构建期切换形态）**：`tools/profileoverlay`（共享实现 `tools/internal/profileoverlay`）把选点文件替换为「只选该形态」的版本，产物落在 gitignored 的 `.overlay/<profile>/`、不改工作区；`PROFILE=minimal make build-server` → `bin/jimu-server-minimal`（`full` 仍是 `bin/jimu-server`）、`docker build --build-arg PROFILE=<name>`；非法形态名非零退出、无产物。
- **已完成（registry 单点）**：形态名与清单的唯一来源收口到 `internal/profiles/registry`（`Names`/`All`/`Lookup`），只被 `tools/*` 与 `scripts/check_profiles.sh` 引用，不进 `cmd/server` 的 import 图。
- **已完成（门禁与报告）**：`make check-capabilities` 改为 **4 条汇总行**（① 能力自描述与 `Owns` ↔ 迁移归属 ② 驱动可用集/选中集/import 闭包一致 ③ 形态生产代码只 import 已声明驱动 ④ 唯一入口与选点包只 import 一个形态）；`make profiles-check` 与 `make compose-report` 的闭包口径同步改为「`./cmd/server` + 该形态 overlay」。
- **实测**：`full` 闭包 **118 个重型依赖包**、四个轻形态 **0**（`go list -deps`，与 P2.5 逐值一致）；`minimal` **84.7 MB** vs `full` **123.1 MB**；路由/迁移/表逐值不变，仅「本仓 Go 文件」+1、「本仓代码行」+25（`full`）/ +30（其余四形态）来自入口文件差异，二进制差 16 KB 量级。
- **边界与不做**：驱动参数化**仅限形态级**（驱动集合仍由各形态 `internal/profiles/<name>/drivers.go` 的 blank import 决定）；`PROFILE=enterprise DRIVERS=local` 式**形态内再选驱动**需要第二份真源清单 + 生成 `drivers.go` + 门禁改读它，**留后续阶段**。开发者可见的行为变更：`go build ./profiles/<name>` 不再可用。

## P2.6 层② 非代码资产模块化（§3.8）（已完成）

执行记录见 `docs/plans/2026-09-23-p2.6-assets-and-conditionals.md`。

- **已完成（资产归属与门禁）**：`contract.Descriptor.Assets` 声明能力的非代码资产（当前只有 `apidocs` → `["docs/openapi"]`）；内核运维/观测资产用**具名资产组** `ops`/`observability` 表达（**不新增 `obs` 能力**，catalog 仍 18 项）；归属判定 = **最长前缀匹配**（具体文件赢过目录），资产根 = `deploy/` 与 `docs/openapi/`（不含 `configs/`）。共享派生 `tools/internal/profileassets` + 查询 `go run ./tools/profileassets <profile>`（`-capabilities` 列能力名）。`make check-capabilities` 新增资产段 → **5 条汇总行**（四条断言：路径存在且在根内 / 同一路径不被两个所有者声明（归一化比较）/ 根下每个文件都有有效所有者 / 每个形态覆盖全部内核资产组）。
- **已完成（本仓条件化）**：① `make swagger`/`swagger-check` 按形态资产集跳过（不含 `apidocs` 的 minimal/saas/enterprise/machine 打印 `SKIP` 并 exit 0，`PROFILE` 非法名仍非零）；② 能力自带 CLI 命令（`internal/capabilities/<name>/cli.Commands()`，实例 `jimu apikey issue|list`，补 `machine` 形态首把 API Key 的带外签发）；③ `jimu` CLI 的 `migrate`/`adopt-capabilities`/`seed` 跟随当前形态（从 `catalog.All()` 过滤、保持拓扑序，`capabilities.enabled` 不参与，`full` 逐值不变）；④ `internal/e2e` 按形态装配（`assembly.Resolve`/`WireFor`，`Run` 复用，单一 wiring）+ `requireCapabilities` 声明依赖；⑤ 4 个非 full 形态的路由面 golden 收在 `internal/profiles/registry/routes_golden_test.go`（32/48/55/28，与 `make compose-report` 逐值吻合）+ 跨形态挂载点一致性断言 `TestShapeMountsMatchFull`，`full` 仍由自己包内的 golden 钉住。
- **实测**：`git ls-files deploy docs/openapi` 共 **48** 个文件 = `cap:apidocs` 3 / `group:observability` 22 / `group:ops` 23（零未覆盖零重叠）；5 形态路由数 **99 / 32 / 48 / 55 / 28**；e2e 跳过矩阵 **full 10/0、minimal 3/7、saas 4/6、enterprise 5/5、machine 3/7，全部 0 FAIL**。`configs/*.yaml` 逐字节不变、catalog 18 项、`go.mod` 未动。
- **行为变更（指针，详见 release note 与 README）**：① 非 full 形态的 `jimu` CLI 只迁移/播种该形态的能力（`migrate status` 表数下降；`full` 逐值不变）；② 结构种子需要 `tenant` 能力 → 不含 `tenant` 的形态（minimal/machine/enterprise）**不提供结构种子**（CLI `seed` 明确报错、启动期 `StructuralSeed` 告警跳过、服务不因此启动失败），替代路径必须**先用 full/saas 把迁移也跑一遍**；`full`/`saas` 但 `capabilities.enabled` 禁用 `tenant` 时启动期跳过、CLI `seed` 仍可用；③ 形态不含 `apidocs` 时 `make swagger`/`swagger-check` 打印 `SKIP` 并成功退出。
- **边界与不做**：**渲染**（`values.yaml`/`configs/app.yaml` 按能力裁剪、生成项目里「未选中资产不出现」、生成项目的 CLI 裁剪）留 **P2.7**；门禁**仍未接入** `make ci`/`release-check`（P2.8 收口）；迁移/种子的**全量清单**语义在 `internal/shared/testutil` 保持（测试要建全部表）。

## P2.7 层① 脚手架

- 复用 `tools/generator`：`jimu module create` 升级为 `jimu new <project> --profile=` / `--with=a,b` 与 `jimu capability add <name>`。
- 解析 `Requires` 闭包 → 只复制选中的能力目录、迁移、配置段、对应文档；生成专属 `catalog.go` 与只含选中能力配置段的 `configs/app.yaml`；生成后 `go mod tidy`；产出 `compose-report.md`。

## P2.8 门禁（§9）

| 门禁 | 检查内容 |
|---|---|
| `check-capabilities` | `capability.go` 声明与实际一致：`Owns` 的表只出现在所有者迁移里、`Requires`/`SoftRequires` 与实际 import 一致、无未声明的跨能力 import、无 `capabilities/A → capabilities/B/internal` 越界、无 `kernel → capabilities` 反向依赖、中间件归属正确 |
| `check-profiles` | 各 profile 的 catalog 子集与入口包一致、能构建 |
| `check-pluggable` | 驱动集合与 profile 实际 import 一致（§3.7 的静态保险） |
| `bench-ci` / 既有门禁 | 保持全绿 |

## 风险（§11 摘录）

- **`adopt-capabilities` 是存量实例的唯一门槛**：迁移搬迁必须先把版本基线登记做对。
- **`tenant_id`/`version` 列保留**：`tenancy` 关闭时列仍在。
- **驱动拆分把「编译期可见」换成「显式 import」**：需两道保险（见 P2.5）。
- **`minimal` 的「省」要防止被内核吃掉**：报告进 CI 归档对比是最直接的刹车。
