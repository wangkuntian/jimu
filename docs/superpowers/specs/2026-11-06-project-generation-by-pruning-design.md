# 按裁剪生成项目：统一 minimal / auth / full / 组合的设计

> 状态：已确认（2026-11-06）
> 日期：2026-11-06
> 范围：`jimu new` 的生成模型、`minimal`/`auth`/`full` 预设化、`basic-api`/`auth-api` 退役

## 1. 背景与意图

Jimu 的交付物是**代码**，不是二进制。用户执行 `jimu new` 得到的是一棵可以独立
`go build` / `go test` / `go run` 的 Go 工程，之后这棵代码归用户所有。

当前仓库有三套生成路径：

| 路径 | 实现 | 能力集 | support |
|---|---|---|---|
| `--template=basic-api` | 手写渲染（`frameworkmanifest/basic.go`） | 0 | certified |
| `--template=auth-api` | 手写渲染（`frameworkmanifest/auth.go`） | user/access/auth | certified |
| `--with=<caps>` | 主仓库复制 + 计划裁剪（`layout.go` + `plan`） | 任意子集 | experimental |

三套路径意味着三套事实源、三套测试、三套升级行为。`basic-api` 和 `auth-api` 的产物
结构与主仓库不一致：它们有自己的 `internal/` 布局、自己的迁移目录、自己的 Makefile
和 Compose，不遵循 `--with` 的裁剪规范。

**目标形态**：只保留一套生成引擎——**从当前主仓库裁剪**。`minimal` / `auth` / `full`
是这套引擎的三个预设，`--with` 是专家入口。

```bash
jimu new app --template=minimal   # 0 能力，最小内核
jimu new app --template=full      # 当前全部能力
jimu new app --with=user,access,queue:kafka   # 专家组合
```

三者产出同一种目录结构，只是能力集不同。

## 2. 已确认的决策

以下由项目所有者确认，不再讨论：

1. **`full` 指当前 23 个能力的全集代码**，不含 `jimu-web`，不是全栈模板。
2. **选择 B**：`minimal` = 0 能力 + 闭包裁剪后的最小内核，不是"0 能力 + 完整内核"。
3. **`basic-api` 和 `auth-api` 立即退役**，不保留 deprecated 过渡版本。
4. 生成源是**当前主代码仓库**，不是 `tools/generator` 里的独立模板树。
5. 已在旧模板上生成的项目，`jimu upgrade` **硬失败**（第 6.2 节方案 A）。
6. **保留 `auth`** 作为第三个 certified 预设（`minimal` / `auth` / `full`）。
7. `full` 首次发布标 **`validated`**，`minimal` 和 `auth` 标 `certified`。

## 3. Spike 结论（已验证）

在分支 `spike/minimal-kernel-decoupling` 上做了丢弃式实验，事实如下。

### 3.1 内核几乎没有能力耦合

`internal/kernel`、`internal/shared`、`internal/config`、`internal/contract` 的生产代码
**完全不 import** `internal/capabilities`。

真正的耦合只有一处：

```
internal/app/seed.go
  → jimu/internal/capabilities/access/domain
  → jimu/internal/capabilities/tenant/domain
  → jimu/internal/capabilities/user/domain
```

这三处只用到 5 个**纯数据模型**：`Permission`、`Role`、`Tenant`、`Plan`、`User`。
它们没有业务逻辑，只有 GORM tag 和 `TableName()`。

### 3.2 替换后全仓可编译

实验把这三个 import 换成本地定义的最小模型，结果：

- `go build ./internal/app/` 通过
- `go build ./...` 通过
- `go list -deps ./internal/app` 不再包含任何 `internal/capabilities` 包

### 3.3 `cmd/cli` 和 `catalog` 不是耦合

之前怀疑 `cmd/cli` 也依赖能力，实测**不成立**：生成器已经按选择重写 `cmd/cli/main.go`，
只注册选中能力自带命令；`internal/capabilities/catalog` 也是按选择生成的专案产物。
两者都跟随选择，不是内核依赖。

### 3.4 结论

`minimal` 的阻塞点只有一个：**`seed.go` 的模型下沉**。不是"重写生成器"。

## 4. 目标模型

### 4.1 三段式生成

```text
选择层
  输入：--template=minimal | --template=full | --with=<caps>[:<drv>]
  产物：能力集 + 驱动集 + 预设预算
       ↓
物化层
  以主仓库为源，复制 kernel/shared/app/contract/config/assembly
  按能力复制能力目录（含迁移携带、domain-only 依赖）
  重写：go.mod、cmd/cli、profiles/active、catalog
       ↓
校验层
  go build ./... + go test ./...（暂存目录内）
  失败整体回滚，不写目标目录
```

### 4.2 裁剪规则

每个文件归入以下之一，归属由**声明**决定，由**闭包**执行：

| 类别 | 归属来源 | 裁剪依据 |
|---|---|---|
| 能力 Go 代码 | `Descriptor` + 目录 | 只复制 `Copy` 集（声明 ∪ 编译闭包 ∪ 迁移携带 ∪ domain-only） |
| 迁移 SQL | `Descriptor.Migrations` | 只复制选中能力及其 `MigrationSchemaDeps` |
| 配置段 | `Descriptor.Configs` | 只保留选中能力的段 |
| 非代码资产 | `Descriptor.Assets` | 只复制有主且被选中的资产 |
| 路由/权限点 | `Descriptor` | 由生成的 `catalog` 驱动 |
| 内核包 | 无声明 | **闭包裁剪**：从 `cmd/server` + `cmd/cli` 出发，只保留可达包 |

内核闭包裁剪是本设计相对现状的**唯一新增机制**。当前 `kernelDirs` 整包复制，
导致 0 能力项目仍携带 `internal/e2e`、未使用的 kernel 子包、完整 `conf`。

### 4.3 `minimal` 的定义

```text
minimal = 0 能力
        + seed.go 已解耦（不 import 任何 capabilities）
        + kernel 闭包裁剪（只保留 HTTP 骨架可达的包）
        + 无数据库、无 Redis、无迁移
```

它替代 `basic-api` 的定位：一个能 `go build`、`go run`、`/livez` `/readyz`
都能用的最小 HTTP 服务（`/ping` 的处理见 4.3.1 ②）。

实测补充：`--with` 生成路径**本来就不产出 `docker-compose.yml`**（Compose 是
`auth-api` 手写模板独有的资产，见 6.4）。因此"无 Compose"不是 `minimal` 要新增的裁剪，
而是 `auth`/`full` 需要决定是否补上。

### 4.3.1 0 能力下必须处理的两个运行时事实

Spike 只验证了"能编译"，没验证"能启动"。逐行核对后有两个真实阻塞点：

**① `StructuralSeed` 在 nil DB 上会 panic。**

`internal/profiles/profiles.go` 的 `StructuralSeed` 无条件调用
`app.WarnIfNoAccounts(ctx.DB(), ctx.Logger())`。0 能力时 `requiredRuntimeServices` 为空，
`NewContainer` 不建 `dbConn`，`ctx.DB()` 是 `nil *gorm.DB`。

已实测确认（临时探针 `TestProbeNilDBSeed`，跑完已删除）：
设置 `ADMIN_PASSWORD` 后调用 `app.RunSeedWithCasbin(nil, nil)` 会
`panic: runtime error: invalid memory address or nil pointer dereference`。
原因：`RunSeed` 只检查密码，不检查 `db`，随后直接 `db.Transaction(...)`。

不设 `ADMIN_PASSWORD` 时不会 panic（`StructuralSeed` 提前返回），
但这条分支同样没有显式处理 nil DB —— 它只是恰好绕过了。

实现要求（二选一，推荐前者）：

- `StructuralSeed` 在 `ctx.DB() == nil` 时直接 `Warnw("structural seed skipped",
  "missing", "database")` 并返回 nil；
- 或让 `RunSeed`/`RunSeedWithCasbin` 对 nil DB 返回明确错误而不是 panic。

三个预设都要覆盖这个用例（重点是无 DB 的 `minimal`）：不设 `ADMIN_PASSWORD`、
设 `ADMIN_PASSWORD` 两种情况都必须能启动。

**② `/ping` 不存在。**

第 4.3 节和原 `basic-api` 文档里写的 `GET /ping` 在框架里没有对应路由：
`internal/kernel/observability/health.go` 只注册 `/livez` 和 `/readyz`，
`/ping` 只在 `internal/capabilities/grpc/ping.go`（gRPC，且只在 `grpc` 能力选中时存在）和
`basic-api` 模板自己的 `internal/api/v1/ping.go.tmpl` 里。

现状下 `minimal` 的健康检查只有：

```bash
curl -fsS http://127.0.0.1:19090/livez
curl -fsS http://127.0.0.1:19090/readyz
```

**已决定：加 `GET /api/v1/ping`**，返回 `{"message":"pong"}`。
上面两条健康检查仍然保留，`/api/v1/ping` 是额外的业务入口冒烟。

落点必须是**内核**，不是模板：

- 在 `internal/kernel/http/server.go` 的 `SetupRouter` 里注册 `r.GET("/api/v1/ping", ...)`，
  复用 `internal/shared/response`。所有预设（含 `minimal`）和框架自身的 full 形态
  都会自动获得这条路由。
- 不放进 `internal/capabilities/*`：它不是能力，没有 `Descriptor`，进能力清单会污染
  `capabilities.enabled` 语义和 23 项计数。
- 不放进生成器的模板动作：那会重现"每个预设各带一份私有业务代码"的旧问题，
  而且 `minimal` 会变成"0 能力但 1 条业务路由"。

代价必须写明：这是一条**公开、无认证**的路由，会出现在所有现有部署里（不只是生成项目）。
它只回常量 `pong`，不读 DB/Redis，不泄漏任何配置。

路由计数口径随之固定：`routeCount`（`tools/generator/frameworkmanifest/export.go`）
只统计**能力的 Module 注册的**路由，内核路由不计入 `report.routes`。
因此 `minimal` 的 `business_routes` 预算是 `0`，而 smoke 仍然断言
`/api/v1/ping` 返回 200。这个"内核路由不计入能力路由预算"的约定要写进
`docs/templates/v1/README.md`，否则下一个人会以为预算算错了。

### 4.4 `full` 的定义

```text
full = 全部 catalog 能力 + 全部 Ungated 能力
     + 框架 full 形态声明的完整驱动集（见 7.2）
     + 完整迁移、配置、资产
```

它替代 `--with=<23 个能力手敲>`，是一个**有名字、有预算、有门禁**的预设。

`full` 不包含 `jimu-web`，不生成全栈项目。

## 5. `seed.go` 解耦方案

### 5.1 问题

`seed.go` 需要写五张表（`permissions`、`roles`、`tenants`、`tenant_plans`、`users`）。
当前直接 import 能力 domain 模型。0 能力项目没有这些包，因此无法编译。

### 5.2 方案

新增 `internal/kernel/seedmodel`，定义 seed 所需的**最小持久化模型**：

```
internal/kernel/seedmodel/
  permission.go   Permission
  role.go         Role
  tenant.go       Tenant
  plan.go         Plan
  user.go         User
```

`seed.go` 只 import `seedmodel`。能力 domain 保持原有定义不变（它们有自己的仓储接口
和方法，不能简单别名）。

### 5.3 漂移防护

两份模型定义会漂移。用**结构断言测试**兜住：

```
TestSeedModelMatchesCapabilityDomains
  对每对 (seedmodel.X, domain.X)：
    比较字段名、类型、GORM tag、TableName()
    任何不一致 → 测试失败，并指出需要同步的文件
```

这个测试必须在主仓库常驻，且纳入 `make check`。

### 5.4 为什么不放到 `contract`

`internal/contract` 是能力之间**端口**的契约层，不应该持有业务表的持久化模型。
`kernel/seedmodel` 语义明确：内核 bootstrap 所需的最小模型。

### 5.5 已知取舍

两份定义是事实上的重复。替代方案是把模型下沉到 `kernel` 并让能力 domain 别名引用，
但那会改变能力包的所有权边界，且需要同时改仓储实现。seed 只有 5 个模型、一次写入，
重复成本可控，因此选择重复 + 结构断言。

## 6. 模板退役策略

### 6.1 立即退役（已确认）

- `jimu new --template=basic-api` → 报错，提示改用 `--template=minimal`
- `jimu new --template=auth-api` → 报错，提示改用 `--with=user,access,auth` 或 `auth` 预设
- 删除 `frameworkmanifest/basic.go`、`frameworkmanifest/auth.go` 及对应模板树
- 删除 `docs/templates/v1/budgets.json` 中的两个条目，改为 `minimal`/`full`

### 6.2 后果（必须接受）

已经用 `basic-api` / `auth-api` 生成的项目，manifest 里记录的是
`selection.template = "basic-api" | "auth-api"`。`jimu upgrade` 依靠这个字段重建目标树
（见 `tools/generator/upgrade.go` 的 `upgradeRequest`）。

模板立即退役后，这些项目**无法再升级**。

三种处理方式：

| 方案 | 行为 | 代价 |
|---|---|---|
| A. 硬失败 | `jimu upgrade` 报错并说明模板已退役 | 用户停留在旧版本，安全修复拿不到 |
| B. 只读重建 | 保留冻结的 v0.9 生成逻辑，仅供 upgrade | 退役不彻底，要长期维护死代码 |
| C. 迁移命令 | 提供 `jimu migrate-template` 一次性切到新布局 | 布局差异大，实际是重新生成 + 人工合并 |

**已确认：A（硬失败）。** 理由：这两个模板没有外部用户（仓库 3 stars、0 forks、0 issues），
冻结死代码的维护成本高于用户价值。

实现要求：

- `upgradeRequest` 在 `selection.template` 为 `basic-api`/`auth-api` 时返回明确错误，
  不进入 staging、不读源树、不写任何文件。
- 错误信息必须包含三件事：旧模板名、退役版本（v0.10.0）、迁移路径。
  例如：`project was generated with retired template "basic-api"; it cannot be upgraded.
  Migrate manually: jimu new <dir> --template=minimal (or --with=<caps>) and merge your
  application code.`  `auth-api` 对应 `--template=auth`。
- 这条检查放在 `upgradeRequest` 最前面，先于模块解析，保证失败不依赖 `go.mod` 可读。

### 6.3 升级 fixture 的影响

`tools/generator/testdata/upgrade/v0.5/{basic-api,auth-api}` 是 v0.5 的固定快照，
用于证明升级事务正确。模板退役后：

- 这些 fixture 不再参与"升级到当前版本"的测试（目标模板已不存在）
- 方案 A 已确认，因此 `upgradeFixtureCases` 需要换成新模板的等价覆盖：
  用 `--template=auth`（能力集与旧 `auth-api` 接近）重建一个 v0.10.0 fixture，
  保留 untouched / user-modified / rollback 三类场景的事务验证
- 旧 fixture 树随 `basic.go`/`auth.go` 一起删除，不留只读副本
- 不能简单删掉 `TestUpgradeFixtures`：它是升级事务（备份、逐文件安装、失败回滚）的唯一门禁

### 6.4 Compose 资产会随 `auth-api` 一起消失

实测：用 `--with=user,access,auth --no-tidy` 生成的项目里**没有 `docker-compose.yml`**，
只有 `deploy/helm`、`deploy/k8s`、`deploy/openobserve`。Compose 文件仅存在于
`tools/generator/render/templates/authapi/docker-compose.yml.tmpl`，是旧模板的私有资产。

后果：旧 `auth-api` 用户习惯的"`make up` 一条命令起 MySQL + Redis + 应用 + 迁移"在新模型里
不存在。新 `auth`/`full` 预设要保留这个体验，必须新增一个**从主仓库资产渲染**的 Compose
（或明确放弃，只保留 Helm/K8s）。

框架仓根的 `docker-compose.yml` 是开发用全栈编排（含 `jimu-web`、observability），
不能直接复制给生成项目。需要一份只含选中能力 runtime services 的最小 Compose。

**我的决定：阶段 4 前不做 Compose 生成。** 理由：没有用户在用，且 Compose 的 service
集合必须从 `Descriptor.RuntimeServices` + 驱动派生，这是一块独立的生成逻辑（约等于
第二个资产管线）。先在 release note 里写明"新预设不提供 Compose，用 `deploy/k8s`
或自己的编排"，把它列为 v0.11 的候选。相应地：

- 第 8 节里 `auth`/`full` 的"Compose 启动 + `/readyz`"门禁改为
  "外部 MySQL/Redis（CI service containers）+ `/readyz`"
- `TestAuthAPIComposeConfigIsValid`、`TestAuthAPIComposeUsesHostUIDForSecrets`
  随模板一起删除，不迁移

## 7. 预设定义

**已确认：保留三个预设。**

| 预设 | 能力集 | 驱动 | 替代 |
|---|---|---|---|
| `minimal` | 空 | 无 | `basic-api` |
| `auth` | user/access/auth（+ 闭包与 Ungated） | 无 | `auth-api` |
| `full` | 全部 23 项 | 见 7.2 | `--with=<23 项>` |

### 7.1 预设名 = 生成形态名

`--template=<preset>` 在生成项目里写入 `internal/profiles/<preset>/`，
manifest 的 `selection.template` 与 `selection.shape` 都记录预设名。

注意：框架仓自己的 `internal/profiles/minimal` 是**认证形态**（user/access/auth），
与新 `--template=minimal`（0 能力）同名不同义。两者不在同一棵树里（生成器只复制
`internal/profiles/profiles.go`，不复制形态目录），不会冲突，但文档和日志里必须
始终写清是"生成预设"还是"框架内部形态"。这与旧 `--profile=minimal` 的命名冲突是
同一类问题，退役说明里要显式提示：`--profile=minimal` 的替代从 `auth-api` 改为
`--template=auth`。

### 7.2 `full` 的驱动集必须与框架 `full` 形态逐值一致

`--with` 的驱动缺省取每个能力的**首项**，与框架 `internal/profiles/full` 的声明不同：

| 能力 | `--with` 缺省 | 框架 full |
|---|---|---|
| storage | `local` | `local`, `s3` |
| queue | `redis` | `redis`, `kafka`, `rabbitmq` |
| dataops | `csv` | `csv`, `excel` |

因此 `full` 预设**不能**实现为"`--with` 传 23 个名字"。否则 `full` 产物会静默丢掉
s3/kafka/rabbitmq/excel 驱动代码，与框架 full 二进制不等价，也就不能声称"替代 full"。

实现要求：`full` 预设的驱动集从框架 `internal/profiles/full/assembly.go` 派生
（或在其旁边维护一份逐值相同的常量，加测试断言一致），不经过 `--with` 的缺省推断。

### 7.3 预设项目必须能继续加能力

现有实现禁止对模板项目做 `jimu capability add`：

```go
// tools/generator/frameworkmanifest/export.go
if previous.Selection.Template != "" {
    return fmt.Errorf("capability add does not extend official template projects (%s); ...")
}
```

这条规则是旧模型（手写模板 = 认证基线，不可扩展）的产物。新模型下预设只是
**能力集的起点**，用户明确要求"`minimal` 是最基础，同时又能组合能力"。
如果保留这条检查，`--template=minimal` 生成的项目永远无法加第二个能力，
新模型就退化成两个不能用 `jimu capability add` 的死模板。

**实现要求**：把"模板项目不可扩展"的判据从 `selection.template != ""` 改为
`selection.template ∈ {retired templates}`，或者直接删除该分支。
manifest 里 `selection.template` 保留（`upgrade` 需要它），但不再决定可扩展性。

支持级别随扩展降级：预设计划由 certified/validated 变为 `experimental`
（与 `--with` 一致），因为组合空间不再受预设门禁覆盖。`capability add` 后的
`support` 写入规则要显式定义，并加测试：

- `minimal` 加一个能力后：`support = experimental`
- `full` 加能力不改变能力集（已全选），应报"已存在"

这条不在阶段 2 落地就会直接违背需求，因此归入阶段 2，不推迟。

## 8. 支持级别与门禁

| 预设 | support | 门禁 |
|---|---|---|
| `minimal` | certified | 预算、生成即 build/test、`/livez` `/readyz` smoke、升级 fixture |
| `auth` | certified | 预算、生成即 build/test、外部 MySQL/Redis + `/readyz`、升级 fixture |
| `full` | **validated** | 预算、生成即 build/test、外部 MySQL/Redis + `/readyz`、升级 fixture |
| `--with` | experimental | 单能力 build/vet、固定 pairwise 矩阵 |

**已确认：`full` 首发标 `validated`。** certified 意味着承诺升级和兼容窗口；
观察一个版本后再决定是否提升。

支持级别与 `docs/templates/v1/matrix.yaml` 的 `support_levels` 存在命名差：
`matrix.yaml` 里是 `official`/`experimental`/`internal`，manifest 里是
`certified`/`validated`/`experimental`。两者当前靠 `official → certified` 的隐式映射对齐。
新增 `validated` 后必须显式补上映射，否则 `jimu check` 会把 `full` 报成 `official`。
实现时要同时改 `matrix.yaml` 的 `support_levels`（新增 `validated` 条目）和
`tools/generator/support` 的映射逻辑，并加一个测试钉住三个预设的级别。

## 8.1 六个已知问题的落地方案

这一节回答"每个问题具体怎么改、改哪里、怎么验收"。编号与问题清单一致。

### 8.1.1 nil DB 下的 seed panic

**改动点**：`internal/profiles/profiles.go` 的 `StructuralSeed`，在函数最前面加：

```go
if ctx.DB() == nil {
    ctx.Logger().Warnw("structural seed skipped", "missing", "database")
    return nil
}
```

**为什么放在这一层而不是 `RunSeed`**：`RunSeed` 的调用方还有 `cmd/cli/main.go:195`
（`jimu seed` 子命令）。CLI 在无 DB 配置时应报错，不应静默返回 nil；
放在 `StructuralSeed` 只影响"启动时自动播种"，语义精确。

**测试**（新增到 `internal/profiles/profiles_test.go`，沿用现有 `assembly.WireFor` 构造
空 `Container{}` 的模式）：

| 用例 | 设置 | 断言 |
|---|---|---|
| 无 DB、无密码 | `DB: nil`，清空 `ADMIN_PASSWORD*` | 返回 nil，日志含 `structural seed skipped` |
| 无 DB、有密码 | `DB: nil`，设 `ADMIN_PASSWORD` | **返回 nil 且不 panic**（这是回归点） |

第二个用例在改之前会 panic，改之后通过 —— 是我实测过的判据。

**不做**：不给 `RunSeed` 加 nil 检查。CLI 路径需要显式错误，加 nil 检查会让
`jimu seed` 在配置缺失时报告一个更含糊的错误。

### 8.1.2 `/api/v1/ping`

见 4.3.1 ②。改动点：`internal/kernel/http/server.go` 的 `SetupRouter`，
在 debug 模式根路径块**之外**注册（不随 `http.mode` 消失）：

```go
r.GET("/api/v1/ping", func(c *gin.Context) {
    response.OK(c, http.StatusOK, gin.H{"message": "pong"})
})
```

**测试**：

- `internal/kernel/http/server_test.go`：新建 router，断言
  `GET /api/v1/ping` 返回 200 且 body 含 `pong`；`release` 模式下同样存在。
- `internal/e2e/auth_routes_parity_test.go`：`/api/v1/ping` 不在 `want` 列表里，
  它的计数循环只扫 `/api/v1/auth` 前缀，不受影响（已核对）。
- 生成项目侧：`minimal` 的 smoke 增加
  `curl -fsS http://127.0.0.1:18080/api/v1/ping`。

**回滚边界**：这是框架级行为变更，所有现有部署都会多一条公开路由，要进
release note 的 "行为变更" 段。

### 8.1.3 `full` 的完整驱动集

**改动点**：新增 `presetDrivers(name string) map[string][]string`，放在
`tools/generator/frameworkmanifest/presets.go`（新文件），`full` 分支直接从
`full.Assembly().Capabilities` 逐项取 `Drivers`：

```go
func fullDrivers() map[string][]string {
    out := map[string][]string{}
    for _, item := range full.Assembly().Capabilities {
        if len(item.Drivers) > 0 {
            out[item.Descriptor.Name] = slices.Clone(item.Drivers)
        }
    }
    return out
}
```

`frameworkmanifest` 已经 import `full`（`profiles.go` 的 `allDescriptors` 用到），
不新增边界违规（`isGeneratorCorePath` 只约束 `frameworkmanifest/` 之外的文件）。

**测试**：`TestFullPresetDriversMatchFrameworkFull`，
断言 `fullDrivers()` 与 `full.Assembly()` 的 `name → Drivers` 逐值相等。
因为两边同源，这个测试的价值在于**防止未来有人把预设改成常量表**：
一旦有人硬编码，测试才会真正开始比较两份事实。

**验收**：`jimu new --template=full` 后 `grep -r "kafka\|rabbitmq\|excel\|s3"`
在 `internal/capabilities/{queue,dataops,storage}` 下均命中。

### 8.1.4 预设项目可以继续 `capability add`

**改动点**：`tools/generator/frameworkmanifest/export.go:149` 的
`if previous.Selection.Template != ""` 分支。改为只拒绝已退役的模板名：

```go
switch previous.Selection.Template {
case "basic-api", "auth-api":
    return manifest.Document{}, fmt.Errorf(
        "template %q is retired and cannot be extended; regenerate with "+
            "--template=minimal|auth|full or --with", previous.Selection.Template)
}
```

**关键：`ExportForAdd` 之后的重建必须保留预设身份。**
当前代码把 `With` 拼成 `strings.Join(requested, ",")` 再调 `Export`，
这会让 `minimal` 加一个能力后 `selection.template` 变成空、`support` 变成
`experimental`（这是**期望的**），但 `selection.shape` 会保留、`template` 丢失。
如果 `template` 丢失，`jimu upgrade` 会走 `composition.requested` 分支 —— 这正是
我们要的行为（组合项目按组合升级）。因此**不需要**改重建逻辑，但要加测试钉住：

| 场景 | 断言 |
|---|---|
| `minimal` + `capability add tenant` | `composition.requested` 含 tenant；`selection.template == ""`；`support == experimental` |
| `auth` + `capability add mfa` | 同上，且装配集含 mfa |
| `full` + `capability add tenant` | 报 "already present as a declared capability" |
| 已退役模板 manifest 调 add | 报 `template "..." is retired` |

**这条是需求的硬要求**：没有它，"`minimal` 最基础 + 组合能力"不成立。

### 8.1.5 Compose 资产

**决定**：阶段 4 不生成 Compose，门禁改用外部 MySQL/Redis。

**改动点**：

- 删除 `tools/generator/render/templates/authapi/docker-compose.yml.tmpl`
- 删除 `TestAuthAPIComposeConfigIsValid`、`TestAuthAPIComposeUsesHostUIDForSecrets`
- `TestAuthAPIEmptyDatabaseEndToEnd` 改写为 `TestAuthPresetEmptyDatabaseEndToEnd`：
  用 `--template=auth` 生成，MySQL 用 `docker run`（或 CI service container）起，
  跑 `make migrate && make seed`，再 `go run ./cmd/server` 验 `/readyz`
- `docs/production/auth-api.md` 的 `make up` 步骤改为直接给
  `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME/REDIS_ADDR` 环境变量

**为什么要写进 release note**：旧 `auth-api` 的 `make up` 是它最主要的卖点之一。
静默消失会让任何存量使用者直接卡住 —— 即使现在没有外部用户，这也是对未来的
使用者的承诺边界，必须显式声明。

**v0.11 候选**：从 `Descriptor.RuntimeServices` + 驱动派生最小 Compose。
只有 `required: true` 的服务进 services，不含 `jimu-web`、不含 observability。

### 8.1.6 official 门禁不能静默通过

**问题本质**：`go test -run '<regex>'` 匹配不到任何用例时退出码是 0。
阶段 4 删掉旧测试名后，`make test-scaffold-official` 会变成一条永远绿的空命令。

**改动点**（两层，缺一不可）：

1. **换 regex**：`Makefile` 的 `SCAFFOLD_OFFICIAL_TESTS` 改为新预设测试名：
   ```
   ^(TestPresetBudgets|TestMinimalPresetStartsWithoutDependencies|TestMinimalPresetPingRoute|TestAuthPresetEmptyDatabaseEndToEnd)$
   ```
2. **加计数断言**：新增一个不收 `JIMU_HEAVY_MATRIX` 门控的轻量测试
   `TestOfficialGateSelectorsMatch(t)`，它读取 `Makefile` 里这三个
   `SCAFFOLD_*_TESTS` 变量，用 `go test -list '<regex>'` 在本包上跑一遍，
   断言每个 regex 至少匹配 1 个用例；同时断言匹配到的名字里不含
   `basic-api`/`auth-api`。这条测试进 `make check`，成本是一次 `go test -list`。

   实现要点：`-list` 只列出**本包**的顶层测试名，所以正则必须只匹配顶层
   `Test*` 函数。当前 regex 都满足。已实测：
   `go test ./tools/generator/ -list 'TestOfficialTemplateBudgets'` 输出
   `TestOfficialTemplateBudgets` 并 `ok`，退出码 0；空正则匹配时只输出
   `ok ... [no tests to run]`，退出码同样是 0 —— 这正是需要额外断言的证据。
   实现时用 `go test -list` 的 **stdout 行数**判定，不看退出码。

   注意：本机 `go` 可能被 RTK 之类的包装器接管，`-list` 输出被吞成
   "No tests found"。门禁必须调用真实二进制（CI 上是 `/usr/local/go/bin/go`），
   或在包装器环境下跳过该断言并显式报 SKIP，不能静默通过。

   判定伪代码（`tools/checkgateselectors/main.go` 或测试内联）：

   ```go
   out := execGo("test", "./tools/generator/", "-list", regex)
   var names []string
   for _, line := range strings.Split(out, "\n") {
       line = strings.TrimSpace(line)
       if strings.HasPrefix(line, "Test") {
           names = append(names, line)
       }
   }
   if len(names) == 0 {
       return fmt.Errorf("selector %q matches no tests (would be a no-op gate)", regex)
   }
   ```

   失败方式就是返回 error，让 `make check` 非零退出；不要靠 `exit 0/1` 分支，
   也不要吞掉 `exec` 的 stderr（编译失败的诊断必须可见）。

3. **升级门禁同理**：`SCAFFOLD_UPGRADE_TESTS` 里的 `TestUpgradeFixturesMatchV050`
   在 fixture 更换后要改成新名字，并纳入上面的计数断言。

**为什么不用 `go test -v` 输出计数**：输出格式不稳定，且需要解析 stdout。
`-list` 是 Go 官方支持的稳定接口，且能在毫秒级完成。

## 9. 实施阶段

### 阶段 1：内核解耦（可独立合并）

- 新增 `internal/kernel/seedmodel`
- `seed.go` 改用 seedmodel
- 新增 `TestSeedModelMatchesCapabilityDomains`
- `StructuralSeed` 的 nil DB 护栏（8.1.1）—— 纯内核改动，与生成器无关
- `GET /api/v1/ping` 内核路由（8.1.2）—— 同上
- 验收：`go build ./...`、`go test ./internal/app/...`、`make check` 全绿
- 不改生成器行为，`--with` 产物不变

### 阶段 2：生成器支持 `minimal` / `auth` / `full` 预设

- `ParseCapabilitySet` 接受 `--template=minimal|auth|full`
- `minimal` = 空能力集；`auth` = user/access/auth + 闭包 + Ungated；
  `full` = 全部 23 项 + 框架 full 的完整驱动集（见 7.2）
- 新增三个预设的预算条目
- 解除预设项目的 `capability add` 禁令（见 7.3）
- 验收：三个预设各自生成后 `go build ./...` 通过；`minimal` 无 `ADMIN_PASSWORD`
  与有 `ADMIN_PASSWORD` 都能启动且 `/livez` `/readyz` 返回 200；
  `jimu capability add tenant` 在 `minimal` 项目上成功且 support 变为 experimental

### 阶段 3：内核闭包裁剪

- `kernelDirs` 从"整包复制"改为"闭包计算"
- 删除 `kernelRequiredDomains` 补丁表（阶段 1 后不再需要）
- 验收：`minimal` 生成物文件数显著下降；`full` 生成物不受影响

### 阶段 4：退役旧模板

- 删除 `basic-api` / `auth-api` 渲染路径和模板树
- 更新 `budgets.json`、`docs/templates/v1`、README、release note
- `jimu upgrade` 对旧 manifest 硬失败，错误信息含退役版本与迁移路径（见 6.2）
- 重建 `upgradeFixtureCases`（见 6.3），保证升级事务门禁不消失
- Compose 资产按 8.1.5 处理：删除模板私有 Compose，E2E 改用外部 MySQL/Redis
- 更新 `test-scaffold-official` 的选择集

### 阶段 5：门禁与文档

- `minimal` / `auth` / `full` 加入 official 门禁
- 新增 `TestOfficialGateSelectorsMatch` 与 `-list` 计数断言（8.1.6）
- 更新 `docs/design/profiles-and-project-generation.md`
- 新增 ADR 记录"生成即裁剪"的决策

阶段 4/5 的具体门禁改动点（都已核对到当前 Makefile 与测试名）：

- `Makefile` 的 `SCAFFOLD_OFFICIAL_TESTS` 现在钉的是六个 `basic-api`/`auth-api`
  专属测试（`TestNewProjectBasicAPIHasNoExternalRuntimeDependencies`、
  `TestNewProjectAuthAPIIsCompactSingleTenantTemplate`、`TestBasicAPIStartsWithoutExternalServices`、
  `TestAuthAPIComposeConfigIsValid`、`TestAuthAPIComposeUsesHostUIDForSecrets`、
  `TestAuthAPIEmptyDatabaseEndToEnd`）。阶段 4 删除这些模板后这些测试名会消失，
  regex 会匹配不到任何用例而**静默通过**（`go test -run` 不报"无匹配"为失败）。
  因此不能只删测试：必须同时把该 regex 换成新预设的测试名，
  并按 8.1.6 加 `-list` 非空断言（不要用 `-v` 输出计数，格式不稳定）。
- `SCAFFOLD_UPGRADE_TESTS` 里的 `TestUpgradeFixturesMatchV050` 绑定 v0.5 fixture，
  退役后要替换为新 fixture 的完整性测试，不能留空 regex。
- `scripts/clean_room_e2e.sh` 和 `docs/production/{basic-api,auth-api}.md` 同步改为新预设。

## 10. 验证方式

每个阶段都必须有可复现命令：

```bash
# 阶段 1
go build ./...
go test ./internal/app/...
make check

# 阶段 2
go run ./cmd/jimu new /tmp/p-min --template=minimal
cd /tmp/p-min && go build ./... && go test ./...

# minimal 启动冒烟（无 ADMIN_PASSWORD / 有 ADMIN_PASSWORD 各跑一次）
APP_ENV=dev HTTP_PORT=18080 MANAGEMENT_PORT=19090 go run ./cmd/server &
curl -fsS http://127.0.0.1:19090/livez
curl -fsS http://127.0.0.1:19090/readyz
curl -fsS http://127.0.0.1:18080/api/v1/ping   # {"message":"pong"}

# auth / full 生成即编译
for t in auth full; do
  go run ./cmd/jimu new "/tmp/p-$t" --template="$t"
  (cd "/tmp/p-$t" && go build ./... && go test ./...)
done

# full 的驱动必须齐（8.1.3）
for d in kafka rabbitmq; do ls "/tmp/p-full/internal/capabilities/queue/$d"; done
for d in excel; do ls "/tmp/p-full/internal/capabilities/dataops/$d"; done
for d in s3; do ls "/tmp/p-full/internal/capabilities/storage/$d"; done

# 预设项目可继续组合（阶段 2 解除禁令后）
cd /tmp/p-min && go run ./cmd/jimu capability add tenant
go build ./...

# official 门禁选择器必须非空（8.1.6）
go test ./tools/generator/ -list '^(TestPresetBudgets|TestMinimalPresetStartsWithoutDependencies|TestMinimalPresetPingRoute|TestAuthPresetEmptyDatabaseEndToEnd)$'

# 阶段 3
go run ./cmd/jimu new /tmp/p --template=minimal --report
wc -l /tmp/p/docs/profiles/generated-report.md

# 阶段 4
make test-scaffold-official
make test-scaffold-upgrade

# 全量
make release-check-fast
```

## 11. 非目标

- 不做 generated full-stack（不生成 `jimu-web`）
- 不做任意组合的 certified 承诺
- 不做全能力幂集的升级门禁
- 不改运行时 `capabilities.enabled` 的语义
- 不支持 Windows

## 11.1 Release 不再发布 server 二进制

**先摆事实，再说结论。**

| 产物 | 用途 | 谁在用 |
|---|---|---|
| `jimu` CLI（4 平台） | 生成/检查/升级工程 | 所有 `jimu new` 用户 |
| `jimu-{full,minimal}-<os>-<arch>` server（8 个） | 预构建的运行形态 | **查不到任何使用者** |

`jimu-web` 与 server 二进制的关系，已逐条核对：

1. `jimu-web/vite.config.ts` 只需要一个 HTTP 端点：`JIMU_API_PROXY_TARGET ?? 'http://localhost:8080'`。
   它不关心这个端点由什么产生。
2. `jimu-web/README.md`「起后端」章节给的是 `cd ../jimu && make compose-up`，
   即 **Docker Compose 从源码构建**，不是下载 release 资产。
3. `scripts/demo.sh` 走 `make up` → `docker-compose.yml` 的 `build: context: .`，
   同样是从源码构建。
4. 全仓文档（README、docs/、jimu-web）没有任何一处让用户下载 `jimu-full-*` / `jimu-minimal-*`。
   只有历史 plan/spec 提到这些名字。

所以：**jimu-web 需要的是"一个跑起来的后端"，而不是"一个 release 里的二进制"**。
今天这个后端由 Compose 从源码构建提供，链路完整、已有验证（`make demo` / `make demo-e2e`）。

还有一个决定性事实：server 二进制**不是自包含的**。`Dockerfile` 的 runtime 阶段显式 COPY 了
`conf/`、`configs/`、`docs/openapi/`，且 `internal/config` 在没有 `app.yaml` 时是硬失败
（`failed to read app.yaml`），没有内置默认配置。因此"下载一个二进制就能起服务"这个卖点
**今天并不成立** —— 用户还得自己准备配置目录、MySQL、Redis、迁移与 seed。
发布一个跑不起来的二进制，比不发布更糟。

**决策：v0.10.0 起 Release 只发布 CLI。**

| 资产 | 处理 |
|---|---|
| `jimu-full-<os>-<arch>` × 4 | 删除 |
| `jimu-minimal-<os>-<arch>` × 4 | 删除 |
| `jimu-<os>-<arch>` CLI × 4 | 保留 |
| `template-bundle.json` + checksums | 保留 |

这条与"交付物是代码"一致：用户要么用 `jimu new` 生成工程自行构建，
要么直接 `make up` 跑框架本体。两条路都不需要预构建的 server 二进制。

改动点：

- `release.yml`：删除整个 `build` job（8 条矩阵）；`publish` 的 `needs` 去掉 `build`；
  `smoke` job 不动（它下载的是 CLI）。
- `scripts/test_makefile_contract.sh` 增加反向断言：release workflow 不得再出现
  `./cmd/server`、`matrix.profile`、`jimu-${PROFILE}`。
- `docs/releases/v0.10.0.md` 写进"不兼容变更"。

**保留不动**：`PROFILE=<name> make build-server`、`Dockerfile` 的 `ARG PROFILE`、
`internal/profiles/{full,minimal}`、`tools/profileoverlay`、`scripts/check_profiles.sh`。
这些是框架自身的开发与自测路径（`make check-capabilities`、`make profiles-check`
依赖它们的双形态对比），与发布物无关。

**如果将来需要"下载即用"的后端**，正确做法是发 **容器镜像** 而不是裸二进制：
镜像才能把 `conf/`、`configs/`、`docs/openapi/` 和启动脚本一起带上。
那是独立的一项工作，不在本次范围内。

## 12. 留给实现者的决定

- `seedmodel` 的字段是否直接复制 GORM tag，还是用接口约束
- 闭包计算的实现方式（`go list -deps` vs `packages.Load`）
- `minimal` 是否保留空的 `internal/capabilities/catalog` 包

## 13. 已确认的决策（原待确认项）

1. **第 6.2 节**：旧 `basic-api`/`auth-api` 项目的升级行为 → **A，硬失败**。
2. **第 7 节**：是否保留 `auth` 预设 → **保留**，共 `minimal`/`auth`/`full` 三个。
3. **第 8 节**：`full` 首发支持级别 → **`validated`**。
