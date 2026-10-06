# v0.3.3 生成器根包收口实施计划

> **For agentic workers:** 本计划按 inline 模式逐项执行；每个 Task 完成后独立验证并创建一个英文 Conventional Commit。

**目标：** 在不改变 `jimu/tools/generator` 公共入口和生成行为的前提下，把独立的模块生成、通用支持原语和模板引擎迁入职责子包。

**架构：** 根包继续作为 CLI facade；`module` 负责能力骨架生成，`support` 负责框架无关的文件与进程原语，`render` 负责模板读取和 manifest 渲染。框架能力、profile 和 descriptor 适配仍由 `frameworkmanifest` 负责。

**技术栈：** Go 1.26+、`go:embed`、标准库 `os/exec`、现有 Go test 与生成器矩阵。

## 全局约束

- 保持 `jimu/tools/generator`、`GenerateModule`、`GenerateModuleAt`、`Template`、`RenderText`、`CopyTree`、`RewriteModule`、`Tidy` 和 `SelfCheck` 的现有调用契约。
- 不改变生成项目内容、manifest schema、CLI flags、回滚语义和 v0.3.2 不兼容策略。
- 不覆盖或回滚工作区现有改动；每个 Task 完成后只提交当前 Task 相关文件。
- 生产代码迁移前先保留或迁移对应测试；聚焦测试通过后再进入下一 Task。
- 修改源码后同步更新 `README.md` 和 `docs/releases/v0.3.3.md`。

---

### Task 1：迁移能力骨架生成器

**文件：**

- 创建：`tools/generator/module/create.go`、`tools/generator/module/templates.go`
- 移动：`tools/generator/module_test.go`、能力骨架模板目录
- 修改：`tools/generator/module.go`、`tools/generator/golden_test.go`、`tools/generator/compile_test.go`
- 测试：`go test ./tools/generator/module ./tools/generator -run 'TestGenerateModule|TestGeneratedModule' -count=1`

**步骤：**

- [x] 将 `GenerateModule`、`GenerateModuleAt` 及其私有预检、渲染、写入和回滚函数迁入 `module` 包。
- [x] 将模块骨架模板迁入 `module/templates`，用独立 `go:embed` 和渲染函数读取。
- [x] 根包的同名函数改为委托 `module` 包，CLI 和原有外部调用不变。
- [x] 将模块单测迁到 `module` 包；为根包保留 golden/compile 测试所需的最小测试仓库 helper。
- [x] 运行聚焦测试并提交 `refactor(generator): isolate module scaffolder`。

### Task 2：迁移通用生成器支持原语

**文件：**

- 创建：`tools/generator/support/files.go`、`tools/generator/support/rewrite.go`、`tools/generator/support/process.go`
- 移动：`tools/generator/copytree.go`、`tools/generator/rewrite.go`、`tools/generator/selfcheck.go`、`tools/generator/tidy.go`
- 修改：根包兼容包装和对应单测包声明
- 测试：`go test ./tools/generator/support ./tools/generator -run 'TestCopyTree|TestRewriteModule|TestTidy|TestSelfCheck' -count=1`

**步骤：**

- [x] 将四组实现迁入 `support`，保持错误文本、路径处理、环境变量和回滚相关行为不变。
- [x] 根包提供同签名的薄包装，避免旧测试和内部兼容调用改变。
- [x] 将原语单测靠近 `support` 包，保留根包集成测试覆盖 facade 委托。
- [x] 运行聚焦测试并提交 `refactor(generator): isolate support primitives`。

### Task 3：收拢模板引擎与文档

**文件：**

- 创建或移动：`tools/generator/render/templates.go`、`tools/generator/render/templates/`
- 修改：`tools/generator/render.go`、`tools/generator/templates.go`、`tools/generator/frameworkmanifest/actions.go`、相关 golden fixture
- 修改文档：`README.md`、`docs/releases/v0.3.3.md`
- 测试：`go test ./tools/generator/render ./tools/generator/... -count=1`

**步骤：**

- [x] 让 `render` 包拥有模板 `embed`、枚举、读取和文本渲染实现。
- [x] 根包保留同签名包装，现有构建文件和旧模板调用保持可用。
- [x] 将框架模板源路径统一到 `tools/generator/render/templates/project`，刷新受影响的 manifest fixture。
- [x] README 增加生成器目录职责树，release note 说明根包 facade 与职责子包的最终边界。
- [x] 运行完整生成器测试、门禁与文档检查，提交 `docs(v0.3.3): document generator package layout`。

### Task 4：全量验证

**步骤：**

- [ ] 运行 `go test ./... -count=1`、`go vet ./...`。
- [ ] 运行 `make check-capabilities`、`make profiles-check`、`make compose-report-check`、`make check-skills`、`git diff --check`。
- [ ] 运行 `make test-scaffold-matrix`；若环境门控跳过，记录明确的跳过条件。
- [ ] 查看 `git status --short`，确认提交只包含本轮整理相关文件。
