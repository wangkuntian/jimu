package queue

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"jimu/internal/platform/queue/domain"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// fakeConsumer 内存消费者，验证 WorkerPool 依赖 Consumer 接口而非具体实现
type fakeConsumer struct {
	jobs chan *JobData
}

func (f *fakeConsumer) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	select {
	case j := <-f.jobs:
		return j, nil
	default:
		return nil, context.DeadlineExceeded
	}
}

func (f *fakeConsumer) Ack(ctx context.Context, j *JobData) error  { return nil }
func (f *fakeConsumer) Nack(ctx context.Context, j *JobData) error { return nil }

// fakeProducer 同时实现 Queue（Submit/SubmitDelayed/MoveDueJobs），验证 WorkerPool 生产者断言路径
type fakeProducer struct {
	submitted     []*JobData
	submittedDels []*JobData
}

func (p *fakeProducer) Submit(ctx context.Context, job *JobData) error {
	p.submitted = append(p.submitted, job)
	return nil
}

func (p *fakeProducer) SubmitDelayed(ctx context.Context, job *JobData, delay time.Duration) error {
	p.submittedDels = append(p.submittedDels, job)
	return nil
}

func (p *fakeProducer) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	return nil, context.DeadlineExceeded
}

func (p *fakeProducer) MoveDueJobs(ctx context.Context) (int, error) { return 0, nil }
func (p *fakeProducer) Ack(ctx context.Context, job *JobData) error  { return nil }
func (p *fakeProducer) Nack(ctx context.Context, job *JobData) error { return nil }

// fakeJobRepo 内存 Job 仓储
type fakeJobRepo struct {
	mu   sync.Mutex
	jobs map[uint64]*domain.Job
	next uint64
}

func newFakeJobRepo() *fakeJobRepo { return &fakeJobRepo{jobs: map[uint64]*domain.Job{}} }

func (r *fakeJobRepo) Create(ctx context.Context, job *domain.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	job.ID = r.next
	r.jobs[job.ID] = job
	return nil
}

func (r *fakeJobRepo) FindByID(ctx context.Context, id uint64) (*domain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return job, nil
}

func (r *fakeJobRepo) Update(ctx context.Context, job *domain.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[job.ID] = job
	return nil
}

func (r *fakeJobRepo) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]domain.Job, int64, error) {
	return nil, 0, nil
}

// fakeHistoryRepo 内存历史仓储
type fakeHistoryRepo struct {
	mu      sync.Mutex
	records []*domain.JobHistory
}

func (r *fakeHistoryRepo) Create(ctx context.Context, h *domain.JobHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, h)
	return nil
}

func (r *fakeHistoryRepo) ListByJobID(ctx context.Context, jobID uint64) ([]domain.JobHistory, error) {
	return nil, nil
}

// fakeDeadRepo 内存死信仓储
type fakeDeadRepo struct {
	mu     sync.Mutex
	items  []*domain.DeadLetter
	nextID uint64
}

func (r *fakeDeadRepo) Create(ctx context.Context, d *domain.DeadLetter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	d.ID = r.nextID
	r.items = append(r.items, d)
	return nil
}

func (r *fakeDeadRepo) List(ctx context.Context, offset, limit int, resolved bool) ([]domain.DeadLetter, int64, error) {
	return nil, 0, nil
}

func (r *fakeDeadRepo) MarkResolved(ctx context.Context, id uint64) error { return nil }

// fakeStore 构造内存版 MySQLStore
func fakeStore() *MySQLStore {
	return NewMySQLStore(newFakeJobRepo(), &fakeHistoryRepo{}, &fakeDeadRepo{})
}

func TestWorkerPoolConsumeJob(t *testing.T) {
	RegisterWorker("echo", func(ctx context.Context, payload string) error { return nil })

	consumer := &fakeConsumer{jobs: make(chan *JobData, 1)}
	consumer.jobs <- &JobData{ID: 1, Type: "echo", Payload: "hello"}

	wp := NewWorkerPool(WorkerConfig{Workers: 1, PollTimeout: 10 * time.Millisecond}, consumer, fakeStore())
	wp.Start()
	time.Sleep(100 * time.Millisecond)
	wp.Stop()
}

func TestWorkerPoolSubmitRejectsConsumerOnly(t *testing.T) {
	// fakeConsumer 只实现 Consumer 不实现 Queue，断言应在持久化前失败，避免孤儿记录
	consumer := &fakeConsumer{jobs: make(chan *JobData, 1)}
	jobRepo := newFakeJobRepo()
	store := NewMySQLStore(jobRepo, &fakeHistoryRepo{}, &fakeDeadRepo{})
	wp := NewWorkerPool(DefaultWorkerConfig, consumer, store)

	_, err := wp.Submit(context.Background(), "echo", `{"x":1}`)
	assert.Error(t, err)
	assert.Len(t, jobRepo.jobs, 0)

	_, err = wp.SubmitDelayed(context.Background(), "echo", `{"x":1}`, time.Minute)
	assert.Error(t, err)
	assert.Len(t, jobRepo.jobs, 0)
}

func TestWorkerPoolOutboxEventFailureWritesDeadLetter(t *testing.T) {
	// outbox 事件无 jobs 表行：执行失败应写死信，且不因状态机缺失而崩
	deadRepo := &fakeDeadRepo{}
	store := NewMySQLStore(newFakeJobRepo(), &fakeHistoryRepo{}, deadRepo)
	RegisterWorker("outbox:user.created", func(ctx context.Context, payload string) error {
		return errors.New("boom")
	})

	consumer := &fakeConsumer{jobs: make(chan *JobData, 1)}
	consumer.jobs <- &JobData{ID: 999, Type: "outbox:user.created", Payload: `{}`}

	wp := NewWorkerPool(WorkerConfig{Workers: 1, PollTimeout: 10 * time.Millisecond}, consumer, store)
	wp.Start()
	time.Sleep(100 * time.Millisecond)
	wp.Stop()

	deadRepo.mu.Lock()
	defer deadRepo.mu.Unlock()
	assert.Len(t, deadRepo.items, 1)
	assert.Equal(t, uint64(999), deadRepo.items[0].JobID)
}

func TestWorkerPoolOutboxEventSuccessSkipsTracking(t *testing.T) {
	// outbox 事件成功：无 jobs 行，跳过历史写入，不崩
	store := NewMySQLStore(newFakeJobRepo(), &fakeHistoryRepo{}, &fakeDeadRepo{})
	RegisterWorker("outbox:user.created", func(ctx context.Context, payload string) error { return nil })

	consumer := &fakeConsumer{jobs: make(chan *JobData, 1)}
	consumer.jobs <- &JobData{ID: 999, Type: "outbox:user.created", Payload: `{}`}

	wp := NewWorkerPool(WorkerConfig{Workers: 1, PollTimeout: 10 * time.Millisecond}, consumer, store)
	wp.Start()
	time.Sleep(100 * time.Millisecond)
	wp.Stop()
}

func TestWorkerPoolSubmitAndDelayed(t *testing.T) {
	producer := &fakeProducer{}
	wp := NewWorkerPool(DefaultWorkerConfig, producer, fakeStore())

	job, err := wp.Submit(context.Background(), "echo", `{"x":1}`)
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.Len(t, producer.submitted, 1)

	delayed, err := wp.SubmitDelayed(context.Background(), "echo", `{"x":1}`, time.Minute)
	assert.NoError(t, err)
	assert.NotNil(t, delayed)
	assert.Len(t, producer.submittedDels, 1)
}
