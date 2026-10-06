# v0.3.3 生成器根包收口设计

## 目标

继续整理 `tools/generator`，保持 `jimu/tools/generator` 的公共导入路径和现有 CLI 行为不变，同时把可以独立理解的实现迁入已有职责目录，避免根包继续承载完整模块生成器、通用文件操作和模板引擎。

## 方案

- `tools/generator/module` 独立承载 `capability create` 的模块骨架生成、回滚逻辑和专属模板。
- `tools/generator/support` 承载目录复制、模块路径重写、`go mod tidy` 和生成项目自检等框架无关原语。
- `tools/generator/render` 承载模板 `embed`、模板读取和文本渲染；原有 manifest 渲染实现继续留在该包。
- 根包保留稳定的 CLI facade 和少量兼容包装，现有调用方无需改 import 路径。

## 边界

本轮不改变 manifest schema、生成项目文件内容、CLI 参数或 v0.3.2 兼容策略；不把框架能力选择、profile 和 descriptor 读取迁入框架无关包。每次迁移都用原有测试和生成器矩阵验证行为不变。

## 验证

每个迁移任务先运行对应聚焦测试，再运行 `go test ./tools/generator/... -count=1`；最终运行 `go test ./... -count=1`、`make check-capabilities`、`make profiles-check`、`make compose-report-check`、`make check-skills` 和 `git diff --check`。
