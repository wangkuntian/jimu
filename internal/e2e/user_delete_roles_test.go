package e2e

import (
	"context"
	stderrors "errors"
	"testing"

	accessapp "jimu/internal/capabilities/access/application"
	accessdomain "jimu/internal/capabilities/access/domain"
	accessinfra "jimu/internal/capabilities/access/infrastructure"
	tenantdomain "jimu/internal/capabilities/tenant/domain"
	userapp "jimu/internal/capabilities/user/application"
	userdomain "jimu/internal/capabilities/user/domain"
	userinfra "jimu/internal/capabilities/user/infrastructure"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type failingUserRoleAssigner struct{}

func (failingUserRoleAssigner) AssignRoleIDs(context.Context, uint64, []uint64) error {
	return nil
}

func (failingUserRoleAssigner) RolesForUsers(context.Context, []uint64) (map[uint64][]contract.UserRole, error) {
	return nil, nil
}

func (failingUserRoleAssigner) EnsureCanDeactivateUser(context.Context, uint64) error {
	return nil
}

func (failingUserRoleAssigner) RemoveUserRoles(context.Context, uint64) error {
	return stderrors.New("role cleanup failed")
}

func TestDeleteUserRemovesRoleAssignmentsAndUnblocksRoleDeletion(t *testing.T) {
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

	targetTenant := tenantdomain.Tenant{ID: 2, Code: "acme", Name: "Acme", Status: 1}
	require.NoError(t, db.Create(&targetTenant).Error)

	user := userdomain.User{
		Username: "alice",
		Password: "hash",
		Status:   1,
		TenantID: targetTenant.ID,
	}
	require.NoError(t, db.Create(&user).Error)

	customRole := accessdomain.Role{
		TenantID: targetTenant.ID,
		Name:     "operator",
		Scope:    contract.RoleScopeTenant,
		Status:   1,
	}
	require.NoError(t, db.Create(&customRole).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)",
		user.ID,
		customRole.ID,
	).Error)

	ctx := tenant.WithTenant(context.Background(), targetTenant.ID)
	userRoles := accessapp.NewUserRoleService(db)
	users := userapp.NewUserService(userinfra.NewMysqlRepository(db), userRoles).WithDB(db)
	roles := accessapp.NewRoleService(accessinfra.NewMysqlRepository(db))

	require.NoError(t, users.Delete(ctx, user.ID))

	var assignments int64
	require.NoError(t, db.Table("user_roles").Where("user_id = ?", user.ID).Count(&assignments).Error)
	require.Zero(t, assignments, "deleting a user must remove user_roles in the same transaction")

	require.NoError(t, roles.Delete(ctx, customRole.ID),
		"a role released by user deletion must be deletable instead of returning 409")
}

func TestDeleteUserRollsBackWhenRoleCleanupFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&userdomain.User{}))

	user := userdomain.User{Username: "alice", Password: "hash", Status: 1, TenantID: 2}
	require.NoError(t, db.Create(&user).Error)

	ctx := tenant.WithTenant(context.Background(), 2)
	users := userapp.NewUserService(userinfra.NewMysqlRepository(db), failingUserRoleAssigner{}).WithDB(db)

	require.Error(t, users.Delete(ctx, user.ID))
	var stored userdomain.User
	require.NoError(t, db.First(&stored, user.ID).Error, "role cleanup failure must roll back user deletion")
}
