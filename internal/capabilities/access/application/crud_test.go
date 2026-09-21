package application

import (
	"context"
	"testing"

	"jimu/internal/capabilities/access/domain"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	mysqlerr "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 角色 CRUD ----

func TestRoleServiceGet(t *testing.T) {
	svc := NewRoleService(&fakeRoleRepository{role: &domain.Role{ID: 3, Name: "ops"}})
	resp, err := svc.Get(context.Background(), 3)
	require.NoError(t, err)
	assert.Equal(t, "ops", resp.Name)

	// 记录不存在 → 404
	svc = NewRoleService(&fakeRoleRepository{findErr: gorm.ErrRecordNotFound})
	_, err = svc.Get(context.Background(), 9)
	assert.Equal(t, apperrors.CodeNotFound, roleAppCode(err))

	// 其他错误 → 500
	svc = NewRoleService(&fakeRoleRepository{findErr: gorm.ErrInvalidDB})
	_, err = svc.Get(context.Background(), 9)
	assert.Equal(t, apperrors.CodeInternalError, roleAppCode(err))
}

func TestRoleServiceUpdate(t *testing.T) {
	repo := &fakeRoleRepository{role: &domain.Role{ID: 3, Name: "ops"}}
	svc := NewRoleService(repo)
	require.NoError(t, svc.Update(context.Background(), 3, UpdateRoleRequest{Name: "ops2", Description: "d"}))
	assert.Equal(t, "ops2", repo.role.Name)

	// 不存在 → 404
	svc = NewRoleService(&fakeRoleRepository{findErr: gorm.ErrRecordNotFound})
	assert.Equal(t, apperrors.CodeNotFound, roleAppCode(svc.Update(context.Background(), 9, UpdateRoleRequest{Name: "x"})))

	// 重名 → 409
	svc = NewRoleService(&fakeRoleRepository{
		role:      &domain.Role{ID: 3, Name: "ops"},
		updateErr: &mysqlerr.MySQLError{Number: 1062, Message: "Duplicate entry"},
	})
	assert.Equal(t, apperrors.CodeConflict, roleAppCode(svc.Update(context.Background(), 3, UpdateRoleRequest{Name: "dup"})))

	// 其他写错误 → 500
	svc = NewRoleService(&fakeRoleRepository{role: &domain.Role{ID: 3}, updateErr: gorm.ErrInvalidDB})
	assert.Equal(t, apperrors.CodeInternalError, roleAppCode(svc.Update(context.Background(), 3, UpdateRoleRequest{Name: "x"})))
}

func TestRoleServiceDelete(t *testing.T) {
	svc := NewRoleService(&fakeRoleRepository{})
	require.NoError(t, svc.Delete(context.Background(), 3))

	svc = NewRoleService(&fakeRoleRepository{deleteErr: gorm.ErrInvalidDB})
	assert.Equal(t, apperrors.CodeInternalError, roleAppCode(svc.Delete(context.Background(), 3)))
}

func TestRoleServicePermissions(t *testing.T) {
	svc := NewRoleService(&fakeRoleRepository{})
	require.NoError(t, svc.AssignPermissions(context.Background(), 3, []uint64{1, 2}))

	perms, err := svc.GetPermissions(context.Background(), 3)
	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestRoleServiceCreateQuota(t *testing.T) {
	quota := &fakeTenantQuota{err: apperrors.New(apperrors.CodeQuotaExceeded, "full")}
	repo := &fakeRoleRepository{}
	svc := NewRoleService(repo).WithQuota(quota)

	_, err := svc.Create(context.Background(), CreateRoleRequest{Name: "ops"})
	assert.Equal(t, apperrors.CodeQuotaExceeded, roleAppCode(err))
	assert.Empty(t, repo.created, "配额超限时不得写库")
}

// ---- 权限 CRUD ----

func TestPermissionServiceCreate(t *testing.T) {
	perm, err := NewPermissionService(&fakePermissionRepository{}).
		Create(context.Background(), CreatePermissionRequest{Name: "读", Resource: "/api/v1/x", Action: "GET"})
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/x", perm.Resource)

	// 重复 → 409
	svc := NewPermissionService(&fakePermissionRepository{createErr: &mysqlerr.MySQLError{Number: 1062}})
	_, err = svc.Create(context.Background(), CreatePermissionRequest{Name: "读", Resource: "/x", Action: "GET"})
	assert.Equal(t, apperrors.CodeConflict, roleAppCode(err))

	// 其他错误 → 500
	svc = NewPermissionService(&fakePermissionRepository{createErr: gorm.ErrInvalidDB})
	_, err = svc.Create(context.Background(), CreatePermissionRequest{Name: "读", Resource: "/x", Action: "GET"})
	assert.Equal(t, apperrors.CodeInternalError, roleAppCode(err))
}

func TestPermissionServiceGetListUpdateDelete(t *testing.T) {
	svc := NewPermissionService(&fakePermissionRepository{permission: &domain.Permission{ID: 2, Name: "读"}})
	got, err := svc.Get(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, "读", got.Name)

	// 不存在 → 404
	svc = NewPermissionService(&fakePermissionRepository{findErr: gorm.ErrRecordNotFound})
	_, err = svc.Get(context.Background(), 2)
	assert.Equal(t, apperrors.CodeNotFound, roleAppCode(err))

	// 其他错误 → 500
	svc = NewPermissionService(&fakePermissionRepository{findErr: gorm.ErrInvalidDB})
	_, err = svc.Get(context.Background(), 2)
	assert.Equal(t, apperrors.CodeInternalError, roleAppCode(err))

	// List
	svc = NewPermissionService(&fakePermissionRepository{permissions: []domain.Permission{{ID: 1}}, total: 1})
	list, total, err := svc.List(context.Background(), pagination.Pagination{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)

	// Update 成功
	repo := &fakePermissionRepository{permission: &domain.Permission{ID: 2, Name: "读"}}
	svc = NewPermissionService(repo)
	require.NoError(t, svc.Update(context.Background(), 2, UpdatePermissionRequest{Name: "写", Resource: "/x", Action: "POST"}))
	assert.Equal(t, "写", repo.permission.Name)

	// Update 不存在 → 404
	svc = NewPermissionService(&fakePermissionRepository{findErr: gorm.ErrRecordNotFound})
	assert.Equal(t, apperrors.CodeNotFound, roleAppCode(svc.Update(context.Background(), 2, UpdatePermissionRequest{})))

	// Update 重名 → 409
	svc = NewPermissionService(&fakePermissionRepository{
		permission: &domain.Permission{ID: 2},
		updateErr:  &mysqlerr.MySQLError{Number: 1062},
	})
	assert.Equal(t, apperrors.CodeConflict, roleAppCode(svc.Update(context.Background(), 2, UpdatePermissionRequest{})))

	// Update 其他错误 → 500
	svc = NewPermissionService(&fakePermissionRepository{permission: &domain.Permission{ID: 2}, updateErr: gorm.ErrInvalidDB})
	assert.Equal(t, apperrors.CodeInternalError, roleAppCode(svc.Update(context.Background(), 2, UpdatePermissionRequest{})))

	// Delete
	require.NoError(t, NewPermissionService(&fakePermissionRepository{}).Delete(context.Background(), 2))
	svc = NewPermissionService(&fakePermissionRepository{deleteErr: gorm.ErrInvalidDB})
	assert.Equal(t, apperrors.CodeInternalError, roleAppCode(svc.Delete(context.Background(), 2)))
}
