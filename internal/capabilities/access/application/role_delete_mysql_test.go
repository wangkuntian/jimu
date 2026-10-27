package application

import (
	"context"
	"strconv"
	"testing"
	"time"

	"jimu/internal/capabilities/access/domain"
	accessinfra "jimu/internal/capabilities/access/infrastructure"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRoleDeleteSerializesWithAssignmentOnMySQL(t *testing.T) {
	db := newConcurrencyTestDB(t)
	role := domain.Role{
		TenantID: tenant.DefaultTenantID,
		Name:     "delete-race-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		Scope:    contract.RoleScopeTenant,
		Status:   1,
	}
	require.NoError(t, db.Create(&role).Error)

	userID := uint64(time.Now().UnixNano())
	require.NoError(t, db.Exec(
		"INSERT INTO users (id, username, password, status, tenant_id, created_at, updated_at) VALUES (?, ?, ?, 1, ?, NOW(), NOW())",
		userID, "delete-race-"+strconv.FormatUint(userID, 36), "hash", tenant.DefaultTenantID,
	).Error)

	roleService := NewRoleService(accessinfra.NewMysqlRepository(db))
	userRoleService := NewUserRoleService(db)
	ctx := tenant.WithPlatformScope(tenant.WithTenant(context.Background(), tenant.DefaultTenantID))

	start := make(chan struct{})
	deleteResult := make(chan error, 1)
	assignResult := make(chan error, 1)
	go func() {
		<-start
		deleteResult <- roleService.Delete(ctx, role.ID)
	}()
	go func() {
		<-start
		assignResult <- userRoleService.AssignRoleIDs(ctx, userID, []uint64{role.ID})
	}()
	close(start)

	deleteErr := <-deleteResult
	assignErr := <-assignResult
	if (deleteErr == nil) == (assignErr == nil) {
		t.Fatalf("exactly one operation must succeed: delete=%v assign=%v", deleteErr, assignErr)
	}

	if assignErr == nil {
		require.Equal(t, apperrors.CodeConflict, roleAppCode(deleteErr))
		var userRoles int64
		require.NoError(t, db.Table("user_roles").Where("user_id = ? AND role_id = ?", userID, role.ID).Count(&userRoles).Error)
		require.Equal(t, int64(1), userRoles)
		var stored domain.Role
		require.NoError(t, db.First(&stored, role.ID).Error)
		return
	}

	require.NoError(t, deleteErr)
	require.Equal(t, apperrors.CodeNotFound, userRoleAppCode(assignErr))
	var userRoles int64
	require.NoError(t, db.Table("user_roles").Where("user_id = ? AND role_id = ?", userID, role.ID).Count(&userRoles).Error)
	require.Zero(t, userRoles)
	var stored domain.Role
	require.ErrorIs(t, db.First(&stored, role.ID).Error, gorm.ErrRecordNotFound)
}
