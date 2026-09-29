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
./bin/jimu new ./myapp --profile=minimal --module=example.com/myapp --report   # 形态清单
./bin/jimu new ./myapp --with=user,access,queue --shape=app                   # 能力名（硬依赖闭包 + 拓扑序）
./bin/jimu new ./myapp --with=queue:kafka --dry-run                           # 只打印计划，不落盘
```

| 参数 | 说明 |
|---|---|
| `--profile=<name>` / `--with=<cap>[:<drv>][,…]` | 能力集二选一（互斥）。`--with` 的名字取自 **catalog 18 ∪ Ungated 7**（`apidocs`/`storage`/`notification`/`retention`/`ws`/`grpc`/`encryption`）；驱动默认取该能力 `Descriptor.Drivers` 首项（`queue→redis`、`storage→local`、`dataops→csv`），用 `<cap>:<drv>` 覆盖 |
| `--shape=<name>` | 生成项目的形态名（缺省由能力集推导） |
| `--module=<path>` | 重写 `go.mod` 的 module 指令与文本里的 `jimu/` 前缀；**不改**框架运行期名字 / CLI 名 / 镜像名 / protobuf 描述符 |
| `--dry-run` | 只打印计划，不落盘 |
| `--force` | 只覆盖带 `.jimu-generated` 标记的产物（识别依据是标记，不是时间戳） |
| `--no-tidy` | 跳过 `go mod tidy`（`go.mod` 保持框架依赖集） |
| `--report` | 写 `<dir>/docs/profiles/generated-report.md`（文件数 / 闭包 Go 文件数与代码行 / 直接依赖数 / 迁移数 / 表数 / 路由数 / 重型依赖 / 资产数） |

**默认自检**：生成后跑 `go build ./...` 与生成项目自己的 `check-capabilities`，任一失败**整体回滚**。

**生成物是子集**：内核目录原样复制；能力按复制集（声明集 ∪ 编译闭包 ∪ schema 依赖的迁移携带目录 ∪ 内核 domain 依赖）落地；`internal/profiles/{registry,<shape>,active}`、`internal/capabilities/catalog/*`、`configs/*.yaml`、`Makefile`/`Dockerfile`/`scripts/check_profiles.sh` 按能力集**渲染**（单形态，无 `PROFILE=`、无 overlay）；测试树按「逐文件 import 可满足性 + 资产依赖」裁剪（清单记入 `.jimu-generated`，该 dot 文件不参与全树扫描）。

## `jimu capability add <name>`

在**已生成项目**上增量追加能力：确定性重渲染、幂等（同一集合重跑逐字节等价）；`configs/*.yaml` 与 `deploy/helm/values.yaml` 以项目现有文件为底补齐新增段，**不覆盖**用户手改的值。若项目里已有 `--report` 产物，成功后同批重算刷新（本来没有则不创建）。

## `jimu module create <name>`

**只**在本仓 `internal/capabilities/<name>/` 落骨架（`module.go` + `domain/` + `application/` + `infrastructure/` + `interfaces/` + 迁移占位），**不改任何注册点** —— 注册见 [capability.md](capability.md)。

## 生成项目与本仓的门禁关系

- 生成项目的 `Makefile`/`Dockerfile`/`scripts/check_profiles.sh` 由 `templates/project/*.tmpl` 渲染，**不受框架 Makefile 改动影响**
- 本仓的模板漂移门禁是 `make check-templates`（生成最小项目并真构建 + 跑生成项目门禁），真实生成项目的重型用例由 `JIMU_HEAVY_MATRIX=1` 门控（本地 `make test-scaffold-matrix`）

## 验收命令与期望输出

```bash
./bin/jimu new /tmp/sk-demo --profile=minimal --module=example.com/demo --dry-run   # 只打印计划
./bin/jimu new /tmp/sk-demo --profile=minimal --module=example.com/demo --report    # 默认 tidy + 自检
cd /tmp/sk-demo && go build ./... && go run ./tools/checkcapabilities && go test ./... -count=1
grep -rn '"jimu/' /tmp/sk-demo --include='*.go'      # 期望 0 命中（module 已重写）
make check-templates                                 # 本仓模板漂移门禁
```

## 常见红与定位

| 症状 | 定位 |
|---|---|
| `frameworkRoot` 找不到源根（错误里带层数与起点） | 从过深目录执行：向上找 `module jimu` 源根有以下上限（`internal/config` 的 `SearchDepthUp`，与找 `configs/` 共用），超限即 fail-closed |
| `go mod tidy` 失败 | 生成的 `go.mod` 与选中能力的 import 不一致；看 `--no-tidy` 是否能构建，定位是 tidy 还是复制集问题 |
| 生成项目门禁红 | 模板漂移：`make check-templates` 复现，改 `templates/**` 或生成器的复制集 |
| 重生成覆盖了手改文件 | 手改的文件不在 `.jimu-generated` 清单里，或用了 `--force`；报告与清单以 `.jimu-generated` 为准 |
