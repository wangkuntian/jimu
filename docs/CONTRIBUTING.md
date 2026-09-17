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
- 由 `githooks/commit-msg` 强制检查（`make hooks` 安装，即 `core.hooksPath` 指向 `githooks/`）；CI (Commits workflow) 兜底；确需跳过用 `git commit --no-verify`

示例：

```
feat(user): add avatar upload endpoint
fix(auth): reject expired refresh token
```

## Pull Request

1. 至少 1 人 review 通过
2. CI (`make release-check`) 必须通过
3. 合并策略：feature/fix/hotfix → release 用 squash merge；release → master 用 merge commit
4. 合并后删除源分支

## Tag 与发布

- 发布当日从 release 分支合并到 master 后，在 master tip 打 `vMAJOR.MINOR.PATCH` tag（SemVer，无预发布标签）
- Tag 经 `make release-check` 通过后打，tag 与 release notes 同步推送；不发布未经 tag 的 commit
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
  go test ./internal/modules/user/... -run Integration -v

# 3. 结束删除容器
docker rm -f jimu-test-mysql
```

注意：

- 若本地 `3306` 已被占用（如已有 mariadb 容器），换映射端口：`-p 3307:3306`，测试命令里 `DB_PORT=3307`
- `MARIADB_DATABASE=jimu_test` 已自动建库，测试直接连接，无需手动建
- 普通单元测试（sqlite/in-memory）不需要此容器

## 模块开发

```bash
./bin/jimu module create product
```

生成完整骨架后在 `cmd/server/main.go` 注册模块。详见 README.md "模块开发" 章节。

## 报告问题

- Bug 报告：使用 [Bug Report 模板](https://github.com/your-org/jimu/issues/new?template=bug_report.md)
- 功能建议：使用 [Feature Request 模板](https://github.com/your-org/jimu/issues/new?template=feature_request.md)

## 行为准则

保持友善、尊重、建设性。详情请参见仓库的 Code of Conduct（如有）。
