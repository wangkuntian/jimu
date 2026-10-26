package access

import (
	"context"
	"testing"

	"jimu/internal/capabilities/access/domain"
	"jimu/internal/contract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvisioningRoleStorePersistsSystemMetadata(t *testing.T) {
	db := newAccessTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.Role{}, &domain.Permission{}))
	require.NoError(t, db.Exec("CREATE TABLE role_permissions (role_id INTEGER, permission_id INTEGER)").Error)
	permission := domain.Permission{Name: "用户列表", Resource: "/api/v1/users", Action: "GET"}
	require.NoError(t, db.Create(&permission).Error)

	store := NewProvisioningRoleStore(db)
	ownerID, err := store.ProvisionRoles(context.Background(), 7, contract.ProvisioningRoles{
		OwnerRole: "custom-owner",
		Roles: []contract.ProvisioningRole{
			{
				Code:   contract.RoleCodeTenantAdmin,
				Scope:  contract.RoleScopeTenant,
				System: true,
				Name:   "租户管理员",
			},
			{
				Code:   contract.RoleCodeTenantMember,
				Scope:  contract.RoleScopeTenant,
				System: true,
				Name:   "租户成员",
			},
			{
				Scope:       contract.RoleScopeTenant,
				Name:        "custom-owner",
				Permissions: []contract.ProvisioningPermission{{Resource: "/api/v1/users", Action: "GET"}},
			},
		},
	})
	require.NoError(t, err)

	var roles []domain.Role
	require.NoError(t, db.Where("tenant_id = ?", 7).Order("id").Find(&roles).Error)
	require.Len(t, roles, 3)
	assert.Equal(t, contract.RoleCodeTenantAdmin, roles[0].Code)
	assert.True(t, roles[0].IsSystem)
	assert.Equal(t, contract.RoleCodeTenantMember, roles[1].Code)
	assert.Empty(t, roles[2].Code)
	assert.False(t, roles[2].IsSystem)
	assert.Equal(t, roles[2].ID, ownerID)

	var count int64
	require.NoError(t, db.Table("role_permissions").Where("role_id = ?", ownerID).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
