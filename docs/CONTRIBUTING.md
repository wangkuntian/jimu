# 贡献指南

感谢你愿意为 Jimu 贡献力量。本文档帮助你快速上手。

## 开发环境

- Go 1.26+
- Docker 24+（含 Compose v2）
- Air（`go install github.com/air-verse/air@latest`，仅 `make dev` 需要）

```bash
# 克隆
git clone https://github.com/your-org/jimu.git
cd jimu
go mod download

# 启动依赖、迁移/种子并以 Air 热重载
make dev

# 或启动容器化核心栈
make up
```

`make dev` 与 `make up` 都会通过 `scripts/init-secrets.sh` 补齐 `secrets/`，且不会传入 `LEGACY_ENV_FILE`。旧 `.env` 的唯一公开迁移入口是 `make init`：它会检测项目根 `.env`，也可通过 `LEGACY_ENV_FILE` 指定其它旧文件。
若已有 MariaDB 数据卷只缺新增的安全密钥，使用 `make init ALLOW_NEW_SECURITY_KEYS=1`；已有安全密钥文件不会被 `FORCE=1` 轮换。

## 跨仓协作

与 `jimu-web` 的纵向需求按[跨仓协作工作流](design/cross-repo-development.md)执行：

- 后端先完成 API、测试和 Swagger，再固定后端提交供前端同步契约。
- 前端只消费固定契约和生成的 TypeScript 类型，不复制或猜测后端规则。
- 任务开始时检查两个仓库的工作区；未经用户明确指令，不执行 commit 或 push。
- 可复制该文档末尾的任务模板描述跨仓需求，分别汇报两端验证结果。

## 分支策略

采用简化 GitHub Flow：`master` 为唯一长期分支（禁止直接 push，仅接受 release 分支合并），`release/x.y.z` 为集成与发布分支，日常开发从 release 切出、经 PR 合回。

- `feature/<issue>-<slug>` — 新功能
- `fix/<issue>-<slug>` — 缺陷修复
- `hotfix/<issue>-<slug>` — 线上紧急修复
- `dependabot/*` — 自动化分支，人工不得基于它开发

分支名小写，单词用短横线，`slug` 不超过 5 个单词。鼓励带 issue ID。

## Commit 规范

Conventional Commits 轻量格式：

```
type(scope): summary
```

`type` 取值：`feat`、`fix`、`docs`、`test`、`refactor`、`chore`

`scope` 建议：`repo`、`server`、`config`、`auth`、`user`、`role`、`permission`、`logger`、`db`、`http`

规则：

- summary 与正文全部使用英文：小写开头、祈使句、不加句号
- 一个 commit 只包含一个清晰主题；多行 commit 可在正文说明 why 和风险
- 由 `githooks/commit-msg` 强制检查（`make hooks` 安装，即 `core.hooksPath` 指向 `githooks/`）；CI 的 `Lint & Policy` job 兜底；确需跳过用 `git commit --no-verify`

示例：

```
feat(user): add avatar upload endpoint
fix(auth): reject expired refresh token
```

## Pull Request

1. 至少 1 人 review 通过
2. feature/fix/hotfix → release PR 必须通过 `PR Gate`（代码质量、门禁、分片、覆盖率、DB 集成、提交消息、版本日志）与 `Image Gate`（镜像构建、漏洞扫描、容器冒烟）；这些 workflow 只响应 PR，不在合并后的 branch push 上重复运行
3. release/* → master 发布候选 PR 还必须通过一次 `Scaffold Gate`；向 master 直接提交的其它来源分支会在该 required check 中失败。master 有 required checks（`PR Gate`/`Image Gate`/`Scaffold Gate`，`strict` 策略要求分支与 master 同步）；release/* 只有 `non_fast_forward` 规则、无 required checks
4. 合并策略：feature/fix/hotfix → release 用 squash merge；release → master 用 merge commit
5. 合并后删除源分支

## Tag 与发布

- 发布当日从 release 分支合并到 master 后，在 master tip 打 `vMAJOR.MINOR.PATCH` tag（SemVer，无预发布标签）
- 发布候选 PR 合并后，在当前 `master` tip 打 `vMAJOR.MINOR.PATCH` tag；tag workflow 会校验 tag 必须指向当前 `master` tip，然后并行构建 `full` 与 `minimal` 两个形态 × 4 平台的 8 个二进制（`jimu-<profile>-<os>-<arch>`，产物内注入该 tag 作为版本号）并创建 GitHub Release，不重复运行完整 CI 或 `Scaffold Gate`
- 依赖更新：dependabot 的 PR **直接面向 `master`**（默认分支），minor/patch 由 `.github/workflows/dependabot-auto-merge.yml` 自动合并（major 留人工评估）。依赖 PR 与发布候选 PR 走同一套 master 门禁 —— `PR Gate`/`Image Gate`/`Scaffold Gate` 三道都要过，因此升级会被**脚手架矩阵当场验证**，而不是攒到发布候选 PR 才第一次跑。自动合并前还有两道守卫（都在该 workflow 内，只调 GitHub API、不 checkout）：①**实测** master 的 ruleset 仍 require 那三道门，读不到或少了任何一门都不启用（并撤销已挂上的 auto-merge），防止 ruleset 被改动后 `--auto` 在零检查下抢跑；②**镜像 tag 形状断言** —— 拒绝裸数字后缀（`-386`）、平台名后缀（`-amd64`/`-arm64`/`-ppc64le`…）、`nightly`/`dev`/`snapshot` 等前缀与完全没有数字的 tag（`latest`），只放行常规版本号（含 `-alpine`/`-noble` 这类发行版后缀）与 `@sha256:` 摘要 pin。②是 PR #167 的产物：它把 otel collector 从 `0.162.0` 换成**同一版本的 i386 单架构 tag** `0.162.0-386`，dependabot 按 tag 字符串当成升级、三道门禁全绿，于是被自动合并
- 发布流程：`Start Release Branch` workflow（手动触发）**只需输入版本号**，总是从 `master` 切出 `release/<version>`；依赖升级平时已经直接进 master，发布分支天然包含它们，不再有「吸收」步骤
- 来源白名单：`Scaffold Gate` 的来源白名单维护在 `.github/workflows/ci-scaffold.yml`（shards job 的 `head_ref` 守卫与聚合 job 的来源判定），**已同时放行 `release/*`（发布候选）与 `dependabot/*`（依赖升级）**；其余来源一律 fail-closed。`dependabot-auto-merge.yml` 的 `branches`/`base.ref` 亦已改为 `master`。注意这两处是**配套**的：若只改 `dependabot.yml` 的 `target-branch` 而不放行白名单，依赖 PR 会在 `Scaffold Gate` 上红且无法合并（master ruleset 没有 bypass actor）
- 历史说明：v0.3.5 曾引入常驻分支 `dependency-updates` 攒批依赖升级；v0.3.6 先移除了它的同步自动化（`sync-dependency-branch.yml`），并在本版本重切时连同该分支与其 ruleset 一并删除。移除原因：该分支必须 require `PR Gate`（否则 `--auto` 会抢跑），而 required check **同时拦截直推** ⇒ 同步只能走 PR；`GITHUB_TOKEN` 默认不允许创建 PR，即使允许，用它创建的 PR 也不会触发工作流 ⇒ 自动化无解；且 `Start Release Branch` 从它切出会漏掉 master 的提交。现在只有 `master` 一条长期分支
- `make release-check` 仍可在本地发布前人工运行；tag 与 release notes 同步推送；不发布未经 tag 的 commit
- 回滚：master 不接受 force push，用 revert commit 或新 hotfix PR；release 分支回滚切 hotfix 分支修复后重复合并流程

## Release Note

发布说明按 release 版本号存放于 `docs/releases/<version>.md`（如 `docs/releases/v0.1.0.md`），同一文件既是版本 changelog 也是 GitHub Release body（`release.yml` 发布时读取）。改动源码的 PR 必须同步更新对应版本文件（CI 强制检查）。

每个 release note 按以下顺序组织：

```markdown
# v0.1.0

## 亮点

- 一句话说明本版本最重要的变化。

## 新增

- 新增能力。

## 变更

- 行为、流程、配置或文档变化。

## 修复

- 修复项。

## 移除

- 删除项。

## 验证

- 发布前跑过的关键命令和结果。

## 说明

- 已知限制、部署提醒和后续事项。
```

规则：

- 版本号使用 SemVer：`vMAJOR.MINOR.PATCH`
- 条目使用中文，面向使用者说明结果，不写内部流水账
- 没有内容的分类可以省略，但 `亮点`、`验证`、`说明` 必须保留
- `验证` 必须包含 `make release-check` 的结果，除非明确说明无法运行

## 代码规范

模块结构、错误码分段、配置校验、数据库迁移与日志调用规范统一维护在 README「[开发规范](../README.md#开发规范)」章节，修改相关代码前先阅读并遵守。

- 所有改动必须通过 `make fmt`、`make vet`、`make lint`、`make test`
- 修改代码后，必须同步更新 README.md 相关章节

## 测试

```bash
make test              # 全量测试
make test-coverage     # 覆盖率报告
```

- 改动必须有对应测试覆盖
- 模块生成器自带 service 和 handler 测试骨架
- 集成测试使用 `internal/shared/testutil` 中的 testdb 辅助
- **测试里定位文件路径必须用 `testutil` 的 helper**：模块根用 `testutil.RepoRoot(t)`（`init()` 等拿不到 `*testing.T` 的场合用 `testutil.MustRepoRoot()`），包内 testdata 用 `testutil.TestdataDir(rel)`（cwd 就是包目录，等价于相对路径）。**不要用 `runtime.Caller(0)` 再向上拼层数**：生成项目的重型矩阵构建一律带 `-trimpath`，编译期路径会被重写成模块相对路径，据此推出的「根」是字符串而不是磁盘目录（实测 `chdir example.com/proj: no such file or directory`）

### 本地数据库集成测试

集成测试（`internal/**/integration_test.go`）用真实 MySQL，通过 `testutil.SkipUnlessMysql`
自动检测：数据库不可达时跳过，CI 的 mariadb service 满足条件。

本地运行方式（临时 mariadb 容器，用完即删）：

```bash
# 1. 启动临时 mariadb（root 密码 root，建 jimu_test 库）
docker run -d --rm --name jimu-test-mysql \
  -e MARIADB_ROOT_PASSWORD=root \
  -e MARIADB_DATABASE=jimu_test \
  -p 3306:3306 \
  mariadb:12.1.2-noble

# 2. 跑集成测试（连接参数与 CI 一致）
DB_HOST=127.0.0.1 DB_PORT=3306 DB_USER=root DB_PASSWORD=root DB_NAME=jimu_test \
  go test ./internal/capabilities/user/... -run Integration -v

# 3. 结束删除容器
docker rm -f jimu-test-mysql
```

注意：

- 若本地 `3306` 已被占用（如已有 mariadb 容器），换映射端口：`-p 3307:3306`，测试命令里 `DB_PORT=3307`
- `MARIADB_DATABASE=jimu_test` 已自动建库，测试直接连接，无需手动建
- 普通单元测试（sqlite/in-memory）不需要此容器

## 模块开发 / 新增能力

```bash
./bin/jimu capability create product # 本仓内生成能力骨架：internal/capabilities/product/
```

`jimu capability create` 只往 `internal/capabilities/<name>/` 落典型 CRUD 骨架（含 `Descriptor` 和 `Wire`），**不改动任何注册点**；旧 `module create` 不再可用。注册分两处：能力清单 `internal/capabilities/catalog`，以及需要该能力的形态清单 `internal/profiles/<name>/assembly.go`（非 catalog 条目按 `Ungated` 声明）。唯一入口 `cmd/server` 只调 `assembly.Run(active.Assembly())`，当前形态由选点包 `internal/profiles/active` 决定，不在这里逐个装配能力。

出货（为使用者生成独立项目）走层①脚手架 `jimu new` / `jimu capability add`；其中 `add` 向已生成项目追加已有能力，不创建框架能力，见 README「生成项目」章节。

改完跑 `make check-capabilities`（覆盖能力声明、迁移、驱动、形态装配、资产归属与生成器边界）与 `make profiles-check`（golden 依赖闭包）；新增能力、新增形态、新增驱动的完整步骤见 README「[开发规范 › 新增能力 / 驱动](../README.md#新增能力--驱动)」。

## 报告问题

- Bug 报告：使用 [Bug Report 模板](https://github.com/your-org/jimu/issues/new?template=bug_report.md)
- 功能建议：使用 [Feature Request 模板](https://github.com/your-org/jimu/issues/new?template=feature_request.md)

## 行为准则

保持友善、尊重、建设性。详情请参见仓库的 Code of Conduct（如有）。
