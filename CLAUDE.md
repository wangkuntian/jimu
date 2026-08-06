# CLAUDE.md

本项目的 Claude 协作规范。全仓库默认遵守本文件；子目录如果以后有自己的
`CLAUDE.md`，以更靠近被修改文件的规范为准。

默认使用中文回复用户，除非用户明确要求使用其他语言。

## 开发前必读

- 项目结构、技术栈、API 等见 [README.md](README.md)
- 修改代码后，必须同步更新 README.md 相关章节

## 架构约束

### 模块结构

每个业务模块必须遵循以下分层：

```text
internal/modules/{name}/
  domain/           # 实体、值对象、仓储接口
  application/      # 用例服务、DTO
  infrastructure/   # 数据库/缓存实现
  interfaces/       # HTTP handler + 路由注册
  module.go         # 实现 contract.Module 接口
```

业务逻辑必须依赖接口，不依赖具体实现。

### 模块注册

所有模块必须实现 `contract.Module` 接口：

```go
type Module interface {
    Name() string
    RegisterHTTP(r Router)
    RegisterJobs(j JobRegistry)
    RegisterEvents(e EventBus)
}
```

HTTP 路由统一注册在 `/api/v1` 前缀下。

## 配置约束

### 枚举值

以下配置字段只接受固定枚举值，非法值启动报错：

- `http.mode`：`debug`、`release`、`test`
- `log.level`：`debug`、`info`、`warn`、`error`
- `log.format`：`json`、`console`

新增枚举值必须在 `internal/config/config.go` 中定义常量并加入校验。

### 环境变量

支持 `_FILE` 后缀从文件读取敏感值（Docker Secrets 兼容）：

```bash
# 直接环境变量
DB_HOST=mariadb
DB_PASSWORD=secret

# 或从文件读取（推荐生产环境）
DB_PASSWORD_FILE=/run/secrets/db_password
JWT_SECRET_FILE=/run/secrets/jwt_secret
```

### 多环境配置

通过 `APP_ENV` 环境变量切换配置文件：

| 环境 | 配置文件 |
|------|----------|
| 开发 | `app.yaml` |
| 生产 | `app.prod.yaml` |

优先级：`环境变量 > app.{env}.yaml > app.yaml`

## 编码约束

### 统一响应

HTTP 状态码始终 200，业务错误通过 `body.code` 体现：

```json
{"code": 0, "message": "ok", "data": {}}
```

错误码定义在 `internal/shared/errors/errors.go`，新增错误码需在对应模块范围内：

- `1xxx` — 通用错误
- `2xxx` — 用户/认证模块
- 后续模块按 `3xxx`、`4xxx` 依次分配

### 日志

- 使用 Zap，通过 `logger.New(cfg.LogConfig)` 创建
- 文件输出使用 lumberjack 滚动，配置项：`max_size`、`max_backups`、`max_age`、`compress`
- `output: stdout` 时直接写终端不滚动

### 数据库

- Gorm + `gorm.io/driver/mysql`
- 所有基础表包含字段：`id`、`created_at`、`updated_at`、`deleted_at`
- 迁移使用 Goose，命名 `{seq}_create_{table}s.sql`
- 迁移文件需为每个字段和表添加 COMMENT（中文说明）
- 支持读写分离（通过 `read_hosts`、`read_ports` 配置）

### CLI

- 基于 Cobra，入口在 `cmd/cli/main.go`
- 迁移和种子命令通过 `jimu migrate` / `jimu seed` 调用

## 文档维护

修改代码后，必须同步更新 README.md：

- 新增配置项 → 更新配置说明表
- 新增 CLI 命令 → 更新命令表
- 新增 API → 更新 API 示例
- 项目结构变化 → 更新目录树
- 新增 Makefile 目标 → 更新命令速查
- 新增 API → 使用中文 swagger 注解（@Summary、@Description、@Param、@Success、@Failure）

## 简单优先

只写当前任务需要的最少代码。

- 不添加未被要求的功能、配置项、扩展点或抽象。
- 不为一次性代码创建接口、工厂或通用框架。
- 能删除本次改动造成的无用代码时直接删除。
- 不重构无关代码，不顺手改格式、命名或注释。

如果实现规模明显超过需求，先停下来简化方案。

## 保护工作区

工作区可能包含用户未提交改动。

- 修改前查看 `git status --short`。
- 不覆盖、回滚或格式化无关文件。
- 不运行 `git reset hard`、`git checkout -- .`、批量删除等破坏性命令，除非用户明确要求。
- 提交前再次查看 `git status --short`，只纳入当前任务相关文件。
- 除非用户明确要求，否则不要创建 commit。

如果无关本地改动影响当前任务，说明冲突并尽量绕开。

## 分支策略

### 分支模型

采用简化的 GitHub Flow，以 `master` 为唯一长期分支，`release/x.y.z` 为集成与发布分支。

- `master`：唯一长期分支，保持可部署、CI 通过。禁止直接 push，仅接受 release 分支合并。
- `release/x.y.z`：发布分支，从 master 切出。日常开发（feature/fix/hotfix）在此集成；稳定后合并回 master 并打 `vX.Y.Z` tag。
- `feature/<issue>-<slug>`：新功能分支，从 release 切出，PR 合入目标 release。
- `fix/<issue>-<slug>`：非紧急缺陷修复，从 release 切出，PR 合入目标 release。
- `hotfix/<issue>-<slug>`：线上紧急修复，从 release 切出，PR 合入目标 release。
- `dependabot/*`：Dependabot 自动化分支，遵循配置的自动合并规则，人工不得基于它开发。

### 命名约定

- 分支名小写，单词用短横线，`<slug>` 不超过 5 个单词。
  例：`feature/123-inventory-api`、`fix/456-login-redirect`。
- 鼓励带 issue ID，便于追溯。
- 标签遵循 SemVer：`v<major>.<minor>.<patch>`。无预发布标签。

### 合并与 PR

- 所有非 Dependabot 改动必须经 PR。
- PR 至少 1 人 review 通过。
- CI (`make release-check`) 必须通过。
- 合并策略：feature/fix/hotfix → release 用 squash merge；release → master 用 merge commit。
- 合并后删除源分支。

### Tag 与发布

- 发布当日从 release 分支合并到 master 后，在 master tip 打 `v<major>.<minor>.<patch>` tag。
- Tag 经 `make release-check` 通过后打，tag 与 release notes 同步推送。
- 不发布未经 tag 的 commit。

### 回滚

- master 不接受 force push。回滚用 revert commit 或新 hotfix PR。
- release 分支回滚：切 hotfix 分支修复后重复合并流程。

## Commit Message 规范

使用 Conventional Commits 的轻量格式：

```text
type(scope): summary
```

`type` 使用以下值：

- `feat`：新增用户可见功能。
- `fix`：修复缺陷。
- `docs`：文档修改。
- `test`：测试新增或调整。
- `refactor`：不改变行为的代码重构。
- `chore`：构建、依赖、脚手架、维护类修改。

`scope` 可选，优先使用：

- `repo`
- `server`
- `console`
- `config`
- `auth`
- `user`
- `role`
- `permission`
- `logger`
- `db`
- `http`

`summary` 规则：

- 使用英文。
- 小写开头。
- 使用祈使句或动词短语。
- 不加句号。
- 简洁说明本次提交做了什么。

示例：

```text
chore(repo): initialize project structure
feat(server): add health check endpoint
feat(auth): add JWT token refresh
fix(config): validate enum values on load
```

多行 commit 可以在正文说明 why 和风险，不重复 summary。一个 commit 只包含一个清晰主题，不混入无关改动。

## Release Note 规范

发布说明使用 Markdown，文件名或 GitHub Release 标题使用版本号，例如 `v0.1.0`。

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

- 版本号使用 SemVer：`vMAJOR.MINOR.PATCH`。
- 条目使用中文，面向使用者说明结果，不写内部流水账。
- 每条尽量以动词开头，简洁、可验证。
- 没有内容的分类可以省略，但 `亮点`、`验证`、`说明` 必须保留。
- `验证` 必须包含 `make release-check COMPOSE_ENV=.env.example` 的结果，除非明确说明无法运行。
- 不把 commit hash 列表直接当 release note；必要时只链接对应 PR 或总结变更。

## 回复格式

长任务中给出简洁进度，说明正在收集什么上下文、准备改什么、验证结果是什么。

最终回复包含：

- 改了什么。
- 跑了什么验证。
- 还有什么风险或未做事项。
