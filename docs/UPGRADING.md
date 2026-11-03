# 升级

`jimu upgrade` 是显式升级入口，不会在生成后自动覆盖用户项目。

## 基本命令

```bash
jimu upgrade <dir> --dry-run   # 只读三方比较，输出完整分类
jimu upgrade <dir> --check     # 有 conflict 时退出非 0，适合 CI
jimu upgrade <dir>             # 备份、安装、tidy/build/test，失败回滚
```

默认升级会读取 `.jimu/manifest.json` 中的 `generated_file_sha256` 基线，把选择集按当前 template bundle 重新渲染到临时目录，再比较：

| 分类 | 判定 | 策略 |
|---|---|---|
| `untouched` | 本地与目标都等于基线 | 不动作 |
| `modified` | 目标变化，本地仍等于基线 | 安装目标文件 |
| `conflict` | 本地偏离基线（修改、删除或路径被占用） | 保留本地，写前中止 |
| `new` | 目标新增且本地不存在 | 安装 |
| `deleted` | 目标不再生成 | 保留本地副本，不自动删除 |

升级只改动生成器管理的文件；用户自有文件和 manifest 之外的路径不触碰。`configs/*.yaml` 是整文件模板，手改会判为 `conflict`；`--with` 项目的配置段按段合并。

## CI 配方

在生成项目的 CI 中加入只读检查，避免有冲突时合并：

```yaml
- name: Check Jimu project upgrade
  run: jimu upgrade . --check
```

`jimu` 应使用与项目 manifest 兼容的 Release CLI，并锁定版本；不要使用 `latest`。

## 数据库 schema

`jimu upgrade` 只更新生成项目中的代码、配置和迁移文件，**不会自动修改正在运行的数据库 schema**。数据库迁移仍需按项目自己的发布流程执行，并在升级前备份。

## 支持边界

当前支持窗口、模板契约和兼容性承诺见[模板契约 v0.1](templates/v1/README.md)。跨大版本升级遇到 `conflict` 时先保留本地改动，人工合并后重新运行 `--check`。
