# Release CLI 只保留 scaffold 命令

GitHub Releases 中发布的 `jimu` binary 只暴露项目生成与升级命令：`new`、`check`、`upgrade`、`capability add` 和 `version`。`migrate`、`seed`、`config`、`apikey` 等 server operations 保留在 framework 或 generated-project CLI。这样 release artifact 与 Jimu 的 scaffold 定位一致，避免在生成工具中携带 database/runtime configuration，也无需让 Release CLI 依赖 server profile 构建。

## 考虑过的方案

- **发布现有完整 CLI**：代码路径更少，但用户会拿到语义依赖 server configuration 的命令，release artifact 也不再是专注的 project generator。
- **分别发布 scaffold 和 operations binary**：运维边界更清晰，但增加 release assets 和文档负担，且不服务 P0 用户旅程。

## 影响

- Release CLI 需要新的 command entry，而不是直接复用现有 `cmd/cli`。
- Generated project 在需要时保留自己的 operational CLI。
- 文档必须区分 Release CLI 与 framework/generated-project CLI。
