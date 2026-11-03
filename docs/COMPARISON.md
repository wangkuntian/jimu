# 与常见方案的区别

这份比较讨论**定位和交付方式**，不是功能打分。Jimu 不试图替代通用 Web 框架或完整微服务平台。

## 一句话定位

Jimu 是 Go 后端项目的脚手架/工程基线：从一个可运行项目开始，按需组合生产级能力；代码归使用者，升级通过 manifest 显式检查。

## 对比

| 方案 | 它主要解决什么 | 与 Jimu 的关系 |
|---|---|---|
| [Gin](https://gin-gonic.com/) / [Echo](https://echo.labstack.com/) | HTTP 路由、中间件和请求上下文 | 它们是 Web 框架；Jimu 生成的 `basic-api` 使用 Gin，但 Jimu 不是它们的替代品 |
| [go-zero](https://go-zero.dev/) | 微服务开发框架、代码生成和治理能力 | Jimu 提供独立项目基线、模板和显式升级；不承诺等价的微服务治理全家桶 |
| [Kratos](https://go-kratos.dev/) | 微服务框架和工程布局 | Jimu 不替换 Kratos；选择 Jimu 意味着接受更窄的脚手架边界 |
| 整仓模板/示例仓库 | 复制一套示例代码开始 | Jimu 生成独立 module，并记录 manifest、支持 `jimu check` / `jimu upgrade`；不是复制后无来源信息的模板 |

## 不适用

- 需要低代码、PaaS、控制面、应用托管或 Kubernetes Operator。
- 需要 Jimu SDK 作为业务运行时；生成项目不依赖 Jimu 运行时。
- 需要任意 experimental 能力组合的生产兼容承诺；`--with` 仍标记为 experimental。
- 需要开箱即用的多租户 SaaS 或全栈模板；当前路线只维护两个 certified 模板。

如果你只是需要一个 HTTP 路由库，直接使用 Gin/Echo 更简单；如果你需要完整的微服务治理平台，应评估 go-zero/Kratos 等方案。Jimu 适合希望保留代码所有权、同时获得可检查升级路径的 Go 后端团队。
