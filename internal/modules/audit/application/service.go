package application

import (
	"context"

	"jimu/internal/modules/audit/domain"
)

type AuditService struct {
	repo domain.AuditRepository
}

func NewAuditService(repo domain.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// Record 记录审计日志
func (s *AuditService) Record(ctx context.Context, log domain.AuditLog) error {
	return s.repo.Create(ctx, &log)
}

// List 查询审计日志
func (s *AuditService) List(ctx context.Context, page, pageSize int) ([]domain.AuditLog, int64, error) {
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, offset, pageSize)
}
