package application

import (
	"context"
	stderrors "errors"

	"jimu/internal/modules/audit/domain"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
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

func (s *AuditService) Get(ctx context.Context, id uint64) (*AuditLogResponse, error) {
	log, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeNotFound, "audit log not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get audit log", err)
	}
	resp := ToAuditLogResponse(*log)
	return &resp, nil
}

// List 查询审计日志
func (s *AuditService) List(ctx context.Context, p pagination.Pagination) ([]AuditLogResponse, int64, error) {
	logs, total, err := s.repo.List(ctx, p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list audit logs", err)
	}
	return ToAuditLogResponses(logs), total, nil
}
