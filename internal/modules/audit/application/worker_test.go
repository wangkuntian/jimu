package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"jimu/internal/config"
	"jimu/internal/modules/audit/domain"
	"jimu/internal/platform/logger"

	"go.uber.org/zap"
)

type fakeBatchRepository struct {
	mu      sync.Mutex
	batches [][]domain.AuditLog
	written chan struct{}
}

func (r *fakeBatchRepository) Create(context.Context, *domain.AuditLog) error { return nil }

func (r *fakeBatchRepository) CreateBatch(ctx context.Context, logs []domain.AuditLog) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	r.batches = append(r.batches, append([]domain.AuditLog(nil), logs...))
	r.mu.Unlock()
	select {
	case r.written <- struct{}{}:
	default:
	}
	return nil
}

func (r *fakeBatchRepository) List(context.Context, int, int) ([]domain.AuditLog, int64, error) {
	return nil, 0, nil
}

func testWorker(repo domain.AuditRepository, queueSize, batchSize int, flush time.Duration) *Worker {
	return NewWorker(repo, config.AuditConfig{
		QueueSize:       queueSize,
		BatchSize:       batchSize,
		FlushIntervalMS: int(flush / time.Millisecond),
	}, &logger.Logger{SugaredLogger: zap.NewNop().Sugar()})
}

func TestWorkerRejectsWhenQueueFull(t *testing.T) {
	repo := &fakeBatchRepository{written: make(chan struct{}, 1)}
	worker := testWorker(repo, 1, 1, time.Hour)
	worker.accepting = true
	worker.queue <- domain.AuditLog{Path: "/first"}

	if worker.Enqueue(domain.AuditLog{Path: "/second"}) {
		t.Fatal("Enqueue() = true for full queue")
	}
}

func TestWorkerFlushesConfiguredBatch(t *testing.T) {
	repo := &fakeBatchRepository{written: make(chan struct{}, 1)}
	worker := testWorker(repo, 4, 2, time.Hour)
	if err := worker.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !worker.Enqueue(domain.AuditLog{Path: "/one"}) || !worker.Enqueue(domain.AuditLog{Path: "/two"}) {
		t.Fatal("Enqueue() rejected available capacity")
	}
	select {
	case <-repo.written:
	case <-time.After(time.Second):
		t.Fatal("batch was not written")
	}
	if err := worker.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.batches) != 1 || len(repo.batches[0]) != 2 {
		t.Fatalf("batches = %#v", repo.batches)
	}
}

func TestWorkerStopDrainsAcceptedRecords(t *testing.T) {
	repo := &fakeBatchRepository{written: make(chan struct{}, 1)}
	worker := testWorker(repo, 4, 4, time.Hour)
	if err := worker.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	worker.Enqueue(domain.AuditLog{Path: "/one"})
	worker.Enqueue(domain.AuditLog{Path: "/two"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := worker.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.batches) != 1 || len(repo.batches[0]) != 2 {
		t.Fatalf("drained batches = %#v", repo.batches)
	}
}
