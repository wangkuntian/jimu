package application

import (
	"context"
	stderrors "errors"
	"testing"

	roledomain "jimu/internal/capabilities/role/domain"
	tenantdomain "jimu/internal/capabilities/tenant/domain"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/config"
	"jimu/internal/platform/db"
	apperrors "jimu/internal/shared/errors"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newProvisionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open("file:provision_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := gdb.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	require.NoError(t, db.InitSnowflake(1))
	db.RegisterSnowflakeHook(gdb)
	require.NoError(t, gdb.AutoMigrate(
		&userdomain.User{},
		&roledomain.Role{},
		&roledomain.Permission{},
		&tenantdomain.Tenant{},
	))
	require.NoError(t, gdb.Exec(`CREATE TABLE IF NOT EXISTS user_roles (user_id INTEGER NOT NULL, role_id INTEGER NOT NULL)`).Error)
	require.NoError(t, gdb.Exec(`CREATE TABLE IF NOT EXISTS role_permissions (role_id INTEGER NOT NULL, permission_id INTEGER NOT NULL)`).Error)
	return gdb
}

func provisionTestConfig() config.ProvisioningConfig {
	return config.ProvisioningConfig{
		Enabled:   true,
		OwnerRole: "管理员",
		Roles: []config.ProvisionRoleTemplate{
			{
				Name:        "管理员",
				Description: "租户管理员",
				Permissions: []config.ProvisionPermission{
					{Resource: "/api/v1/users", Action: "GET"},
					{Resource: "/api/v1/users", Action: "POST"},
					{Resource: "/api/v1/roles", Action: "GET"},
					{Resource: "/api/v1/missing", Action: "GET"}, // 未 seed，应跳过
				},
			},
			{
				Name:        "成员",
				Description: "普通成员",
				Permissions: []config.ProvisionPermission{
					{Resource: "/api/v1/audits", Action: "GET"},
				},
			},
		},
	}
}

func seedProvisionPermissions(t *testing.T, gdb *gorm.DB) {
	t.Helper()
	perms := []roledomain.Permission{
		{Name: "用户列表", Resource: "/api/v1/users", Action: "GET"},
		{Name: "用户创建", Resource: "/api/v1/users", Action: "POST"},
		{Name: "角色列表", Resource: "/api/v1/roles", Action: "GET"},
		{Name: "审计列表", Resource: "/api/v1/audits", Action: "GET"},
	}
	for i := range perms {
		require.NoError(t, gdb.Create(&perms[i]).Error)
	}
}

func TestGormTenantProvisionerProvision(t *testing.T) {
	gdb := newProvisionTestDB(t)
	seedProvisionPermissions(t, gdb)
	provisioner := NewGormTenantProvisioner(gdb, provisionTestConfig())

	res, err := provisioner.Provision(context.Background(), ProvisionParams{
		Username:     "alice",
		PasswordHash: "hashed",
		Email:        "alice@example.com",
		TenantName:   "Acme Inc",
		TenantCode:   "ACME", // 大写输入应归一化为小写
	})
	require.NoError(t, err)

	// 租户已创建且编码正确（统一小写）
	assert.Equal(t, "acme", res.Tenant.Code)
	assert.Equal(t, "Acme Inc", res.Tenant.Name)
	assert.NotZero(t, res.Tenant.ID)

	// owner 用户归属新租户
	assert.Equal(t, res.Tenant.ID, res.User.TenantID)
	assert.Equal(t, "alice", res.User.Username)

	// 模板角色已创建且归属新租户
	var roles []roledomain.Role
	require.NoError(t, gdb.Where("tenant_id = ?", res.Tenant.ID).Order("id").Find(&roles).Error)
	require.Len(t, roles, 2)
	assert.Equal(t, "管理员", roles[0].Name)
	assert.Equal(t, "成员", roles[1].Name)

	// 权限绑定：管理员 3 条（缺失权限跳过）、成员 1 条
	var adminPermCount, memberPermCount int64
	require.NoError(t, gdb.Table("role_permissions").Where("role_id = ?", roles[0].ID).Count(&adminPermCount).Error)
	require.NoError(t, gdb.Table("role_permissions").Where("role_id = ?", roles[1].ID).Count(&memberPermCount).Error)
	assert.Equal(t, int64(3), adminPermCount)
	assert.Equal(t, int64(1), memberPermCount)

	// owner 绑定“管理员”角色（owner_role 指定）
	var userRoleCount int64
	require.NoError(t, gdb.Table("user_roles").Where("user_id = ? AND role_id = ?", res.User.ID, roles[0].ID).Count(&userRoleCount).Error)
	assert.Equal(t, int64(1), userRoleCount)
}

func TestGormTenantProvisionerOwnerRoleDefaultsToFirst(t *testing.T) {
	gdb := newProvisionTestDB(t)
	seedProvisionPermissions(t, gdb)
	cfg := provisionTestConfig()
	cfg.OwnerRole = "" // 缺省 = 第一个模板角色
	provisioner := NewGormTenantProvisioner(gdb, cfg)

	res, err := provisioner.Provision(context.Background(), ProvisionParams{
		Username:     "bob",
		PasswordHash: "hashed",
		TenantName:   "Bob Corp",
	})
	require.NoError(t, err)

	var roles []roledomain.Role
	require.NoError(t, gdb.Where("tenant_id = ?", res.Tenant.ID).Order("id").Find(&roles).Error)
	require.Len(t, roles, 2)
	var userRoleCount int64
	require.NoError(t, gdb.Table("user_roles").Where("user_id = ? AND role_id = ?", res.User.ID, roles[0].ID).Count(&userRoleCount).Error)
	assert.Equal(t, int64(1), userRoleCount)

	// 自动生成的编码以 t 开头且匹配格式
	assert.Regexp(t, `^t[0-9a-f]{12}$`, res.Tenant.Code)
}

func TestGormTenantProvisionerErrors(t *testing.T) {
	gdb := newProvisionTestDB(t)
	seedProvisionPermissions(t, gdb)
	provisioner := NewGormTenantProvisioner(gdb, provisionTestConfig())
	ctx := context.Background()

	t.Run("租户名缺失", func(t *testing.T) {
		_, err := provisioner.Provision(ctx, ProvisionParams{Username: "x", PasswordHash: "h"})
		assert.Equal(t, apperrors.CodeInvalidParam, appCodeOf(err))
	})

	t.Run("编码格式无效", func(t *testing.T) {
		_, err := provisioner.Provision(ctx, ProvisionParams{Username: "x", PasswordHash: "h", TenantName: "T", TenantCode: "bad code"})
		assert.Equal(t, apperrors.CodeTenantCodeFormat, appCodeOf(err))
	})

	t.Run("编码冲突", func(t *testing.T) {
		// 先开通一个租户占用编码，再以相同编码开通 → 冲突
		_, err := provisioner.Provision(ctx, ProvisionParams{Username: "first", PasswordHash: "h", TenantName: "First", TenantCode: "taken"})
		require.NoError(t, err)
		_, err = provisioner.Provision(ctx, ProvisionParams{Username: "second", PasswordHash: "h", TenantName: "Second", TenantCode: "taken"})
		assert.Equal(t, apperrors.CodeTenantExists, appCodeOf(err))
	})

	t.Run("用户名冲突（DB unique 兜底）", func(t *testing.T) {
		// 预置同名用户，触发 user 表唯一约束 → 事务回滚，租户不应残留
		require.NoError(t, gdb.Create(&userdomain.User{Username: "alice", Password: "h", Status: 1, TenantID: 1}).Error)
		_, err := provisioner.Provision(ctx, ProvisionParams{Username: "alice", PasswordHash: "h", TenantName: "T2"})
		assert.Equal(t, apperrors.CodeUserExists, appCodeOf(err))

		// 事务已回滚：T2 租户不存在
		var count int64
		require.NoError(t, gdb.Model(&tenantdomain.Tenant{}).Where("name = ?", "T2").Count(&count).Error)
		assert.Zero(t, count)
	})
}

func appCodeOf(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}
