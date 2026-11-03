# 安全政策

## 报告安全问题

我们重视所有用户和研究人员报告的安全漏洞。

### 如何报告

**请勿通过公开 issue 报告安全漏洞。**

请发送邮件至：

- **wangkuntian@...**

邮件请包含：

1. 漏洞描述（受影响的组件、触发条件）
2. 复现步骤（最小化示例）
3. 影响评估（机密性 / 完整性 / 可用性）
4. 建议修复方案（如有）

不要在公开渠道发布可直接利用的细节，直到修复版本发布并预留合理的升级时间。

### 响应时间

- **确认收到**：2 个工作日内
- **初步评估**：5 个工作日内
- **修复时间**：根据严重程度，严重漏洞 30 天内

### 披露政策

- 修复完成后公开致谢 Reporter（除非要求匿名）
- 遵循负责任的披露原则：给我们合理时间修复后再公开
- 不追究善意研究人员的法律责任

## 支持版本

| 版本 | 安全修复 | 说明 |
|---|---|---|
| 0.7.x 最新 patch | ✅ | 只保证当前 0.7.x 的最新 patch |
| 0.7.x 旧 patch、0.6.x、0.5.x | ❌ | 不 backport；先升级到当前最新 patch |
| 0.4.x 及更早 | ❌ | 重新生成项目并人工迁移 |
| `--with` / `capability add` | ❌ | experimental，不承诺安全修复或升级兼容 |

升级支持窗口、manifest schema 和兼容承诺见[升级](UPGRADING.md)与[ADR 0003](adr/0003-upgrade-support-window.md)。

## 安全修复进入生成项目

适用对象：由 `jimu new` 生成的 `basic-api` / `auth-api` 项目，以及使用 `jimu upgrade` 的 certified 项目。

### 处理流程

1. **确认影响范围**：记录受影响模板、Jimu 版本、manifest schema、可达路径和建议严重级别。
2. **修复模板或框架源**：优先修复生成模板中的可升级文件，并加入回归测试。不要在公开 issue 中提前披露利用方式。
3. **发布 patch**：在最新 patch 的 Release Note 中写明修复内容、受影响版本、升级方式和无法自动迁移的部分。
4. **用户判断**：用户运行 `jimu version`、`jimu check <dir>`、`jimu upgrade <dir> --check`，确认项目和 CLI 是否需要处理。
5. **执行升级**：用户运行 `jimu upgrade <dir>`。事务会先备份、逐文件安装，再执行 tidy/build/test；失败按原字节回滚。
6. **人工完成无法自动迁移的部分**：数据库 schema、Secret 轮换、反向代理/部署配置、业务数据修复仍由项目维护者执行。

### 升级能做什么

- 安装 manifest 管理且本地未被修改的模板文件。
- 安装模板新增的安全中间件、配置键、测试和迁移文件。
- 在安全修复涉及配置结构时，按 manifest 声明的策略合并受支持配置段。
- 在写入前后保持 `conflict` 不覆盖，并报告可定位的路径和原因。

### 升级不能做什么

- 不自动修改已运行的数据库 schema；`auth-api` 的迁移仍需 `./jimu migrate up`。
- 不覆盖用户修改、删除或占用的生成文件；这些路径会判为 `conflict`。
- 不轮换已经泄露的 JWT Secret、数据库密码或管理员密码。
- 不修改反向代理、TLS、限流、Secret manager 或部署平台配置。
- 不保证 `--with` / `capability add` 项目的安全修复兼容性。

### 公告格式

安全修复的 Release Note 至少包含：

- 修复摘要与受影响版本
- 修复版本和升级命令
- 是否需要数据库迁移或 Secret 轮换
- 需要人工处理的冲突/部署项
- 用户如何用 `jimu check` / `jimu upgrade --check` 自查

## 安全修复演练

**日期**：2026-11-03

**发布版本**：`v0.7.1`

**修复**：生成模板现在真正执行 `http.allowed_origins` 和 `http.trusted_proxies`，并补充基线安全 Header；`allowed_origins` 中出现 `*` 会在启动时拒绝。

**旧项目样本**：`tools/generator/testdata/upgrade/v0.5/basic-api`（v0.5.0，manifest schema v2）

演练命令：

```bash
bash scripts/security_upgrade_drill.sh
JIMU_HEAVY_MATRIX=1 go test ./tools/generator -run TestUpgradeFixtures -count=1
```

结果：

- 旧项目 `jimu check` 通过。
- `upgrade --check` 报告 5 个 `modified`、0 个 `conflict`；变更路径为 `internal/app/app.go`、`internal/app/router.go`、`internal/app/router_test.go`、`internal/config/config.go`、`internal/platform/http/middleware.go`。
- 实际 `upgrade` 完成 tidy/build/test，`verified: true`，manifest 升级到 schema v3。
- 升级后的项目包含 `SecurityHeaders()`、CORS 装配和可信代理配置，生成项目测试验证了允许/拒绝来源与安全 Header。
- `TestUpgradeFixtures` 对 `basic-api` 和 `auth-api` 的 20 个升级场景通过。

该演练使用真实 v0.5 项目快照验证发布候选的升级链路；它不是 CVE 公告演练，不伪造漏洞编号。

## 安全最佳实践

部署生成项目时，请确保：

1. **生产 JWT Secret**：`auth-api` 使用至少 32 字符的强随机值，通过 `JWT_SECRET_FILE` 注入。
2. **数据库密码**：通过 Docker Secrets、Secret manager 或 `DB_PASSWORD_FILE` 注入。
3. **CORS**：只填写明确来源；`http.allowed_origins` 不允许 `*`。
4. **可信代理**：`http.trusted_proxies` 只填写实际代理 IP/CIDR。
5. **HTTPS**：在反向代理/负载均衡器终止 TLS，不要把明文 HTTP 暴露到公网。
6. **限流**：在网关或项目代码中配置，至少覆盖登录、刷新和公开路由。
7. **日志脱敏**：不要记录密码、token、验证码或未脱敏 PII。
8. **依赖更新**：关注 Dependabot 和 Release Note 中的安全修复。
9. **数据库**：升级前备份，迁移由独立发布步骤执行，并验证恢复流程。

## 依赖漏洞处置

CI 的 `govulncheck` 门禁（`make govulncheck`）对**可达**漏洞零容忍：发现即红，不在代码里静默放行。

若某可达漏洞上游尚无修复版本（govulncheck 报 `Fixed in: N/A`），处理方式是**显式记录 + 缓解**，而不是加白名单：

1. 在代码侧对不可信输入加边界（例如 `internal/capabilities/dataops/excel` 的 `readSheet` recover），让异常只表现为错误而非打挂请求/任务；
2. 在 PR 描述与对应 `docs/releases/<version>.md` 中写明漏洞 ID、可达路径、缓解措施与复核条件；
3. 只有在上述记录齐备时，才由维护者在审查中显式接受该风险。

**图像扫描的显式例外**：CI 的镜像扫描（`Image Gate` 的 Trivy 步骤）只对**有修复版本**的 CRITICAL/HIGH 失败（`ignore-unfixed: true`）。理由：对上游尚未修复的漏洞卡红，会把这些 PR 永久堵死。这不是静默放行；这些条目仍写入 SBOM artifact（`jimu-sbom.json`）供审查。

## 生成模板边界

- `auth-api` 的用户密码只强制最小长度 8；更复杂的强度策略需要项目自行实现。
- `auth-api` 不内置登录失败锁定、全局限流、审计日志、指标端点或 OpenTelemetry。
- `basic-api` 是无状态 HTTP 基线，不内置限流、指标端点或 OpenTelemetry。
- 生成模板的安全 Header 由应用写入；反向代理不得删除这些 Header。
