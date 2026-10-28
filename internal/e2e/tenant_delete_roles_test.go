package e2e

import (
	"context"
	stderrors "errors"
	"testing"

	accessapp "jimu/internal/capabilities/access/application"
	accessdomain "jimu/internal/capabilities/access/domain"
	accessinfra "jimu/internal/capabilities/access/infrastructure"
	tenantapp "jimu/internal/capabilities/tenant/application"
	tenantdomain "jimu/internal/capabilities/tenant/domain"
	tenantinfra "jimu/internal/capabilities/tenant/infrastructure"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type failingTenantRoleLifecycle struct{}

func (failingTenantRoleLifecycle) DeleteTenantRoles(context.Context, uint64) error {
	return stderrors.New("tenant role cleanup failed")
}

func TestDeleteTenantCleansTenantRolesAndHidesThemFromRoleList(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&tenantdomain.Tenant{},
		&userdomain.User{},
		&accessdomain.Role{},
		&accessdomain.Permission{},
	))
	require.NoError(t, db.Exec(`CREATE TABLE user_roles (
		user_id INTEGER NOT NULL,
		role_id INTEGER NOT NULL,
		PRIMARY KEY (user_id, role_id)
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE role_permissions (
		role_id INTEGER NOT NULL,
		permission_id INTEGER NOT NULL,
		PRIMARY KEY (role_id, permission_id)
	)`).Error)

	defaultTenant := tenantdomain.Tenant{ID: 1, Code: "default", Name: "Default", Status: 1}
	require.NoError(t, db.Create(&defaultTenant).Error)
	targetTenant := tenantdomain.Tenant{ID: 2, Code: "acme", Name: "Acme", Status: 1}
	require.NoError(t, db.Create(&targetTenant).Error)

	platformRole := accessdomain.Role{
		TenantID: defaultTenant.ID,
		Name:     "platform_super_admin",
		Code:     contract.RoleCodePlatformSuperAdmin,
		Scope:    contract.RoleScopePlatform,
		IsSystem: true,
		Status:   1,
	}
	require.NoError(t, db.Create(&platformRole).Error)

	adminRole := accessdomain.Role{
		TenantID: targetTenant.ID,
		Name:     "tenant_admin",
		Code:     contract.RoleCodeTenantAdmin,
		Scope:    contract.RoleScopeTenant,
		IsSystem: true,
		Status:   1,
	}
	memberRole := accessdomain.Role{
		TenantID: targetTenant.ID,
		Name:     "tenant_member",
		Code:     contract.RoleCodeTenantMember,
		Scope:    contract.RoleScopeTenant,
		IsSystem: true,
		Status:   1,
	}
	require.NoError(t, db.Create(&adminRole).Error)
	require.NoError(t, db.Create(&memberRole).Error)

	permission := accessdomain.Permission{Name: "users", Resource: "/users", Action: "GET"}
	require.NoError(t, db.Create(&permission).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)",
		adminRole.ID,
		permission.ID,
	).Error)

	member := userdomain.User{
		Username: "bob",
		Password: "hash",
		Status:   1,
		TenantID: targetTenant.ID,
	}
	require.NoError(t, db.Create(&member).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)",
		member.ID,
		memberRole.ID,
	).Error)

	tenantRoles := accessapp.NewTenantRoleService(db)
	tenants := tenantapp.NewTenantService(tenantinfra.NewMysqlRepository(db)).
		WithDB(db).
		WithRoleLifecycle(tenantRoles)
	roles := accessinfra.NewMysqlRepository(db)

	before, totalBefore, err := roles.List(context.Background(), accessdomain.RoleListFilter{}, 0, 20, "id", "asc")
	require.NoError(t, err)
	require.Len(t, before, 3)
	require.EqualValues(t, 3, totalBefore)

	require.NoError(t, tenants.Delete(context.Background(), targetTenant.ID))

	var activeTenantRoles int64
	require.NoError(t, db.Model(&accessdomain.Role{}).
		Where("tenant_id = ?", targetTenant.ID).
		Count(&activeTenantRoles).Error)
	require.Zero(t, activeTenantRoles, "tenant deletion must soft-delete all roles owned by the tenant")

	var rolePermissions int64
	require.NoError(t, db.Table("role_permissions").
		Where("role_id IN ?", []uint64{adminRole.ID, memberRole.ID}).
		Count(&rolePermissions).Error)
	require.Zero(t, rolePermissions)

	var assignments int64
	require.NoError(t, db.Table("user_roles").
		Where("user_id = ? OR role_id IN ?", member.ID, []uint64{adminRole.ID, memberRole.ID}).
		Count(&assignments).Error)
	require.Zero(t, assignments)

	after, totalAfter, err := roles.List(context.Background(), accessdomain.RoleListFilter{}, 0, 20, "id", "asc")
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.EqualValues(t, 1, totalAfter)
	require.Equal(t, platformRole.ID, after[0].ID,
		"tenant_admin and tenant_member must no longer appear in the role list")
}

func TestDeleteTenantRollsBackWhenRoleCleanupFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&tenantdomain.Tenant{}))

	targetTenant := tenantdomain.Tenant{ID: 2, Code: "acme", Name: "Acme", Status: 1}
	require.NoError(t, db.Create(&targetTenant).Error)

	tenants := tenantapp.NewTenantService(tenantinfra.NewMysqlRepository(db)).
		WithDB(db).
		WithRoleLifecycle(failingTenantRoleLifecycle{})

	require.Error(t, tenants.Delete(context.Background(), targetTenant.ID))
	var stored tenantdomain.Tenant
	require.NoError(t, db.First(&stored, targetTenant.ID).Error, "role cleanup failure must roll back tenant deletion")
}
