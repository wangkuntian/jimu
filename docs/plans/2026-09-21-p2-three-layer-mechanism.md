# 能力可插拔 P2：三层机制 实现计划（总纲）

**Goal:** 落实设计 §6「三层机制」——让能力在**建项目时（层①）**、**编译时（层②）**、**运行时（层③）**三个时机都可组合，并把 4 项验收判据做到可测：5 个 profile 均能构建启动、`minimal` 报告数字显著低于 `full`、只用本地存储/Redis 队列/CSV 时对应重型依赖不出现、四道门禁在 CI 生效。

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md` §6（三层机制）、§6.1（能力自描述）、§3.7（驱动级可插拔）、§3.8（非代码资产）、§8（配置归属）、§9（门禁）、§10 P2/P3

> **范围澄清（本轮修正）**：P2 不只是「配置下沉」。`contract.Capability` 的声明形态（§6.1 的 `Name`/`Tags`/`Requires`/`SoftRequires`/`Owns`/`Config`/`Migrations`/`Permissions`/`Mount`）是 P2 的**核心产物**，因为层①的目录复制、层②的 profile 校验、层③的门禁都以它为唯一元数据来源。此前把它当作「P3 再说」是错的。

## 子阶段与依赖

```
P2.1 运行时配置归属（§8）              ← 已完成 8/9 段（auth 段待做）
P2.2 能力自描述契约（§6.1）             ← 依赖 P2.1 的 Config 建模；产出 Tags/SoftRequires/Owns
P2.3 层③ 运行时：capabilities.enabled   ← 依赖 P2.2（启用闭包改用契约的 Requires/SoftRequires）
P2.4 层② 构建：profiles 入口包          ← 依赖 P2.2；5 个 profile + compose-report
P2.5 层② 驱动级可插拔（§3.7）           ← 依赖 P2.4（profile 决定 import 哪些驱动）
P2.6 层② 非代码资产模块化（§3.8）        ← deploy/Helm/CLI/契约测试随 profile 裁剪
P2.7 层① 脚手架：jimu new / capability add ← 依赖 P2.2 + P2.4（生成专属 catalog 与 app.yaml）
P2.8 门禁（§9）：四道 check-*            ← 贯穿 P2.2–P2.7，最后在 CI 生效
```

每个子阶段独立 PR、保持 `full` 全绿。feature→release 用 squash merge。

## P2.1 运行时配置归属（进行中）

详见 `docs/plans/2026-09-21-config-ownership.md`（含执行记录）。

- **已完成**：机制（`config.LoadWithSections` + `LoadSection`）、`captcha`、`audit`、`storage`+`upload`、`queue`+`outbox`+`scheduler`、`email`+`sms`+`notification`、`oauth`、`retention`；`configs/*.yaml` 全程零改动。
- **已完成（契约侧）**：`contract.ConfigSpec{Section, New}` + `Descriptor.Configs` + 框架加载器 `app.LoadCapabilityConfigs`（按启用集解码→默认值→校验，prod 加严走可选 `ProdValidate` 钩子），已带测试。
- **待做**：
  1. **`auth` 段**：按 §8 ¶2 **不拆段** —— `auth.Config` 拥有整个 `auth` 段（含嵌套 `webauthn`/`provisioning`），对应 §6.1「一个能力一份 Config」。`passkey` 可收 `auth.Config`（`Requires` 含 auth）；`tenant` 由 `main` 构造自己的 `ProvisioningConfig`（auth 依赖 tenant，反向 import 越界）。`mfa` 改为装配期传参（`Requires user`，不得 import auth）。两条跨能力校验（`provisioning.enabled`→`auth.public_registration`、`jwt_secret` 的 prod 加严）留组合根 / 机制钩子。
  2. **把 9 个段的 `Load(dec)` 换成 `Descriptor.Configs` 声明**，`main`/`container` 改从 `CapabilityConfigs` 取（`app.SectionOf[*T]`）。
  3. **非 catalog 包的配置段归属**：`storage`/`notification`/`retention` 目前**没有 Descriptor**，无法被框架按启用集遍历。需与 §6.1 一起定：给它们 `Descriptor`（成为 catalog 可见、可被 `capabilities.enabled` 开关）还是继续由组合根显式加载。**这是 P2.2 的第一个决定点。**

## P2.2 能力自描述契约（§6.1）

- `contract.Descriptor` 补齐 `Tags` / `SoftRequires` / `Owns`（`Config` 已在 P2.1 建立），并决定是否改名为设计所称的 `Capability`（`Migrations` 是否从 `fs.FS` 回到路径形态：P1.5 选 `fs.FS` 是为 embed 进二进制，改动需评估）。
- 18 个能力逐个补声明（`Owns` 取自 §7 表归属，`SoftRequires` 取实际软依赖：`auth`↔`captcha`/`breach` 等）。
- `catalog` 的启用闭包算法从「只用 Requires」扩展为「Requires + SoftRequires 降级」。
- 非 catalog 包（`storage`/`notification`/`retention`/`uploadsec` 之外的 `ws`/`grpc`/`apidocs`/`encryption`）是否catalogize：按 §6.1「catalog 是全仓唯一列出能力的地方」逐一定夺。

## P2.3 层③ 运行时

- `capabilities.enabled` 与 `SoftRequires` 降级路径逐条测试（`auth` 缺 `captcha`/`breach`、`notify` 缺真实渠道等）。
- 未启用能力不挂路由/不注册任务事件/不启动后台组件（P0 已建，补 SoftRequires 分支）。
- 配置合并与校验：P2.1 完成后，未启用能力的配置段「既不出现也不校验」需补一条装配级回归用例。

## P2.4 层② 构建：profiles 入口包

- `profiles/{full,minimal,saas,enterprise,machine}` 入口包，各自 `catalog` 子集 + `main`。
- `compose-report`：代码行数 / 文件数 / `go.mod` 直接依赖数 / 二进制大小 / 路由数 / 迁移数 / 表数，进 CI 归档对比。
- **验收**：5 个 profile 均能构建启动；`minimal` 报告数字显著低于 `full`。
- 层②边界要在文档写清：profile 入口**不减小 `go.mod`**，避免「以为换 profile 依赖就少了」的误解（§11）。

## P2.5 层② 驱动级可插拔（§3.7）

- `storage/{local,s3}`、`queue/{redis,kafka,rabbitmq}`、`dataops/{csv,excel}` 拆驱动包。
- 注册机制从同包 `switch` 改为「显式 import + 注册」；漏 import 会在**运行时**报「未知驱动」而非编译失败 → 两道保险：`check-capabilities` 静态校验「能力声明的驱动集合 = profile 实际 import 的驱动包」，启动时校验配置里的驱动已在编译期注册并给出明确错误。
- **验收**：只用本地存储/Redis 队列/CSV 时，对应重型依赖不出现（`go.mod`/二进制报告可证）。

## P2.6 层② 非代码资产模块化（§3.8）

- deploy 资产、Helm values、CLI 子命令、契约测试随 profile 裁剪。

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
