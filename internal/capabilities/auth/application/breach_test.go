package application

import (
	"context"
	"sync"
	"testing"
	"time"

	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/platform/auth"
	"jimu/internal/platform/encryption"
	"jimu/internal/platform/tenant"
	apperrors "jimu/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// fakeBreachChecker 记录调用并按配置返回命中结果
type fakeBreachChecker struct {
	breached map[string]bool
	err      error
	calls    []string
}

func (f *fakeBreachChecker) IsBreached(_ context.Context, password string) (bool, error) {
	f.calls = append(f.calls, password)
	if f.err != nil {
		return false, f.err
	}
	return f.breached[password], nil
}

// fakeProvisioner 记录开通参数，用于隔离开通式注册的其余逻辑
type fakeProvisioner struct {
	mu     sync.Mutex
	params []ProvisionParams
}

func (f *fakeProvisioner) Provision(_ context.Context, params ProvisionParams) (*ProvisionResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.params = append(f.params, params)
	return &ProvisionResult{User: &userdomain.User{ID: 7, Username: params.Username}}, nil
}

func (f *fakeProvisioner) called() []ProvisionParams {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ProvisionParams{}, f.params...)
}

func newBreachService(t *testing.T, checker *fakeBreachChecker) (*AuthService, *fakeUserRepo) {
	t.Helper()
	repo := &fakeUserRepo{users: map[string]*userdomain.User{}}
	svc := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30, checker)
	return svc, repo
}

func TestRegisterRejectsBreachedPassword(t *testing.T) {
	ctx := context.Background()
	checker := &fakeBreachChecker{breached: map[string]bool{"password123": true}}
	svc, repo := newBreachService(t, checker)

	_, err := svc.Register(ctx, "alice", "password123", "alice@example.com", "")
	assert.Equal(t, apperrors.CodePasswordBreached, appCode(err))
	assert.Equal(t, []string{"password123"}, checker.calls)
	assert.Empty(t, repo.created, "命中泄露库时不得创建用户")

	// 未命中的口令正常注册，且同样经过检查
	user, err := svc.Register(ctx, "bob", "a-clean-passphrase", "bob@example.com", "")
	require.NoError(t, err)
	assert.Equal(t, "bob", user.Username)
	assert.Equal(t, []string{"password123", "a-clean-passphrase"}, checker.calls)
}

func TestRegisterProvisionedRejectsBreachedPassword(t *testing.T) {
	ctx := context.Background()
	checker := &fakeBreachChecker{breached: map[string]bool{"password123": true}}
	provisioner := &fakeProvisioner{}
	repo := &fakeUserRepo{users: map[string]*userdomain.User{}}
	svc := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7),
		newFakeSessionStore(), nil, 30, checker, provisioner)

	_, err := svc.RegisterProvisioned(ctx, RegisterTenantRequest{
		Username: "alice", Password: "password123", TenantName: "Acme",
	})
	assert.Equal(t, apperrors.CodePasswordBreached, appCode(err))
	assert.Empty(t, provisioner.called(), "命中泄露库时不得开通租户")

	_, err = svc.RegisterProvisioned(ctx, RegisterTenantRequest{
		Username: "alice", Password: "a-clean-passphrase", TenantName: "Acme",
	})
	require.NoError(t, err)
	require.Len(t, provisioner.called(), 1)
}

func TestRegisterAllowsPasswordWhenCheckFails(t *testing.T) {
	ctx := context.Background()
	checker := &fakeBreachChecker{err: context.DeadlineExceeded}
	svc, repo := newBreachService(t, checker)

	user, err := svc.Register(ctx, "alice", "password123", "alice@example.com", "")
	require.NoError(t, err, "检查服务不可用时应放行")
	assert.Equal(t, "alice", user.Username)
	assert.Len(t, repo.created, 1)
}

func TestRegisterSkipsCheckWhenDisabled(t *testing.T) {
	ctx := context.Background()
	svc, _ := newBreachService(t, &fakeBreachChecker{})
	svc.breachChecker = nil

	_, err := svc.Register(ctx, "alice", "password123", "alice@example.com", "")
	require.NoError(t, err, "未启用检查器时放行")
}

func TestResetPasswordRejectsBreachedPassword(t *testing.T) {
	ctx := context.Background()
	_, rclient := newResetRedis(t)
	resetStore := NewResetStore(rclient, 15*time.Minute)
	cipher := encryption.New(resetTestKey)

	repo := &fakeUserRepo{users: map[string]*userdomain.User{}}
	alice := userWithPassword(t, 42, "alice", "correct", 1)
	repo.users["alice"] = alice
	repo.findByEmailHash = func(_ context.Context, hash string) (*userdomain.User, error) {
		if hash != cipher.BlindIndex("alice@example.com") {
			return nil, gorm.ErrRecordNotFound
		}
		return alice, nil
	}
	checker := &fakeBreachChecker{breached: map[string]bool{"password123": true}}
	svc := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7),
		newFakeSessionStore(), nil, 30, cipher, &fakeDispatcher{}, resetStore, checker)
	svc.resetGen = func() string { return "123456" }

	require.NoError(t, svc.ForgotPassword(ctx, "alice@example.com"))
	err := svc.ResetPassword(ctx, "alice@example.com", "123456", "password123")
	assert.Equal(t, apperrors.CodePasswordBreached, appCode(err))
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(alice.Password), []byte("correct")), "被拒时密码保持不变")
}

// fakeTenantQuota 只返回预设的配额校验结果
type fakeTenantQuota struct {
	err     error
	checked []uint64
}

func (f *fakeTenantQuota) CheckUserQuota(_ context.Context, tenantID uint64) error {
	f.checked = append(f.checked, tenantID)
	return f.err
}

func TestRegisterRejectsWhenQuotaExceeded(t *testing.T) {
	ctx := context.Background()
	quota := &fakeTenantQuota{err: apperrors.New(apperrors.CodeQuotaExceeded, "users quota exceeded")}
	svc, repo := newBreachService(t, &fakeBreachChecker{})
	svc.quota = quota

	_, err := svc.Register(ctx, "alice", "password123", "alice@example.com", "")
	assert.Equal(t, apperrors.CodeQuotaExceeded, appCode(err))
	assert.Equal(t, []uint64{tenant.DefaultTenantID}, quota.checked, "公开注册的用户归默认租户，配额按默认租户校验")
	assert.Empty(t, repo.created, "配额超限时不应创建用户")
}
