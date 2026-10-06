# 门禁与报告排障

**何时读**：任一能力门禁红了、`compose-report-check` 漂移、或不确定某条断言在管什么时。

**权威口径**：README「[质量门禁](../../../README.md#质量门禁)」、「[Makefile 命令](../../../README.md#makefile-命令)」、[docs/profiles/compose-report.md](../../../docs/profiles/compose-report.md)。

## `make check-capabilities`

输出中的每行 `✅ check-capabilities: …` 对应一类断言；具体实现位于 `tools/checkcapabilities`：

| # | 断言 | 管什么 | 红了看 |
|---|---|---|---|
| ① | 能力自描述与迁移归属一致 | `Descriptor.Owns` ↔ mysql 迁移：单表唯一归属、无孤儿表（建了表没人声明）、无未声明的建表 | [migration.md](migration.md) |
| ② | 驱动可用集/选中集/import 闭包一致 | `Descriptor.Drivers` ↔ 驱动目录存在；能力核心生产闭包零驱动包、零重型依赖；形态选中集 == 该形态**生产** import 闭包（集合比较，不比顺序） | [profile-driver.md](profile-driver.md) |
| ③ | 形态生产代码只 import 已声明的驱动 | 形态代码的 capabilities 子包 import 只能是能力根包或已声明的驱动包 | [profile-driver.md](profile-driver.md) |
| ④ | 唯一入口与选点包只 import 一个形态 | `cmd/server` 只 import `internal/assembly` + 选点包；`internal/profiles/active` **恰好**选一个形态；两者都不得 import `internal/profiles/registry` | [capability.md](capability.md) / [profile-driver.md](profile-driver.md) |
| ⑤ | 资产归属唯一且无未声明资产 | 声明路径非空/存在/在资产根内（`deploy/`、`docs/openapi/`）；同一路径不被两个所有者声明（归一化比较）；资产根下每个文件都有所有者；每个形态覆盖全部内核资产组 | README「[非代码资产归属（P2.6）](../../../README.md#非代码资产归属p26)」 |
| ⑥ | 能力树仅 catalog 允许跨能力 import | 能力生产、测试和驱动子包只依赖本能力或 `internal/contract`；`catalog` 是唯一组合根 | README「[开发规范](../../../README.md#开发规范)」 |
| ⑦ | 生成器核心只经 `frameworkmanifest` 读取框架内部包 | `tools/generator` 的 manifest、plan、render、workspace、report 等核心子包（含测试）不得 import `internal/capabilities`、`internal/profiles` 或 `internal/contract`；`frameworkmanifest` 是唯一适配层。生成项目不携带 `tools/generator`，自动跳过此仓库侧扫描 | README「[生成项目](../../../README.md#生成项目jimu-new--jimu-capability-add)」 |

## `make profiles-check`：形态构建 + golden 依赖闭包

- 用 `tools/profileoverlay` 把选点文件替换为「只选该形态」的版本（产物落在 gitignored 的 `.overlay/<profile>/`，**不动工作区**），再逐个构建
- **golden 依赖闭包**：`scripts/check_profiles.sh` 里每个形态的 `EXPECTED_<name>` / `FORBIDDEN_<name>` 逐值锁定能力根包集合 —— 新增/删除能力必须同步更新它，这是「层②裁剪不能被悄悄改胖」的刹车
- 未设 `JIMU_PROFILES_SMOKE=1` 时，启动与 `/readyz` 冒烟逐形态打印 `SKIP`；**SKIP 不等于通过**，要真验证启动得设该变量（需 DB+Redis）

## `make compose-report-check`：报告漂移

- 重新实测并比对入库报告 [docs/profiles/compose-report.md](../../../docs/profiles/compose-report.md) 的**平台无关列**：路由 / 迁移 / 表 / 本仓闭包文件数与代码行 / 重型依赖 / 直接依赖数
- **二进制大小列是平台相关的**（darwin/arm64 与 linux/amd64 不同），掩码后比对、只打印到日志归档 —— 不要让本地平台差异去改入库值
- 红了先看首个差异行属于哪一列：路由/迁移/表变化 → 能力或形态改动；闭包文件数/代码行变化 → 本仓代码改动；重型依赖 → 驱动集合变化

## 模板与重型矩阵

- `make check-templates`：用生成器在临时目录生成最小项目并真构建 + 跑生成项目自己的 `check-capabilities`
- 所有「真实生成项目 + `go build/vet/test/run`」的用例由 **`JIMU_HEAVY_MATRIX=1`** 门控（只认字面量 `1`；`-short`/未设都跳过）；本地入口 `make test-scaffold-matrix`
- CI 侧在独立 workflow `.github/workflows/ci-scaffold.yml` 的 `Scaffold Matrix` job，**默认不在 PR 上跑**：只在 `push release/**`、tag、`workflow_dispatch`、PR 打 `heavy-ci` 标签（只认 `labeled` 事件）时运行

## 聚合目标成员

```bash
make check-capabilities profiles-check compose-report-check   # CI 的 Capability Gates job（不需要 DB/Redis）
make ci                # fmt-check vet lint check-log-usage + 上述三道 + test-cover/coverage/race/swagger/smoke/build/govulncheck + test-scaffold-matrix
make release-check     # fmt-check vet check-log-usage + 上述三道 + test + govulncheck + compose-check + test-scaffold-matrix
```

- `make check-skills`（本 skill 自身的契约校验）**不在**任何聚合目标里 —— 刻意如此
- `make compose-check` 需要 Docker：隔离 Compose 运行时安全校验 + API 契约冒烟

## 常见红与定位

| 症状 | 定位 |
|---|---|
| 闭包出现没声明的能力 | 形态清单、`scripts/check_profiles.sh` golden、或某能力的 import 变了（检查能力树是否残留跨能力 import，再核对声明依赖） |
| `compose-report-check` 报本仓代码行变化 | 你改了本仓 Go 代码 → 跑 `make compose-report` 重新入库（确认差异只来自本次改动） |
| `profiles-check` 全部形态构建失败 | 先用 `go build ./...` 排除基础编译问题，再看 overlay 是否生成成功 |
| `check-templates` 红 | 模板/复制口径与真实结构漂移：看 `tools/generator` 的复制集与 `templates/**` |
