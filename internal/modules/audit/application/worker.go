package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"jimu/internal/config"
	"jimu/internal/modules/audit/domain"
	"jimu/internal/platform/logger"
)

const flushTimeout = 5 * time.Second

type Worker struct {
	repo  domain.AuditRepository
	cfg   config.AuditConfig
	log   *logger.Logger
	queue chan domain.AuditLog

	mu        sync.RWMutex
	accepting bool
	started   bool
	stopOnce  sync.Once
	stop      chan context.Context
	done      chan struct{}
	result    error
}

func NewWorker(repo domain.AuditRepository, cfg config.AuditConfig, log *logger.Logger) *Worker {
	return &Worker{
		repo:  repo,
		cfg:   cfg,
		log:   log,
		queue: make(chan domain.AuditLog, cfg.QueueSize),
		stop:  make(chan context.Context, 1),
		done:  make(chan struct{}),
	}
}

func (w *Worker) Start(context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started {
		return errors.New("audit worker already started")
	}
	w.started = true
	w.accepting = true
	go w.run()
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	started := w.started
	w.accepting = false
	w.mu.Unlock()
	if !started {
		return nil
	}
	w.stopOnce.Do(func() { w.stop <- ctx })
	select {
	case <-w.done:
		return w.result
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) Enqueue(log domain.AuditLog) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if !w.accepting {
		return false
	}
	select {
	case w.queue <- log:
		return true
	default:
		return false
	}
}

func (w *Worker) run() {
	ticker := time.NewTicker(time.Duration(w.cfg.FlushIntervalMS) * time.Millisecond)
	defer ticker.Stop()
	defer close(w.done)
	batch := make([]domain.AuditLog, 0, w.cfg.BatchSize)

	flush := func(ctx context.Context) error {
		if len(batch) == 0 {
			return nil
		}
		logs := append([]domain.AuditLog(nil), batch...)
		batch = batch[:0]
		return w.repo.CreateBatch(ctx, logs)
	}
	regularFlush := func() {
		ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
		defer cancel()
		if err := flush(ctx); err != nil && w.log != nil {
			w.log.Error("audit batch write failed", "error", err)
		}
	}

	for {
		select {
		case item := <-w.queue:
			batch = append(batch, item)
			if len(batch) >= w.cfg.BatchSize {
				regularFlush()
			}
		case <-ticker.C:
			regularFlush()
		case stopCtx := <-w.stop:
			for {
				select {
				case item := <-w.queue:
					batch = append(batch, item)
					if len(batch) >= w.cfg.BatchSize {
						w.result = errors.Join(w.result, flush(stopCtx))
					}
				default:
					w.result = errors.Join(w.result, flush(stopCtx))
					return
				}
			}
		}
	}
}
