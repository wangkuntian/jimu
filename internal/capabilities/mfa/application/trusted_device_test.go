package application

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	mfadomain "jimu/internal/capabilities/mfa/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 测试替身 ----

// fakeMFARepo 内存 TOTP 状态仓储
type fakeMFARepo struct {
	records map[uint64]*mfadomain.UserMFA
	err     error
}

func newFakeMFARepo() *fakeMFARepo {
	return &fakeMFARepo{records: map[uint64]*mfadomain.UserMFA{}}
}

func (r *fakeMFARepo) FindByUser(_ context.Context, userID uint64) (*mfadomain.UserMFA, error) {
	if r.err != nil {
		return nil, r.err
	}
	rec, ok := r.records[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return rec, nil
}

func (r *fakeMFARepo) UpsertSecret(_ context.Context, tenantID, userID uint64, secret string, enabled bool) error {
	if r.err != nil {
		return r.err
	}
	r.records[userID] = &mfadomain.UserMFA{ID: userID, TenantID: tenantID, UserID: userID, TOTPSecret: secret, TOTPEnabled: enabled}
	return nil
}

func (r *fakeMFARepo) Clear(_ context.Context, userID uint64) error {
	if r.err != nil {
		return r.err
	}
	delete(r.records, userID)
	return nil
}

// fakeTrustedDeviceRepo 内存可信设备仓储
type fakeTrustedDeviceRepo struct {
	devices  []*mfadomain.TrustedDevice
	nextID   uint64
	touched  []uint64
	deletedA []uint64
}

func newFakeTrustedDeviceRepo() *fakeTrustedDeviceRepo {
	return &fakeTrustedDeviceRepo{nextID: 1}
}

func (r *fakeTrustedDeviceRepo) Create(_ context.Context, device *mfadomain.TrustedDevice) error {
	device.ID = r.nextID
	r.nextID++
	r.devices = append(r.devices, device)
	return nil
}

func (r *fakeTrustedDeviceRepo) FindByTokenHash(_ context.Context, tokenHash string) (*mfadomain.TrustedDevice, error) {
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

func (r *fakeTrustedDeviceRepo) ListByUser(_ context.Context, tenantID, userID uint64) ([]mfadomain.TrustedDevice, error) {
	var out []mfadomain.TrustedDevice
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

// newTestMFAService 构造启用 TOTP + 可信设备的服务（ttlDays<=0 关闭可信设备）。
func newTestMFAService(ttlDays int) (*MFAService, *fakeMFARepo, *fakeTrustedDeviceRepo) {
	repo := newFakeMFARepo()
	devices := newFakeTrustedDeviceRepo()
	svc := NewMFAService(repo, nil, devices, ttlDays, "jimu")
	return svc, repo, devices
}

func enableTOTP(t *testing.T, svc *MFAService, repo *fakeMFARepo, userID uint64) string {
	t.Helper()
	secret, _, err := svc.SetupTOTP(context.Background(), userID, "alice")
	require.NoError(t, err)
	require.NoError(t, svc.EnableTOTP(context.Background(), userID, totpCurrentCode(t, secret)))
	require.True(t, repo.records[userID].TOTPEnabled)
	return secret
}

func appCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}

// ---- TOTP 生命周期 ----

func TestSetupEnableDisableTOTP(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newTestMFAService(30)

	secret, uri, err := svc.SetupTOTP(ctx, 42, "alice")
	require.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.Contains(t, uri, "otpauth://totp/")
	assert.False(t, repo.records[42].TOTPEnabled, "SetupTOTP 应只存密钥不启用")

	// 错误码不能启用
	assert.Equal(t, apperrors.CodeInvalidMFA, appCode(svc.EnableTOTP(ctx, 42, "000000")))
	// 正确码启用
	require.NoError(t, svc.EnableTOTP(ctx, 42, totpCurrentCode(t, secret)))
	assert.True(t, repo.records[42].TOTPEnabled)

	// 未绑定时启用报错
	err = svc.EnableTOTP(ctx, 99, "123456")
	assert.Equal(t, apperrors.CodeInvalidMFA, appCode(err))

	// 空码报 MFARequired
	assert.Equal(t, apperrors.CodeMFARequired, appCode(svc.EnableTOTP(ctx, 42, "")))
	assert.Equal(t, apperrors.CodeMFARequired, appCode(svc.DisableTOTP(ctx, 42, "")))

	// 错误码不能关闭
	assert.Equal(t, apperrors.CodeInvalidMFA, appCode(svc.DisableTOTP(ctx, 42, "000000")))
	// 正确码关闭并清记录
	require.NoError(t, svc.DisableTOTP(ctx, 42, totpCurrentCode(t, secret)))
	_, ok := repo.records[42]
	assert.False(t, ok, "DisableTOTP 应清除记录")

	// 未启用时关闭报错
	assert.Equal(t, apperrors.CodeInvalidMFA, appCode(svc.DisableTOTP(ctx, 42, "123456")))
}

func TestEnabledReportsTOTPState(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newTestMFAService(30)

	enabled, err := svc.Enabled(ctx, 42)
	require.NoError(t, err)
	assert.False(t, enabled, "无记录视为未启用")

	enableTOTP(t, svc, repo, 42)
	enabled, err = svc.Enabled(ctx, 42)
	require.NoError(t, err)
	assert.True(t, enabled)

	repo.err = stderrors.New("db down")
	_, err = svc.Enabled(ctx, 42)
	assert.Error(t, err)
}

// ---- VerifyTOTP（登录第二因子） ----

func TestVerifyTOTPDisabledUserPasses(t *testing.T) {
	svc, _, _ := newTestMFAService(30)
	// 未启用 MFA 的用户：任何码都放行
	assert.NoError(t, svc.VerifyTOTP(context.Background(), 42, 1, ""))
}

func TestVerifyTOTPRequiresCodeWhenEnabled(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newTestMFAService(30)
	secret := enableTOTP(t, svc, repo, 42)

	assert.Equal(t, apperrors.CodeMFARequired, appCode(svc.VerifyTOTP(ctx, 42, 1, "")))
	assert.Equal(t, apperrors.CodeInvalidMFA, appCode(svc.VerifyTOTP(ctx, 42, 1, "000000")))
	assert.NoError(t, svc.VerifyTOTP(ctx, 42, 1, totpCurrentCode(t, secret)))
}

func TestVerifyTOTPSkipsForTrustedDevice(t *testing.T) {
	ctx := context.Background()
	svc, repo, devices := newTestMFAService(30)
	secret := enableTOTP(t, svc, repo, 42)

	// 签发设备令牌
	issueCtx := contract.WithLoginDevice(contract.WithClientInfo(ctx, "203.0.113.7", "curl/8.0"), "", true)
	token := svc.MaybeIssueDevice(issueCtx, 42, 1)
	require.NotEmpty(t, token)
	require.Equal(t, 1, devices.count())
	device := devices.devices[0]
	assert.Equal(t, uint64(42), device.UserID)
	assert.Equal(t, uint64(1), device.TenantID)
	assert.Equal(t, hashDeviceToken(token), device.TokenHash, "只存哈希")
	assert.Equal(t, "203.0.113.7", device.IP)

	// 携带设备令牌、不带 TOTP 码即可通过
	useCtx := contract.WithLoginDevice(contract.WithClientInfo(ctx, "203.0.113.7", "curl/8.0"), token, false)
	require.NoError(t, svc.VerifyTOTP(useCtx, 42, 1, ""))
	assert.Equal(t, []uint64{1}, devices.touched, "跳过时应更新最近使用时间")

	// 不携带令牌仍要求 TOTP
	assert.Equal(t, apperrors.CodeMFARequired, appCode(svc.VerifyTOTP(ctx, 42, 1, "")))
	// 正确码仍可用
	assert.NoError(t, svc.VerifyTOTP(ctx, 42, 1, totpCurrentCode(t, secret)))
}

func TestTrustedDeviceOwnershipExpiryAndUnknown(t *testing.T) {
	ctx := context.Background()
	svc, repo, devices := newTestMFAService(30)
	enableTOTP(t, svc, repo, 42)

	// 他人设备令牌无效
	require.NoError(t, devices.Create(ctx, &mfadomain.TrustedDevice{ID: 1, TenantID: 1, UserID: 43, TokenHash: hashDeviceToken("jimu_dev_other"), ExpiresAt: time.Now().Add(time.Hour)}))
	otherCtx := contract.WithLoginDevice(ctx, "jimu_dev_other", false)
	assert.Equal(t, apperrors.CodeMFARequired, appCode(svc.VerifyTOTP(otherCtx, 42, 1, "")), "设备令牌只对签发它的用户有效")

	// 过期令牌不可用且被清理
	require.NoError(t, devices.Create(ctx, &mfadomain.TrustedDevice{ID: 2, TenantID: 1, UserID: 42, TokenHash: hashDeviceToken("jimu_dev_expired"), ExpiresAt: time.Now().Add(-time.Minute)}))
	expiredCtx := contract.WithLoginDevice(ctx, "jimu_dev_expired", false)
	assert.Equal(t, apperrors.CodeMFARequired, appCode(svc.VerifyTOTP(expiredCtx, 42, 1, "")))
	assert.Equal(t, 1, devices.count(), "过期设备应被顺手清理")

	// 未知令牌不可用
	unknownCtx := contract.WithLoginDevice(ctx, "jimu_dev_unknown", false)
	assert.Equal(t, apperrors.CodeMFARequired, appCode(svc.VerifyTOTP(unknownCtx, 42, 1, "")))
}

func TestTrustedDeviceDisabledWhenTTLIsZero(t *testing.T) {
	svc, repo, devices := newTestMFAService(0)
	enableTOTP(t, svc, repo, 42)

	ctx := contract.WithLoginDevice(context.Background(), "", true)
	assert.Empty(t, svc.MaybeIssueDevice(ctx, 42, 1), "ttl=0 关闭签发")
	assert.Zero(t, devices.count())
	// 即使外部塞入设备令牌也不跳过
	_ = devices.Create(context.Background(), &mfadomain.TrustedDevice{ID: 1, TenantID: 1, UserID: 42, TokenHash: hashDeviceToken("jimu_dev_x"), ExpiresAt: time.Now().Add(time.Hour)})
	assert.Equal(t, apperrors.CodeMFARequired, appCode(svc.VerifyTOTP(contract.WithLoginDevice(context.Background(), "jimu_dev_x", false), 42, 1, "")))
}

func TestMaybeIssueDeviceOnlyWhenRememberAndEnabled(t *testing.T) {
	ctx := context.Background()
	svc, repo, devices := newTestMFAService(30)

	// 未勾选记住 → 不签发
	notRemember := contract.WithLoginDevice(ctx, "", false)
	assert.Empty(t, svc.MaybeIssueDevice(notRemember, 42, 1))

	// 勾选记住但未启用 TOTP → 不签发
	remember := contract.WithLoginDevice(ctx, "", true)
	assert.Empty(t, svc.MaybeIssueDevice(remember, 42, 1), "仅启用 TOTP 的账号才签发")
	assert.Zero(t, devices.count())

	// 启用后勾选记住 → 签发
	enableTOTP(t, svc, repo, 42)
	assert.NotEmpty(t, svc.MaybeIssueDevice(remember, 42, 1))
	assert.Equal(t, 1, devices.count())
}

// ---- 可信设备 CRUD ----

func TestListAndRevokeTrustedDevices(t *testing.T) {
	ctx := tenant.WithTenant(context.Background(), 1)
	svc, _, devices := newTestMFAService(30)

	require.NoError(t, devices.Create(ctx, &mfadomain.TrustedDevice{TenantID: 1, UserID: 42, TokenHash: "h1", ExpiresAt: time.Now().Add(time.Hour)}))
	require.NoError(t, devices.Create(ctx, &mfadomain.TrustedDevice{TenantID: 2, UserID: 42, TokenHash: "h2", ExpiresAt: time.Now().Add(time.Hour)}))

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

func TestRevokeDevicesQuietly(t *testing.T) {
	svc, _, devices := newTestMFAService(30)
	require.NoError(t, devices.Create(context.Background(), &mfadomain.TrustedDevice{TenantID: 1, UserID: 42, TokenHash: "h1", ExpiresAt: time.Now().Add(time.Hour)}))

	svc.RevokeDevices(context.Background(), 42)
	assert.Zero(t, devices.count())
	assert.Equal(t, []uint64{42}, devices.deletedA)
}

func TestCleanupExpiredTrustedDevices(t *testing.T) {
	svc, _, devices := newTestMFAService(30)
	require.NoError(t, devices.Create(context.Background(), &mfadomain.TrustedDevice{TenantID: 1, UserID: 42, TokenHash: "h1", ExpiresAt: time.Now().Add(-time.Minute)}))
	removed, err := svc.CleanupExpiredTrustedDevices(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), removed)

	// 未配置仓储（nil）时：列表报内部错误、清理返回 0
	noRepo := NewMFAService(newFakeMFARepo(), nil, nil, 0, "jimu")
	_, err = noRepo.ListTrustedDevices(context.Background(), 42)
	assert.Equal(t, apperrors.CodeInternalError, appCode(err))
	removed, err = noRepo.CleanupExpiredTrustedDevices(context.Background())
	require.NoError(t, err)
	assert.Zero(t, removed)
	assert.NoError(t, noRepo.RevokeAllTrustedDevices(context.Background(), 42))
	assert.Equal(t, apperrors.CodeInternalError, appCode(noRepo.RevokeTrustedDevice(context.Background(), 42, 1)))
}

// ---- 异常分支 ----

// failingTrustedDeviceRepo 让查询/写入全部报错
type failingTrustedDeviceRepo struct {
	fakeTrustedDeviceRepo
	findErr   error
	createErr error
	deleteErr error
}

func (r *failingTrustedDeviceRepo) FindByTokenHash(context.Context, string) (*mfadomain.TrustedDevice, error) {
	return nil, r.findErr
}
func (r *failingTrustedDeviceRepo) Create(context.Context, *mfadomain.TrustedDevice) error {
	return r.createErr
}
func (r *failingTrustedDeviceRepo) Delete(context.Context, uint64, uint64, uint64) error {
	return r.deleteErr
}
func (r *failingTrustedDeviceRepo) ListByUser(context.Context, uint64, uint64) ([]mfadomain.TrustedDevice, error) {
	return nil, r.findErr
}
func (r *failingTrustedDeviceRepo) DeleteAllByUser(context.Context, uint64) error { return r.deleteErr }
func (r *failingTrustedDeviceRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	return 0, r.findErr
}

func TestTrustedDeviceServiceErrorBranches(t *testing.T) {
	ctx := context.Background()
	repo := &failingTrustedDeviceRepo{
		findErr:   stderrors.New("boom"),
		deleteErr: stderrors.New("boom"),
		createErr: stderrors.New("boom"),
	}
	svc := NewMFAService(newFakeMFARepo(), nil, repo, 30, "jimu")

	if _, err := svc.ListTrustedDevices(ctx, 42); err == nil {
		t.Fatal("仓储报错时 ListTrustedDevices 应返回错误")
	}
	assert.Equal(t, apperrors.CodeInternalError, appCode(svc.RevokeTrustedDevice(ctx, 42, 1)))
	assert.Equal(t, apperrors.CodeInternalError, appCode(svc.RevokeAllTrustedDevices(ctx, 42)))
	// 签发失败返回空串，不阻断登录
	assert.Empty(t, svc.MaybeIssueDevice(contract.WithLoginDevice(ctx, "", true), 42, 1))

	// 过期设备视为不可信，并顺手清理（清理失败不影响判定）
	expired := &failingTrustedDeviceRepo{deleteErr: stderrors.New("cleanup failed")}
	expired.devices = []*mfadomain.TrustedDevice{{ID: 9, UserID: 42, TokenHash: hashDeviceToken("jimu_dev_expired"), ExpiresAt: time.Now().Add(-time.Minute)}}
	svc.trustedDevices = expired
	assert.False(t, svc.isTrustedDevice(ctx, 42, 1), "过期设备应视为不可信")

	svc.trustedDevices = &failingTrustedDeviceRepo{findErr: stderrors.New("down")}
	if _, err := svc.CleanupExpiredTrustedDevices(ctx); err == nil {
		t.Fatal("清理失败应返回错误")
	}
}
