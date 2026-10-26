package application

import (
	"context"
	stderrors "errors"
	"testing"

	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newUserRoleTestDB 建 users/roles/user_roles 三表（assigner 会读 users.tenant_id、
// 解析 roles、写 user_roles）。
func newUserRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, tenant_id INTEGER NOT NULL DEFAULT 0)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE roles (id INTEGER PRIMARY KEY, name TEXT NOT NULL, tenant_id INTEGER NOT NULL DEFAULT 0, status INTEGER NOT NULL DEFAULT 1, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_roles (user_id INTEGER NOT NULL, role_id INTEGER NOT NULL, PRIMARY KEY (user_id, role_id))`).Error)
	return db
}

func seedAssigner(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id) VALUES (7, 1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO roles (id, name, tenant_id) VALUES (1, 'admin', 1), (2, 'viewer', 1)`).Error)
}

func TestAssignRoleIDsReplacesRoles(t *testing.T) {
	db := newUserRoleTestDB(t)
	seedAssigner(t, db)
	svc := NewUserRoleService(db)
	ctx := tenant.WithTenant(context.Background(), 1)

	require.NoError(t, svc.AssignRoleIDs(ctx, 7, []uint64{1}))
	var n int64
	require.NoError(t, db.Table("user_roles").Where("user_id = ?", 7).Count(&n).Error)
	assert.Equal(t, int64(1), n)

	// 再次分配应替换（不是累加）
	require.NoError(t, svc.AssignRoleIDs(ctx, 7, []uint64{2}))
	var roleID uint64
	require.NoError(t, db.Table("user_roles").Select("role_id").Where("user_id = ?", 7).Scan(&roleID).Error)
	assert.Equal(t, uint64(2), roleID)

	// 空列表清空全部角色
	require.NoError(t, svc.AssignRoleIDs(ctx, 7, nil))
	require.NoError(t, db.Table("user_roles").Where("user_id = ?", 7).Count(&n).Error)
	assert.Zero(t, n)
}

func TestAssignRoleIDsUnknownRole(t *testing.T) {
	db := newUserRoleTestDB(t)
	seedAssigner(t, db)
	svc := NewUserRoleService(db)
	err := svc.AssignRoleIDs(context.Background(), 7, []uint64{1, 99})
	assert.Equal(t, apperrors.CodeNotFound, userRoleAppCode(err))
}

func TestAssignRoleIDsMissingUser(t *testing.T) {
	db := newUserRoleTestDB(t)
	svc := NewUserRoleService(db)
	err := svc.AssignRoleIDs(context.Background(), 999, []uint64{1})
	assert.Equal(t, apperrors.CodeNotFound, userRoleAppCode(err))
}

// TestAssignRoleIDsTenantIsolation 跨租户的目标用户按不存在处理（不得越权写角色）。
func TestAssignRoleIDsTenantIsolation(t *testing.T) {
	db := newUserRoleTestDB(t)
	seedAssigner(t, db)
	svc := NewUserRoleService(db)

	ctx := tenant.WithTenant(context.Background(), 2) // 目标是租户 1 的用户
	err := svc.AssignRoleIDs(ctx, 7, []uint64{1})
	assert.Equal(t, apperrors.CodeNotFound, userRoleAppCode(err))
	var n int64
	require.NoError(t, db.Table("user_roles").Where("user_id = ?", 7).Count(&n).Error)
	assert.Zero(t, n, "越权调用不得写入任何角色")
}

func TestAssignRoleIDsPlatformScopeCanManageOtherTenant(t *testing.T) {
	db := newUserRoleTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id) VALUES (7, 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO roles (id, name, tenant_id) VALUES (1, 'admin', 2)`).Error)
	svc := NewUserRoleService(db)
	ctx := tenant.WithPlatformScope(tenant.WithTenant(context.Background(), tenant.DefaultTenantID))

	require.NoError(t, svc.AssignRoleIDs(ctx, 7, []uint64{1}))
	var n int64
	require.NoError(t, db.Table("user_roles").Where("user_id = ?", 7).Count(&n).Error)
	assert.Equal(t, int64(1), n)
}

// TestAssignRoleIDsUsesInjectedTenantReader 注入端口时走端口读租户。
func TestAssignRoleIDsUsesInjectedTenantReader(t *testing.T) {
	db := newUserRoleTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id) VALUES (7, 9)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO roles (id, name, tenant_id) VALUES (1, 'admin', 9)`).Error)
	svc := NewUserRoleService(db).WithUsers(fakeTenantReader{tenantID: 9})

	require.NoError(t, svc.AssignRoleIDs(tenant.WithTenant(context.Background(), 9), 7, []uint64{1}))
	var n int64
	require.NoError(t, db.Table("user_roles").Where("user_id = ?", 7).Count(&n).Error)
	assert.Equal(t, int64(1), n)

	// 端口报错透传
	svc = NewUserRoleService(db).WithUsers(fakeTenantReader{err: gorm.ErrRecordNotFound})
	assert.Equal(t, apperrors.CodeNotFound, userRoleAppCode(svc.AssignRoleIDs(context.Background(), 7, nil)))
	svc = NewUserRoleService(db).WithUsers(fakeTenantReader{err: stderrors.New("boom")})
	assert.Equal(t, apperrors.CodeInternalError, userRoleAppCode(svc.AssignRoleIDs(context.Background(), 7, nil)))
}

// TestAssignRolesWithoutDB db 缺失时报内部错误。
func TestAssignRoleIDsWithoutDB(t *testing.T) {
	svc := NewUserRoleService(nil)
	assert.Equal(t, apperrors.CodeInternalError, userRoleAppCode(svc.AssignRoleIDs(context.Background(), 1, nil)))
}

func TestRolesForUsers(t *testing.T) {
	db := newUserRoleTestDB(t)
	seedAssigner(t, db)
	require.NoError(t, db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (7, 1), (7, 2)`).Error)

	got, err := NewUserRoleService(db).RolesForUsers(context.Background(), []uint64{7})
	require.NoError(t, err)
	assert.Equal(t, []contract.UserRole{
		{ID: 1, Name: "admin"},
		{ID: 2, Name: "viewer"},
	}, got[7])
}

type fakeTenantReader struct {
	tenantID uint64
	err      error
}

func (f fakeTenantReader) TenantOf(context.Context, uint64) (uint64, error) {
	return f.tenantID, f.err
}

func userRoleAppCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}
