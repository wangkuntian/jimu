package application

import (
	"context"
	"log"

	authdomain "jimu/internal/capabilities/auth/domain"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
)

const userAgentMaxLen = 256

type clientInfoKey struct{}

// ClientInfo 登录请求的客户端信息，由 interfaces 层从 gin 上下文提取后注入，
// 供登录历史与可信设备使用（保持 AuthService 的方法签名不感知 HTTP）。
type ClientInfo struct {
	IP        string
	UserAgent string
	// DeviceToken 登录时携带的可信设备令牌（密码仍需正确，仅用于跳过 TOTP）
	DeviceToken string
	// RememberDevice 登录成功后是否签发新的可信设备令牌
	RememberDevice bool
}

// WithClientInfo 把客户端信息写入上下文（保留已设置的其他字段）
func WithClientInfo(ctx context.Context, ip, userAgent string) context.Context {
	info := clientInfoFrom(ctx)
	info.IP = ip
	info.UserAgent = userAgent
	return context.WithValue(ctx, clientInfoKey{}, info)
}

// WithLoginDevice 写入可信设备相关参数（保留已设置的客户端信息）
func WithLoginDevice(ctx context.Context, deviceToken string, remember bool) context.Context {
	info := clientInfoFrom(ctx)
	info.DeviceToken = deviceToken
	info.RememberDevice = remember
	return context.WithValue(ctx, clientInfoKey{}, info)
}

func clientInfoFrom(ctx context.Context) ClientInfo {
	if v, ok := ctx.Value(clientInfoKey{}).(ClientInfo); ok {
		return v
	}
	return ClientInfo{}
}

// recordLoginHistory 记录一次登录尝试（成功/失败/锁定）。
// 记录失败不影响登录主流程，仅打印日志，避免审计旁路拖垮认证。
func (s *AuthService) recordLoginHistory(ctx context.Context, userID, tenantID uint64, username, status, reason string) {
	if s.loginHistory == nil {
		return
	}
	info := clientInfoFrom(ctx)
	record := &authdomain.LoginHistory{
		TenantID:  tenantID,
		UserID:    userID,
		Username:  username,
		Status:    status,
		Reason:    reason,
		IP:        info.IP,
		UserAgent: truncateString(info.UserAgent, userAgentMaxLen),
	}
	if err := s.loginHistory.Create(ctx, record); err != nil {
		log.Printf("auth: record login history for %s: %v", username, err)
	}
}

// ListLoginHistory 查询当前用户的登录历史（按上下文租户隔离）
func (s *AuthService) ListLoginHistory(ctx context.Context, userID uint64, p pagination.Pagination) ([]authdomain.LoginHistory, int64, error) {
	if s.loginHistory == nil {
		return nil, 0, errors.New(errors.CodeInternalError, "login history is not configured")
	}
	return s.loginHistory.ListByUser(ctx, tenant.FromContext(ctx), userID, p.GetOffset(), p.GetLimit())
}

func truncateString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
