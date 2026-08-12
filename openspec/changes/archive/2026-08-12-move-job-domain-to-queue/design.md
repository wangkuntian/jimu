## Context

动机见 proposal.md — Why。当前状态与约束：

- `internal/platform/queue` 依赖 `internal/modules/admin/domain` 的三个实体族：`Job`（+`JobStatus`+`JobRepository`）、`JobHistory`（+`JobHistoryRepository`）、`DeadLetter`（+`DeadLetterRepository`）。
- `admin/domain` 还含 `apikey.go` / `import_job.go` 两族实体，与 queue 无关，本次不动。
- 表 `jobs` / `job_history` / `dead_letters`（migrations 010 等）结构不变。
- queue 已依赖 gorm tag 实体，无新增依赖。
- 涉及调用方：queue 自身 2 个文件、admin 侧 4 个文件、测试 4 个。

## Goals / Non-Goals

**Goals:**
- 反转依赖：`platform/queue` 不再 import `modules/admin/domain`。
- 实体及仓储接口归位到 `internal/platform/queue/domain/`。
- 行为完全不变：编译通过、既有测试通过、lint 通过。

**Non-Goals:**
- 不改表结构 / migrations。
- 不改对外 API 与任务状态机语义。
- 不搬 `apikey` / `import_job` 实体。
- 不重写 repository 实现逻辑（仅改 import 路径）。
- 不沉淀 queue 能力规格（由后续独立 change 完成）。

## Decisions

1. **新包位置 `internal/platform/queue/domain/`**。备选：放 `internal/shared/` — 否，`Job` 是队列能力专属，非通用件；留在 `admin/domain` 只改 import — 否，问题根源就是跨层依赖，必须物理搬离。包名 `domain` 遵循项目既有约定（各模块均为 `domain`）。

2. **仓储接口随实体一起移动，实现留在 admin**。`JobRepository` 等接口是领域契约，与实体同住；其 gorm 实现（`admin/infrastructure/*_repository.go`）是 admin 的 jobs 管理能力实现，依赖方向变为 `admin → queue/domain`，符合"业务依赖平台"的约束。

3. **admin 继续承担 jobs 管理 API**（`interfaces/jobs.go`）。它对外暴露任务/死信管理端点，依赖 queue 领域接口。管理界面归属 admin、领域实体归属 queue，是本次明确的边界；jobs 管理 API 是否整体迁出 admin 属后续演进，不阻塞本次。

4. **文件级搬迁，保持内容最小差异**：仅改 package 声明与 import，不动字段、gorm tag、注释结构。

## Risks / Trade-offs

- [import 遗漏 → 编译失败] → 完成后 `go build ./...` 全量验证。
- [测试引用遗漏 → 测试失败] → 跑 `go test ./internal/platform/queue/... ./internal/modules/admin/...`。
- [`worker.go` 引用 `admin/domain` 的 import_job 等误带] → 先 grep 确认仅引用目标三族，再搬迁。
- [实体移动后 admin 侧语义归属混淆] → design Decisions #3 已明确边界，spec 沉淀时进一步钉死。

## Migration Plan

- 纯代码搬迁，无部署 / 数据迁移步骤。
- 验证顺序：`go build ./...` → 相关包测试 → lint。
- 回滚：revert commit（表结构与行为均无变化，回滚无副作用）。

## Open Questions

无。jobs 管理 API 是否迁出 admin 是可安全延后的演进，不影响本次 spec/approach/task 划分。
