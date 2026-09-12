package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"time"

	authdomain "jimu/internal/modules/auth/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/tenant"
	"jimu/internal/shared/errors"

	"gorm.io/gorm"
)

const (
	// deviceTokenPrefix 设备令牌前缀，便于识别与误提交排查
	deviceTokenPrefix = "jimu_dev_"
	deviceLabelMaxLen = 64
)

// hashDeviceToken 计算设备令牌哈希（仅存哈希，明文只在签发时返回一次）
func hashDeviceToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// isTrustedDevice 判断本次登录是否来自该用户的可信设备。
// 任何异常（未启用、令牌缺失/过期/不属于该用户）都按「不可信」处理，回退到 TOTP 校验。
func (s *AuthService) isTrustedDevice(ctx context.Context, user *userdomain.User) bool {
	if s.trustedDevices == nil || s.trustedDeviceDays <= 0 {
		return false
	}
	token := clientInfoFrom(ctx).DeviceToken
	if token == "" {
		return false
	}

	device, err := s.trustedDevices.FindByTokenHash(ctx, hashDeviceToken(token))
	if err != nil {
		return false
	}
	// 令牌只对签发它的用户有效：即使泄露也无法用于他人账号
	if device.UserID != user.ID {
		return false
	}
	if time.Now().After(device.ExpiresAt) {
		// 顺手清理过期记录（归属校验用 userID 兜底）
		_ = s.trustedDevices.Delete(ctx, 0, user.ID, device.ID)
		return false
	}
	_ = s.trustedDevices.Touch(ctx, device.ID, time.Now())
	return true
}

// issueTrustedDevice 签发新的可信设备令牌（仅登录成功后调用）
func (s *AuthService) issueTrustedDevice(ctx context.Context, user *userdomain.User) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := deviceTokenPrefix + hex.EncodeToString(raw)

	info := clientInfoFrom(ctx)
	device := &authdomain.TrustedDevice{
		TenantID:  user.TenantID,
		UserID:    user.ID,
		TokenHash: hashDeviceToken(token),
		Label:     truncateString(info.UserAgent, deviceLabelMaxLen),
		IP:        info.IP,
		UserAgent: truncateString(info.UserAgent, userAgentMaxLen),
		ExpiresAt: time.Now().AddDate(0, 0, s.trustedDeviceDays),
	}
	if err := s.trustedDevices.Create(ctx, device); err != nil {
		return "", err
	}
	return token, nil
}

// ListTrustedDevices 列出当前用户的可信设备（按上下文租户隔离）
func (s *AuthService) ListTrustedDevices(ctx context.Context, userID uint64) ([]authdomain.TrustedDevice, error) {
	if s.trustedDevices == nil {
		return nil, errors.New(errors.CodeInternalError, "trusted devices are not configured")
	}
	return s.trustedDevices.ListByUser(ctx, tenant.FromContext(ctx), userID)
}

// RevokeTrustedDevice 注销单个可信设备
func (s *AuthService) RevokeTrustedDevice(ctx context.Context, userID, id uint64) error {
	if s.trustedDevices == nil {
		return errors.New(errors.CodeInternalError, "trusted devices are not configured")
	}
	if err := s.trustedDevices.Delete(ctx, tenant.FromContext(ctx), userID, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to revoke trusted device", err)
	}
	return nil
}

// RevokeAllTrustedDevices 注销该用户全部可信设备
func (s *AuthService) RevokeAllTrustedDevices(ctx context.Context, userID uint64) error {
	if s.trustedDevices == nil {
		return nil
	}
	if err := s.trustedDevices.DeleteAllByUser(ctx, userID); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to revoke trusted devices", err)
	}
	return nil
}

// CleanupExpiredTrustedDevices 清理过期设备记录，返回删除条数
func (s *AuthService) CleanupExpiredTrustedDevices(ctx context.Context) (int64, error) {
	if s.trustedDevices == nil {
		return 0, nil
	}
	return s.trustedDevices.DeleteExpired(ctx, time.Now())
}

// revokeTrustedDevicesQuietly 吊销设备但不影响主流程（改密/登出全部设备时调用）
func (s *AuthService) revokeTrustedDevicesQuietly(ctx context.Context, userID uint64) {
	if s.trustedDevices == nil {
		return
	}
	if err := s.trustedDevices.DeleteAllByUser(ctx, userID); err != nil && !isNotFound(err) {
		log.Printf("auth: revoke trusted devices for user %d: %v", userID, err)
	}
}

func isNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
