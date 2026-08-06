package application

import (
	"context"

	"jimu/internal/modules/admin/domain"
)

// AdminAuditService 审计日志服务
type AdminAuditService struct {
	repo domain.AuditRepository
}

// NewAdminAuditService 创建审计日志服务
func NewAdminAuditService(repo domain.AuditRepository) *AdminAuditService {
	return &AdminAuditService{repo: repo}
}

// Log 记录审计日志
func (s *AdminAuditService) Log(ctx context.Context, adminID uint64, adminName, action, resource, detail, ip string) error {
	log := &domain.AuditLog{
		AdminID:   adminID,
		AdminName: adminName,
		Action:    action,
		Resource:  resource,
		Detail:    detail,
		IP:        ip,
	}
	return s.repo.Create(ctx, log)
}

// ListAuditLogs 查询审计日志
func (s *AdminAuditService) ListAuditLogs(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]domain.AuditLog, int64, error) {
	return s.repo.List(ctx, offset, limit, filters)
}
