# 配置收敛与 Makefile 生命周期重构设计

- 日期：2026-10-23
- 状态：已确认
- 范围：本仓的本地/Compose 配置入口、secret 生命周期、Makefile，以及生成项目的 Makefile 模板

## 1. 背景与目的

当前 `.env.example` 有 33 个变量，`Makefile` 通过 `include .env + export` 把其中所有变量注入子进程，Compose 又自动读取同一个 `.env`。应用配置、部署接线、首次启动凭据和可选组件凭据混在一个文件里，导致：

- 用户不知道哪些值必须修改，哪些可以直接保留。
- `DB_*` 与 `COMPOSE_DB_*`、`REDIS_*` 与 `COMPOSE_REDIS_*` 表达同一含义，形成两套配置。
- `ADMIN_PASSWORD` 写在 `.env`，但 Compose 的常驻 server 不注入它，容易误以为启动时会自动建管理员。
- `ENCRYPTION_KEY` 没有出现在 `.env.example`，Compose 也没有挂载对应 secret；文档却宣称生产必须配置。
- `ZO_OBSERVE_ROOT_USER_PASSWORD` 存在固定默认密码，开启 observability 时没有形成明确的“先设置凭据再启动”约束。
- `Makefile` 有 484 行、70 多个 target，帮助文本手工重复维护，`compose-up`、`docker-run`、`make run` 各自表达部分启动路径。
- 镜像构建同时存在 `make docker-build` 和 `docker compose up` 的隐式构建两条路径，用户无法判断哪条是权威入口。

本设计的目标不是做一次 `.env` 文案整理，而是明确四类配置的生命周期，并把日常操作收敛为少量稳定命令。

## 2. 目标与非目标

### 目标

1. 用户日常只需要理解和维护 `configs/app.yaml`，不再手写 `.env`。
2. 首次启动所需的敏感值由脚本生成到 `secrets/`，目录权限 `0700`、文件权限 `0600`，幂等且默认不覆盖已有文件。
3. 正常启动完全不读取项目根 `.env`；Compose 通过 `--env-file /dev/null` 禁止自动加载它。
4. `make up` 一条命令完成 secret 检查、镜像构建、迁移、种子和服务启动。
5. OpenObserve 改为显式可选栈，只有执行 `make up-observability` 时才生成并校验管理员凭据。
6. `make image` 是唯一显式构建容器镜像的入口，`VERSION` 同时生成版本标签和 `latest` 标签。
7. 根 Makefile 与生成项目的 Makefile 模板共享同一套命名、帮助和“构建二进制 / 构建镜像”边界。
8. 修复字段加密密钥长度校验缺陷：非空 `security.encryption_key` 必须恰好 32 字节，而不是“不少于 32 字节”。

### 非目标

- 不移除应用对环境变量覆盖的支持。Helm、Kubernetes、CI 和临时测试仍可注入 `DB_*`、`JWT_SECRET_FILE` 等运行时值。
- 不实现 Vault、云 Secret Manager 或密钥托管服务接入。
- 不自动轮换数据库密码、管理员密码、JWT 密钥或字段加密密钥。
- 不改 Helm/Kubernetes 的 Secret 模型。
- 不便携现有 `.env` 中的非敏感应用配置到运行时；这些值应由 `configs/app.yaml` 表达。
- 不在本轮处理 OAuth client secret、SMTP 密码、SMS AccessKey、S3 secret、Webhook 签名密钥和 CSRF secret 的自动生成。这些可选集成默认关闭，继续沿用现有 YAML/部署注入方式；为它们补全 `_FILE` 支持属于后续独立工作。
- 不为删除的旧命令保留别名。

### 反目标

- 不引入启动前 YAML 转 Compose 环境变量的生成器。
- 不保留一套“推荐配置”和一套“真实配置”长期并存。
- 不让用户在多个文件中填写同一个连接地址、端口或密码。
- 不把生成后的 secret 提交到 Git，也不在帮助文本或日志中打印密钥值。

## 3. 配置分层

| 层 | 唯一职责 | 用户是否编辑 |
|---|---|---|
| `configs/app.yaml` | 应用行为、连接默认值、安全与功能开关 | 是 |
| `configs/app.prod.yaml` | 仅保存生产环境相对基线的差异 | 按部署需要 |
| `secrets/` | 机器生成或外部注入的凭据与加密材料 | 否 |
| `docker-compose.yml` | 本地容器拓扑、固定容器内地址、secret 挂载 | 否 |
| `compose.observability.yml` | OpenObserve 可选栈及其运行时接线 | 否 |
| `Makefile` | 用户入口和生命周期编排 | 维护者编辑 |

`configs/app.yaml` 是业务配置事实源；`docker-compose.yml` 只表达“容器里如何接线”。例如 Compose 内固定使用 `DB_HOST=mariadb`、`REDIS_ADDR=redis:6379`，这些不是用户业务配置，不再暴露为 `COMPOSE_DB_*` 变量。

保留现有环境变量覆盖代码，是为了让部署平台能在不修改镜像内 YAML 的情况下注入 secret 路径和临时连接参数。它们不是日常使用说明的一部分。

## 4. 目标文件布局

```text
configs/
  app.yaml
  app.prod.yaml
docker-compose.yml
compose.observability.yml
Makefile
scripts/
  init-secrets.sh
  dev.sh
  observability.sh
secrets/                    # gitignored
  db_root_password.txt
  db_password.txt
  jwt_secret.txt
  encryption_key.txt
  audit_hash_secret.txt
  admin_password.txt
  otel_auth_password.txt   # 启用 observability 时生成，作为 OpenObserve 密码源
  zo_auth_token.txt        # 启用 observability 时从密码源派生
  openobserve.env           # 仅在启用 observability 时生成
```

删除 `.env.example`。保留 `.env` 的 gitignore 规则，避免用户已有文件被误提交；正常入口不再读取它。

## 5. Secret 模型

### 5.1 生成规则

`scripts/init-secrets.sh` 是唯一生成入口：

- 使用 `umask 077`，创建 `secrets/` 为 `0700`。
- 已存在的文件不覆盖；只有显式传入 `FORCE=1` 才允许重建。
- 生成失败立即退出，不留下半写入文件。
- 输出只列文件名与用途，不输出 secret 值。
- 在 TTY 中首次生成时允许用户设置 admin 初始密码；非 TTY 环境生成随机强密码。
- 检测到旧 `.env` 时，按**单个缺失文件**导入已知的 secret 键（DB root、DB 用户、JWT、admin、OpenObserve），并提示该 `.env` 已停用；不导入端口、地址、卷等普通配置。不能因为 `secrets/` 已存在就跳过缺失项。
- 新安装（没有对应数据卷）自动生成所有缺失的核心 secret。
- 已有 MariaDB 数据卷时，缺少数据库凭据必须终止并要求恢复原值，不能静默生成新密码把已有数据锁死。
- 已有 MariaDB 数据卷、但缺少原本未使用的 JWT/字段加密/审计 secret 时，初始化不得静默生成这些会改变既定语义的值；必须要求操作者显式传入 `ALLOW_NEW_SECURITY_KEYS=1` 完成一次性迁移。错误信息必须说明该参数的后果。
- 已有 OpenObserve 数据卷时，必须能从 `otel_auth_password.txt`、旧 `openobserve.env` 或旧 `.env` 恢复出原密码；找不到则终止，不能用新密码覆盖已有实例。

### 5.2 分类与格式

| 文件 | 用途 | 生成要求 |
|---|---|---|
| `db_root_password.txt` | MariaDB 首次初始化、指标采集 | 高熵随机串 |
| `db_password.txt` | Jimu 应用数据库用户 | 高熵随机串 |
| `jwt_secret.txt` | Access/Refresh Token 签名 | 至少 32 字节 |
| `encryption_key.txt` | AES-256-GCM 字段加密 | 恰好 32 个可打印 ASCII 字节，无换行 |
| `audit_hash_secret.txt` | 审计链 HMAC | 至少 32 字节高熵随机串 |
| `admin_password.txt` | 首次结构种子 | 用户输入或随机强密码 |
| `openobserve.env` | OpenObserve 管理员账号 | 仅在 observability 初始化时生成；符合 OpenObserve 密码强度要求 |

`ENCRYPTION_KEY` 不能按当前“≥32 字节”规则生成任意长度字符串。Go 的 `aes.NewCipher` 只接受 16、24 或 32 字节；现有 `configs/app.yaml` 中的 39 字节开发密钥实际会导致加密写入时报 `invalid key size`。设计将其修正为：

- 生成器写入恰好 32 字节。
- 配置校验在非空时断言长度恰好为 32 字节。
- 增加覆盖 31/32/33 字节和现有开发占位值的回归测试。

### 5.3 注入方式

Compose 使用 secret 文件，不把密码展开到命令行：

- MariaDB：`MARIADB_ROOT_PASSWORD_FILE`、`MARIADB_PASSWORD_FILE`
- Jimu server/init：`DB_PASSWORD_FILE`、`JWT_SECRET_FILE`、`ENCRYPTION_KEY_FILE`、`AUDIT_HASH_SECRET_FILE`、`ADMIN_PASSWORD_FILE`
- OpenObserve：官方仅支持环境变量读取 `ZO_ROOT_USER_EMAIL` / `ZO_ROOT_USER_PASSWORD`，因此使用自动生成、权限 `0600` 的 `secrets/openobserve.env`；这是唯一保留的 `env_file`，且不是用户编辑入口。`otel_auth_password.txt` 是密码源，`openobserve.env` 和 `zo_auth_token.txt` 都从它派生，避免用户维护三份密码。

`internal/app/seed.go` 与 `internal/profiles/profiles.go` 的 `ADMIN_PASSWORD` 读取统一支持 `_FILE`，避免“CLI seed 支持文件、启动期种子仍只认明文环境变量”的分裂行为。

## 6. 应用配置

`configs/app.yaml` 保持完整注释和可读性，用户修改它即可调整业务行为。`configs/app.prod.yaml` 改为稀疏覆盖文件，只保留生产差异，不重复整份 dev 配置。

具体约定：

- `configs/app.yaml` 不存放真实 secret；密码字段留空，由 secret 文件覆盖。
- `configs/app.prod.yaml` 不重复默认值；Viper 已支持在当前配置上 merge。
- Compose 内固定服务地址和内部端口，不再要求用户填 `DB_HOST`、`COMPOSE_DB_HOST` 两套值。
- `APP_ENV` 仅作为运行环境选择器保留；`make up` 默认传 `dev`，高级部署可显式覆盖。
- `PROFILE` 是构建形态选择器，不是应用配置；`build`、`image`、`migrate`、`seed`、`dev` 和 Compose init 都必须使用同一个 `PROFILE`，避免二进制、迁移集和种子权限集不一致。`VERSION` 只决定镜像标签。
- 端口、卷名等 Compose 运行参数可以有默认值，但不进入 `.env.example`，需要时通过 Compose override 或一次性命令变量覆盖。

## 7. Compose 启动流程

`docker-compose.yml` 新增一次性 `jimu-init` 服务，职责固定为：

1. 等待 MariaDB 与 Redis 健康。
2. 执行 `./jimu migrate up`。
3. 从 `/run/secrets/admin_password` 读取初始密码并执行 `./jimu seed`。
4. 成功退出。

`server` 通过 `depends_on: condition: service_completed_successfully` 依赖 `jimu-init`。因此首次启动不再要求用户依次执行迁移和种子；迁移失败时 server 不会错误启动。

核心栈只包含 `mariadb`、`redis`、`jimu-init`、`server`。Adminer 不再跟随 `APP_ENV=dev` 隐式启动；需要时通过显式开发工具入口启动。

`compose.observability.yml` 叠加 OpenObserve、OTel Collector，并只在该 profile 下给 server 设置：

- `OTEL_ENABLED=true`
- `OTEL_ENDPOINT=openobserve:5081`
- `OTEL_AUTH_EMAIL=admin@jimu.local`
- `OTEL_AUTH_PASSWORD_FILE=/run/secrets/otel_auth_password`

核心 `make up` 不启动 observability，也不要求存在 OpenObserve 密码。`make up-observability` 在启动前调用 secret 初始化并校验 OpenObserve 凭据。

Observability secret 的权威关系固定为：

```text
otel_auth_password.txt
  ├── openobserve.env  (ZO_ROOT_USER_EMAIL + ZO_ROOT_USER_PASSWORD，供 OpenObserve 容器)
  └── zo_auth_token.txt (base64(email:password)，供 collector)
```

初始化脚本只在文件缺失时生成它们；已有自定义 token 不被静默覆盖。

所有 Compose 调用统一带 `--env-file /dev/null`，防止项目根旧 `.env` 被自动加载。服务需要的运行时变量由 Makefile 显式传入；容器自身的 `env_file` 只允许指向 `secrets/` 下自动生成的文件。

## 8. Makefile 设计

### 8.1 公共入口

| 命令 | 行为 |
|---|---|
| `make up` | 初始化缺失 secret，构建镜像，启动核心栈；包含迁移与种子 |
| `make up-observability` | 在 `up` 基础上启动 OpenObserve 栈 |
| `make down` | 停止核心与 observability 栈，保留数据卷 |
| `make restart` | 重启已存在容器，不重建镜像 |
| `make status` / `make logs` | 查看核心栈状态 / 日志 |
| `make adminer` | 显式启动 Adminer 开发工具，不影响核心启动 |
| `make dev` | 启动本地依赖、执行迁移/种子，以热重载方式运行 Go server |
| `make image` | 唯一显式镜像构建入口 |
| `make build` | 只构建本机 Go 二进制，不构建镜像 |
| `make migrate` / `make seed` | 高级运维入口，作用于当前 Compose 栈 |
| `make check` / `make ci` / `make release-check` | 开发与发布门禁 |

直接删除以下旧入口，不保留别名：

- `compose-up`、`compose-down`、`compose-restart`、`compose-logs`
- `compose-migrate`、`compose-seed`
- `docker-run`、`docker-stop`、`docker-logs`
- `docker-build`
- `run`
- 公开的 `secrets`

### 8.2 镜像构建与标签

镜像构建只有 `make image` 一个入口：

```text
IMAGE_REPO ?= jimu
VERSION ?= dev
IMAGE_TAG = $(IMAGE_REPO):$(VERSION)
LATEST_TAG = $(IMAGE_REPO):latest
```

`make image` 构建 `IMAGE_TAG`，随后将同一镜像标记为 `LATEST_TAG`。`make up` 调用 `make image`，并以 `JIMU_IMAGE=$(IMAGE_TAG)` 启动 Compose，因此运行的是明确的版本号，而不是碰巧存在的旧 `latest`。

示例：

```text
make image VERSION=v0.4.0   # jimu:v0.4.0 + jimu:latest
make up VERSION=v0.4.0      # 使用 jimu:v0.4.0
make up                     # 使用 jimu:dev
```

本地 Docker tag 本身没有不可变保证；语义版本标签的不可覆盖约束由发布 CI/镜像仓库负责。Makefile 不假装本地 tag 能提供注册表级不可变性。

### 8.3 Makefile 结构优化

- 删除 `include .env` 和 `export`。
- 使用 `COMPOSE = docker compose --env-file /dev/null`，并在需要时显式传入环境变量。
- 帮助文本从 `##` target 注释自动生成，删除约 60 行重复 `echo`。
- `.PHONY` 以公共 target 清单集中维护，避免分散在多行。
- Makefile 只做编排；复杂逻辑移入 `scripts/init-secrets.sh`、`scripts/dev.sh`、`scripts/observability.sh`。
- `make up` 每次重建镜像，保证代码和容器一致；只想重启时使用 `make restart`。
- `make dev` 通过 `scripts/dev.sh` 从 `secrets/` 读取 secret 文件路径，不读取 `.env`。
- `PROFILE` 默认 `full`，并贯穿本地构建、镜像构建、迁移、种子和开发运行；`VERSION` 只影响镜像标签。

### 8.4 生成项目同步

`tools/generator/render/templates/project/Makefile.tmpl` 同步以下约定：

- 自动帮助；
- 不 `include .env`；
- `build` 与本机二进制、`image` 与容器镜像严格分离；
- `image` 同样生成版本标签和 `latest`；
- 生成项目没有 Compose 栈，因此不生成 `up`、`down`、`up-observability` 等 Compose 专属入口。

`tools/generator/buildfiles_test.go` 增加模板断言，防止后续重新引入 `.env` 或旧 target。

## 9. 文档与迁移

实施时必须同步更新：

- `README.md`：安装、配置、快速开始、命令表、配置项说明、目录树、OpenObserve、备份恢复。
- `docs/CONTRIBUTING.md`：开发环境启动命令和 release-check 说明中移除 `COMPOSE_ENV=.env.example`。
- `docs/design/configuration-and-data-lifecycle.md`：补充配置分层与 secret 生命周期。
- `docs/releases/v0.4.0.md`：记录 `.env` 移除、Makefile target 直接替换、旧 `.env` 迁移行为及镜像标签规则。

`scripts/smoke_api_contract.sh`、`scripts/test_runtime_security.sh`、`scripts/observability.sh` 等隔离脚本不再依赖根 `.env`；全部通过临时 `secrets/`、显式变量和独立 Compose project 工作。

## 10. 验证要求

至少覆盖：

1. 从无 `secrets/` 的干净工作区执行 `make up`，能生成 secret、构建镜像、完成迁移/种子并启动。
2. 重复执行 `make up` 不覆盖已有 secret，不重复创建 admin，不因 `.env` 存在而改变配置。
3. `secrets/` 目录为 `0700`，文件为 `0600`；失败路径不留下半生成文件。
4. `make image VERSION=v0.4.0` 同时存在 `jimu:v0.4.0` 与 `jimu:latest`，`make up VERSION=v0.4.0` 使用版本标签。
5. `make up-observability` 会自动生成强 OpenObserve 凭据；未启用 observability 时不要求该凭据。
6. 已有 OpenObserve 数据卷但缺少旧密码时，启动失败且不会改写数据卷；`otel_auth_password.txt`、`openobserve.env`、`zo_auth_token.txt` 三者保持一致。
7. `ADMIN_PASSWORD_FILE` 同时被 CLI seed 和启动期结构种子识别；密码缺失时不创建弱默认账号。
8. `security.encryption_key` 在 31/33 字节时启动失败，32 字节时正常；现有 39 字节占位值不再可用。
9. 生成项目的 Makefile 有自动帮助、无 `.env` include、有 `image` target，并继续保持单形态构建。
10. `make compose-check`、`make check-capabilities`、`make profiles-check`、`make test`、`make test-race` 通过。
11. README 的从零启动步骤可由新用户在只编辑 `configs/app.yaml` 的前提下完成。

## 11. 风险与约束

- 删除 `.env` 对已有本地部署是破坏性变更；通过一次性 secret 导入和明确错误信息降低数据锁死风险。
- 数据库 secret 与已有 MariaDB 数据卷强绑定；secret 丢失时不能自动恢复。
- 字段加密密钥泄露没有自动轮换机制；本设计只保证生成、持久化和校验，不宣称支持热轮换。
- OpenObserve 仍需通过环境变量接收管理员密码，因此只能做到“文件自动生成且不让用户维护”，无法像 Jimu 一样全程只传 secret 文件路径。
- 本地镜像 tag 不是注册表不可变标签；版本发布仍必须在 CI/镜像仓库层保证不可覆盖。
