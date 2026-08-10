# Scheduler Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scheduler 任务定义持久化到 MySQL，重启恢复；支持 memory/mysql 切换；多实例用分布式锁防重复执行。

**Architecture:** 抽 `Store` 接口（List/Save/Delete），`MemoryStore` 保持现有内存行为，`MySQLStore` 落 `scheduled_jobs` 表。`CronScheduler.AddNamedFunc` 注册时若启用 mysql store 则写库；启动时从 store 加载任务恢复 cron。多实例协调复用 `platform/redis/lock.go` 分布式锁（每任务执行前加锁，防多实例重复执行）。

**Tech Stack:** Go 1.26.5, 现有 `robfig/cron`, `gorm.io/gorm`, 现有 `platform/redis/lock.go`

## Global Constraints

- 模块：`jimu`
- 迁移序号：`013`（现有最大 `012`）
- `scheduler.store` 枚举：`memory`（默认）、`mysql`
- 任务定义表 `scheduled_jobs`：`id`、`name`、`cron`、`enabled`、`created_at`、`updated_at`、`deleted_at`
- `AddNamedFunc` 已存在（`internal/platform/scheduler/scheduler.go:56`），持久化复用该入口
- `AddFunc`（匿名任务）保持不持久化，向后兼容
- 遵循现有代码风格：中文注释、迁移字段 COMMENT

---

### Task 1: Store 接口 + JobDef + MemoryStore

**Files:**
- Create: `internal/platform/scheduler/store.go`
- Create: `internal/platform/scheduler/memory_store.go`
- Create: `internal/platform/scheduler/store_test.go`

**Interfaces:**
- Consumes: 无
- Produces:
  - `JobDef` struct：`ID string`、`Name string`、`Cron string`、`Enabled bool`、`CreatedAt time.Time`、`UpdatedAt time.Time`
  - `Store` 接口：`List(ctx) ([]JobDef, error)`、`Save(ctx, JobDef) error`、`Delete(ctx, string) error`
  - `NewMemoryStore() *MemoryStore` 实现 `Store`

- [ ] **Step 1: 写失败测试**

```go
// internal/platform/scheduler/store_test.go
package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	job := JobDef{ID: "job1", Name: "Test", Cron: "@every 1m", Enabled: true}
	assert.NoError(t, store.Save(ctx, job))

	list, err := store.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "job1", list[0].ID)

	assert.NoError(t, store.Delete(ctx, "job1"))
	list, err = store.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 0)
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/platform/scheduler/ -run TestMemoryStore -v`
Expected: 编译失败，`JobDef`/`Store`/`NewMemoryStore` 未定义

- [ ] **Step 3: 写 `store.go`**

```go
// internal/platform/scheduler/store.go
package scheduler

import (
	"context"
	"time"
)

// JobDef 持久化任务定义
type JobDef struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Cron      string    `json:"cron"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store 任务定义存储接口
type Store interface {
	// List 列出所有任务定义
	List(ctx context.Context) ([]JobDef, error)
	// Save 保存任务定义（新增或更新）
	Save(ctx context.Context, job JobDef) error
	// Delete 删除任务定义
	Delete(ctx context.Context, id string) error
}
```

- [ ] **Step 4: 写 `memory_store.go`**

```go
// internal/platform/scheduler/memory_store.go
package scheduler

import (
	"context"
	"sync"
	"time"
)

// MemoryStore 内存任务定义存储（不持久化，重启丢失）
type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]JobDef
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{jobs: make(map[string]JobDef)}
}

// List 列出所有任务定义
func (s *MemoryStore) List(ctx context.Context) ([]JobDef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]JobDef, 0, len(s.jobs))
	for _, job := range s.jobs {
		result = append(result, job)
	}
	return result, nil
}

// Save 保存任务定义
func (s *MemoryStore) Save(ctx context.Context, job JobDef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if _, ok := s.jobs[job.ID]; ok {
		job.UpdatedAt = now
	} else {
		job.CreatedAt = now
		job.UpdatedAt = now
	}
	s.jobs[job.ID] = job
	return nil
}

// Delete 删除任务定义
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, id)
	return nil
}

var _ Store = (*MemoryStore)(nil)
```

- [ ] **Step 5: 运行测试验证通过**

Run: `go test ./internal/platform/scheduler/ -run TestMemoryStore -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/platform/scheduler/store.go internal/platform/scheduler/memory_store.go internal/platform/scheduler/store_test.go
git commit -m "feat(scheduler): add Store interface with memory implementation"
```

---

### Task 2: MySQLStore + 迁移

**Files:**
- Create: `internal/platform/scheduler/mysql_store.go`
- Create: `migrations/013_create_scheduled_jobs.sql`
- Create: `internal/platform/scheduler/mysql_store_test.go`

**Interfaces:**
- Consumes: `Store` 接口（Task 1）、`gorm.DB`
- Produces: `NewMySQLStore(db *gorm.DB) *MySQLStore` 实现 `Store`

- [ ] **Step 1: 写迁移**

```sql
-- migrations/013_create_scheduled_jobs.sql
-- +goose Up
CREATE TABLE scheduled_jobs (
    id VARCHAR(64) NOT NULL COMMENT '任务 ID',
    name VARCHAR(128) NOT NULL COMMENT '任务名称',
    cron VARCHAR(64) NOT NULL COMMENT 'cron 表达式',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at DATETIME DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='定时任务定义表';

-- +goose Down
DROP TABLE scheduled_jobs;
```

- [ ] **Step 2: 写失败测试**

```go
// internal/platform/scheduler/mysql_store_test.go
package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// testDB 提供内存 SQLite 或 mock gorm.DB
func TestMySQLStore(t *testing.T) {
	db, err := testDB()
	assert.NoError(t, err)
	store := NewMySQLStore(db)
	ctx := context.Background()

	job := JobDef{ID: "job1", Name: "Test", Cron: "@every 1m", Enabled: true}
	assert.NoError(t, store.Save(ctx, job))

	list, err := store.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
}
```

- [ ] **Step 3: 运行测试验证失败**

Run: `go test ./internal/platform/scheduler/ -run TestMySQLStore -v`
Expected: 编译失败，`NewMySQLStore` 未定义

- [ ] **Step 4: 写 `mysql_store.go`**

```go
// internal/platform/scheduler/mysql_store.go
package scheduler

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// jobModel 数据库映射模型
type jobModel struct {
	ID        string         `gorm:"primaryKey;size:64" json:"id"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Cron      string         `gorm:"size:64;not null" json:"cron"`
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名
func (jobModel) TableName() string { return "scheduled_jobs" }

// MySQLStore 基于 MySQL 的任务定义存储
type MySQLStore struct {
	db *gorm.DB
}

// NewMySQLStore 创建 MySQL 存储
func NewMySQLStore(db *gorm.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

// List 列出所有任务定义
func (s *MySQLStore) List(ctx context.Context) ([]JobDef, error) {
	var models []jobModel
	if err := s.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]JobDef, 0, len(models))
	for _, m := range models {
		result = append(result, JobDef{
			ID:        m.ID,
			Name:      m.Name,
			Cron:      m.Cron,
			Enabled:   m.Enabled,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		})
	}
	return result, nil
}

// Save 保存任务定义
func (s *MySQLStore) Save(ctx context.Context, job JobDef) error {
	return s.db.WithContext(ctx).Save(&jobModel{
		ID:      job.ID,
		Name:    job.Name,
		Cron:    job.Cron,
		Enabled: job.Enabled,
	}).Error
}

// Delete 删除任务定义
func (s *MySQLStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&jobModel{}).Error
}

var _ Store = (*MySQLStore)(nil)
```

- [ ] **Step 5: 写测试辅助 `testDB`**

在 `mysql_store_test.go` 补 `testDB`（用 `github.com/glebarez/go-sqlite` 内存库，已在 go.mod indirect）：

```go
func testDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&jobModel{}); err != nil {
		return nil, err
	}
	return db, nil
}
```

- [ ] **Step 6: 运行测试验证通过**

Run: `go test ./internal/platform/scheduler/ -run TestMySQLStore -v`
Expected: PASS

- [ ] **Step 7: 提交**

```bash
git add internal/platform/scheduler/mysql_store.go internal/platform/scheduler/mysql_store_test.go migrations/013_create_scheduled_jobs.sql
git commit -m "feat(scheduler): add MySQL store and migration"
```

---

### Task 3: Scheduler 持久化改造 + 配置

**Files:**
- Modify: `internal/platform/scheduler/scheduler.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/validate.go`
- Modify: `internal/app/container.go`
- Modify: `configs/app.yaml`
- Modify: `configs/app.prod.yaml`
- Modify: `README.md`

**Interfaces:**
- Consumes: `Store` 接口（Task 1）、`MySQLStore`（Task 2）、`redis.Lock`（现有）
- Produces: `CronScheduler` 增加 `store Store` 字段与 `NewWithStore(log, store, lock) *CronScheduler`；`AddNamedFunc` 持久化；`RestoreFromStore(ctx) error` 启动恢复

- [ ] **Step 1: 修改 config**

`internal/config/config.go` 加：

```go
// Scheduler 存储类型枚举
const (
	SchedulerStoreMemory = "memory"
	SchedulerStoreMySQL  = "mysql"
)

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	Store string `mapstructure:"store"` // memory / mysql
}
```

`Config` struct 加：`Scheduler SchedulerConfig \`mapstructure:"scheduler"\``

`validate.go` `validateCommon()` 加：

```go
if !contains(validSchedulerStores, c.Scheduler.Store) {
	return fmt.Errorf("invalid scheduler.store: %q, must be one of %v", c.Scheduler.Store, validSchedulerStores)
}
```

config.go 定义：

```go
var validSchedulerStores = []string{SchedulerStoreMemory, SchedulerStoreMySQL}
```

- [ ] **Step 2: 修改 scheduler.go**

`CronScheduler` struct 加字段：

```go
type CronScheduler struct {
	cron    *cron.Cron
	logger  *logger.Logger
	errors  chan error
	entries []cron.EntryID
	store   Store
	lock    *redis.Lock // 多实例协调（可选）

	mu     sync.RWMutex
	jobs   map[string]*JobInfo
	byName map[string]string
}
```

`New` 保持内存行为，新增 `NewWithStore`：

```go
// New 创建内存调度器（store 默认 MemoryStore，不持久化）
func New(log *logger.Logger) *CronScheduler {
	return NewWithStore(log, NewMemoryStore(), nil)
}

// NewWithStore 创建调度器，指定任务定义存储与分布式锁（锁可选，nil 时不加锁）
func NewWithStore(log *logger.Logger, store Store, lock *redis.Lock) *CronScheduler {
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
		store:  store,
		lock:   lock,
	}
}
```

`AddNamedFunc` 持久化（在注册 cron 前写 store）：

```go
func (s *CronScheduler) AddNamedFunc(id, name, spec string, cmd func()) error {
	if s.store != nil {
		if err := s.store.Save(context.Background(), JobDef{ID: id, Name: name, Cron: spec, Enabled: true}); err != nil {
			return fmt.Errorf("persist job %q: %w", id, err)
		}
	}
	info := &JobInfo{ID: id, Name: name, Cron: spec, Enabled: true}
	entryID, err := s.cron.AddFunc(spec, func() {
		s.recordRun(info, cmd)
	})
	...
}
```

执行前加锁（若配置锁）：

```go
	entryID, err := s.cron.AddFunc(spec, func() {
		if s.lock != nil {
			lockCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			ok, err := s.lock.TryLock(lockCtx, "job:"+id, 30*time.Second)
			cancel()
			if err != nil || !ok {
				s.logger.Debug("job skipped, lock not acquired", "id", id)
				return
			}
			defer s.lock.Unlock(context.Background(), "job:"+id)
		}
		s.recordRun(info, cmd)
	})
```

`RestoreFromStore` 新增：

```go
// RestoreFromStore 从 store 加载任务并恢复注册（启动时调用）
func (s *CronScheduler) RestoreFromStore(ctx context.Context, cmdFactory func(id string) func()) error {
	jobs, err := s.store.List(ctx)
	if err != nil {
		return fmt.Errorf("list scheduled jobs: %w", err)
	}
	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		fn := cmdFactory(job.ID)
		if fn == nil {
			continue
		}
		if err := s.AddNamedFunc(job.ID, job.Name, job.Cron, fn); err != nil {
			return fmt.Errorf("restore job %q: %w", job.ID, err)
		}
	}
	return nil
}
```

- [ ] **Step 3: 修改 container.go**

`NewScheduler` 处按配置选择 store：

```go
	var schedStore scheduler.Store = scheduler.NewMemoryStore()
	if cfg.Scheduler.Store == config.SchedulerStoreMySQL {
		schedStore = scheduler.NewMySQLStore(dbConn)
	}
	var sched *scheduler.CronScheduler
	if cfg.Scheduler.Store == config.SchedulerStoreMySQL {
		sched = scheduler.NewWithStore(log, schedStore, lock)
	} else {
		sched = scheduler.NewWithStore(log, schedStore, nil)
	}
```

- [ ] **Step 4: 更新配置文件**

`configs/app.yaml` 加：

```yaml
scheduler:
  store: memory
```

- [ ] **Step 5: 验证编译**

Run: `go build ./...`
Expected: 编译通过

- [ ] **Step 6: 更新 README.md**

配置表加 `scheduler.store`，特性列表注明 Scheduler 支持 MySQL 持久化 + 多实例协调。

- [ ] **Step 7: 提交**

```bash
git add internal/platform/scheduler/scheduler.go internal/config/config.go internal/config/validate.go internal/app/container.go configs/app.yaml configs/app.prod.yaml README.md
git commit -m "feat(scheduler): persist named jobs and restore on startup"
```

---

### Self-Review 记录

**1. Spec 覆盖：**
- Store 接口 + Memory/MySQL 实现 → Task 1/2
- `scheduler.store` 枚举 → Task 3
- 启动加载 → Task 3 `RestoreFromStore`
- 多实例协调（分布式锁）→ Task 3 `AddNamedFunc` 锁
- 迁移 → Task 2

**2. Placeholder 扫描：** 无 TBD/TODO。`RestoreFromStore` 的 `cmdFactory` 由调用方提供恢复函数映射——这是必要解耦，非占位。

**3. Type 一致性：**
- `JobDef`/`Store` Task 1 定义，Task 2/3 引用一致。
- `NewWithStore` 签名 Task 3 定义，container 装配一致。
- `jobModel` Task 2 定义，`MySQLStore` 方法引用一致。
