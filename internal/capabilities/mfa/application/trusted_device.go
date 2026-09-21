package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"time"

	mfadomain "jimu/internal/capabilities/mfa/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
)

const (
	// deviceTokenPrefix 设备令牌前缀，便于识别与误提交排查
	deviceTokenPrefix = "jimu_dev_"
	deviceLabelMaxLen = 64
	userAgentMaxLen   = 256
)

// hashDeviceToken 计算设备令牌哈希（仅存哈希，明文只在签发时返回一次）
func hashDeviceToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// isTrustedDevice 判断本次登录是否来自该用户的可信设备。
// 任何异常（未启用、令牌缺失/过期/不属于该用户）都按「不可信」处理，回退到 TOTP 校验。
func (s *MFAService) isTrustedDevice(ctx context.Context, userID, tenantID uint64) bool {
	if s.trustedDevices == nil || s.trustedDeviceDays <= 0 {
		return false
	}
	token := contract.ClientInfoFrom(ctx).DeviceToken
	if token == "" {
		return false
	}

	device, err := s.trustedDevices.FindByTokenHash(ctx, hashDeviceToken(token))
	if err != nil {
		return false
	}
	// 令牌只对签发它的用户有效：即使泄露也无法用于他人账号
	if device.UserID != userID {
		return false
	}
	if time.Now().After(device.ExpiresAt) {
		// 顺手清理过期记录（归属校验用 userID 兜底）
		_ = s.trustedDevices.Delete(ctx, 0, userID, device.ID)
		return false
	}
	_ = s.trustedDevices.Touch(ctx, device.ID, time.Now())
	return true
}

// issueTrustedDevice 签发新的可信设备令牌（仅登录成功后调用）
func (s *MFAService) issueTrustedDevice(ctx context.Context, userID, tenantID uint64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := deviceTokenPrefix + hex.EncodeToString(raw)

	info := contract.ClientInfoFrom(ctx)
	device := &mfadomain.TrustedDevice{
		TenantID:  tenantID,
		UserID:    userID,
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
func (s *MFAService) ListTrustedDevices(ctx context.Context, userID uint64) ([]mfadomain.TrustedDevice, error) {
	if s.trustedDevices == nil {
		return nil, errors.New(errors.CodeInternalError, "trusted devices are not configured")
	}
	return s.trustedDevices.ListByUser(ctx, tenant.FromContext(ctx), userID)
}

// RevokeTrustedDevice 注销单个可信设备
func (s *MFAService) RevokeTrustedDevice(ctx context.Context, userID, id uint64) error {
	if s.trustedDevices == nil {
		return errors.New(errors.CodeInternalError, "trusted devices are not configured")
	}
	if err := s.trustedDevices.Delete(ctx, tenant.FromContext(ctx), userID, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to revoke trusted device", err)
	}
	return nil
}

// RevokeAllTrustedDevices 注销该用户全部可信设备
func (s *MFAService) RevokeAllTrustedDevices(ctx context.Context, userID uint64) error {
	if s.trustedDevices == nil {
		return nil
	}
	if err := s.trustedDevices.DeleteAllByUser(ctx, userID); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to revoke trusted devices", err)
	}
	return nil
}

// CleanupExpiredTrustedDevices 清理过期设备记录，返回删除条数
func (s *MFAService) CleanupExpiredTrustedDevices(ctx context.Context) (int64, error) {
	if s.trustedDevices == nil {
		return 0, nil
	}
	return s.trustedDevices.DeleteExpired(ctx, time.Now())
}

// RevokeTrustedDevicesQuietly 吊销设备但不影响主流程（改密/登出全部设备时调用）
func (s *MFAService) RevokeTrustedDevicesQuietly(ctx context.Context, userID uint64) {
	if s.trustedDevices == nil {
		return
	}
	if err := s.trustedDevices.DeleteAllByUser(ctx, userID); err != nil && !isNotFound(err) {
		log.Printf("mfa: revoke trusted devices for user %d: %v", userID, err)
	}
}

// RevokeDevices 实现 contract.MFAVerifier：静默吊销该用户全部可信设备。
func (s *MFAService) RevokeDevices(ctx context.Context, userID uint64) {
	s.RevokeTrustedDevicesQuietly(ctx, userID)
}

// truncateString 截断字符串到最大长度（超出部分丢弃）
func truncateString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
