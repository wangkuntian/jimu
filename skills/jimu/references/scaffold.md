# 脚手架：生成项目 / 追加能力

**何时读**：要为一个新项目生成只含选中能力的代码库、在已生成项目上追加能力、或在本仓生成能力骨架时。

**权威口径**：README「[生成项目（`jimu new` / `jimu capability add`）](../../../README.md#生成项目jimu-new--jimu-capability-add)」、「[CLI 工具](../../../README.md#cli-工具)」。

## 两条裁剪路径（别混淆）

| 路径 | 做什么 | `go.mod` |
|---|---|---|
| 层① 脚手架 `jimu new` | 生成**独立 module**（复制内核 + 按复制集落地能力 + 渲染四处派生文件 + `go mod tidy`） | **真的变小** |
| 层② 形态（profile） | 只决定哪些包与符号编进二进制 | **不变** |

## `jimu new <dir>`

```bash
./bin/jimu new ./myapp --module=example.com/myapp --report                   # 默认 basic-api
./bin/jimu new ./myapp --template=basic-api --module=example.com/myapp       # 官方模板
./bin/jimu new ./myapp --template=auth-api --module=example.com/myapp        # 单租户认证模板
./bin/jimu new ./myapp --with=user,access,auth,queue --shape=app              # 能力名（硬依赖闭包 + 拓扑序）
./bin/jimu new ./myapp --with=auth,queue:kafka --dry-run                      # 只打印计划，不落盘
./bin/jimu check ./myapp                                                      # 只读检查 manifest 与组合一致性
./bin/jimu upgrade ./myapp                                                    # 显式升级：备份、安装、验证、回滚、更新 manifest
./bin/jimu upgrade ./myapp --dry-run                                          # 只读三方 diff，不写项目目录
./bin/jimu upgrade ./myapp --check                                            # 有 conflict 时退出非 0
```

| 参数 | 说明 |
|---|---|
| `--template=basic-api\|auth-api` | 官方模板；省略选择器时默认 `basic-api` |
| `--profile=minimal` | 弃用兼容别名，映射为 `auth-api`，v0.6.0 移除；其他 profile 由 CLI 拒绝 |
| `--with=<cap>[:<drv>][,…]` | experimental 专家模式；名字取自 catalog 与 Ungated 能力，驱动默认取 `Descriptor.Drivers` 首项，用 `<cap>:<drv>` 覆盖。受保护能力需要同时选择 middleware provider（如 `auth`） |
| `--shape=<name>` | 仅用于 `--with` 的形态名（默认 `app`） |
| `--module=<path>` | 重写 `go.mod` 的 module 指令与文本里的 `jimu/` 前缀；**不改**框架运行期名字 / CLI 名 / 镜像名 / protobuf 描述符 |
| `--dry-run` | 只打印 requested/resolved/auto-added、驱动、外部服务、迁移、status 与文件/资产计划，不落盘 |
| `--force` | 只覆盖带 `.jimu/manifest.json` 的产物（识别依据是 manifest，不是时间戳） |
| `--no-tidy` | 跳过 `go mod tidy`（`go.mod` 保持框架依赖集） |
| `--report` | 写 `<dir>/docs/profiles/generated-report.md`（文件数 / 闭包 Go 文件数与代码行 / 直接依赖数 / 迁移数 / 表数 / 路由数 / 重型依赖 / 资产数） |

**默认自检**：生成后跑 `go build ./...` 与生成项目自己的 `check-capabilities`，任一失败**整体回滚**。

**组合解析顺序（P0-2）**：输入校验 → 硬依赖闭包 → 冲突 → `Provides`/`Consumes` → 驱动 → 迁移 schema 依赖 → 表/迁移 → 运行时服务/配置 → 中间件 → plan。诊断统一含 `code`/`capability`/`path`/`message`/`fix` 并稳定排序；非法组合在写目标目录前失败。`jimu check <dir>` 用同一套预检只读检查已有项目，合法退出 0、非法退出非 0。

**Provenance**：`.jimu/manifest.json` 为 schema v3，`framework.commit` 来自 `git rev-parse --verify HEAD^{commit}`（支持普通 clone 与 worktree），同时记录 `framework.dirty`、官方模板的 `selection.template_version`、覆盖全部 `generated_files` 的 `generated_file_sha256`，以及 `composition` 组合块（契约版本、requested/resolved/auto-added、驱动、validity/support）。`--dry-run` 与 `jimu check` 的 support 读取 `composition.support`；schema v2 只读用于升级，schema v1 fail-closed。

## `jimu upgrade <dir>`

默认执行显式升级事务：以 manifest 的 `generated_file_sha256` 为基线做三方比较，冲突在写盘前中止；变换文件先备份再逐个安装，受支持配置以现有段值为底合并；随后在项目里跑 tidy/build/test，失败按原字节回滚；成功后更新 manifest，重复升级幂等。`--dry-run` 打印完整报告；`--check` 复用报告并在有冲突时退出非 0。两种检查模式都不写项目目录，`--no-tidy` 只跳过临时规划副本中的 tidy，升级后验证始终执行 tidy/build/test。

**生成物是子集**：内核目录原样复制；能力按复制集（声明集 ∪ 编译闭包 ∪ schema 依赖的迁移携带目录 ∪ 内核 domain 依赖）落地；`internal/profiles/{registry,<shape>,active}`、`internal/capabilities/catalog/*`、`configs/*.yaml`、`Makefile`/`Dockerfile`/`scripts/check_profiles.sh` 按能力集**渲染**（单形态，无 `PROFILE=`、无 overlay）；测试树按「逐文件 import 可满足性 + 资产依赖」裁剪（规则写入 `.jimu/manifest.json` 的 `prune` 段，文件清单写入 `generated_files`）。manifest 只支持 JSON，路径必须为相对路径。

`basic-api` 是独立的零外部服务模板：只生成 HTTP/配置/日志/健康检查/示例路由，不包含能力目录、Compose 或数据库/Redis 配置。v0.4.0 已移除受跟踪的 `.env.example`，basic-api 不重新引入它；配置与 `HTTP_*`/`MANAGEMENT_*`/`LOG_LEVEL` 环境变量写在生成项目 README。

## `jimu capability add <name>`

在**已生成项目**上增量追加能力：确定性重渲染、幂等（同一集合重跑逐字节等价）；`configs/*.yaml` 与 `deploy/helm/values.yaml` 以项目现有文件为底补齐新增段，**不覆盖**用户手改的值。若项目里已有 `--report` 产物，成功后同批重算刷新（本来没有则不创建）。

## `jimu capability create <name>`

**只**在本仓 `internal/capabilities/<name>/` 落典型 CRUD 骨架（`Descriptor` + `Wire` + 按职责分层代码 + 迁移），**不改任何注册点** —— 注册见 [capability.md](capability.md)。旧 `module create` 已移除；`capability add` 是向已生成项目追加已有能力，职责不同。

## 生成项目与本仓的门禁关系

- 生成项目的 `Makefile`/`Dockerfile`/`scripts/check_profiles.sh` 由 `templates/project/*.tmpl` 渲染，**不受框架 Makefile 改动影响**
- 本仓的模板漂移门禁是 `make check-templates`（生成最小项目并真构建 + 跑生成项目门禁），真实生成项目的重型用例由 `JIMU_HEAVY_MATRIX=1` 门控（本地 `make test-scaffold-matrix`）；重型入口同时执行 `budgets.json` 预算检查、`basic-api` smoke 和 `auth-api` 新空库 Compose 端到端（需 Docker Compose）

## 验收命令与期望输出

```bash
./bin/jimu new /tmp/sk-demo --template=basic-api --module=example.com/demo --dry-run   # 只打印计划
./bin/jimu new /tmp/sk-demo --template=auth-api --module=example.com/demo --report     # 单租户认证模板
./bin/jimu check /tmp/sk-demo                                                          # 只读组合检查（退出 0）
cd /tmp/sk-demo && go build ./... && go run ./tools/checkcapabilities && go test ./... -count=1
grep -rn '"jimu/' /tmp/sk-demo --include='*.go'      # 期望 0 命中（module 已重写）
make check-templates                                 # 本仓模板漂移门禁
make test-scaffold-matrix                            # 官方模板预算 + basic smoke + auth 空库 E2E
```

## 常见红与定位

| 症状 | 定位 |
|---|---|
| `frameworkRoot` 找不到源根（错误里带层数与起点） | 从过深目录执行：向上找 `module jimu` 源根有以下上限（`internal/config` 的 `SearchDepthUp`，与找 `configs/` 共用），超限即 fail-closed |
| `go mod tidy` 失败 | 生成的 `go.mod` 与选中能力的 import 不一致；看 `--no-tidy` 是否能构建，定位是 tidy 还是复制集问题 |
| 生成项目门禁红 | 模板漂移：`make check-templates` 复现，改 `templates/**` 或生成器的复制集 |
| 重生成覆盖了手改文件 | 手改的文件不在 manifest 的 `generated_files` 清单里，或用了 `--force`；报告与动作清单以 `.jimu/manifest.json` 为准 |
