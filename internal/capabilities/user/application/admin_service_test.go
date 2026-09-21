package application

import (
	"context"
	"testing"

	"jimu/internal/capabilities/user/domain"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTenantQuota 只按预设结果返回配额校验结果
type fakeTenantQuota struct {
	userErr error
	checked []uint64
}

func (f *fakeTenantQuota) CheckUserQuota(_ context.Context, tenantID uint64) error {
	f.checked = append(f.checked, tenantID)
	return f.userErr
}

func TestCreateUserRejectsWhenQuotaExceeded(t *testing.T) {
	ctx := tenant.WithTenant(context.Background(), 7)
	quota := &fakeTenantQuota{userErr: apperrors.New(apperrors.CodeQuotaExceeded, "users quota exceeded")}
	repo := &fakeUserRepository{create: func(_ context.Context, user *domain.User) error {
		t.Fatal("配额超限时不应写入用户")
		return nil
	}}
	svc := NewAdminUserService(repo, nil).WithQuota(quota)

	_, err := svc.CreateUser(ctx, AdminCreateUserRequest{Username: "bob", Password: "password123"})
	assert.Equal(t, apperrors.CodeQuotaExceeded, quotaCode(err))
	assert.Equal(t, []uint64{7}, quota.checked, "应按上下文租户校验配额")
}

func TestCreateUserChecksDefaultTenantWithoutContext(t *testing.T) {
	quota := &fakeTenantQuota{}
	created := false
	repo := &fakeUserRepository{create: func(_ context.Context, user *domain.User) error {
		created = true
		assert.Equal(t, uint64(tenant.DefaultTenantID), user.TenantID)
		return nil
	}}
	svc := NewAdminUserService(repo, nil).WithQuota(quota)

	_, err := svc.CreateUser(context.Background(), AdminCreateUserRequest{Username: "bob", Password: "password123"})
	require.NoError(t, err)
	require.True(t, created)
	assert.Equal(t, []uint64{tenant.DefaultTenantID}, quota.checked, "无租户上下文时校验默认租户配额")
}

// TestAssignRolesDelegatesToPort 角色分配委托 access 能力（user_roles 表所有者）
func TestAssignRolesDelegatesToPort(t *testing.T) {
	svc := NewAdminUserService(&fakeUserRepository{}, nil)
	err := svc.AssignRoles(context.Background(), 1, []string{"admin"})
	assert.Equal(t, apperrors.CodeInternalError, quotaCode(err), "未注入端口时报内部错误")

	assigner := &fakeRoleAssigner{}
	svc.WithRoles(assigner)
	require.NoError(t, svc.AssignRoles(context.Background(), 7, []string{"admin", "viewer"}))
	assert.Equal(t, uint64(7), assigner.userID)
	assert.Equal(t, []string{"admin", "viewer"}, assigner.roles)
}

type fakeRoleAssigner struct {
	userID uint64
	roles  []string
	err    error
}

func (f *fakeRoleAssigner) AssignRoles(_ context.Context, userID uint64, roleNames []string) error {
	f.userID, f.roles = userID, roleNames
	return f.err
}

func quotaCode(err error) int {
	var appErr *apperrors.AppError
	if apperrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}
