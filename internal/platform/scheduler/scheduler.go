package scheduler

import (
	"context"
	"fmt"

	"jimu/internal/contract"
	"jimu/internal/platform/logger"

	"github.com/robfig/cron/v3"
)

// CronScheduler 基于 robfig/cron 的定时任务调度器
type CronScheduler struct {
	cron    *cron.Cron
	logger  *logger.Logger
	errors  chan error
	entries []cron.EntryID
}

// New 创建 CronScheduler
func New(log *logger.Logger) *CronScheduler {
	c := cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger),
		cron.SkipIfStillRunning(cron.DefaultLogger),
	))

	return &CronScheduler{
		cron:   c,
		logger: log,
		errors: make(chan error, 16),
	}
}

// AddFunc 实现 contract.JobRegistry 接口
// spec 支持标准 cron 表达式或 @every 1m / @hourly / @daily 等简写
func (s *CronScheduler) AddFunc(spec string, cmd func()) error {
	entryID, err := s.cron.AddFunc(spec, func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("job panic recovered",
					"spec", spec,
					"panic", fmt.Sprintf("%v", r),
				)
				select {
				case s.errors <- fmt.Errorf("job panic: %v", r):
				default:
				}
			}
		}()
		s.logger.Debug("job starting", "spec", spec)
		cmd()
		s.logger.Debug("job completed", "spec", spec)
	})
	if err != nil {
		return fmt.Errorf("add job %q: %w", spec, err)
	}
	s.entries = append(s.entries, entryID)
	s.logger.Info("job registered", "spec", spec, "entry_id", entryID)
	return nil
}

// Start 实现 contract.Component 接口
func (s *CronScheduler) Start(_ context.Context) error {
	s.logger.Info("scheduler starting", "jobs", len(s.entries))
	s.cron.Start()
	return nil
}

// Stop 实现 contract.Component 接口
func (s *CronScheduler) Stop(_ context.Context) error {
	s.logger.Info("scheduler stopping")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("scheduler stopped")
	return nil
}

// Errors 返回任务错误通道（实现 contract.ErrorSource）
func (s *CronScheduler) Errors() <-chan error {
	return s.errors
}

// EntryCount 返回已注册任务数量
func (s *CronScheduler) EntryCount() int {
	return len(s.entries)
}

// 确保 CronScheduler 实现了所需接口
var _ contract.JobRegistry = (*CronScheduler)(nil)
var _ contract.Component = (*CronScheduler)(nil)
var _ contract.ErrorSource = (*CronScheduler)(nil)
