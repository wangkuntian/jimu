package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/capabilities/user/domain"
	"jimu/internal/shared/testutil"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}))
	return db
}

func newTestUser() *domain.User {
	return &domain.User{
		Username: testutil.RandomString(10),
		Password: testutil.RandomString(16),
		Status:   1,
	}
}

func TestMysqlRepositoryCreateAndFindByID(t *testing.T) {
	db := newUserTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	u := newTestUser()
	require.NoError(t, repo.Create(ctx, u))
	assert.NotZero(t, u.ID)

	got, err := repo.FindByID(ctx, u.ID)
	assert.NoError(t, err)
	assert.Equal(t, u.Username, got.Username)
	assert.Equal(t, u.Password, got.Password)
}

func TestMysqlRepositoryFindByUsername(t *testing.T) {
	db := newUserTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	u := newTestUser()
	require.NoError(t, repo.Create(ctx, u))

	got, err := repo.FindByUsername(ctx, u.Username)
	assert.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	_, err = repo.FindByUsername(ctx, "nonexistent-"+testutil.RandomString(6))
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestMysqlRepositoryListCountAndPagination(t *testing.T) {
	db := newUserTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, newTestUser()))
	}

	users, total, err := repo.List(ctx, domain.UserListFilter{}, 0, 2, "id", "desc")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, users, 2)
	// desc 排序：最新创建的 ID 最大，应排第一
	assert.True(t, users[0].ID > users[1].ID)
}

func TestMysqlRepositoryListFilters(t *testing.T) {
	db := newUserTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()
	statusDisabled := int8(0)

	require.NoError(t, repo.Create(ctx, &domain.User{Username: "alice_admin", Password: "x", Status: 1, TenantID: 1}))
	disabled := &domain.User{Username: "alice_disabled", Password: "x", Status: 1, TenantID: 1}
	require.NoError(t, repo.Create(ctx, disabled))
	require.NoError(t, db.Model(disabled).Update("status", 0).Error)
	require.NoError(t, repo.Create(ctx, &domain.User{Username: "bob", Password: "x", Status: 1, TenantID: 2}))

	users, total, err := repo.List(ctx, domain.UserListFilter{
		TenantID: 1,
		Username: "alice",
		Status:   &statusDisabled,
	}, 0, 20, "id", "desc")
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, users, 1)
	assert.Equal(t, "alice_disabled", users[0].Username)
}

func TestMysqlRepositoryFindByEmailHash(t *testing.T) {
	db := newUserTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	u := newTestUser()
	hash := testutil.RandomString(32)
	u.EmailHash = &hash
	require.NoError(t, repo.Create(ctx, u))

	got, err := repo.FindByEmailHash(ctx, hash)
	assert.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	_, err = repo.FindByEmailHash(ctx, "missing-hash")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestMysqlRepositoryFindByPhoneHash(t *testing.T) {
	db := newUserTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	u := newTestUser()
	hash := testutil.RandomString(32)
	u.PhoneHash = &hash
	require.NoError(t, repo.Create(ctx, u))

	got, err := repo.FindByPhoneHash(ctx, hash)
	assert.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	_, err = repo.FindByPhoneHash(ctx, "missing-hash")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestMysqlRepositoryUpdatePassword(t *testing.T) {
	db := newUserTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	u := newTestUser()
	require.NoError(t, repo.Create(ctx, u))

	require.NoError(t, repo.UpdatePassword(ctx, u.ID, "new-hash"))
	got, err := repo.FindByID(ctx, u.ID)
	assert.NoError(t, err)
	assert.Equal(t, "new-hash", got.Password)
}

func TestMysqlRepositoryUpdateAndDelete(t *testing.T) {
	db := newUserTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	u := newTestUser()
	require.NoError(t, repo.Create(ctx, u))

	newStatus := int8(0)
	u.Status = newStatus
	require.NoError(t, repo.Update(ctx, u))

	got, err := repo.FindByID(ctx, u.ID)
	assert.NoError(t, err)
	assert.Equal(t, newStatus, got.Status)

	require.NoError(t, repo.Delete(ctx, u.ID))
	_, err = repo.FindByID(ctx, u.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "软删除后查询应返回未找到")
}
