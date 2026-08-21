package queue

import (
	"testing"

	"jimu/internal/platform/queue/domain"
)

// BenchmarkExecuteJobDedup 基准：已成功任务的去重路径开销。
// 覆盖 FindByID 查询 + status 检查 + Ack 跳过，量化 at-least-once 幂等引入的固定成本。
func BenchmarkExecuteJobDedup(b *testing.B) {
	jobRepo := newFakeJobRepo()
	store := NewMySQLStore(jobRepo, &fakeHistoryRepo{}, &fakeDeadRepo{})
	// 预置已成功任务：每次 executeJob 应 Ack 跳过，不执行 handler
	jobRepo.jobs[1] = &domain.Job{ID: 1, Type: "noop", Status: domain.JobStatusSuccess, Attempts: 1, MaxAttempts: 3}

	consumer := &fakeConsumer{jobs: make(chan *JobData, 1)}
	wp := NewWorkerPool(WorkerConfig{Workers: 1}, consumer, store)
	data := &JobData{ID: 1, Type: "noop", Payload: "{}"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wp.executeJob(data)
	}
}
