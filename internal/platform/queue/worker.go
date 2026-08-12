package queue

import (
	"context"
	"log"
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
			// Consumer 接口无 MoveDueJobs/RequeueExpired，支持延迟队列与
			// 可见性超时恢复的实现（如 Redis）通过断言触发
			if m, ok := p.queue.(interface {
				MoveDueJobs(context.Context) (int, error)
			}); ok {
				_, _ = m.MoveDueJobs(p.ctx)
			}
			if r, ok := p.queue.(interface {
				RequeueExpired(context.Context) (int, error)
			}); ok {
				_, _ = r.RequeueExpired(p.ctx)
			}
		}
	}
}

func (p *WorkerPool) executeJob(data *JobData) {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Minute)
	defer cancel()

	if p.store != nil {
		if err := p.store.MarkRunning(ctx, data.ID); err != nil {
			log.Printf("queue: mark running job %d: %v", data.ID, err)
		}
	}

	fn, ok := GetWorker(data.Type)
	if !ok {
		if p.store != nil {
			if err := p.store.MarkFailed(ctx, data.ID, data.Type, data.Payload,
				apperrors.New(apperrors.CodeInternalError, "no worker for type: "+data.Type), 0); err != nil {
				log.Printf("queue: mark failed job %d: %v", data.ID, err)
			}
		}
		_ = p.queue.Nack(ctx, data)
		return
	}

	start := time.Now()
	err := fn(ctx, data.Payload)
	duration := time.Since(start).Milliseconds()

	if p.store != nil {
		if err != nil {
			if merr := p.store.MarkFailed(ctx, data.ID, data.Type, data.Payload, err, duration); merr != nil {
				log.Printf("queue: mark failed job %d: %v", data.ID, merr)
			}
		} else {
			if merr := p.store.MarkSuccess(ctx, data.ID, duration); merr != nil {
				log.Printf("queue: mark success job %d: %v", data.ID, merr)
			}
		}
	}

	// 任务语义：成功 Ack；失败时按 store 决策是否重试。
	// store 在失败且未耗尽重试次数时返回 requeue=true，worker 对 Redis 重新入队；
	// Kafka/RabbitMQ 的 Nack 为 no-op，但 store 已把 job 状态置回 pending，
	// 重试提交由 Submit/SubmitDelayed 的下次调用驱动。
	if err != nil {
		if p.store != nil {
			// store 判定是否需要重试（看 Attempts vs MaxAttempts）
			requeue := p.needRetry(ctx, data)
			if requeue {
				_ = p.queue.Nack(ctx, data)
			} else {
				_ = p.queue.Ack(ctx, data)
			}
		} else {
			// 无 store 时也重试，避免静默丢任务
			_ = p.queue.Nack(ctx, data)
		}
	} else {
		_ = p.queue.Ack(ctx, data)
	}
}

// needRetry 查询 store 判断任务是否还有重试机会。
// 无 jobs 行（outbox 事件）或查询失败时保守返回 false（不无限重试）。
func (p *WorkerPool) needRetry(ctx context.Context, data *JobData) bool {
	job, err := p.store.jobRepo.FindByID(ctx, data.ID)
	if err != nil {
		return false
	}
	return job.Attempts < job.MaxAttempts
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
