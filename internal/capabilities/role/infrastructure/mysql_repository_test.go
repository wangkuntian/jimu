package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/capabilities/role/domain"
	dbutil "jimu/internal/platform/db"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Role{}, &rolePermission{}))
	return db
}

func TestRoleRepositoryUpdateUsesOptimisticLock(t *testing.T) {
	db := newRoleTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	role := &domain.Role{TenantID: 1, Name: "viewer", Status: 1}
	require.NoError(t, repo.Create(ctx, role))

	// 两个请求读到同一版本
	first, err := repo.FindByID(ctx, role.ID)
	require.NoError(t, err)
	second, err := repo.FindByID(ctx, role.ID)
	require.NoError(t, err)

	first.Name = "viewer-a"
	require.NoError(t, repo.Update(ctx, first))
	assert.Equal(t, int64(1), first.Version, "更新后版本号应自增")

	second.Name = "viewer-b"
	assert.ErrorIs(t, repo.Update(ctx, second), dbutil.ErrConcurrentUpdate, "陈旧版本应报冲突")

	got, err := repo.FindByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, "viewer-a", got.Name, "后到的写入不应覆盖先到的")
	assert.Equal(t, int64(1), got.Version)
}

func TestRoleRepositoryAssignPermissionsReplacesAll(t *testing.T) {
	db := newRoleTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	role := &domain.Role{TenantID: 1, Name: "viewer", Status: 1}
	require.NoError(t, repo.Create(ctx, role))

	require.NoError(t, repo.AssignPermissions(ctx, role.ID, []uint64{1, 2}))
	var count int64
	require.NoError(t, db.Model(&rolePermission{}).Where("role_id = ?", role.ID).Count(&count).Error)
	assert.Equal(t, int64(2), count)

	// 再次分配为整体替换
	require.NoError(t, repo.AssignPermissions(ctx, role.ID, []uint64{3}))
	require.NoError(t, db.Model(&rolePermission{}).Where("role_id = ?", role.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
