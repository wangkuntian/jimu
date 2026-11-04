# 本地实验指标

本地实验用于建立项目生命周期工具的可重复耗时与稳定性基线。采集由操作者显式启动，结果只写入指定文件，不会上传，也不包含真实项目标识或业务数据。

## 前置条件

- Go 版本满足 `go.mod` 声明。
- Python 3.9 或更高版本。
- 当前环境无需外部数据库或 Redis；默认实验只使用 `basic-api`。
- 在 Jimu checkout 中先构建带目标版本戳的开发 CLI：

```bash
make build-cli APP_VERSION=v0.8.0
```

## 采集

```bash
python3 scripts/experiment_v0_8_0.py \
  --jimu bin/jimu-cli \
  --framework-root . \
  --runs 3 \
  --version v0.8.0 \
  --output docs/experiments/v0.8.0.json
```

脚本先执行内建检查：

```bash
python3 scripts/experiment_v0_8_0.py --self-test
```

少于三次运行会被拒绝。每次运行在独立临时目录中生成 `basic-api`、构建并启动服务，等待管理端口 `/readyz` 返回成功，然后依次测量 `jimu check --json`、`jimu check --json --deep` 和 `jimu upgrade --check --json --diff`。

## 固定指标

| 指标 | 定义 |
|---|---|
| `new_seconds` | `jimu new` 从启动到成功返回 |
| `new_to_ready_seconds` | 从开始生成到首次通过 `/readyz` |
| `check_fast_seconds` | `jimu check --json` fast 检查 |
| `check_deep_seconds` | `jimu check --json --deep` 含配置、build、test |
| `upgrade_check_json_diff_seconds` | `jimu upgrade --check --json --diff` |

汇总记录中位数、最小值、最大值和波动范围，同时记录平台、架构、CPU 数、Python、Go、CLI 版本和源码提交。

## 确定性与隐私

- `check` 和 `upgrade` JSON 会在同一项目上重复执行并逐字节比较。
- `generated_at` 和耗时是实验文件声明的易变字段；比较机器诊断时必须排除它们。
- 脚本不保存生成项目内容、日志正文、业务数据或用户输入。
- 脚本只写 `--output` 指定的本地文件，不包含上传逻辑。
- 错误文本会移除临时目录、仓库根目录和用户主目录。

## 版本基线

首版基线见 [v0.8.0](experiments/v0.8.0.md)。后续版本使用同一模板、命令、指标定义和至少三次运行的口径更新，并保留上一版本文件。

每个版本至少检查一次异常列表。失败、确定性下降或明显离群值必须分流为带编号的需求、缺陷或文档问题；仅展示图表或数字不算完成。

当前口径只覆盖本机 `basic-api` 路径，不替代干净机器 CI、真实项目升级验证或生产环境性能承诺。
