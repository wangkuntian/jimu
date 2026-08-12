## Why

平台基础设施层 `internal/platform/queue` 直接依赖业务模块 `internal/modules/admin/domain`
（`Job` / `JobHistory` / `DeadLetter` 实体及仓储接口），依赖方向颠倒：平台层不该依赖业务层。
这阻碍 queue 作为独立平台件被复用，也让架构边界模糊。趁将 queue 沉淀为能力规格之前修正，
避免把错误的依赖方向固化进 spec。

## What Changes

- 新建 `internal/platform/queue/domain/` 包，容纳队列领域实体。
- 从 `internal/modules/admin/domain/` 移入 3 个实体文件（含仓储接口）：
  - `job.go` — `Job`、`JobStatus`、`JobRepository`
  - `history.go` — `JobHistory`、`JobHistoryRepository`
  - `dead_letter.go` — `DeadLetter`、`DeadLetterRepository`
- 更新 6 个调用方 import 路径：
  - queue 自身 2 个：`mysql_store.go`、`worker.go`
  - admin 侧 4 个：`infrastructure/job_repository.go`、`infrastructure/job_history_repository.go`、
    `infrastructure/dead_letter_repository.go`、`interfaces/jobs.go`
- 更新 4 个测试引用：admin 侧 3 个 repository 测试、queue `worker_test.go`。
- `admin/domain` 保留 `apikey.go` / `import_job.go`，不受影响。
- 纯代码搬迁：无行为变化、无表结构变化、无 migrations 变更。

## Capabilities

### New Capabilities

无。纯重构，`skip_specs: true`。queue 能力规格由后续独立 change 沉淀。

### Modified Capabilities

无。无 spec 级行为变化。

## Impact

- **依赖方向反转**：

  ```
  之前: platform/queue ──▶ modules/admin/domain     (反了)
  之后: admin/interfaces & infrastructure ──▶ platform/queue/domain
  ```

- **代码**：6 个现有文件改 import + 3 个新文件（`platform/queue/domain/`）+ 4 个测试调整；`admin/domain` 删除 3 个文件。
- **表结构 / migrations**：无变化（`jobs`、`job_history`、`dead_letters` 表原样）。
- **对外 API / 配置**：无变化。
- **outbox**：不受影响（仅依赖 `queue.Queue` 接口与 `queue.JobData`）。
