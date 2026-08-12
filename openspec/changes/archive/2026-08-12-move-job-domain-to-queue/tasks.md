## 1. 搬迁实体到 queue/domain

- [x] 1.1 创建 `internal/platform/queue/domain/` 包目录
- [x] 1.2 从 `internal/modules/admin/domain/job.go` 移动 `Job` / `JobStatus` / `JobRepository` 到 `internal/platform/queue/domain/job.go`，package 改为 `domain`
- [x] 1.3 从 `internal/modules/admin/domain/history.go` 移动 `JobHistory` / `JobHistoryRepository` 到 `internal/platform/queue/domain/history.go`
- [x] 1.4 从 `internal/modules/admin/domain/dead_letter.go` 移动 `DeadLetter` / `DeadLetterRepository` 到 `internal/platform/queue/domain/dead_letter.go`
- [x] 1.5 删除 `internal/modules/admin/domain/` 下对应的 3 个源文件

## 2. 更新调用方 import

- [x] 2.1 `internal/platform/queue/mysql_store.go` 改用新包
- [x] 2.2 `internal/platform/queue/worker.go` 改用新包
- [x] 2.3 `internal/modules/admin/infrastructure/job_repository.go` 改用新包
- [x] 2.4 `internal/modules/admin/infrastructure/job_history_repository.go` 改用新包
- [x] 2.5 `internal/modules/admin/infrastructure/dead_letter_repository.go` 改用新包
- [x] 2.6 `internal/modules/admin/interfaces/jobs.go` 改用新包

## 3. 更新测试引用

- [x] 3.1 `internal/modules/admin/infrastructure/` 下 2 个 repository 测试（job_history、dead_letter）改 import
- [x] 3.2 `internal/platform/queue/worker_test.go` 改 import

## 4. 验证

- [x] 4.1 `go build ./...` 通过
- [x] 4.2 `go test ./internal/platform/queue/... ./internal/modules/admin/...` 通过
- [x] 4.3 `golangci-lint run` 通过（本次改动文件零告警；4 个预存告警在 csrf.go / s3.go，与本 change 无关）
