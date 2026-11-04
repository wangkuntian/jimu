# 真实项目升级验证

本规范保留给未来的真实项目升级验证：在真实团队使用过的 generated project 上验证 `jimu upgrade`，并留下可审计证据。普通用户升级自己的项目仍以[升级](UPGRADING.md)为准。

截至 2026-11-04，当前 v0.8.0 路线不包含真实项目验证需求。本规范作为未来重新立项时的工具保留，不阻塞 M1 的当前活动范围，也不把仓库内 fixture 计为真实项目。

## 项目资格

一个 upgrade validation project 必须同时满足：

- 由 Release CLI 基于 `basic-api` 或 `auth-api` 生成。
- 已被真实团队用于实际服务，而不是临时目录、仓库内 demo 或测试 fixture。
- 包含团队对生成文件或受管配置的修改。
- 可以从一个已发布版本实际升级到目标版本，并允许维护者记录匿名摘要。

至少 3 个项目分别覆盖：

| 覆盖类别 | 最低要求 |
|---|---|
| 纯 HTTP/BFF | 无状态 API 或无数据库依赖的 BFF |
| 数据库加认证 | 使用数据库、认证和权限能力，并执行独立迁移窗口 |
| 队列/异步任务 | 包含队列、消费者或异步任务；团队自有代码和 experimental composition 均可贡献观察证据 |

experimental composition 的观察结果不改变其 support 级别，也不扩张升级支持窗口。

## 隐私边界

原始 CLI 输出、仓库地址、内部域名、客户名称、业务数据和真实项目标识保留在维护者控制的本地目录。提交到本仓库的内容只包含经项目方确认的匿名摘要。

每个匿名摘要使用稳定别名，例如 `validation-http-01`。不得从别名、路径、截图或日志摘要反推真实团队。公开案例与 P1-04 验收分离，必须另行取得项目方授权。

## 采集流程

1. 从项目默认分支创建临时升级分支，并备份项目目录与业务数据。
2. 记录项目形态、official template、能力组合、support 级别、manifest schema 和本地修改范围。
3. 用目标 Release CLI 执行只读检查，保存完整输出和耗时。
4. 在实际升级前确认用户自有文件和代表性业务数据均可核验。
5. 在临时分支执行实际升级，保存完整输出、耗时和退出状态。
6. 执行项目自己的构建、测试和业务数据校验，比较升级前后的用户自有文件。
7. 将冲突路径、冲突原因、自动回滚、人工恢复和问题链接填入匿名摘要。

建议至少保留以下命令输出：

```bash
date -u +%Y-%m-%dT%H:%M:%SZ
jimu version
jimu check .
/usr/bin/time -p jimu upgrade . --check
/usr/bin/time -p jimu upgrade .
date -u +%Y-%m-%dT%H:%M:%SZ
git status --short
```

命令中的 `jimu` 必须是待验证的目标 Release CLI；在输出中保留 `jimu version` 给出的 CLI、template bundle 和 commit。`jimu check` 和 `jimu upgrade --check` 是只读检查；实际升级应始终使用可丢弃的分支或恢复点，并且不得用 `--no-tidy` 跳过正式验证。数据库 schema 不由 `jimu upgrade` 自动迁移，数据库加认证项目必须按自己的发布流程单独验证迁移。

## 匿名摘要模板

将每个项目的匿名摘要提交到 `docs/upgrade-validation/<version>.md`，原始日志留在本地。

```markdown
### <validation-alias>

- 覆盖类别：
- official template：
- composition 与 support：
- 项目使用场景（匿名）：
- 已发布起始版本：
- 目标 CLI / template bundle：
- manifest schema：
- 验证时间窗口：

| 检查项 | 结果 |
|---|---|
| `jimu check` 退出状态与耗时 | |
| `jimu upgrade --check` 退出状态与耗时 | |
| check 分类计数 | |
| 实际升级退出状态与耗时 | |
| 实际升级文件与 manifest 结果 | |
| 冲突路径与原因 | |
| 自动回滚次数 | |
| 人工恢复次数 | |
| 用户自有文件是否保持不变 | |
| 代表性业务数据是否保持不变 | |
| 最终 build / test / 业务验证 | |
| 回流 issue 或文档 | |
| 匿名摘要已由项目方确认 | |
```

“回滚次数”包含 `jimu upgrade` 事务自动回滚的次数，以及升级尝试后人工恢复项目或数据的总次数。`--check` 的只读失败不产生自动回滚。

## 恢复验证后的关闭条件

真实项目验证如果重新纳入活动路线，只有同时满足以下条件才能标记完成：

- 3 个合格项目的匿名摘要均已提交。
- 三个覆盖类别均至少有一个项目。
- 每个项目都有 `jimu upgrade --check` 和实际升级记录。
- 每个项目最终成功到达目标版本并完成 build / test / 业务验证。
- 没有用户自有文件或业务数据丢失。
- 冲突、耗时和回滚记录完整。
- 发现的问题已经回流到 issue、模板、诊断或文档。

未满足时，`REQUIREMENTS.md`、版本里程碑和 Release Note 不得声称存在真实项目采用证据。
