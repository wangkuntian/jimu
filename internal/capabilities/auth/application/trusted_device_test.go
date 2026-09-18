package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	authdomain "jimu/internal/capabilities/auth/domain"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/platform/auth"
	"jimu/internal/platform/encryption"
	"jimu/internal/platform/tenant"
	apperrors "jimu/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeTrustedDeviceRepo 内存可信设备仓储
type fakeTrustedDeviceRepo struct {
	devices  []*authdomain.TrustedDevice
	nextID   uint64
	touched  []uint64
	deletedA []uint64
}

func newFakeTrustedDeviceRepo() *fakeTrustedDeviceRepo {
	return &fakeTrustedDeviceRepo{nextID: 1}
}

func (r *fakeTrustedDeviceRepo) Create(_ context.Context, device *authdomain.TrustedDevice) error {
	device.ID = r.nextID
	r.nextID++
	r.devices = append(r.devices, device)
	return nil
}

func (r *fakeTrustedDeviceRepo) FindByTokenHash(_ context.Context, tokenHash string) (*authdomain.TrustedDevice, error) {
	for _, device := range r.devices {
		if device.TokenHash == tokenHash {
			return device, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeTrustedDeviceRepo) Touch(_ context.Context, id uint64, usedAt time.Time) error {
	r.touched = append(r.touched, id)
	for _, device := range r.devices {
		if device.ID == id {
			device.LastUsedAt = &usedAt
		}
	}
	return nil
}

func (r *fakeTrustedDeviceRepo) ListByUser(_ context.Context, tenantID, userID uint64) ([]authdomain.TrustedDevice, error) {
	var out []authdomain.TrustedDevice
	for _, device := range r.devices {
		if device.UserID != userID {
			continue
		}
		if tenantID != 0 && device.TenantID != tenantID {
			continue
		}
		out = append(out, *device)
	}
	return out, nil
}

func (r *fakeTrustedDeviceRepo) Delete(_ context.Context, tenantID, userID, id uint64) error {
	kept := r.devices[:0]
	for _, device := range r.devices {
		if device.ID == id && device.UserID == userID && (tenantID == 0 || device.TenantID == tenantID) {
			continue
		}
		kept = append(kept, device)
	}
	r.devices = kept
	return nil
}

func (r *fakeTrustedDeviceRepo) DeleteAllByUser(_ context.Context, userID uint64) error {
	r.deletedA = append(r.deletedA, userID)
	kept := r.devices[:0]
	for _, device := range r.devices {
		if device.UserID == userID {
			continue
		}
		kept = append(kept, device)
	}
	r.devices = kept
	return nil
}

func (r *fakeTrustedDeviceRepo) DeleteExpired(_ context.Context, now time.Time) (int64, error) {
	var removed int64
	kept := r.devices[:0]
	for _, device := range r.devices {
		if device.ExpiresAt.Before(now) {
			removed++
			continue
		}
		kept = append(kept, device)
	}
	r.devices = kept
	return removed, nil
}

func (r *fakeTrustedDeviceRepo) count() int { return len(r.devices) }

// newTrustedDeviceService 构造启用了 TOTP + 可信设备的服务，并返回启用 TOTP 的用户。
func newTrustedDeviceService(t *testing.T, ttlDays int) (*AuthService, *fakeUserRepo, *fakeTrustedDeviceRepo, *userdomain.User, string) {
	t.Helper()
	repo := &fakeUserRepo{users: map[string]*userdomain.User{}}
	devices := newFakeTrustedDeviceRepo()
	svc := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7),
		newFakeSessionStore(), nil, 30, devices, WithTrustedDeviceTTL(ttlDays))

	alice := userWithPassword(t, 42, "alice", "correct", 1)
	alice.TenantID = 1
	repo.users["alice"] = alice

	secret, _, err := svc.SetupTOTP(context.Background(), 42, "alice")
	require.NoError(t, err)
	require.NoError(t, svc.EnableTOTP(context.Background(), 42, totpCurrentCode(t, secret)))
	require.True(t, alice.TOTPEnabled)
	return svc, repo, devices, alice, secret
}

func TestLoginWithRememberDeviceIssuesToken(t *testing.T) {
	ctx := context.Background()
	svc, _, devices, _, secret := newTrustedDeviceService(t, 30)

	ctx = WithLoginDevice(WithClientInfo(ctx, "203.0.113.7", "curl/8.0"), "", true)
	pair, err := svc.LoginWithTOTP(ctx, "alice", "correct", totpCurrentCode(t, secret))
	require.NoError(t, err)

	require.NotEmpty(t, pair.DeviceToken, "勾选记住设备时应签发设备令牌")
	assert.True(t, strings.HasPrefix(pair.DeviceToken, deviceTokenPrefix))
	require.Equal(t, 1, devices.count())

	device := devices.devices[0]
	assert.Equal(t, uint64(42), device.UserID)
	assert.Equal(t, uint64(1), device.TenantID, "设备应随用户归属租户")
	assert.Equal(t, hashDeviceToken(pair.DeviceToken), device.TokenHash, "只存哈希，不存明文")
	assert.NotContains(t, device.TokenHash, pair.DeviceToken)
	assert.Equal(t, "203.0.113.7", device.IP)
	assert.WithinDuration(t, time.Now().AddDate(0, 0, 30), device.ExpiresAt, time.Minute)
}

func TestLoginWithoutRememberDeviceKeepsDeviceTokenEmpty(t *testing.T) {
	ctx := context.Background()
	svc, _, devices, _, secret := newTrustedDeviceService(t, 30)

	// 未勾选记住设备
	ctx = WithLoginDevice(WithClientInfo(ctx, "203.0.113.7", "curl/8.0"), "", false)
	pair, err := svc.LoginWithTOTP(ctx, "alice", "correct", totpCurrentCode(t, secret))
	require.NoError(t, err)
	assert.Empty(t, pair.DeviceToken)
	assert.Zero(t, devices.count())

	// 未启用 TOTP 的账号即使勾选也不签发
	bob := userWithPassword(t, 43, "bob", "correct", 1)
	svc.userRepo.(*fakeUserRepo).users["bob"] = bob
	ctx = WithLoginDevice(WithClientInfo(ctx, "203.0.113.7", "curl/8.0"), "", true)
	pair, err = svc.LoginWithTOTP(ctx, "bob", "correct", "")
	require.NoError(t, err)
	assert.Empty(t, pair.DeviceToken, "仅启用 TOTP 的账号才签发设备令牌")
	assert.Zero(t, devices.count())
}

func TestLoginSkipsTOTPForTrustedDevice(t *testing.T) {
	svc, _, devices, _, secret := newTrustedDeviceService(t, 30)
	base := WithClientInfo(context.Background(), "203.0.113.7", "curl/8.0")

	pair, err := svc.LoginWithTOTP(WithLoginDevice(base, "", true), "alice", "correct", totpCurrentCode(t, secret))
	require.NoError(t, err)
	require.NotEmpty(t, pair.DeviceToken)

	// 携带设备令牌、不带 TOTP 码即可登录
	pair, err = svc.LoginWithTOTP(WithLoginDevice(base, pair.DeviceToken, false), "alice", "correct", "")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.Empty(t, pair.DeviceToken, "复用时不再签发新令牌")
	assert.Equal(t, []uint64{1}, devices.touched, "跳过 TOTP 时应更新最近使用时间")

	// 不携带令牌仍然要求 TOTP
	_, err = svc.LoginWithTOTP(base, "alice", "correct", "")
	assert.Equal(t, apperrors.CodeMFARequired, appCode(err))
}

func TestLoginTrustedDeviceChecksOwnershipAndExpiry(t *testing.T) {
	svc, repo, devices, _, _ := newTrustedDeviceService(t, 30)
	base := WithClientInfo(context.Background(), "203.0.113.7", "curl/8.0")

	// 为其他用户签发令牌，alice 不可用它跳过 TOTP
	bob := userWithPassword(t, 43, "bob", "correct", 1)
	repo.users["bob"] = bob
	require.NoError(t, devices.Create(context.Background(), &authdomain.TrustedDevice{
		TenantID:  1,
		UserID:    43,
		TokenHash: hashDeviceToken("jimu_dev_other"),
		ExpiresAt: time.Now().Add(time.Hour),
	}))
	_, err := svc.LoginWithTOTP(WithLoginDevice(base, "jimu_dev_other", false), "alice", "correct", "")
	assert.Equal(t, apperrors.CodeMFARequired, appCode(err), "设备令牌只对签发它的用户有效")

	// 过期令牌不可用，且会被清理
	require.NoError(t, devices.Create(context.Background(), &authdomain.TrustedDevice{
		TenantID:  1,
		UserID:    42,
		TokenHash: hashDeviceToken("jimu_dev_expired"),
		ExpiresAt: time.Now().Add(-time.Minute),
	}))
	_, err = svc.LoginWithTOTP(WithLoginDevice(base, "jimu_dev_expired", false), "alice", "correct", "")
	assert.Equal(t, apperrors.CodeMFARequired, appCode(err))
	assert.Equal(t, 1, devices.count(), "过期设备应被顺手清理")

	// 未知令牌不可用
	_, err = svc.LoginWithTOTP(WithLoginDevice(base, "jimu_dev_unknown", false), "alice", "correct", "")
	assert.Equal(t, apperrors.CodeMFARequired, appCode(err))
}

func TestLoginTrustedDeviceDisabledWhenTTLIsZero(t *testing.T) {
	svc, _, _, _, secret := newTrustedDeviceService(t, 0)
	ctx := WithLoginDevice(WithClientInfo(context.Background(), "203.0.113.7", "curl/8.0"), "", true)

	pair, err := svc.LoginWithTOTP(ctx, "alice", "correct", totpCurrentCode(t, secret))
	require.NoError(t, err)
	assert.Empty(t, pair.DeviceToken, "ttl=0 表示关闭该能力")
}

func TestWrongPasswordStillFailsWithTrustedDevice(t *testing.T) {
	svc, _, devices, _, _ := newTrustedDeviceService(t, 30)
	base := WithClientInfo(context.Background(), "203.0.113.7", "curl/8.0")
	require.NoError(t, devices.Create(context.Background(), &authdomain.TrustedDevice{
		TenantID:  1,
		UserID:    42,
		TokenHash: hashDeviceToken("jimu_dev_known"),
		ExpiresAt: time.Now().Add(time.Hour),
	}))

	_, err := svc.LoginWithTOTP(WithLoginDevice(base, "jimu_dev_known", false), "alice", "wrong", "")
	assert.Equal(t, apperrors.CodeInvalidCredentials, appCode(err), "可信设备只替代 TOTP，不免除密码校验")
}

func TestListAndRevokeTrustedDevices(t *testing.T) {
	svc, _, devices, _, _ := newTrustedDeviceService(t, 30)
	ctx := tenant.WithTenant(context.Background(), 1)

	require.NoError(t, devices.Create(context.Background(), &authdomain.TrustedDevice{
		TenantID: 1, UserID: 42, TokenHash: "h1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	require.NoError(t, devices.Create(context.Background(), &authdomain.TrustedDevice{
		TenantID: 2, UserID: 42, TokenHash: "h2", ExpiresAt: time.Now().Add(time.Hour),
	}))

	list, err := svc.ListTrustedDevices(ctx, 42)
	require.NoError(t, err)
	assert.Len(t, list, 1, "按上下文租户隔离")

	require.NoError(t, svc.RevokeTrustedDevice(ctx, 42, list[0].ID))
	assert.Equal(t, 1, devices.count(), "只删除当前租户下的设备")

	// 越权删除他人设备无效
	require.NoError(t, svc.RevokeTrustedDevice(ctx, 43, devices.devices[0].ID))
	assert.Equal(t, 1, devices.count())

	require.NoError(t, svc.RevokeAllTrustedDevices(ctx, 42))
	assert.Equal(t, []uint64{42}, devices.deletedA)
}

func TestLogoutAllRevokesTrustedDevices(t *testing.T) {
	svc, _, devices, _, _ := newTrustedDeviceService(t, 30)
	require.NoError(t, devices.Create(context.Background(), &authdomain.TrustedDevice{
		TenantID: 1, UserID: 42, TokenHash: "h1", ExpiresAt: time.Now().Add(time.Hour),
	}))

	require.NoError(t, svc.LogoutAll(context.Background(), 42))
	assert.Zero(t, devices.count(), "登出全部设备应吊销可信设备")
}

func TestResetPasswordRevokesTrustedDevices(t *testing.T) {
	ctx := context.Background()
	_, rclient := newResetRedis(t)
	resetStore := NewResetStore(rclient, 15*time.Minute)
	notifier := &fakeDispatcher{}
	devices := newFakeTrustedDeviceRepo()
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
	svc := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7),
		newFakeSessionStore(), nil, 30, cipher, notifier, resetStore, devices)
	svc.resetGen = func() string { return "123456" }

	require.NoError(t, devices.Create(ctx, &authdomain.TrustedDevice{
		TenantID: 1, UserID: 42, TokenHash: "h1", ExpiresAt: time.Now().Add(time.Hour),
	}))

	require.NoError(t, svc.ForgotPassword(ctx, "alice@example.com"))
	require.NoError(t, svc.ResetPassword(ctx, "alice@example.com", "123456", "brand-new"))
	assert.Zero(t, devices.count(), "改密后旧设备不得继续跳过 TOTP")
}

func TestCleanupExpiredTrustedDevicesWithoutRepo(t *testing.T) {
	svc := newTestService(t, map[string]*userdomain.User{}, newFakeSessionStore())
	removed, err := svc.CleanupExpiredTrustedDevices(context.Background())
	require.NoError(t, err)
	assert.Zero(t, removed)

	_, err = svc.ListTrustedDevices(context.Background(), 42)
	assert.Equal(t, apperrors.CodeInternalError, appCode(err), "未配置仓储时应报内部错误")
}

// failingTrustedDeviceRepo 让查询/写入全部报错，用于覆盖服务的异常分支
type failingTrustedDeviceRepo struct {
	fakeTrustedDeviceRepo
	findErr   error
	createErr error
	deleteErr error
	touchErr  error
}

func (r *failingTrustedDeviceRepo) FindByTokenHash(context.Context, string) (*authdomain.TrustedDevice, error) {
	return nil, r.findErr
}
func (r *failingTrustedDeviceRepo) Create(context.Context, *authdomain.TrustedDevice) error {
	return r.createErr
}
func (r *failingTrustedDeviceRepo) Delete(context.Context, uint64, uint64, uint64) error {
	return r.deleteErr
}
func (r *failingTrustedDeviceRepo) Touch(context.Context, uint64, time.Time) error { return r.touchErr }
func (r *failingTrustedDeviceRepo) ListByUser(context.Context, uint64, uint64) ([]authdomain.TrustedDevice, error) {
	return nil, r.findErr
}
func (r *failingTrustedDeviceRepo) DeleteAllByUser(context.Context, uint64) error { return r.deleteErr }
func (r *failingTrustedDeviceRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	return 0, r.findErr
}

func TestTrustedDeviceServiceErrorBranches(t *testing.T) {
	ctx := context.Background()
	repo := &failingTrustedDeviceRepo{
		findErr:   errors.New("boom"),
		deleteErr: errors.New("boom"),
		createErr: errors.New("boom"),
	}
	svc := NewAuthService(&fakeUserRepo{users: map[string]*userdomain.User{}},
		auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30,
		repo, WithTrustedDeviceTTL(30))

	// 列表查询失败：直接透传仓储错误（响应层按 500 处理）
	if _, err := svc.ListTrustedDevices(ctx, 42); err == nil {
		t.Fatal("仓储报错时 ListTrustedDevices 应返回错误")
	}

	// 吊销失败 → 内部错误
	assert.Equal(t, apperrors.CodeInternalError, appCode(svc.RevokeTrustedDevice(ctx, 42, 1)))
	assert.Equal(t, apperrors.CodeInternalError, appCode(svc.RevokeAllTrustedDevices(ctx, 42)))

	// 签发失败时返回错误，由登录流程降级为「不返回设备令牌」
	alice := userWithPassword(t, 42, "alice", "correct", 1)
	assert.Error(t, func() error { _, err := svc.issueTrustedDevice(ctx, alice); return err }())

	// 过期设备视为不可信，并顺手清理（清理失败不影响判定）
	expired := &failingTrustedDeviceRepo{deleteErr: errors.New("cleanup failed")}
	expired.devices = []*authdomain.TrustedDevice{{
		ID: 9, UserID: 42, TokenHash: hashDeviceToken("jimu_dev_expired"), ExpiresAt: time.Now().Add(-time.Minute),
	}}
	svc.trustedDevices = expired
	assert.False(t, svc.isTrustedDevice(ctx, alice), "过期设备应视为不可信")

	// 清理过期设备：错误透传
	svc.trustedDevices = &failingTrustedDeviceRepo{findErr: errors.New("down")}
	if _, err := svc.CleanupExpiredTrustedDevices(ctx); err == nil {
		t.Fatal("清理失败应返回错误")
	}
}
