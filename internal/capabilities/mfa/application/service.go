package application

import (
	"context"
	"log"
	"time"

	mfadomain "jimu/internal/capabilities/mfa/domain"
	"jimu/internal/capabilities/mfa/totp"
	"jimu/internal/contract"
	"jimu/internal/shared/errors"

	"gorm.io/gorm"
)

// MFAService TOTP 二次验证 + 可信设备（跳过 MFA）用例。
type MFAService struct {
	repo              mfadomain.MFARepository
	users             contract.UserinfoSource // 取用户名兜底 otpauth account（可 nil）
	trustedDevices    mfadomain.TrustedDeviceRepository
	trustedDeviceDays int
	issuer            string
}

// NewMFAService 创建 MFA 服务。trustedDeviceDays<=0 或 trustedDevices==nil 时关闭可信设备。
func NewMFAService(repo mfadomain.MFARepository, users contract.UserinfoSource, trustedDevices mfadomain.TrustedDeviceRepository, trustedDeviceDays int, issuer string) *MFAService {
	return &MFAService{
		repo:              repo,
		users:             users,
		trustedDevices:    trustedDevices,
		trustedDeviceDays: trustedDeviceDays,
		issuer:            issuer,
	}
}

// SetupTOTP 为用户生成新的 TOTP 密钥并返回 otpauth URI（未启用，需 EnableTOTP 确认）。
// 重复调用会轮换密钥（旧密钥立即失效）。返回数据仅此一次全量可见。
func (s *MFAService) SetupTOTP(ctx context.Context, userID uint64, account string) (secret string, uri string, err error) {
	if account == "" && s.users != nil {
		// 未显式提供 account 时用用户名兜底（otpauth URI 的可读标识）
		if u, loadErr := s.users.GetByID(ctx, userID); loadErr == nil && u != nil {
			account = u.Username
		}
	}
	secret, err = totp.Secret()
	if err != nil {
		return "", "", errors.Wrap(errors.CodeInternalError, "failed to generate totp secret", err)
	}
	// 保存密钥但暂不启用（enabled=false），等待 EnableTOTP 用首次验证码确认
	if err := s.repo.UpsertSecret(ctx, 0, userID, secret, false); err != nil {
		return "", "", errors.Wrap(errors.CodeInternalError, "failed to save totp secret", err)
	}
	issuer := "jimu"
	if s.issuer != "" {
		issuer = s.issuer
	}
	return secret, totp.ProvisioningURI(secret, account, issuer), nil
}

// EnableTOTP 用首次生成的验证码确认启用 TOTP。码验证通过后方可开启，防误绑。
func (s *MFAService) EnableTOTP(ctx context.Context, userID uint64, code string) error {
	if code == "" {
		return errors.New(errors.CodeMFARequired, "TOTP code required")
	}
	record, err := s.repo.FindByUser(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return errors.New(errors.CodeInvalidMFA, "TOTP not set up, call setup first")
		}
		return errors.Wrap(errors.CodeInternalError, "failed to load mfa state", err)
	}
	if record.TOTPSecret == "" {
		return errors.New(errors.CodeInvalidMFA, "TOTP not set up, call setup first")
	}
	if !totp.Validate(record.TOTPSecret, code, time.Now(), totp.DefaultPeriod, totp.DefaultDigits, totp.DefaultSkew) {
		return errors.New(errors.CodeInvalidMFA, "invalid TOTP code")
	}
	if err := s.repo.UpsertSecret(ctx, record.TenantID, userID, record.TOTPSecret, true); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to enable totp", err)
	}
	return nil
}

// DisableTOTP 校验当前验证码后关闭 TOTP 并清除密钥。
func (s *MFAService) DisableTOTP(ctx context.Context, userID uint64, code string) error {
	if code == "" {
		return errors.New(errors.CodeMFARequired, "TOTP code required")
	}
	record, err := s.repo.FindByUser(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return errors.New(errors.CodeInvalidMFA, "TOTP not enabled")
		}
		return errors.Wrap(errors.CodeInternalError, "failed to load mfa state", err)
	}
	if !record.TOTPEnabled || record.TOTPSecret == "" {
		return errors.New(errors.CodeInvalidMFA, "TOTP not enabled")
	}
	if !totp.Validate(record.TOTPSecret, code, time.Now(), totp.DefaultPeriod, totp.DefaultDigits, totp.DefaultSkew) {
		return errors.New(errors.CodeInvalidMFA, "invalid TOTP code")
	}
	if err := s.repo.Clear(ctx, userID); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to disable totp", err)
	}
	return nil
}

// Enabled 实现 contract.MFAVerifier：用户是否启用 TOTP。
func (s *MFAService) Enabled(ctx context.Context, userID uint64) (bool, error) {
	record, err := s.repo.FindByUser(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, errors.Wrap(errors.CodeInternalError, "failed to load mfa state", err)
	}
	return record.TOTPEnabled, nil
}

// VerifyTOTP 实现 contract.MFAVerifier：校验登录第二因子。
// 未启用 MFA 时直接通过；未携带验证码但来自可信设备时跳过；
// 缺码返回 CodeMFARequired，码错返回 CodeInvalidMFA。
func (s *MFAService) VerifyTOTP(ctx context.Context, userID, tenantID uint64, code string) error {
	record, err := s.repo.FindByUser(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return errors.Wrap(errors.CodeInternalError, "failed to load mfa state", err)
	}
	if !record.TOTPEnabled || record.TOTPSecret == "" {
		return nil
	}
	if code == "" {
		if s.isTrustedDevice(ctx, userID, tenantID) {
			return nil
		}
		return errors.New(errors.CodeMFARequired, "TOTP code required")
	}
	if !totp.Validate(record.TOTPSecret, code, time.Now(), totp.DefaultPeriod, totp.DefaultDigits, totp.DefaultSkew) {
		return errors.New(errors.CodeInvalidMFA, "invalid TOTP code")
	}
	return nil
}

// MaybeIssueDevice 实现 contract.MFAVerifier：登录成功后按「记住此设备」签发设备令牌。
// 未启用 MFA、未请求记住、未配置可信设备或签发失败时返回空串（不阻断登录）。
func (s *MFAService) MaybeIssueDevice(ctx context.Context, userID, tenantID uint64) string {
	if s.trustedDevices == nil || s.trustedDeviceDays <= 0 {
		return ""
	}
	if !contract.ClientInfoFrom(ctx).RememberDevice {
		return ""
	}
	enabled, err := s.Enabled(ctx, userID)
	if err != nil || !enabled {
		return ""
	}
	token, err := s.issueTrustedDevice(ctx, userID, tenantID)
	if err != nil {
		log.Printf("mfa: issue trusted device for user %d: %v", userID, err)
		return ""
	}
	return token
}

func isNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
