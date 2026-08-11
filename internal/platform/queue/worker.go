package queue

import (
	"context"
	"sync"
	"time"

	"jimu/internal/modules/admin/domain"
	apperrors "jimu/internal/shared/errors"
)

var (
	workersMu    sync.RWMutex
	jobWorkerMap = map[string]WorkerFunc{}
)

// WorkerFunc 任务处理函数
type WorkerFunc func(ctx context.Context, payload string) error

// RegisterWorker 注册任务处理器
func RegisterWorker(jobType string, fn WorkerFunc) {
	workersMu.Lock()
	defer workersMu.Unlock()
	jobWorkerMap[jobType] = fn
}

// GetWorker 获取任务处理器
func GetWorker(jobType string) (WorkerFunc, bool) {
	workersMu.RLock()
	defer workersMu.RUnlock()
	fn, ok := jobWorkerMap[jobType]
	return fn, ok
}

// WorkerConfig Worker 池配置
type WorkerConfig struct {
	Workers     int
	QueueName   string
	PollTimeout time.Duration
	MaxRetries  int
}

// DefaultWorkerConfig 默认配置
var DefaultWorkerConfig = WorkerConfig{
	Workers:     10,
	QueueName:   QueueKey,
	PollTimeout: 5 * time.Second,
	MaxRetries:  3,
}

// WorkerPool Worker 池
type WorkerPool struct {
	config   WorkerConfig
	queue    Consumer // 依赖接口而非具体实现
	store    *MySQLStore
	strategy RetryStrategy
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewWorkerPool 创建 Worker 池
func NewWorkerPool(config WorkerConfig, queue Consumer, store *MySQLStore) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		config:   config,
		queue:    queue,
		store:    store,
		strategy: DefaultRetry,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动 Worker 池
func (p *WorkerPool) Start() {
	for i := 0; i < p.config.Workers; i++ {
		p.wg.Add(1)
		go p.workerLoop(i)
	}
	p.wg.Add(1)
	go p.delayedJobScanner()
}

// Stop 停止 Worker 池
func (p *WorkerPool) Stop() {
	p.cancel()
	p.wg.Wait()
}

func (p *WorkerPool) workerLoop(id int) {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}
		jobData, err := p.queue.Consume(p.ctx, p.config.PollTimeout)
		if err != nil {
			continue
		}
		p.executeJob(jobData)
	}
}

func (p *WorkerPool) delayedJobScanner() {
	defer p.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			// Consumer 接口无 MoveDueJobs，支持延迟队列的实现（如 Redis）通过断言触发
			if m, ok := p.queue.(interface {
				MoveDueJobs(context.Context) (int, error)
			}); ok {
				_, _ = m.MoveDueJobs(p.ctx)
			}
		}
	}
}

func (p *WorkerPool) executeJob(data *JobData) {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Minute)
	defer cancel()

	_ = p.store.MarkRunning(ctx, data.ID)

	fn, ok := GetWorker(data.Type)
	if !ok {
		_ = p.store.MarkFailed(ctx, data.ID, data.Type, data.Payload,
			apperrors.New(apperrors.CodeInternalError, "no worker for type: "+data.Type), 0)
		return
	}

	start := time.Now()
	err := fn(ctx, data.Payload)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		_ = p.store.MarkFailed(ctx, data.ID, data.Type, data.Payload, err, duration)
	} else {
		_ = p.store.MarkSuccess(ctx, data.ID, duration)
	}
}

// Submit 提交任务
func (p *WorkerPool) Submit(ctx context.Context, jobType, payload string) (*domain.Job, error) {
	producer, ok := p.queue.(Queue)
	if !ok {
		return nil, apperrors.New(apperrors.CodeInternalError, "queue does not support submit")
	}
	job, err := p.store.CreateJob(ctx, jobType, payload, p.config.MaxRetries)
	if err != nil {
		return nil, err
	}
	if err := producer.Submit(ctx, &JobData{ID: job.ID, Type: jobType, Payload: payload}); err != nil {
		return nil, err
	}
	return job, nil
}

// SubmitDelayed 提交延迟任务
func (p *WorkerPool) SubmitDelayed(ctx context.Context, jobType, payload string, delay time.Duration) (*domain.Job, error) {
	producer, ok := p.queue.(Queue)
	if !ok {
		return nil, apperrors.New(apperrors.CodeInternalError, "queue does not support submit")
	}
	job, err := p.store.CreateJob(ctx, jobType, payload, p.config.MaxRetries)
	if err != nil {
		return nil, err
	}
	if err := producer.SubmitDelayed(ctx, &JobData{ID: job.ID, Type: jobType, Payload: payload}, delay); err != nil {
		return nil, err
	}
	return job, nil
}
