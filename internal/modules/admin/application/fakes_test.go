package application

import (
	"context"
	"sync"
	"testing"

	admindomain "jimu/internal/modules/admin/domain"
	userdomain "jimu/internal/modules/user/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

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

func (f *fakeEventBus) published() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.events...)
}

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

func (f *fakeUserRepository) FindByEmailHash(context.Context, string) (*userdomain.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeUserRepository) FindByPhoneHash(context.Context, string) (*userdomain.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeUserRepository) UpdatePassword(context.Context, uint64, string) error { return nil }

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
