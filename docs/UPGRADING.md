# 升级

`jimu upgrade` 是显式升级入口，不会在生成后自动覆盖用户项目。

## 支持窗口

| 项目类型 | 是否支持升级 | 说明 |
|---|---|---|
| `v0.5.0` 及之后生成的官方模板项目（manifest schema v2/v3） | 支持 | 使用与项目兼容的 Release CLI；升级前先运行 `--check` |
| `v0.4.x` 或 schema v1 项目 | 不支持直接升级 | 用目标版本重新生成到新目录，人工 diff 后迁移 |
| `--with` 组合与 `capability add` 项目 | 不承诺 | 保持 `experimental`；只保证显式可检查，不承诺组合稳定性 |
| 已有数据库 schema | 不自动迁移 | 升级只更新生成项目文件；数据库迁移按项目发布流程单独执行 |

安全修复只保证当前 0.7.x 的最新 patch。旧版本不提供安全 backport；需要修复时先升级到最新 patch。策略与取舍见 [ADR 0003](adr/0003-upgrade-support-window.md)。

跨大版本升级仍建议先 `--check`，再在分支或临时副本上实际执行一次。遇到 `conflict` 时保留本地改动，人工合并后重新检查；不要用 `--force` 或其他方式绕过冲突。

## 基本命令

```bash
jimu upgrade <dir> --dry-run   # 只读三方比较，输出完整分类
jimu upgrade <dir> --check     # 有 conflict 时退出非 0，适合 CI
jimu upgrade <dir>             # 备份、安装、tidy/build/test，失败回滚
```

默认升级会读取 `.jimu/manifest.json` 中的 `generated_file_sha256` 基线，把选择集按当前 template bundle 重新渲染到临时目录，再比较：

| 分类 | 判定 | 用户处理方式 |
|---|---|---|
| `untouched` | 本地与目标都等于基线 | 不动作；继续使用本地文件 |
| `modified` | 目标变化，本地仍等于基线 | 检查报告后可让事务安装目标文件 |
| `conflict` | 本地偏离基线（修改、删除或路径被占用） | 先保留本地，手动合并或恢复基线后重新运行 `--check` |
| `new` | 目标新增且本地不存在 | 检查新文件后让事务安装 |
| `deleted` | 目标不再生成 | 保留本地副本，不自动删除；确认不再需要后自行清理 |

升级只改动生成器管理的文件；用户自有文件和 manifest 之外的路径不触碰。`configs/*.yaml` 是整文件模板，手改会判为 `conflict`；`--with` 项目的配置段按段合并。

## CI 配方

### GitHub Actions

在生成项目的 CI 中锁定 Release CLI 版本，并加入只读检查。`--check` 在存在 `conflict` 时退出非 0；没有冲突但可升级时仍返回 0，便于合并前发现风险而不阻塞所有构建。

```yaml
name: jimu-upgrade-check

on:
  pull_request:

jobs:
  upgrade-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Jimu CLI
        env:
          JIMU_VERSION: v0.7.1
          JIMU_OS: linux
          JIMU_ARCH: amd64
        run: |
          set -euo pipefail
          base="https://github.com/wangkuntian/jimu/releases/download/${JIMU_VERSION}"
          curl -fL "${base}/jimu-${JIMU_OS}-${JIMU_ARCH}" -o jimu
          curl -fL "${base}/jimu-${JIMU_VERSION}-checksums.txt" -o checksums.txt
          grep " jimu-${JIMU_OS}-${JIMU_ARCH}$" checksums.txt | sha256sum -c -
          chmod +x jimu

      - name: Check project upgrade
        run: ./jimu upgrade . --check
```

Windows 不受支持；macOS runner 使用 `darwin` 和 `amd64`/`arm64`。不要把 `latest` 作为 CLI 版本。

### Makefile

如果团队已有 CLI 缓存目录，可以在生成项目中加入：

```make
JIMU ?= jimu

.PHONY: jimu-upgrade-check
jimu-upgrade-check:
	$(JIMU) check .
	$(JIMU) upgrade . --check
```

本地可用 `make jimu-upgrade-check JIMU=/path/to/jimu`。CI 应把缓存目录注入同一个变量，并锁定 CLI 版本。

## 数据库 schema

`jimu upgrade` 只更新生成项目中的代码、配置和迁移文件，**不会自动修改正在运行的数据库 schema**。数据库迁移仍需按项目自己的发布流程执行，并在升级前备份。

`auth-api` 的迁移和 seed 命令由生成项目自己的 CLI 提供：

```bash
./jimu migrate up
./jimu seed
```

升级事务的 `go build` / `go test` 通过不代表线上数据库已经迁移；部署顺序应保持“备份 -> 迁移 -> 启动应用”或项目既定的兼容顺序。

## 安全修复

安全修复进入生成项目的流程、公告格式和演练记录见[安全政策](SECURITY.md#安全修复进入生成项目)。用户应至少执行：

```bash
jimu check <dir>
jimu upgrade <dir> --check
```

`jimu check` 验证 manifest 和组合一致性；`upgrade --check` 判断是否存在需要人工处理的冲突。两者都不写项目目录。

## 支持边界

当前支持窗口、模板契约和兼容性承诺见[模板契约 v0.1](templates/v1/README.md)。跨大版本升级遇到 `conflict` 时先保留本地改动，人工合并后重新运行 `--check`。

真实项目参与跨版本升级验证的资格、证据字段和隐私边界见[真实项目升级验证](UPGRADE-VALIDATION.md)。
