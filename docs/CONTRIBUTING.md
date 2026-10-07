# 贡献指南

感谢你愿意为 Jimu 贡献力量。本文档帮助你快速上手。

## 开发环境

- Go 1.26+
- MariaDB 10.5+ / Redis 6+
- (可选) Docker Compose 一键起依赖

```bash
# 克隆
git clone https://github.com/your-org/jimu.git
cd jimu
go mod download

# 启动依赖
docker-compose up -d mariadb redis

# 迁移 + 种子
make migrate && make seed

# 启动
make run
```

## 分支策略

采用简化 GitHub Flow：`master` 为唯一长期分支（禁止直接 push，仅接受 release 分支合并），`release/vx.y.z` 为集成与发布分支，日常开发从 release 切出、经 PR 合回。

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
- 由 `githooks/commit-msg` 强制检查（`make hooks` 安装，即 `core.hooksPath` 指向 `githooks/`）；CI (Commits workflow) 兜底；确需跳过用 `git commit --no-verify`

示例：

```
feat(user): add avatar upload endpoint
fix(auth): reject expired refresh token
```

## Pull Request

1. 至少 1 人 review 通过
2. feature/fix/hotfix → release PR 必须通过常规 CI、Docker 和提交消息检查；这些 workflow 只响应 PR，不在合并后的 branch push 上重复运行
3. release/* → master 发布候选 PR 还必须通过一次 Scaffold Matrix；Dependabot 安全更新直达 master 时该检查成功跳过；master ruleset 的 required checks 阻止未验证合并，release ruleset 只保留 non-fast-forward 保护
4. 合并策略：feature/fix/hotfix → release 用 squash merge；release → master 用 merge commit
5. 合并后删除源分支

## Dependabot 发布周期

每个版本周期由 Release Issue 自动启动。创建标题严格为 `release: vX.Y.Z` 的 Issue，Actions 会校验版本并创建 `release/vX.Y.Z` 与固定汇总分支 `dependabot-updates`。同一时间只接受一个 active release cycle；重复事件会恢复已有资源，不会覆盖非本自动化管理的分支。

Dependabot 普通版本更新指向 `dependabot-updates`，只运行 `CI (Dependabot Focused)` 中的格式、`go vet` 和普通 Go 测试。focused checks 成功后，GitHub App 自动 squash merge。这里使用检查完成后的 App merge，不要求启用 GitHub auto-merge。安全更新不受 `target-branch` 控制，仍直接指向 `master`；现有 CI 照常运行，`Scaffold Matrix` 以成功 skip 满足 required check，PR 链接会记录到活跃的 Release Issue。

每日收集器至少等待 8 天，并且要求 `dependabot-updates` 最近 24 小时没有提交、没有待处理的 Dependabot PR。条件满足后会创建 `updates/vX.Y.Z -> release/vX.Y.Z` 汇总 PR，并将 Issue 标为 `release: candidate`。`CI (Go)`、`CI (Docker)`、`CI (Commits)` 都必须完成；各自检查成功或正常跳过且 PR head SHA 未变化后，GitHub App 才会自动 squash merge。汇总 PR 合入后，Actions 会自动创建 `release/vX.Y.Z -> master` 候选 PR。该 PR 使用 master ruleset 的完整 required checks，仍需维护者人工 review 和 merge。

候选 PR 合入后，GitHub App 会验证 Release Issue 状态、PR marker 和合并提交，创建 `vX.Y.Z` tag。现有 `release.yml` 发布成功后，Issue 才标记为 `release: published`。发布失败或 tag 校验失败会保留现场并标记为 `release: blocked`。

首次启用需要仓库管理员创建并安装 GitHub App，权限为 `contents: write`、`pull_requests: write`、`issues: write`、`metadata: read`，并设置 Actions secrets `JIMU_RELEASE_APP_ID`、`JIMU_RELEASE_APP_PRIVATE_KEY`。仓库默认 workflow token 可保持 `read`。当前 release ruleset 保持 `non_fast_forward`，不添加 release required checks；汇总 PR 的 App workflow 会在 merge 前检查三个 CI workflow。无需开启仓库 auto-merge。正常流程由 Issue、Dependabot PR、每日 schedule 和 PR/workflow 完成事件驱动，不需要手动触发 workflow。

## Tag 与发布

- 发布当日从 release 分支合并到 master 后，在 master tip 打 `vMAJOR.MINOR.PATCH` tag（SemVer，无预发布标签）
- 发布候选 PR 合并后，在当前 `master` tip 打 `vMAJOR.MINOR.PATCH` tag；tag workflow 会校验 tag 必须指向当前 `master` tip，然后只构建四平台二进制并创建 GitHub Release，不重复运行完整 CI 或 Scaffold Matrix
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
- `验证` 必须包含 `make release-check COMPOSE_ENV=.env.example` 的结果，除非明确说明无法运行

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
