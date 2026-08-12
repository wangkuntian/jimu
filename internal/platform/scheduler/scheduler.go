package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"jimu/internal/contract"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/redis"

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
	store   Store
	lock    *redis.Lock // 多实例协调（可选）

	mu        sync.RWMutex
	jobs      map[string]*JobInfo // id -> job info
	byName    map[string]string   // name -> id
	cmdByID   map[string]func()   // id -> 原始命令（手动触发用）
	entryByID map[string]cron.EntryID
}

// NewWithStore 创建调度器，指定任务定义存储与分布式锁（锁可选，nil 时不加锁）
func NewWithStore(log *logger.Logger, store Store, lock *redis.Lock) *CronScheduler {
	c := cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger),
		cron.SkipIfStillRunning(cron.DefaultLogger),
	))

	return &CronScheduler{
		cron:      c,
		logger:    log,
		errors:    make(chan error, 16),
		jobs:      make(map[string]*JobInfo),
		byName:    make(map[string]string),
		cmdByID:   make(map[string]func()),
		entryByID: make(map[string]cron.EntryID),
		store:     store,
		lock:      lock,
	}
}

// AddNamedFunc 注册带名称的任务（支持管理）；注册前将任务定义持久化到 store
func (s *CronScheduler) AddNamedFunc(id, name, spec string, cmd func()) error {
	if s.store != nil {
		if err := s.store.Save(context.Background(), JobDef{ID: id, Name: name, Cron: spec, Enabled: true}); err != nil {
			return fmt.Errorf("persist job %q: %w", id, err)
		}
	}
	info := &JobInfo{ID: id, Name: name, Cron: spec, Enabled: true}
	entryID, err := s.cron.AddFunc(spec, func() {
		if s.lock != nil {
			// 多实例协调：只有成功获取分布式锁的实例才执行任务
			lockCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			result, err := s.lock.TryAcquire(lockCtx, "job:"+id, 30*time.Second)
			cancel()
			if err != nil {
				s.logger.Debug("job skipped, lock not acquired", "id", id, "error", err.Error())
				return
			}
			defer s.lock.Release(context.Background(), result)
		}
		s.recordRun(info, cmd)
	})
	if err != nil {
		return fmt.Errorf("add job %q: %w", spec, err)
	}
	s.mu.Lock()
	s.jobs[id] = info
	s.byName[name] = id
	s.cmdByID[id] = cmd
	s.entryByID[id] = entryID
	s.entries = append(s.entries, entryID)
	s.mu.Unlock()
	return nil
}

// RestoreFromStore 从 store 加载任务并恢复注册（启动时调用），返回已恢复的 enabled 任务 id 列表
func (s *CronScheduler) RestoreFromStore(ctx context.Context, cmdFactory func(id string) func()) ([]string, error) {
	jobs, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scheduled jobs: %w", err)
	}
	var restored []string
	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		fn := cmdFactory(job.ID)
		if fn == nil {
			continue
		}
		if err := s.AddNamedFunc(job.ID, job.Name, job.Cron, fn); err != nil {
			return restored, fmt.Errorf("restore job %q: %w", job.ID, err)
		}
		restored = append(restored, job.ID)
	}
	return restored, nil
}

// recordRun 记录任务执行并调用命令。
// 状态字段读写均持锁，与 Jobs()/SetEnabled 并发安全。
func (s *CronScheduler) recordRun(info *JobInfo, cmd func()) {
	s.mu.Lock()
	info.LastRun = time.Now()
	info.RunCount++
	s.mu.Unlock()
	start := time.Now()
	func() {
		defer func() {
			if r := recover(); r != nil {
				s.mu.Lock()
				info.LastStatus = "failed"
				info.LastError = fmt.Sprintf("%v", r)
				s.mu.Unlock()
				s.logger.Error("job panic", "name", info.Name, "panic", fmt.Sprintf("%v", r))
			}
		}()
		cmd()
		s.mu.Lock()
		info.LastStatus = "success"
		info.LastError = ""
		s.mu.Unlock()
	}()
	s.logger.Debug("job completed", "name", info.Name, "duration", time.Since(start))
}

// TriggerJob 手动触发任务（不依赖 cron 调度，立即执行）
func (s *CronScheduler) TriggerJob(ctx context.Context, id string) error {
	s.mu.RLock()
	info, ok := s.jobs[id]
	cmd := s.cmdByID[id]
	s.mu.RUnlock()
	if !ok || cmd == nil {
		return fmt.Errorf("job %q not found", id)
	}
	if !info.Enabled {
		return fmt.Errorf("job %q disabled", id)
	}
	// 与 cron 触发共用 recordRun，保证状态追踪一致
	go s.recordRun(info, cmd)
	return nil
}

// SetEnabled 暂停/恢复任务
// 暂停：从 cron 移除该 entry；恢复：按保存的 spec 与命令重新注册
func (s *CronScheduler) SetEnabled(ctx context.Context, id string, enabled bool) error {
	s.mu.Lock()
	info, ok := s.jobs[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("job %q not found", id)
	}
	cmd := s.cmdByID[id]
	info.Enabled = enabled
	s.mu.Unlock()

	if !enabled {
		s.mu.Lock()
		entryID, hasEntry := s.entryByID[id]
		if hasEntry {
			s.cron.Remove(entryID)
			delete(s.entryByID, id)
		}
		s.mu.Unlock()
		return nil
	}

	// 恢复：重新注册（若 entry 已移除）
	s.mu.RLock()
	_, hasEntry := s.entryByID[id]
	s.mu.RUnlock()
	if hasEntry || cmd == nil {
		return nil // 仍注册中，无需操作
	}
	entryID, err := s.cron.AddFunc(info.Cron, func() {
		s.recordRun(info, cmd)
	})
	if err != nil {
		return fmt.Errorf("re-add job %q: %w", id, err)
	}
	s.mu.Lock()
	s.entryByID[id] = entryID
	s.entries = append(s.entries, entryID)
	s.mu.Unlock()
	return nil
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

// EntryCount 返回已注册且启用的任务数量
func (s *CronScheduler) EntryCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entryByID)
}

// 确保 CronScheduler 实现了所需接口
var _ contract.JobRegistry = (*CronScheduler)(nil)
var _ contract.Component = (*CronScheduler)(nil)
var _ contract.ErrorSource = (*CronScheduler)(nil)
