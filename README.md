# Jimu

Jimu 是面向生产团队的 **Go 后端项目脚手架/工程基线**：

> 从一个可运行的 Go 服务开始，按需组合生产级能力；代码归使用者，升级可检查、可回滚。

Jimu 不是运行时框架，也不要求业务代码依赖 Jimu SDK。`jimu new` 生成的是独立 Go module，生成后由使用者拥有和维护。

## 适合谁

适合：

- 需要一个独立 Go API/BFF，希望从健康的工程基线开始。
- 希望保留代码所有权，同时有明确的 `check` / `upgrade` 路径。
- 需要认证、租户、队列、审计或可观测性等生产能力的团队。

不适合：

- 需要低代码、PaaS、应用托管或 Kubernetes Operator。
- 需要任意 experimental 能力组合的生产兼容承诺。
- 需要开箱即用的多租户 SaaS 或全栈模板。

## 选择预设

| 预设 | 能力集 | 外部依赖 |
|---|---|---|
| `minimal` | 0 能力，只含最小内核 | 无数据库、Redis 或消息队列 |
| `auth` | user + access + auth（含 tenant 闭包） | MySQL 8.4、Redis 7 |
| `full` | 全部 24 个能力 | MySQL 8.4、Redis 7、Kafka、RabbitMQ 或 S3（按驱动） |

默认预设是 `minimal`。`full` 首发支持级别是 `validated`，不是 `certified`。
预设只是裁剪起点：生成后可用 `jimu capability add` 继续组合能力。
追加带驱动的能力时用 `--driver=<name>` 指定实现；省略则取该能力声明的首个驱动。
`--with` 是 experimental 专家模式，不承诺组合级生产兼容。

生成时按需编译底层组件：`--db=mysql|postgres`（默认 `mysql`）只带所选数据库驱动；
`minimal` 产物不含 Redis、Casbin 与 OTel 导出器，server 二进制约 27 MB。

> `basic-api` 与 `auth-api` 已在 v0.10.0 退役。由它们生成的项目执行 `jimu upgrade` 会硬失败，
> 需要按 `minimal`/`auth` 重新生成。

## 安装 Release CLI

从 GitHub Release 下载对应平台的 `jimu` 二进制：

```bash
# 以 Linux amd64 为例；macOS 将 OS/ARCH 改为 darwin/arm64
OS=linux ARCH=amd64; curl -fL "https://github.com/wangkuntian/jimu/releases/download/v0.7.1/jimu-${OS}-${ARCH}" -o jimu
chmod +x jimu
```

| 操作系统 | 架构 |
|---|---|
| Linux | amd64 / arm64 |
| macOS | amd64 / arm64 |

Release asset 名为 `jimu-<os>-<arch>`；不支持 Windows。

Release 同时提供 `template-bundle.json` 和 SHA-256 checksum 文件。需要时先校验下载，再把 `jimu` 放入 `PATH`。

## 快速开始：minimal

```bash
OS=linux ARCH=amd64; curl -fL "https://github.com/wangkuntian/jimu/releases/download/v0.7.1/jimu-${OS}-${ARCH}" -o jimu && chmod +x jimu
./jimu new demo --template=minimal
cd demo && make dev
```

服务启动后可访问：

- `GET http://127.0.0.1:8080/api/v1/ping`
- `GET http://127.0.0.1:9090/livez`
- `GET http://127.0.0.1:9090/readyz`

`minimal` 生成项目不需要数据库、Redis 或其他外部服务。

生成项目默认编译 MySQL 驱动。需要 PostgreSQL 时加 `--db=postgres`：
只编译所选驱动与对应方言的迁移，另一套驱动不会进入产物。

```bash
./jimu new demo --template=auth --db=postgres
```

## 快速开始：auth

```bash
OS=linux ARCH=amd64; curl -fL "https://github.com/wangkuntian/jimu/releases/download/v0.7.1/jimu-${OS}-${ARCH}" -o jimu && chmod +x jimu
./jimu new demo --template=auth
cd demo
make migrate
make seed
make dev
```

`auth` 预设不生成 `docker-compose.yml`；数据库与 Redis 需自行提供，连接信息通过 `DB_*`
与 `REDIS_*` 环境变量覆盖。首次启动前运行 `make migrate` 与 `make seed`，
seed 会读取 `ADMIN_PASSWORD`（或 `ADMIN_PASSWORD_FILE`）创建管理员账号。

## 检查与升级

```bash
./jimu version
./jimu check demo
./jimu check demo --deep
./jimu check demo --json
./jimu upgrade demo --check
./jimu upgrade demo --check --json --diff
./jimu upgrade demo --dry-run
./jimu upgrade demo
```

`jimu check` 默认只读检查 manifest、能力组合、受管文件漂移、目标版本、升级状态和下一步命令。受管文件漂移、可用升级或 conflict 只产生 warning；非法 manifest/组合退出非 0。`--deep` 在私有临时副本中串行执行配置校验、`go build ./...` 和 `go test ./...`，不会改写项目。

`upgrade --check --json --diff` 输出 schema `v1` 的确定性机器结果和 local→target 内容差异；二进制文件只报告存在状态、哈希和分类。升级只修改 manifest 管理的文件；数据库 schema 不在自动升级范围内。详见[升级](docs/UPGRADING.md)。

`jimu version` 同时显示 CLI、template bundle、template contract 和 commit。

## 文档

- [能力目录](docs/CAPABILITIES.md)
- [配置](docs/CONFIGURATION.md)
- [运维](docs/OPERATIONS.md)
- [升级](docs/UPGRADING.md)
- [本地实验指标](docs/EXPERIMENTS.md)
- [本地全栈 demo](docs/DEMO.md)
- [真实项目升级验证](docs/UPGRADE-VALIDATION.md)
- [与常见方案的区别](docs/COMPARISON.md)
- [生产上线与模板检查清单](docs/PRODUCTION.md)

模板预算、兼容窗口和发布阻断规则见[模板契约 v0.1](docs/templates/v1/README.md)。生成器、manifest 和能力组合的稳定设计见[设计文档索引](docs/design/README.md)。

## 从本仓开发

本仓自身是完整服务，保留 `full`/`minimal` 两个内部 profile、运维与集成测试：

```bash
make init
make up
make check
```

验证发布 CLI 的仓库外生成链路（需要 Docker）：

```bash
make clean-room-e2e
```

贡献流程见[CONTRIBUTING](docs/CONTRIBUTING.md)；默认分支、发布分支和 tag 规则以该文档为准。

## License

MIT
