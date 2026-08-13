package interfaces

import (
	"context"
	"sync"
	"testing"

	admindomain "jimu/internal/modules/admin/domain"
	userdomain "jimu/internal/modules/user/domain"
	qdomain "jimu/internal/platform/queue/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// newSqliteDB 创建内存 sqlite 并迁移给定模型
func newSqliteDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	if len(models) > 0 {
		assert.NoError(t, db.AutoMigrate(models...))
	}
	return db
}

// 测试用角色模型（roles 表）
type testRole struct {
	ID   uint64 `gorm:"primaryKey"`
	Name string
}

func (testRole) TableName() string { return "roles" }

// 测试用用户角色关联（user_roles 表）
type testUserRole struct {
	UserID uint64 `gorm:"primaryKey"`
	RoleID uint64 `gorm:"primaryKey"`
}

func (testUserRole) TableName() string { return "user_roles" }

// fakeUserRepository 可配置的用户仓储 mock
type fakeUserRepository struct {
	findByID func(ctx context.Context, id uint64) (*userdomain.User, error)
	list     func(ctx context.Context, offset, limit int, sort, order string) ([]userdomain.User, int64, error)
	create   func(ctx context.Context, user *userdomain.User) error
	update   func(ctx context.Context, user *userdomain.User) error
}

func (f *fakeUserRepository) FindByID(ctx context.Context, id uint64) (*userdomain.User, error) {
	if f.findByID != nil {
		return f.findByID(ctx, id)
	}
	return &userdomain.User{ID: id, Username: "alice", Status: 1}, nil
}

func (f *fakeUserRepository) FindByUsername(ctx context.Context, username string) (*userdomain.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeUserRepository) List(ctx context.Context, offset, limit int, sort, order string) ([]userdomain.User, int64, error) {
	if f.list != nil {
		return f.list(ctx, offset, limit, sort, order)
	}
	return []userdomain.User{{ID: 1, Username: "alice", Status: 1}}, 1, nil
}

func (f *fakeUserRepository) Create(ctx context.Context, user *userdomain.User) error {
	if f.create != nil {
		return f.create(ctx, user)
	}
	user.ID = 7
	return nil
}

func (f *fakeUserRepository) Update(ctx context.Context, user *userdomain.User) error {
	if f.update != nil {
		return f.update(ctx, user)
	}
	return nil
}

func (f *fakeUserRepository) Delete(ctx context.Context, id uint64) error { return nil }

// fakeAPIKeyRepo 可配置的 API Key 仓储 mock
type fakeAPIKeyRepo struct {
	create   func(ctx context.Context, key *admindomain.APIKey) error
	findByID func(ctx context.Context, id uint64) (*admindomain.APIKey, error)
	list     func(ctx context.Context, offset, limit int) ([]admindomain.APIKey, int64, error)
	delete   func(ctx context.Context, id uint64) error
}

func (f *fakeAPIKeyRepo) Create(ctx context.Context, key *admindomain.APIKey) error {
	if f.create != nil {
		return f.create(ctx, key)
	}
	key.ID = 9
	return nil
}

func (f *fakeAPIKeyRepo) FindByID(ctx context.Context, id uint64) (*admindomain.APIKey, error) {
	if f.findByID != nil {
		return f.findByID(ctx, id)
	}
	return &admindomain.APIKey{ID: id, Name: "web"}, nil
}

func (f *fakeAPIKeyRepo) FindByKeyHash(ctx context.Context, hash string) (*admindomain.APIKey, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeAPIKeyRepo) List(ctx context.Context, offset, limit int) ([]admindomain.APIKey, int64, error) {
	if f.list != nil {
		return f.list(ctx, offset, limit)
	}
	return []admindomain.APIKey{{ID: 1, Name: "web"}}, 1, nil
}

func (f *fakeAPIKeyRepo) Update(ctx context.Context, key *admindomain.APIKey) error { return nil }
func (f *fakeAPIKeyRepo) Delete(ctx context.Context, id uint64) error {
	if f.delete != nil {
		return f.delete(ctx, id)
	}
	return nil
}

func (f *fakeAPIKeyRepo) IncrementUseCount(ctx context.Context, id uint64) error { return nil }

// fakeImportJobRepo 可配置的导入任务仓储 mock
type fakeImportJobRepo struct {
	create   func(ctx context.Context, job *admindomain.ImportJob) error
	findByID func(ctx context.Context, id uint64) (*admindomain.ImportJob, error)
	update   func(ctx context.Context, job *admindomain.ImportJob) error
}

func (f *fakeImportJobRepo) Create(ctx context.Context, job *admindomain.ImportJob) error {
	if f.create != nil {
		return f.create(ctx, job)
	}
	job.ID = 42
	return nil
}

func (f *fakeImportJobRepo) FindByID(ctx context.Context, id uint64) (*admindomain.ImportJob, error) {
	if f.findByID != nil {
		return f.findByID(ctx, id)
	}
	return &admindomain.ImportJob{ID: id, Type: "users", Status: admindomain.ImportJobCompleted}, nil
}

func (f *fakeImportJobRepo) Update(ctx context.Context, job *admindomain.ImportJob) error {
	if f.update != nil {
		return f.update(ctx, job)
	}
	return nil
}

// fakeJobRepo 可配置的任务仓储 mock
type fakeJobRepo struct {
	create   func(ctx context.Context, job *qdomain.Job) error
	findByID func(ctx context.Context, id uint64) (*qdomain.Job, error)
	update   func(ctx context.Context, job *qdomain.Job) error
	list     func(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]qdomain.Job, int64, error)
}

func (f *fakeJobRepo) Create(ctx context.Context, job *qdomain.Job) error {
	if f.create != nil {
		return f.create(ctx, job)
	}
	job.ID = 5
	return nil
}

func (f *fakeJobRepo) FindByID(ctx context.Context, id uint64) (*qdomain.Job, error) {
	if f.findByID != nil {
		return f.findByID(ctx, id)
	}
	return &qdomain.Job{ID: id, Type: "email", Status: qdomain.JobStatusPending}, nil
}

func (f *fakeJobRepo) Update(ctx context.Context, job *qdomain.Job) error {
	if f.update != nil {
		return f.update(ctx, job)
	}
	return nil
}

func (f *fakeJobRepo) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]qdomain.Job, int64, error) {
	if f.list != nil {
		return f.list(ctx, offset, limit, filters)
	}
	return []qdomain.Job{{ID: 1, Type: "email"}}, 1, nil
}

// fakeDeadLetterRepo 可配置的死信仓储 mock
type fakeDeadLetterRepo struct {
	list         func(ctx context.Context, offset, limit int, resolved bool) ([]qdomain.DeadLetter, int64, error)
	markResolved func(ctx context.Context, id uint64) error
}

func (f *fakeDeadLetterRepo) Create(ctx context.Context, d *qdomain.DeadLetter) error { return nil }

func (f *fakeDeadLetterRepo) List(ctx context.Context, offset, limit int, resolved bool) ([]qdomain.DeadLetter, int64, error) {
	if f.list != nil {
		return f.list(ctx, offset, limit, resolved)
	}
	return []qdomain.DeadLetter{{ID: 1, JobID: 1}}, 1, nil
}

func (f *fakeDeadLetterRepo) MarkResolved(ctx context.Context, id uint64) error {
	if f.markResolved != nil {
		return f.markResolved(ctx, id)
	}
	return nil
}

// fakeEventBus 记录发布事件的事件总线 mock
type fakeEventBus struct {
	mu     sync.Mutex
	events []string
}

func (f *fakeEventBus) Subscribe(event string, handler func(payload interface{})) {}
func (f *fakeEventBus) Publish(event string, payload interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}
