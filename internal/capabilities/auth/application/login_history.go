package application

import (
	"context"
	"log"

	authdomain "jimu/internal/capabilities/auth/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
)

const userAgentMaxLen = 256

// recordLoginHistory 记录一次登录尝试（成功/失败/锁定）。
// 记录失败不影响登录主流程，仅打印日志，避免审计旁路拖垮认证。
func (s *AuthService) recordLoginHistory(ctx context.Context, userID, tenantID uint64, username, status, reason string) {
	if s.loginHistory == nil {
		return
	}
	info := contract.ClientInfoFrom(ctx)
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
