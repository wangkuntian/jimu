# 将 template bundle 内嵌到 Release CLI

Release CLI 必须在 Jimu checkout 之外生成和升级项目，包括 experimental compositions。它将按平台发布单个二进制，并通过根包 `//go:embed` 内嵌版本锁定的 template bundle，而不是下载独立 bundle 或读取相邻 source directory。这样 CLI 与 bundle 版本天然一致，避免 cache corruption、download failure 和 installation path ambiguity。

根包嵌入显式列出的 tracked source、template、migration 和 asset paths。`bundle-check` gate 阻止 untracked files 进入 bundle。Release tag 同时提供 CLI version 和 template bundle version；生成的 manifest 记录二者及 source commit。External source root 仅作为隐藏的开发/测试路径，必须在写入任何项目文件前提供匹配的 `template-bundle.json`。

## 考虑过的方案

- **由 CLI 下载并缓存独立 template bundle**：二进制更小，但引入网络、完整性、cache corruption 和 version drift 等失败模式。
- **在 Release archive 中同时放置 CLI 和相邻 template directory**：打包简单，但依赖用户保留目录结构，单独复制 binary 后就不完整。
- **构建时生成 archive**：可复现且紧凑，但引入额外的 generated build step 和第二套 source-of-truth 流程。

## 影响

- 每个 CLI binary 会增加约 tracked bundle 大小（当前候选约 3.5 MiB），对 P0 可接受。
- Embedded mode 下的 framework source provenance 来自 build metadata，而不是 `git`；generator 不能要求 checkout。
- 本地构建若无 gate，可能嵌入 untracked files，因此 bundle gate 是 release 和 CI contract 的一部分。
