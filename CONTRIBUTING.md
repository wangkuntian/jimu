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

采用简化 GitHub Flow，`master` 为唯一长期分支，`release/x.y.z` 为发布分支。

- `feature/<issue>-<slug>` — 新功能
- `fix/<issue>--slug>` — 缺陷修复
- `hotfix/<issue>-<slug>` — 线上紧急修复

分支名小写，单词用短横线，`slug` 不超过 5 个单词。鼓励带 issue ID。

## Commit 规范

Conventional Commits 轻量格式：

```
type(scope): summary
```

`type` 取值：`feat`、`fix`、`docs`、`test`、`refactor`、`chore`

`scope` 建议：`server`、`config`、`auth`、`user`、`role`、`permission`、`http`、`db`、`logger`

规则：summary 英文、小写开头、祈使句、不加句号。

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

## 代码规范

- 所有改动必须通过 `make fmt`、`make vet`、`make lint`
- 新增错误码在对应模块范围（`1xxx` 通用、`2xxx` 用户/认证、`3xxx` 角色/权限）
- 新增配置项必须在 `internal/config/config.go` 定义常量并加入校验
- 业务模块遵循 Clean Architecture 分层（domain / application / infrastructure / interfaces / module.go）
- HTTP 路由统一注册在 `/api/v1` 下
- 修改代码后，必须同步更新 README.md 相关章节

## 测试

```bash
make test              # 全量测试
make test-coverage     # 覆盖率报告
```

- 改动必须有对应测试覆盖
- 模块生成器自带 service 和 handler 测试骨架
- 集成测试使用 `internal/shared/testutil` 中的 testdb 辅助

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
