# 配置与数据生命周期设计

## 背景与范围

配置、数据库表、迁移、历史数据清理和字段保护需要跨能力组合。若共享机制同时决定业务配置或清理策略，能力拆分后仍会留下隐式的数据所有者。

本文定义这些资源的所有权如何与能力边界对应。具体 YAML 键、表名和清理策略以能力描述符、配置结构、迁移及任务实现为准。

### 设计目标与非目标

设计目标：

- 配置段和业务数据由声明它们的能力负责。
- schema 迁移可以独立于运行时业务能力集合解析。
- 共享工具提供通用机制，不维护跨能力的数据策略。

非目标：把所有业务配置搬到新的 YAML 顶层，或由共享清理服务集中决定不同能力的数据保留期限。

## 设计方案

### 配置分层与边界

运行时配置按职责分为四层，入口之间不复制同一含义的值：

| 层 | 载体 | 职责 | 日常是否编辑 |
|---|---|---|---|
| 业务基线 | `configs/app.yaml` | 应用行为、连接默认值、安全与功能开关 | 是 |
| 环境差异 | `configs/app.prod.yaml` | 只声明相对基线的生产差异，未出现的键继承 `app.yaml` | 按部署需要 |
| 运行时注入 | 进程环境变量及 `_FILE` | 部署平台注入连接参数、Secret 路径和临时覆盖 | 部署时 |
| 敏感材料 | `secrets/*.txt`、`secrets/openobserve.env` | 数据库、JWT、字段加密、审计、管理员和观测凭据 | 否，由脚本生成 |

`docker-compose.yml`、`compose.observability.yml` 与 `Makefile` 属于运行接线和编排，不是业务配置入口。Compose 内固定使用 `DB_HOST=mariadb`、`REDIS_ADDR=redis:6379` 等容器地址；`PROFILE` 约束构建、迁移与种子使用同一形态，`VERSION` 只决定镜像标签。

正常入口不读取项目根 `.env`。Compose 调用统一携带 `--env-file /dev/null`，避免旧文件被自动加载；项目根 `.env` 仅作为一次性迁移来源。

### Secret 生成与迁移

`scripts/init-secrets.sh` 是 core/observability Secret 的唯一生成入口。`make init` 是旧 `.env` 的唯一公开迁移入口；`make up` / `make up-observability` 只调用不带 `LEGACY_ENV_FILE` 的内部准备目标：

- `secrets/` 使用 `0700`，文件使用 `0600`；已存在的非空文件默认保留，只有显式 `FORCE=1` 才重建。已有 MariaDB 数据卷时，安全 Secret 仍只补缺失项，`FORCE=1` 不得轮换已有 JWT、加密、审计或管理员密码。
- 写入采用临时文件加原子替换，失败不留下半写入文件；命令输出只列文件名和用途，不打印值。
- 首次非交互初始化生成随机强管理员密码；TTY 环境允许操作者输入初始密码。`ADMIN_PASSWORD_FILE` 同时供 CLI seed、启动期结构种子和 Compose `jimu-init` 使用。
- `encryption_key.txt` 恰好 32 字节；非空 `security.encryption_key` 也必须是 32 字节，其他长度启动失败。
- 旧 `.env` 只通过 `LEGACY_ENV_FILE=/path/to/.env scripts/init-secrets.sh` 导入已知 Secret 键；普通端口、地址和卷名不会迁移。
- 缺少数据库 Secret 且已有 MariaDB 数据卷时初始化失败，避免生成新密码锁死已有数据；为既有数据卷新增安全密钥必须显式设置 `ALLOW_NEW_SECURITY_KEYS=1`（公开入口为 `make init ALLOW_NEW_SECURITY_KEYS=1`）。
- OpenObserve 凭据只在 `--observability` 路径生成。`openobserve.env` 是 server、OpenObserve 容器和管理脚本共用的运行时账号来源；`otel_auth_password.txt` 保留为初始化/迁移校验源，`zo_auth_token.txt` 从最终账号派生。旧 `.env` 的自定义邮箱会迁入 `openobserve.env`；若与既有配置冲突则显式失败。已有 OpenObserve 数据卷但找不到原密码时初始化失败，不覆盖原实例。

### `make up` 启动流程

```mermaid
flowchart LR
    U[make up] --> P[prepare-secrets]
    P --> S[scripts/init-secrets.sh]
    S --> B[make image]
    B --> C[compose up -d --no-build]
    C --> J[jimu-init]
    J --> H[等待 MariaDB / Redis 健康]
    H --> M[./jimu migrate up]
    M --> D[./jimu seed]
    D --> V[server]
```

`prepare-secrets` 不传 `LEGACY_ENV_FILE`，因此 `make up` 不会因项目根存在旧 `.env` 而迁移凭据。需要迁移时，操作者必须显式执行 `make init`。

`jimu-init` 是一次性服务，固定执行 `./jimu migrate up && ./jimu seed`，失败时不会启动 `server`。`server` 通过 `depends_on.condition: service_completed_successfully` 等待它成功退出。`make up-observability` 复用同一流程，再叠加 `compose.observability.yml`，最后调用 `scripts/observability.sh start` 初始化 dashboard 和默认告警；核心 `make up` 不要求 OpenObserve 凭据。

### 配置加载流程

```mermaid
flowchart TD
    P[profile assembly] --> S[解析运行时能力集]
    K[加载内核配置与 YAML sections] --> S
    E[capabilities.enabled] --> S
    S --> D[按 Descriptor.Configs 解码已选配置]
    D --> DF[ApplyDefaults]
    DF --> V[Validate / ValidateProd]
    V --> C[创建内核容器并 Wire]
    C --> R[启动业务服务]
    V -. 配置错误 .-> F[启动失败]
```

### 配置所有权

内核和基础设施配置由 `internal/config` 定义。能力通过 `Descriptor.Configs` 声明自己拥有的配置段，并提供段实例、默认值和校验。组合根先解析当前 profile 的启用集，再按该集合解码能力配置、应用默认值并校验；生产环境还会调用可选的生产校验。未启用能力的配置段不会被解码或校验。

配置段的 YAML 路径可以保持现状，不要求为了代码所有权而移动对外配置键。当前 `auth`、`mfa`、`passkey` 与 `tenant` 分别声明自己负责的配置；例如 WebAuthn 参数属于 `passkey`，可信设备期限属于 `mfa`，开通式注册属于 `tenant`。能力配置变更需重启服务；配置文件热更新目前只应用日志级别。

### 表与迁移

能力在 `Descriptor.Owns` 声明自己负责创建的表，迁移文件由 `Descriptor.Migrations` 提供并嵌入二进制。`Owns` 约束 `CREATE TABLE` 的唯一归属，不表示该能力的所有迁移只能修改自己拥有的表。迁移按 catalog 依赖顺序执行，版本状态按能力分别记录。

迁移集合与运行时装配集合分开解析。profile 的迁移集合可以包含为 schema 完整性所需的额外迁移；这不会把对应业务能力加入运行时装配。迁移边界变化时，同时检查各能力迁移、catalog 的 schema 依赖及 `jimu migrate` 使用的集合。存量库从旧全局版本表迁移到能力版本表时，由 `adopt-capabilities` 登记已有迁移基线，不执行迁移 SQL。

`jimu migrate` 根据选定 profile 构造迁移集合，补上 schema 依赖后按能力拓扑顺序执行；每个能力使用自己的 goose version table。向前迁移按拓扑顺序，回滚按相反顺序。迁移错误带能力名返回；`MigrateWithRetry` 达到配置的重试上限后仍返回错误。

```mermaid
flowchart LR
    P[profile 迁移声明集] --> D[补齐 schema 依赖]
    D --> O[按能力拓扑排序]
    O --> U[Up 按正序执行]
    O --> R[Down / redo 按逆序执行]
    U --> V[每能力独立版本表]
    R --> V
    D -. 仅 schema 依赖 .-> M[不增加运行时装配能力]
```

### 种子数据

种子执行位于 app 层，接收当前 profile 的解析能力描述符，不导入具体能力包。能力权限点从这些描述符聚合；默认租户、结构性套餐、两个平台系统角色和管理员数据由 app 的结构性 seed 维护。开发阶段不提供旧角色 adoption；seed 结束前必须存在至少一个有效平台超管。profile seed 在能力 Wire 完成后、Bootstrap 之前运行，失败会阻止服务启动。

### 历史数据清理

[`internal/shared/dbpurge`](../../internal/shared/dbpurge) 只提供调用方指定模型、时间列、过滤条件和批大小的分批删除原语，不拥有表清单，也不注册通用清理任务。清理策略、调度任务和过滤条件由产生数据的能力负责，因此清理规则随该能力装配。

当前审计记录、队列历史、已发布 outbox 事件、已结束导入任务和过期可信设备的清理分别归 `audit`、`queue`、`outbox`、`dataops` 与 `mfa`。不存在独立的 retention 能力；共享原语不能形成跨能力的数据所有权。

### 字段保护

`encryption` 能力通过 GORM hooks 为标记字段提供 AES-GCM 加密和 HMAC 盲索引，支持密文存储及等值查询。密钥由内核安全配置提供；当前实现未配置密钥时会以明文保存这些字段。密码使用不可逆哈希，不经过字段加密流程。全库透明加密属于数据库或部署层责任。

启用加密能力后，它从内核安全配置构造 Cipher 并在 GORM 注册写入加密、读取解密和盲索引 hooks。无密钥时 Cipher 按现行兼容行为退化为明文；因此仅编入能力不代表敏感字段已获得静态加密保护，部署方必须配置恰好 32 字节的有效密钥。

## 关键决策

- **配置键路径与代码所有权分离。** 配置段归属由 `Descriptor.Configs` 表达，对外 YAML 路径可以保持稳定，避免内部模块重组迫使部署配置迁移。
- **迁移集合与运行装配集合分离。** profile 可以为 schema 完整性选择额外迁移，但迁移依赖不会自动引入对应业务路由或用例。
- **清理策略由数据所有者执行。** `dbpurge` 是通用删除原语，不注册统一 retention 作业；这样清理任务随产生数据的能力启停。
- **字段加密作为应用字段保护能力。** GORM hooks、密文与盲索引属于 `encryption`；密钥来源由内核安全配置提供。
- **配置只加载当前启用集。** 未选能力的配置不会被解码，也不会因无关配置错误阻止启动；已选能力配置错误则 fail-fast，避免以不完整配置启动。
- **迁移使用每能力版本表。** 迁移进度和回滚边界与数据所有权对齐；代价是迁移工具必须按能力顺序运行，不能只看一个全局版本号。

## 约束与不变量

改变配置、表或数据生命周期的所有者时，同时检查 `Descriptor.Configs`、`Descriptor.Owns`、迁移声明和对应任务。共享清理工具不能登记所有者以外的表或策略；生成清理任务应随数据所有者一起启停。同一数据库始终使用同一 profile 执行迁移、回滚、重做和存量基线登记，避免按不同迁移集合操作导致版本记录与实际 schema 分离。

清理规则的 `Days <= 0` 表示跳过；执行按有界批次删除，单条规则失败会被收集并返回，不应记录成全部成功。改变 GORM 加密标记或盲索引源字段时，必须同时检查写入、读取、唯一约束和查询路径；密钥轮换与既有密文兼容策略目前不由该能力自动完成。

## 实现映射

- 配置加载：[internal/config](../../internal/config)、[internal/app.LoadCapabilityConfigs](../../internal/app/capconfig.go)
- 表与迁移：[internal/contract.Descriptor](../../internal/contract/capability.go)、[internal/capabilities/catalog](../../internal/capabilities/catalog)
- 种子与迁移执行：[internal/app/seed.go](../../internal/app/seed.go)、[internal/kernel/db/migrate.go](../../internal/kernel/db/migrate.go)、[catalog.MigrationSchemaDeps](../../internal/capabilities/catalog/migration.go)
- 清理与加密：[internal/shared/dbpurge](../../internal/shared/dbpurge)、[internal/capabilities/encryption](../../internal/capabilities/encryption)
- 验证：配置段单测、迁移集/迁移集成测试、各数据所有者清理测试，以及 `make check-capabilities`
- 相关设计：[能力架构](capability-architecture.md)、[身份与租户](identity-and-tenancy.md)
