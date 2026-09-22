# 形态编译面报告

> 由 `make compose-report`（`tools/composereport`）生成，**请勿手工编辑**：改动形态组成后
> 重跑该命令并提交本文件。设计依据见[能力可插拔设计](../design/2026-09-18-capability-plugins-design.md) §6.3 / §11。

## 指标口径

| 指标 | 口径 |
|---|---|
| 二进制 | `go build -o <tmp> ./profiles/<name>` 的产物大小 |
| 路由数 | 形态解析集在裸 `gin.Engine` 上 `RegisterHTTP` 后的 `r.Routes()` 条数（不启动监听） |
| 迁移数 | 各 `Descriptor.Migrations` 中 `migrations/mysql/*.sql` 的文件数（postgres 同名同数） |
| 表数 | 各 `Descriptor.Owns` 的并集大小 |
| 本仓 Go 文件 / 代码行 | `golang.org/x/tools/go/packages` 载入 `./profiles/<name>` 的 import 闭包，只统计本模块（`jimu/...`）的非 `_test.go` 文件 |
| 重型依赖 | 同一闭包（含第三方包）命中 `tools/internal/heavydeps` 前缀表的展示名，`-` 表示零 |
| go.mod 直接依赖 | `go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all` 的非空行数（不含主模块 `jimu` 自身） |

「本仓闭包」严格大于「形态组成」：`user`/`auth` 直接 import 了 `outbox`/`queue`/`notification`/`ws` 的
具体类型（`*outbox.Outbox`、`notification.Message`、`outbox.Event`），编译期会链上这些能力包，
但装配期一个都不构造（详见 README「形态（profile）」的编译期脚注）。

## 编译面

| 形态 | 二进制 (MB) | 相对 full | 路由数 | 迁移数 | 表数 | 本仓 Go 文件 | 本仓代码行 | 重型依赖 |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| `full` | 123.1 | 100.0% | 99 | 25 | 23 | 330 | 34477 | amqp091-go, aws-sdk-go-v2, excelize, kafka-go |
| `minimal` | 84.7 | 68.8% | 32 | 7 | 7 | 179 | 17464 | - |
| `saas` | 85.0 | 69.0% | 48 | 13 | 11 | 205 | 20110 | - |
| `enterprise` | 85.3 | 69.3% | 55 | 13 | 11 | 245 | 22906 | - |
| `machine` | 83.3 | 67.7% | 28 | 7 | 6 | 181 | 17740 | - |

## 验收断言

`go test ./tools/composereport/` 的 `TestMinimalCompiledSurfaceIsMateriallySmaller` 按下面的**实测关系**断言
（不写死数字，内核膨胀或能力增减只会让真实的裁剪失效暴露出来）：

- 二进制：`minimal` 是 `full` 的 68.8%（要求 ≤ 85%）
- 路由数：`minimal` 32 < `full` 99
- 表数：`minimal` 7 < `full` 23
- 本仓 Go 文件：`minimal` 179 < `full` 330
- 本仓代码行：`minimal` 17464 < `full` 34477

## 层②边界：go.mod 直接依赖

五个形态的 go.mod 直接依赖数**逐形态完全相同**（各 64 个）。这不是漏测：Go 的依赖裁剪
作用于整个 module —— `go.mod`/`go.sum` 描述 module 而非包，profile 入口包只改变**编进二进制的
包与符号集合**，不改变 module 依赖图。真正让 `go.mod` 变小的是层①（`jimu new` 生成专属 module
并 `go mod tidy`），不是层②。设计原文见 §11「层②的边界」。

## 形态组成（解析集）

| 形态 | 装配的能力（按装配顺序） |
|---|---|
| `full` | encryption storage notification queue outbox breach tenant access user captcha mfa auth passkey audit console oauth apikey dataops feature uploadsec search retention apidocs grpc ws |
| `minimal` | encryption notification access user auth |
| `saas` | encryption notification tenant access user auth audit |
| `enterprise` | encryption storage notification access user auth audit console oauth dataops |
| `machine` | encryption access user apikey grpc |
