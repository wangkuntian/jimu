package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/capabilities/access/domain"
	"jimu/internal/contract"
	dbutil "jimu/internal/kernel/db"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Role{}, &domain.Permission{}, &rolePermission{}))
	permissions := []domain.Permission{
		{ID: 1, Name: "perm1", Resource: "/test", Action: "GET"},
		{ID: 2, Name: "perm2", Resource: "/test", Action: "POST"},
		{ID: 3, Name: "perm3", Resource: "/test", Action: "PUT"},
	}
	for i := range permissions {
		require.NoError(t, db.Create(&permissions[i]).Error)
	}
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

func TestRoleRepositoryAssignPermissionsRejectsUnknownAtomically(t *testing.T) {
	db := newRoleTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	role := &domain.Role{TenantID: 1, Name: "viewer", Status: 1}
	require.NoError(t, repo.Create(ctx, role))
	require.NoError(t, repo.AssignPermissions(ctx, role.ID, []uint64{1}))

	assert.ErrorIs(t, repo.AssignPermissions(ctx, role.ID, []uint64{2, 999}), domain.ErrPermissionNotFound)

	var ids []uint64
	require.NoError(t, db.Model(&rolePermission{}).Where("role_id = ?", role.ID).Pluck("permission_id", &ids).Error)
	assert.Equal(t, []uint64{1}, ids, "校验失败时不得删除原有权限")
}

func TestRoleRepositoryListFilters(t *testing.T) {
	db := newRoleTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()

	roles := []domain.Role{
		{TenantID: 1, Name: "ops-admin", Status: 1},
		{TenantID: 1, Name: "ops-disabled", Status: 1},
		{TenantID: 2, Name: "ops-other", Status: 1},
		{TenantID: 2, Name: "viewer", Status: 1},
	}
	for i := range roles {
		require.NoError(t, repo.Create(ctx, &roles[i]))
	}
	require.NoError(t, db.Model(&domain.Role{}).Where("id = ?", roles[1].ID).Update("status", 0).Error)

	disabled := int8(0)
	got, total, err := repo.List(ctx, domain.RoleListFilter{
		TenantID: 1,
		Keyword:  "ops",
		Status:   &disabled,
	}, 0, 20, "id", "asc")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, got, 1)
	assert.Equal(t, "ops-disabled", got[0].Name)
}

func TestRoleRepositoryListOptions(t *testing.T) {
	db := newRoleTestDB(t)
	repo := NewMysqlRepository(db)
	ctx := context.Background()
	roles := []domain.Role{
		{TenantID: 1, Name: "custom-a", Scope: contract.RoleScopeTenant, Status: 1},
		{TenantID: 1, Name: "system", Code: contract.RoleCodeTenantAdmin, Scope: contract.RoleScopeTenant, IsSystem: true, Status: 1},
		{TenantID: 1, Name: "disabled", Scope: contract.RoleScopeTenant, Status: 1},
		{TenantID: 1, Name: "platform", Scope: contract.RoleScopePlatform, Status: 1},
		{TenantID: 2, Name: "other-tenant", Scope: contract.RoleScopeTenant, Status: 1},
	}
	for i := range roles {
		require.NoError(t, repo.Create(ctx, &roles[i]))
	}
	require.NoError(t, db.Model(&domain.Role{}).Where("id = ?", roles[2].ID).Update("status", 0).Error)

	got, err := repo.ListOptions(ctx, 1)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "system", got[0].Name, "system roles sort first")
	assert.Equal(t, "custom-a", got[1].Name)
}
