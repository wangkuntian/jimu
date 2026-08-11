# MQ 消费者接线 + Scheduler 持久化恢复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 兑现 outbox→MQ→消费→业务处理链路（统一事件走 outbox），并在启动时恢复 MySQL 持久化的定时任务定义。

**Architecture:** Task 1 补两个缺失仓储（job_history/dead_letters），Task 2 改 `CronScheduler.RestoreFromStore` 返回恢复的 id 列表，Task 3 重排 service 事件走 outbox + user module 改订阅全局总线，Task 4 在 bootstrap 注册 outbox 桥接 worker 与 event_bus 模式 `outbox:*` 桥接器，Task 5 在 container 接线 WorkerPool 并纳入生命周期，Task 6 在 bootstrap 用 RestoreFromStore 去重注册内置任务。改造顺序：先基础设施，再装配层。

**Tech Stack:** Go 1.26、Gorm（sqlite 测试）、robfig/cron v3、gin。测试用 `github.com/glebarez/sqlite` 内存库。

## Global Constraints

- 事件统一走 outbox：`Create`/`Update`/`Delete` 的 `PublishAsync` 直发全部删除，`Update`/`Delete` 补写 `outbox.Add`。
- 桥接 worker 发布到全局总线**裸业务主题**（`evt.EventType`），不是 `outbox:` 前缀。event_bus 模式用全局总线 `outbox:*` 桥接器转裸主题。
- `outboxTypeConverters` 转换表只覆盖三种事件：`user.created`、`user.updated`、`user.deleted`。未知类型返回错误。
- `RestoreFromStore` 签名改为 `([]string, error)`，返回已恢复的 enabled 任务 id 列表。
- 只改本计划列出的文件；不顺手重构无关代码。改动后必须同步更新 README.md。
- Commit message 用 Conventional Commits 英文，小写开头，无句号。

---

### Task 1: 补全 job_history / dead_letters 仓储实现

**Files:**
- Create: `internal/modules/admin/infrastructure/job_history_repository.go`
- Create: `internal/modules/admin/infrastructure/job_history_repository_test.go`
- Create: `internal/modules/admin/infrastructure/dead_letter_repository.go`
- Create: `internal/modules/admin/infrastructure/dead_letter_repository_test.go`

**Interfaces:**
- Consumes: `internal/modules/admin/domain/history.go` 的 `JobHistory` / `JobHistoryRepository`（Create/ListByJobID）；`internal/modules/admin/domain/dead_letter.go` 的 `DeadLetter` / `DeadLetterRepository`（Create/List/MarkResolved）。
- Produces: `NewMysqlJobHistoryRepository(db *gorm.DB) domain.JobHistoryRepository`、`NewMysqlDeadLetterRepository(db *gorm.DB) domain.DeadLetterRepository`。Task 5 的 `queue.NewMySQLStore` 用它们构造。

**先决知识**：迁移 `migrations/010_create_jobs.sql` 已建 `job_history`（id/job_id/status/error/duration_ms/started_at/ended_at）与 `dead_letters`（id/job_id/type/payload/fail_reason/failed_at/resolved/resolved_at）表。Gorm 模型 struct 的 `TableName()` 分别返回 `job_history`、`dead_letters`。现有模式参考 `job_repository.go`（`mysqlJobRepository`，`r.db.WithContext(ctx).Create`）。

- [ ] **Step 1: 写失败测试 `job_history_repository_test.go`**

```go
package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/modules/admin/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func newHistoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&domain.JobHistory{}, &domain.DeadLetter{}, &domain.Job{}))
	return db
}

func TestMysqlJobHistoryRepositoryCreateAndList(t *testing.T) {
	db := newHistoryTestDB(t)
	repo := NewMysqlJobHistoryRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &domain.JobHistory{JobID: 1, Status: "success", Duration: 10})
	assert.NoError(t, err)

	history, err := repo.ListByJobID(ctx, 1)
	assert.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "success", history[0].Status)

	// 不存在的 job 返回空列表
	empty, err := repo.ListByJobID(ctx, 999)
	assert.NoError(t, err)
	assert.Len(t, empty, 0)
}
```

- [ ] **Step 2: 运行测试确认编译失败（`NewMysqlJobHistoryRepository` 未定义）**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/modules/admin/infrastructure/ -run TestMysqlJobHistoryRepositoryCreateAndList`
Expected: 编译错误 `undefined: NewMysqlJobHistoryRepository`

- [ ] **Step 3: 写最小实现 `job_history_repository.go`**

```go
package infrastructure

import (
	"context"

	"jimu/internal/modules/admin/domain"

	"gorm.io/gorm"
)

type mysqlJobHistoryRepository struct {
	db *gorm.DB
}

// NewMysqlJobHistoryRepository 创建任务历史 MySQL 仓储
func NewMysqlJobHistoryRepository(db *gorm.DB) domain.JobHistoryRepository {
	return &mysqlJobHistoryRepository{db: db}
}

func (r *mysqlJobHistoryRepository) Create(ctx context.Context, h *domain.JobHistory) error {
	return r.db.WithContext(ctx).Create(h).Error
}

func (r *mysqlJobHistoryRepository) ListByJobID(ctx context.Context, jobID uint64) ([]domain.JobHistory, error) {
	var history []domain.JobHistory
	err := r.db.WithContext(ctx).Where("job_id = ?", jobID).Order("id DESC").Find(&history).Error
	return history, err
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/modules/admin/infrastructure/ -run TestMysqlJobHistoryRepositoryCreateAndList`
Expected: PASS

- [ ] **Step 5: 写失败测试 `dead_letter_repository_test.go`**

```go
package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/modules/admin/domain"

	"github.com/stretchr/testify/assert"
)

func TestMysqlDeadLetterRepositoryCRUD(t *testing.T) {
	db := newHistoryTestDB(t)
	repo := NewMysqlDeadLetterRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &domain.DeadLetter{JobID: 1, Type: "outbox:user.created", Payload: "{}", FailReason: "boom"})
	assert.NoError(t, err)
	err = repo.Create(ctx, &domain.DeadLetter{JobID: 2, Type: "outbox:user.updated", Payload: "{}", FailReason: "boom", Resolved: true})
	assert.NoError(t, err)

	unresolved, total, err := repo.List(ctx, 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, unresolved, 1)
	assert.Equal(t, "outbox:user.created", unresolved[0].Type)

	resolved, total, err := repo.List(ctx, 0, 10, true)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, resolved, 1)

	assert.NoError(t, repo.MarkResolved(ctx, 1))
	after, total, err := repo.List(ctx, 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, after, 0)
}
```

- [ ] **Step 6: 运行测试确认编译失败**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/modules/admin/infrastructure/ -run TestMysqlDeadLetterRepositoryCRUD`
Expected: 编译错误 `undefined: NewMysqlDeadLetterRepository`

- [ ] **Step 7: 写最小实现 `dead_letter_repository.go`**

```go
package infrastructure

import (
	"context"
	"time"

	"jimu/internal/modules/admin/domain"

	"gorm.io/gorm"
)

type mysqlDeadLetterRepository struct {
	db *gorm.DB
}

// NewMysqlDeadLetterRepository 创建死信 MySQL 仓储
func NewMysqlDeadLetterRepository(db *gorm.DB) domain.DeadLetterRepository {
	return &mysqlDeadLetterRepository{db: db}
}

func (r *mysqlDeadLetterRepository) Create(ctx context.Context, d *domain.DeadLetter) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *mysqlDeadLetterRepository) List(ctx context.Context, offset, limit int, resolved bool) ([]domain.DeadLetter, int64, error) {
	var letters []domain.DeadLetter
	var total int64
	query := r.db.WithContext(ctx).Model(&domain.DeadLetter{}).Where("resolved = ?", resolved)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&letters).Error
	return letters, total, err
}

func (r *mysqlDeadLetterRepository) MarkResolved(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&domain.DeadLetter{}).Where("id = ?", id).
		Updates(map[string]interface{}{"resolved": true, "resolved_at": time.Now()}).Error
}
```

- [ ] **Step 8: 运行测试确认通过**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/modules/admin/infrastructure/ -run TestMysqlDeadLetterRepositoryCRUD`
Expected: PASS

- [ ] **Step 9: 全包回归 + Commit**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go build ./... && go test ./internal/modules/admin/infrastructure/...`
Expected: build 通过，测试 PASS

```bash
cd /Users/king/Documents/Projects/Mine/jimu
git add internal/modules/admin/infrastructure/job_history_repository.go internal/modules/admin/infrastructure/job_history_repository_test.go internal/modules/admin/infrastructure/dead_letter_repository.go internal/modules/admin/infrastructure/dead_letter_repository_test.go
git commit -m "feat(admin): add job history and dead letter repositories"
```

---

### Task 2: RestoreFromStore 返回已恢复 id 列表

**Files:**
- Modify: `internal/platform/scheduler/scheduler.go:99-117`
- Modify: `internal/platform/scheduler/scheduler_store_test.go:48-67`

**Interfaces:**
- Consumes: 现有 `Store.List(ctx) ([]JobDef, error)`、`AddNamedFunc(id, name, spec string, cmd func()) error`。
- Produces: `RestoreFromStore(ctx context.Context, cmdFactory func(id string) func()) ([]string, error)`。Task 6 用它去重内置任务实时注册。

- [ ] **Step 1: 更新失败测试 `TestRestoreFromStore` 断言返回值**

修改 `internal/platform/scheduler/scheduler_store_test.go` 的 `TestRestoreFromStore`：

```go
func TestRestoreFromStore(t *testing.T) {
	log := newTestLogger()
	store := NewMemoryStore()
	_ = store.Save(context.Background(), JobDef{ID: "r1", Name: "Restored", Cron: "@every 10s", Enabled: true})
	_ = store.Save(context.Background(), JobDef{ID: "r2", Name: "Disabled", Cron: "@every 10s", Enabled: false})
	s := NewWithStore(log, store, nil)

	restored, err := s.RestoreFromStore(context.Background(), func(id string) func() {
		if id == "r1" {
			return func() {}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RestoreFromStore() error: %v", err)
	}
	if len(restored) != 1 || restored[0] != "r1" {
		t.Fatalf("RestoreFromStore() restored = %v, want [r1]", restored)
	}
	if got := s.EntryCount(); got != 1 {
		t.Fatalf("EntryCount() = %d, want 1 (only enabled r1)", got)
	}
}
```

- [ ] **Step 2: 运行测试确认编译失败**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/platform/scheduler/ -run TestRestoreFromStore`
Expected: 编译错误 `RestoreFromStore ... used as value`（返回类型未改）

- [ ] **Step 3: 改签名并收集恢复 id**

修改 `RestoreFromStore`，返回已恢复的 enabled 任务 id：

```go
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
			return nil, fmt.Errorf("restore job %q: %w", job.ID, err)
		}
		restored = append(restored, job.ID)
	}
	return restored, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/platform/scheduler/...`
Expected: PASS（全部 scheduler 测试，含 `TestMySQLStore`）

- [ ] **Step 5: Commit**

```bash
cd /Users/king/Documents/Projects/Mine/jimu
git add internal/platform/scheduler/scheduler.go internal/platform/scheduler/scheduler_store_test.go
git commit -m "refactor(scheduler): return restored job ids from RestoreFromStore"
```

---

### Task 3: 统一事件走 outbox

**Files:**
- Modify: `internal/modules/user/application/service.go:86-109,176-196`
- Modify: `internal/modules/user/module.go:62-83`
- Create: `internal/modules/user/application/service_test.go`

**Interfaces:**
- Consumes: `outbox.Outbox.Add(ctx, tx interface{}, events ...outbox.Event) error`；`outbox.Event{ID, AggregateID, EventType, Payload json.RawMessage, Metadata, CreatedAt}`；`contract.EventUserCreated/EventUserUpdated/EventUserDeleted`；`contract.UserCreatedEvent{UserID, Username}`、`UserUpdatedEvent{UserID, Changes}`、`UserDeletedEvent{UserID}`。
- Produces: 事件只经 outbox 投递；`Update`/`Delete` 补写 outbox。user module `RegisterEvents` 订阅全局总线 `e` 的裸业务主题。

**先决知识**：`contract.EventBus` 接口只有 `Subscribe(event, handler func(payload interface{}))` 与 `Publish`。service 内 `s.eventBus` 是模块私有总线（`event.New()`），`s.outbox` 由 `NewUserService(repo, cache, eb, ob)` 注入，可能为 nil（nil 时跳过 outbox，测试可直接注 nil 观察）。`outbox.Add` 的 `tx` 参数 service 现有调用传 `nil`。domain.User 有 `ID`、`Username`。

- [ ] **Step 1: 写失败测试 `service_test.go`——Create/Update/Delete 不再直发事件，走 outbox**

`NewUserService` 的 deps 只接受 `*outbox.Outbox`（`case *outbox.Outbox`）。用 recording Store 构造真实 `*outbox.Outbox` 注入。在 `internal/modules/user/application/service_test.go` 追加：

```go
package application

import (
	"context"
	stderrors "errors"
	"testing"

	"jimu/internal/contract"
	"jimu/internal/modules/user/domain"
	"jimu/internal/platform/outbox"

	"github.com/stretchr/testify/assert"
)

// recordingOutboxStore 记录 Add 的事件，其余方法无操作
type recordingOutboxStore struct {
	events []outbox.Event
}

func (o *recordingOutboxStore) Add(_ context.Context, _ interface{}, events ...outbox.Event) error {
	o.events = append(o.events, events...)
	return nil
}
func (o *recordingOutboxStore) FetchUnpublish(context.Context, int) ([]outbox.Event, error) { return nil, nil }
func (o *recordingOutboxStore) MarkPublished(context.Context, []uint64) error               { return nil }
func (o *recordingOutboxStore) MarkFailed(context.Context, uint64, error) error             { return nil }

// createOutboxUserService 构造带 recording outbox 的 UserService
func createOutboxUserService() (*UserService, *recordingOutboxStore) {
	store := &recordingOutboxStore{}
	ob := outbox.New(store, nil)
	svc := NewUserService(&fakeOutboxUserRepo{}, nil, ob)
	return svc, store
}

type fakeOutboxUserRepo struct{}

func (r *fakeOutboxUserRepo) FindByID(context.Context, uint64) (*domain.User, error) {
	return &domain.User{ID: 1, Username: "alice"}, nil
}
func (r *fakeOutboxUserRepo) FindByUsername(context.Context, string) (*domain.User, error) {
	return nil, stderrors.New("not found")
}
func (r *fakeOutboxUserRepo) List(context.Context, int, int, string, string) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (r *fakeOutboxUserRepo) Create(_ context.Context, u *domain.User) error {
	u.ID = 1
	return nil
}
func (r *fakeOutboxUserRepo) Update(context.Context, *domain.User) error { return nil }
func (r *fakeOutboxUserRepo) Delete(context.Context, uint64) error       { return nil }

func TestCreateWritesOutbox(t *testing.T) {
	svc, store := createOutboxUserService()
	_, err := svc.Create(context.Background(), CreateUserRequest{Username: "alice", Password: "password123"})
	assert.NoError(t, err)
	assert.Len(t, store.events, 1)
	assert.Equal(t, contract.EventUserCreated, store.events[0].EventType)
	assert.Equal(t, "user:1", store.events[0].AggregateID)
}

func TestUpdateAndDeleteWriteOutbox(t *testing.T) {
	svc, store := createOutboxUserService()

	status := int8(0)
	assert.NoError(t, svc.Update(context.Background(), 1, UpdateUserRequest{Status: &status}))
	assert.Len(t, store.events, 1)
	assert.Equal(t, contract.EventUserUpdated, store.events[0].EventType)

	assert.NoError(t, svc.Delete(context.Background(), 1))
	assert.Len(t, store.events, 2)
	assert.Equal(t, contract.EventUserDeleted, store.events[1].EventType)
}
```

**注意**：现有 `service_test.go` 已定义 `fakeUserRepository`，此测试新增 `fakeOutboxUserRepo` 避免命名冲突。`UserService.List` 测试在 `NewUserService(repo, nil)` 下 `s.outbox` 为 nil 不影响现有断言。

- [ ] **Step 2: 运行测试确认失败（事件未走 outbox）**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/modules/user/application/ -run 'TestCreateWritesOutbox|TestUpdateAndDeleteWriteOutbox'`
Expected: FAIL——`store.events` 为 0（事件仍走 `PublishAsync`）

- [ ] **Step 3: 改 service.go 删除 PublishAsync，Update/Delete 补写 outbox**

`Create` 中删除 `PublishAsync` 块，保留 outbox.Add：

```go
	// 写入 Outbox（统一事件投递路径，确保可靠投递）
	if s.outbox != nil {
		payload, _ := json.Marshal(contract.UserCreatedEvent{
			UserID:   user.ID,
			Username: user.Username,
		})
		_ = s.outbox.Add(ctx, nil, outbox.Event{
			AggregateID: fmt.Sprintf("user:%d", user.ID),
			EventType:   contract.EventUserCreated,
			Payload:     payload,
		})
	}
```

`Update` 中把 `PublishAsync` 块替换为 outbox.Add（保留 `invalidateUserCache`）：

```go
	if s.outbox != nil {
		payload, _ := json.Marshal(contract.UserUpdatedEvent{
			UserID:  id,
			Changes: []string{"status"},
		})
		_ = s.outbox.Add(ctx, nil, outbox.Event{
			AggregateID: fmt.Sprintf("user:%d", id),
			EventType:   contract.EventUserUpdated,
			Payload:     payload,
		})
	}
```

`Delete` 中把 `PublishAsync` 块替换为 outbox.Add：

```go
	if s.outbox != nil {
		payload, _ := json.Marshal(contract.UserDeletedEvent{
			UserID: id,
		})
		_ = s.outbox.Add(ctx, nil, outbox.Event{
			AggregateID: fmt.Sprintf("user:%d", id),
			EventType:   contract.EventUserDeleted,
			Payload:     payload,
		})
	}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/modules/user/application/...`
Expected: PASS

- [ ] **Step 5: 改 user module RegisterEvents 订阅全局总线**

`internal/modules/user/module.go` 的 `RegisterEvents` 改为直接订阅全局总线 `e` 的裸业务主题：

```go
// RegisterEvents 注册用户事件处理器（订阅全局事件总线的裸业务主题）
func (m *Module) RegisterEvents(e contract.EventBus) {
	// 订阅全局总线的用户创建事件，桥接到通知系统
	e.Subscribe(contract.EventUserCreated, func(payload interface{}) {
		if evt, ok := payload.(contract.UserCreatedEvent); ok {
			e.Publish(contract.UserCreatedEmailNotification, notification.Message{
				Channel: notification.ChannelEmail,
				To:      "", // 实际应从用户服务获取邮箱
				Subject: "Welcome to Jimu",
				Body:    fmt.Sprintf("Hi %s, your account has been created successfully.", evt.Username),
				Data: map[string]string{
					"username": evt.Username,
				},
			})
		}
	})

	e.Subscribe(contract.EventUserDeleted, func(payload interface{}) {
		if evt, ok := payload.(contract.UserDeletedEvent); ok {
			e.Publish(contract.UserDeletedEventLog, evt)
		}
	})
}
```

如果 `m.eventBus` 字段因此不再被使用，检查 module.go 是否还有其他引用；若只剩构造赋值，保留字段与构造（避免无关改动），但若编译报 `m.eventBus` 未使用，删除该字段及其构造赋值。

- [ ] **Step 6: 全量构建 + 相关测试 + Commit**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go build ./... && go test ./internal/modules/user/...`
Expected: build 通过，测试 PASS

```bash
cd /Users/king/Documents/Projects/Mine/jimu
git add internal/modules/user/application/service.go internal/modules/user/module.go internal/modules/user/application/service_test.go
git commit -m "refactor(user): deliver events through outbox only"
```

---

### Task 4: outbox 桥接 worker + event_bus 模式桥接器

**Files:**
- Modify: `internal/app/bootstrap.go:17-135`
- Create: `internal/app/outbox_bridge_test.go`

**Interfaces:**
- Consumes: `queue.RegisterWorker(jobType string, fn queue.WorkerFunc)`（全局注册）；`outbox.EventPayload{ID, AggregateID, EventType, Payload json.RawMessage, Metadata, CreatedAt}`；`contract.EventUserCreated/Updated/Deleted` 与强类型事件；`container.EventBus *event.EventBus`（有 `Subscribe`/`Publish`）。
- Produces: 函数 `bridgeFn(c *Container) queue.WorkerFunc`、`registerOutboxWorkers(c *Container)`、`registerEventBusBridge(c *Container)`（event_bus 模式用）。Task 5 的 WorkerPool 消费依赖 `registerOutboxWorkers` 已注册的 worker。

**先决知识**：`outbox.EventPayload.CreatedAt` 是 `interface{}` 类型，`json.Unmarshal` 后为 string。强类型转换只处理 `user.created`/`user.updated`/`user.deleted`。转换表返回 `interface{}`，订阅方做类型断言。event_bus 模式下 `EventBusPublisher` 发布 `outbox:user.created` 到全局总线——新增 `outbox:*` 订阅把 `outbox.EventPayload` 转强类型发裸主题。

- [ ] **Step 1: 写失败测试 `outbox_bridge_test.go`**

```go
package app

import (
	"context"
	"encoding/json"
	"testing"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/event"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/outbox"
	"jimu/internal/platform/queue"

	"github.com/stretchr/testify/assert"
)

// fakeContainer 最小 Container：EventBus + Logger（registerEventBusBridge 需 Logger）
func fakeContainer() *Container {
	return &Container{
		EventBus: event.New(),
		Logger:   newTestLogger(),
	}
}

func newTestLogger() *logger.Logger {
	return logger.New(config.LogConfig{Level: "warn", Format: "console", Output: "stdout"})
}

func TestBridgeWorkerPublishesStrongTypeToBareTopic(t *testing.T) {
	c := fakeContainer()
	c.EventBus.Subscribe(contract.EventUserCreated, func(payload interface{}) {
		if evt, ok := payload.(contract.UserCreatedEvent); ok {
			assert.Equal(t, uint64(7), evt.UserID)
			assert.Equal(t, "bob", evt.Username)
		} else {
			t.Fatalf("expected contract.UserCreatedEvent, got %T", payload)
		}
	})

	payload, _ := json.Marshal(contract.UserCreatedEvent{UserID: 7, Username: "bob"})
	evtPayload, _ := json.Marshal(outbox.EventPayload{
		ID:          1,
		AggregateID: "user:7",
		EventType:   contract.EventUserCreated,
		Payload:     payload,
	})

	err := bridgeFn(c)(context.Background(), string(evtPayload))
	assert.NoError(t, err)
}

func TestBridgeWorkerUnknownTypeErrors(t *testing.T) {
	c := fakeContainer()
	evtPayload, _ := json.Marshal(outbox.EventPayload{
		ID:        1,
		EventType: "order.created",
		Payload:   json.RawMessage(`{}`),
	})
	err := bridgeFn(c)(context.Background(), string(evtPayload))
	assert.Error(t, err)
}

func TestEventBusBridgePublishesToBareTopic(t *testing.T) {
	c := fakeContainer()
	c.EventBus.Subscribe(contract.EventUserCreated, func(payload interface{}) {
		if evt, ok := payload.(contract.UserCreatedEvent); ok {
			assert.Equal(t, "carol", evt.Username)
		} else {
			t.Fatalf("expected contract.UserCreatedEvent, got %T", payload)
		}
	})
	registerEventBusBridge(c)

	payload, _ := json.Marshal(contract.UserCreatedEvent{UserID: 9, Username: "carol"})
	c.EventBus.Publish("outbox:"+contract.EventUserCreated, outbox.EventPayload{
		ID:        1,
		EventType: contract.EventUserCreated,
		Payload:   payload,
	})
}
```

- [ ] **Step 2: 运行测试确认编译失败（`bridgeFn`/`registerEventBusBridge` 未定义）**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/app/ -run 'TestBridgeWorker|TestEventBusBridge'`
Expected: 编译错误 `undefined: bridgeFn` / `undefined: registerEventBusBridge`

- [ ] **Step 3: 在 bootstrap.go 顶部加转换表与桥接函数**

在 `Bootstrap` 函数前新增：

```go
// outboxTypeConverters 按事件类型将 outbox 内层 Payload 还原为强类型事件
var outboxTypeConverters = map[string]func(json.RawMessage) interface{}{
	contract.EventUserCreated: func(p json.RawMessage) interface{} {
		var e contract.UserCreatedEvent
		_ = json.Unmarshal(p, &e)
		return e
	},
	contract.EventUserUpdated: func(p json.RawMessage) interface{} {
		var e contract.UserUpdatedEvent
		_ = json.Unmarshal(p, &e)
		return e
	},
	contract.EventUserDeleted: func(p json.RawMessage) interface{} {
		var e contract.UserDeletedEvent
		_ = json.Unmarshal(p, &e)
		return e
	},
}

// bridgeFn 反序列化 outbox 载荷并发布强类型事件到全局业务主题（裸主题）
func bridgeFn(c *Container) queue.WorkerFunc {
	return func(ctx context.Context, payload string) error {
		var evt outbox.EventPayload
		if err := json.Unmarshal([]byte(payload), &evt); err != nil {
			return fmt.Errorf("unmarshal outbox event: %w", err)
		}
		conv, ok := outboxTypeConverters[evt.EventType]
		if !ok {
			return fmt.Errorf("no converter for outbox event type: %s", evt.EventType)
		}
		c.EventBus.Publish(evt.EventType, conv(evt.Payload))
		return nil
	}
}

// registerOutboxWorkers 注册 MQ 消费端的 outbox 桥接 worker
func registerOutboxWorkers(c *Container) {
	for eventType := range outboxTypeConverters {
		eventType := eventType
		queue.RegisterWorker("outbox:"+eventType, bridgeFn(c))
	}
}

// registerEventBusBridge 订阅全局总线 outbox:* 主题，转强类型后发布到裸业务主题（event_bus 模式）
func registerEventBusBridge(c *Container) {
	for eventType := range outboxTypeConverters {
		eventType := eventType
		c.EventBus.Subscribe("outbox:"+eventType, func(payload interface{}) {
			evt, ok := payload.(outbox.EventPayload)
			if !ok {
				c.Logger.Error("outbox bridge: unexpected payload type")
				return
			}
			conv := outboxTypeConverters[evt.EventType]
			c.EventBus.Publish(evt.EventType, conv(evt.Payload))
		})
	}
}
```

注意 `bootstrap.go` 需要新增 import：`encoding/json`、`jimu/internal/platform/outbox`、`jimu/internal/platform/queue`。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/app/ -run 'TestBridgeWorker|TestEventBusBridge'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/king/Documents/Projects/Mine/jimu
git add internal/app/bootstrap.go internal/app/outbox_bridge_test.go
git commit -m "feat(server): add outbox bridge workers and event bus bridge"
```

---

### Task 5: container 接线 WorkerPool

**Files:**
- Modify: `internal/app/container.go:30-48,130-184`

**Interfaces:**
- Consumes: `queue.New(cfg queue.Config) (queue.Queue, error)`；`queue.Consumer`（断言 `q.(queue.Consumer)`）；`queue.NewMySQLStore(jobRepo, historyRepo, deadRepo) *queue.MySQLStore`；`queue.NewWorkerPool(config queue.WorkerConfig, q Consumer, store *MySQLStore) *WorkerPool`；`admininfra.NewMysqlJobRepository/NewMysqlJobHistoryRepository/NewMysqlDeadLetterRepository`；`queue.TypeKafka/TypeRabbitMQ`；`config.OutboxPublisherMQ`。
- Produces: `Container.WorkerPool *queue.WorkerPool`（MQ 模式非 nil）。Task 6 bootstrap 检查并启动它。

**先决知识**：`queue.NewWorkerPool` 需要 `Consumer` 与 `*queue.MySQLStore`；`q` 同时是 `Queue`（发布用）与 `Consumer`（Kafka/RabbitMQ 实现）。`queue` 包已 import 为 `"jimu/internal/platform/queue"`。`admininfra` 需新增 import（现有 container.go 未 import admin infrastructure）。WorkerPool 的 `Start()`/`Stop()` 已存在。`queue.DefaultWorkerConfig` 存在。

- [ ] **Step 1: Container 加 WorkerPool 字段**

在 `Container` struct（`internal/app/container.go:30-48`）加字段：

```go
	EventBus       *event.EventBus
	Outbox         *outbox.Outbox
	DBCollector    *observability.DBCollector
	Captcha        *captcha.Service
	WorkerPool     *queue.WorkerPool
```

在 `NewContainer` 函数顶部（`log := logger.New(cfg.Log)` 之后）声明 `var pendingWorkerPool *queue.WorkerPool`，供 outbox 分支写入、Container 构造读取。

- [ ] **Step 2: 重构 outbox 分支接线 WorkerPool**

把 `container.go` 中 outbox MQ 分支（现 L134-152）改为同时构造 WorkerPool。注意 `q` 是局部变量，当前只用于 `outbox.NewMQPublisher(q)`。改后：

```go
	switch cfg.Outbox.Publisher {
	case config.OutboxPublisherMQ:
		q, err := queue.New(queue.Config{
			Type:  queue.Type(cfg.Queue.Type),
			Redis: rdb,
			Kafka: queue.KafkaConfig{
				Brokers: cfg.Queue.Kafka.Brokers,
				Topic:   cfg.Queue.Kafka.Topic,
				GroupID: cfg.Queue.Kafka.GroupID,
			},
			RabbitMQ: queue.RabbitMQConfig{
				URL:       cfg.Queue.RabbitMQ.URL,
				QueueName: cfg.Queue.RabbitMQ.Queue,
				Exchange:  cfg.Queue.RabbitMQ.Exchange,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("init outbox queue: %w", err)
		}
		outboxPublisher = outbox.NewMQPublisher(q)
		if cfg.Queue.Type == string(queue.TypeKafka) || cfg.Queue.Type == string(queue.TypeRabbitMQ) {
			consumer, ok := q.(queue.Consumer)
			if !ok {
				return nil, fmt.Errorf("queue %s does not implement consumer", cfg.Queue.Type)
			}
			store := queue.NewMySQLStore(
				admininfra.NewMysqlJobRepository(dbConn),
				admininfra.NewMysqlJobHistoryRepository(dbConn),
				admininfra.NewMysqlDeadLetterRepository(dbConn),
			)
			workerPool := queue.NewWorkerPool(queue.DefaultWorkerConfig, consumer, store)
			// 延迟到 Container 构造后赋值（见 Step 3）
			pendingWorkerPool = workerPool
		}
	default:
		outboxPublisher = outbox.NewEventBusPublisher(eventBus)
	}
```

在函数顶部声明 `var pendingWorkerPool *queue.WorkerPool`，构造 `&Container{...}` 时加字段 `WorkerPool: pendingWorkerPool,`。新增 import `admininfra "jimu/internal/modules/admin/infrastructure"`。

- [ ] **Step 3: 编译验证**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go build ./...`
Expected: build 通过

- [ ] **Step 4: 用现有测试回归 + Commit**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go test ./internal/app/... ./internal/platform/queue/...`
Expected: PASS（现有测试应不受影响；若 container 测试不存在则仅 build 验证）

```bash
cd /Users/king/Documents/Projects/Mine/jimu
git add internal/app/container.go
git commit -m "feat(server): wire outbox worker pool in container"
```

---

### Task 6: bootstrap 接线——启动 WorkerPool + RestoreFromStore 去重

**Files:**
- Modify: `internal/app/bootstrap.go:17-135`
- Modify: `internal/app/outbox_bridge_test.go`

**Interfaces:**
- Consumes: `container.WorkerPool`（Task 5）、`registerOutboxWorkers(c)`/`registerEventBusBridge(c)`（Task 4）、`container.Scheduler.RestoreFromStore(ctx, cmdFactory) ([]string, error)`（Task 2）、`Scheduler.AddNamedFunc(id, name, spec, fn)`。
- Produces: 装配完整。内置任务 `outbox_process`/`metrics_collect`/`cleanup` 恢复去重注册；WorkerPool 随 Application 生命周期启动/停止。

**先决知识**：`bootstrap.go` 现有 3 个静态任务 `AddNamedFunc` 调用（outbox_process/metrics_collect/cleanup）用闭包内嵌。改为 `jobDef{name, spec, fn}` 结构 + `jobFns` map，RestoreFromStore 先恢复，剩余实时注册。WorkerPool 非 `contract.Component`，用匿名 component 包装 `Start`/`Stop`。`cfg.Outbox.Publisher == config.OutboxPublisherMQ` 时启动 WorkerPool 并注册 workers；event_bus 模式注册全局总线桥接器。

- [ ] **Step 1: 重构任务注册为 jobFns 结构**

在 `Bootstrap` 内、`container.JobRegistry != nil` 块前，定义内置任务 map。替换现有 3 个 `AddNamedFunc` 块。最终代码形态：

```go
	// 注册各模块的定时任务
	if container.JobRegistry != nil {
		for _, module := range modules {
			module.RegisterJobs(container.JobRegistry)
			container.Logger.Info("module jobs registered", "name", module.Name())
		}

		type jobDef struct {
			name string
			spec string
			fn   func()
		}
		jobFns := map[string]jobDef{}
		if container.Outbox != nil {
			jobFns["outbox_process"] = jobDef{name: "Process Outbox Events", spec: "@every 10s", fn: func() {
				n, err := container.Outbox.Process(context.Background(), 100)
				if err != nil {
					container.Logger.Error("outbox process error", "error", err.Error())
				} else if n > 0 {
					container.Logger.Debug("outbox processed", "count", n)
				}
			}}
		}
		if container.DBCollector != nil {
			jobFns["metrics_collect"] = jobDef{name: "Collect DB Metrics", spec: "@every 15s", fn: func() {
				container.DBCollector.Collect()
				observability.CollectRuntime()
			}}
		}
		cleanupSvc := db.NewCleanupService(container.DB, db.DefaultCleanupConfig())
		jobFns["cleanup"] = jobDef{name: "Data Cleanup", spec: "0 3 * * *", fn: func() {
			results, err := cleanupSvc.Run(context.Background())
			if err != nil {
				container.Logger.Error("cleanup job failed", "error", err.Error())
				return
			}
			for _, r := range results {
				if r.Deleted > 0 {
					container.Logger.Info("cleanup completed", "table", r.Table, "deleted", r.Deleted)
				}
			}
		}}

		// 从 store 恢复持久化任务，跳过已恢复 id，防双注册
		restored, err := container.Scheduler.RestoreFromStore(context.Background(), func(id string) func() {
			if def, ok := jobFns[id]; ok {
				return def.fn
			}
			return nil
		})
		if err != nil {
			container.Logger.Error("restore scheduled jobs failed", "error", err.Error())
		}
		restoredSet := make(map[string]struct{}, len(restored))
		for _, id := range restored {
			restoredSet[id] = struct{}{}
		}
		for id, def := range jobFns {
			if _, ok := restoredSet[id]; ok {
				continue
			}
			if err := container.Scheduler.AddNamedFunc(id, def.name, def.spec, def.fn); err != nil {
				container.Logger.Error("register job failed", "id", id, "error", err.Error())
			}
		}
	}
```

**注意**：`db.NewCleanupService` 与 `db.DefaultCleanupConfig()` 现已在文件使用；`cleanupSvc` 需在 `container.DB != nil` 时才构造（现有代码假设 DB 非 nil，保持原行为）。原 WebSocket Hub 启动 `go container.WebSocketHub.Run(...)` 保留原位置。

- [ ] **Step 2: 接线 WorkerPool 启动 + outbox 桥接注册**

在 `Bootstrap` 中、模块事件注册后、`components := []contract.Component{container}` 前，插入：

```go
	// 接线 outbox 事件消费：MQ 模式注册 worker 并启动 WorkerPool；event_bus 模式注册全局总线桥接器
	switch cfg.Outbox.Publisher {
	case config.OutboxPublisherMQ:
		registerOutboxWorkers(container)
	case config.OutboxPublisherEventBus:
		registerEventBusBridge(container)
	}
```

在 `components` 构造后，追加 WorkerPool 匿名 component：

```go
	components := []contract.Component{container}
	if container.WorkerPool != nil {
		components = append(components, workerPoolComponent{pool: container.WorkerPool})
	}
	if container.Scheduler != nil {
		components = append(components, container.Scheduler)
	}
```

在文件底部新增：

```go
// workerPoolComponent 包装 WorkerPool，实现 contract.Component 以纳入应用生命周期
type workerPoolComponent struct {
	pool *queue.WorkerPool
}

func (w workerPoolComponent) Start(context.Context) error {
	go w.pool.Start()
	return nil
}

func (w workerPoolComponent) Stop(context.Context) error {
	w.pool.Stop()
	return nil
}
```

新增 import：`"jimu/internal/platform/queue"`（若 Step 4 已加则复用）、`"jimu/internal/config"`（switch 用 `config.OutboxPublisherMQ`/`config.OutboxPublisherEventBus`）。

- [ ] **Step 3: 写/更新装配测试**

在 `outbox_bridge_test.go` 追加测试验证 RestoreFromStore 去重逻辑与 worker 注册，或更新现有测试编译。若 `Bootstrap` 函数需要完整 config/db，测试用 mock Container 直接验证 `restoredSet` 逻辑片段不可行——**改测 Task 2 已有覆盖**，此处补一个对 `registerOutboxWorkers` 的断言：

```go
func TestRegisterOutboxWorkersRegistersAll(t *testing.T) {
	// 清空全局 worker map（queue 包无导出清理，测试只验证三个类型可注册后 GetWorker 命中）
	registerOutboxWorkers(fakeContainer())
	for _, et := range []string{"outbox:user.created", "outbox:user.updated", "outbox:user.deleted"} {
		fn, ok := queue.GetWorker(et)
		assert.True(t, ok, "worker %s not registered", et)
		assert.NotNil(t, fn)
	}
}
```

若 `queue.GetWorker` 全局 map 在多次测试间累积，确认该测试在测试顺序中幂等（只检查存在性）。**若 test binary 内无法重置全局 worker map，此测试可与 Task 4 测试合并**，或在测试注释说明全局注册副作用。

- [ ] **Step 4: 编译 + 全量测试**

Run: `cd /Users/king/Documents/Projects/Mine/jimu && go build ./... && go test ./internal/app/... ./internal/platform/scheduler/...`
Expected: build 通过，测试 PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/king/Documents/Projects/Mine/jimu
git add internal/app/bootstrap.go internal/app/outbox_bridge_test.go
git commit -m "feat(server): start worker pool and restore scheduled jobs on boot"
```

---

## Self-Review

**1. Spec coverage:**
- Task 1 覆盖 spec "新增文件"（两个仓储）。
- Task 2 覆盖 spec Task 2 "RestoreFromStore 签名"。
- Task 3 覆盖 spec 统一走 outbox + user module 订阅全局（新决策）。
- Task 4 覆盖 spec "事件桥接细节" + event_bus 模式桥接器。
- Task 5 覆盖 spec "container.go WorkerPool 接线"。
- Task 6 覆盖 spec "bootstrap.go 启动顺序/去重" + WorkerPool 生命周期。
- 全部任务覆盖 README 更新要求。

**2. Placeholder scan:** 无 TBD/TODO；每步含具体代码。Task 3 测试依赖现有 DTO 字段，已给出调整说明兜底。

**3. Type consistency:**
- `RestoreFromStore` 返回 `([]string, error)` 在 Task 2 定义、Task 6 消费，一致。
- `outboxTypeConverters` / `bridgeFn` / `registerOutboxWorkers` / `registerEventBusBridge` 在 Task 4 定义、Task 6 调用，签名一致。
- `Container.WorkerPool *queue.WorkerPool` 在 Task 5 定义、Task 6 消费，一致。
- 桥接发布用 `evt.EventType`（裸主题），与 Task 3 的 user module 订阅 `contract.EventUserCreated` 匹配。
- `outbox.EventPayload.Payload` 是 `json.RawMessage`，转换表入参一致。
