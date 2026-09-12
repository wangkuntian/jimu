package application

import (
	"context"
	"log"

	authdomain "jimu/internal/modules/auth/domain"
	"jimu/internal/platform/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
)

const userAgentMaxLen = 256

type clientInfoKey struct{}

// ClientInfo 登录请求的客户端信息，由 interfaces 层从 gin 上下文提取后注入，
// 供登录历史记录使用（保持 AuthService 的方法签名不感知 HTTP）。
type ClientInfo struct {
	IP        string
	UserAgent string
}

// WithClientInfo 把客户端信息写入上下文
func WithClientInfo(ctx context.Context, ip, userAgent string) context.Context {
	return context.WithValue(ctx, clientInfoKey{}, ClientInfo{IP: ip, UserAgent: userAgent})
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
