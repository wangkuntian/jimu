# AGENTS.md

本文件由 `CLAUDE.md` 提炼生成，是本工作区的协作规范。全仓库默认遵守；
子目录若有自己的 `CLAUDE.md`/`AGENTS.md`，以更靠近被修改文件的规范为准。

## 语言与沟通

- 默认使用中文回复用户，除非用户明确要求其他语言。
- 长任务中给出简洁进度：正在收集什么上下文、准备改什么、验证结果是什么。
- 最终回复包含：改了什么、跑了什么验证、还有什么风险或未做事项。

## 开发前必读

- 项目结构、技术栈、API 见 `README.md`；修改代码后必须同步更新 README 相关章节。
- 代码库问题优先查询知识图谱 `graphify-out/`（见文末 graphify 一节），再查源码。

## 架构约束

- 业务模块必须分层：`domain/`（实体、值对象、仓储接口）、`application/`（用例服务、DTO）、
  `infrastructure/`（数据库/缓存实现）、`interfaces/`（HTTP handler + 路由注册）、`module.go`。
- 业务逻辑依赖接口，不依赖具体实现。
- 模块必须实现 `contract.Module` 接口（`Name/RegisterHTTP/RegisterJobs/RegisterEvents`）。
- HTTP 路由统一注册在 `/api/v1` 前缀下。
- **设计边界（非目标）**：不做多租户。项目定位单租户/单组织；数据表不预留
  `tenant_id`/`tenant` 字段，不引入租户中间件。新增表、接口、中间件前先对照此清单。

## 配置约束

- 枚举字段只接受固定值，非法值启动报错：`http.mode`（debug/release/test）、
  `log.level`（debug/info/warn/error）、`log.format`（json/console）。
  新增枚举值必须在 `internal/config/config.go` 定义常量并加入校验。
- 敏感配置支持 `_FILE` 后缀从文件读取（Docker Secrets 兼容）。
- 通过 `APP_ENV` 切换配置文件（app.yaml / app.prod.yaml），优先级：
  环境变量 > app.{env}.yaml > app.yaml。
- 集成测试（`internal/**/integration_test.go`）用真实 MySQL，通过
  `testutil.SkipUnlessMysql` 自动跳过不可达环境；本地用临时 mariadb 容器。

## 编码约束

- 统一响应：HTTP 状态码始终 200，业务错误通过 `body.code` 体现。
- 错误码定义在 `internal/shared/errors/errors.go`，按模块分段（1xxx 通用、2xxx 用户/认证、后续依次分配）。
- 日志使用 Zap（`logger.New(cfg.LogConfig)`），文件输出用 lumberjack 滚动。
- 数据库：Gorm + mysql 驱动；基础表含 `id/created_at/updated_at/deleted_at`；
  主键为应用生成的雪花 ID，不用 `AUTO_INCREMENT`；支持读写分离。
- 迁移使用 Goose，命名 `{seq}_create_{table}s.sql`，每个字段和表加中文 COMMENT。
- CLI 基于 Cobra，入口 `cmd/cli/main.go`。

## 文档维护

修改代码后必须同步更新 README.md：

- 新增配置项 → 配置说明表；新增 CLI 命令 → 命令表；新增 API → API 示例；
  项目结构变化 → 目录树；新增 Makefile 目标 → 命令速查。
- 新增 API 使用中文 swagger 注解（@Summary/@Description/@Param/@Success/@Failure）。

## 简单优先

- 只写当前任务需要的最少代码；不添加未被要求的功能、配置项、扩展点或抽象。
- 不为一次性代码创建接口、工厂或通用框架；能删的无用代码直接删。
- 不重构无关代码，不顺手改格式、命名或注释。
- 实现规模明显超过需求时，先停下来简化方案。

## 保护工作区

- 修改前查看 `git status --short`；不覆盖、回滚或格式化无关文件。
- 不运行 `git reset hard`、`git checkout -- .`、批量删除等破坏性命令，除非用户明确要求。
- 提交前再次查看 `git status --short`，只纳入当前任务相关文件。
- 除非用户明确要求，否则不创建 commit。

## 分支策略

- 采用简化 GitHub Flow：`master` 为唯一长期分支（保持可部署、CI 通过、禁止直接 push），
  `release/x.y.z` 为集成与发布分支。
- 开发分支：`feature/<issue>-<slug>`、`fix/<issue>-<slug>`、`hotfix/<issue>-<slug>`，
  从 release 切出，PR 合入目标 release；分支名小写、短横线、slug 不超过 5 个单词。
- 所有非 Dependabot 改动必须经 PR，至少 1 人 review，CI（`make release-check`）必须通过。
- 合并策略：feature/fix/hotfix → release 用 squash merge；release → master 用 merge commit；合并后删源分支。
- 发布：从 release 合并到 master 后在 tip 打 `vX.Y.Z` tag；master 不接受 force push，回滚用 revert 或 hotfix PR。

## Commit Message 规范

- 使用 Conventional Commits 轻量格式：`type(scope): summary`。
- type：`feat`/`fix`/`docs`/`test`/`refactor`/`chore`。
- scope 优先：repo/server/console/config/auth/user/role/permission/logger/db/http。
- summary：英文、小写开头、祈使句或动词短语、不加句号、简洁说明做了什么。
- 多行 commit 正文说明 why 和风险，不重复 summary；一个 commit 只含一个清晰主题。

## Release Note 规范

- 文件名或标题使用版本号（SemVer，如 `v0.1.0`），按「亮点/新增/变更/修复/移除/验证/说明」组织。
- 条目使用中文，面向使用者，以动词开头；无内容分类可省略，但「亮点/验证/说明」必须保留。
- 验证必须包含 `make release-check COMPOSE_ENV=.env.example` 的结果。
- 不把 commit hash 列表直接当 release note。

## graphify

- 知识图谱位于 `graphify-out/`。代码库问题先运行 `graphify query "<question>"`
  （存在 `graphify-out/graph.json` 时），关系用 `graphify path "<A>" "<B>"`，
  概念用 `graphify explain "<concept>"`；`graphify-out/wiki/index.md` 用于宏观导航。
- 修改代码后运行 `graphify update .` 保持图谱最新（仅 AST，无 API 成本）。
