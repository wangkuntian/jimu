package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"jimu/internal/contract"
	"jimu/internal/platform/logger"

	"github.com/robfig/cron/v3"
)

// JobInfo 注册的任务信息
type JobInfo struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Cron       string    `json:"cron"`
	Enabled    bool      `json:"enabled"`
	LastRun    time.Time `json:"last_run,omitempty"`
	LastStatus string    `json:"last_status"`
	LastError  string    `json:"last_error,omitempty"`
	RunCount   int64     `json:"run_count"`
}

// CronScheduler 基于 robfig/cron 的定时任务调度器
type CronScheduler struct {
	cron    *cron.Cron
	logger  *logger.Logger
	errors  chan error
	entries []cron.EntryID

	mu     sync.RWMutex
	jobs   map[string]*JobInfo // id -> job info
	byName map[string]string   // name -> id
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
		jobs:   make(map[string]*JobInfo),
		byName: make(map[string]string),
	}
}

// AddNamedFunc 注册带名称的任务（支持管理）
func (s *CronScheduler) AddNamedFunc(id, name, spec string, cmd func()) error {
	info := &JobInfo{ID: id, Name: name, Cron: spec, Enabled: true}
	entryID, err := s.cron.AddFunc(spec, func() {
		s.recordRun(info, cmd)
	})
	if err != nil {
		return fmt.Errorf("add job %q: %w", spec, err)
	}
	s.mu.Lock()
	s.jobs[id] = info
	s.byName[name] = id
	s.entries = append(s.entries, entryID)
	s.mu.Unlock()
	return nil
}

// recordRun 记录任务执行并调用命令
func (s *CronScheduler) recordRun(info *JobInfo, cmd func()) {
	info.LastRun = time.Now()
	info.RunCount++
	start := time.Now()
	func() {
		defer func() {
			if r := recover(); r != nil {
				info.LastStatus = "failed"
				info.LastError = fmt.Sprintf("%v", r)
				s.logger.Error("job panic", "name", info.Name, "panic", fmt.Sprintf("%v", r))
			}
		}()
		cmd()
		info.LastStatus = "success"
		info.LastError = ""
	}()
	s.logger.Debug("job completed", "name", info.Name, "duration", time.Since(start))
}

// Jobs 返回所有注册任务的信息
func (s *CronScheduler) Jobs() []JobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]JobInfo, 0, len(s.jobs))
	for _, info := range s.jobs {
		result = append(result, *info)
	}
	return result
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
