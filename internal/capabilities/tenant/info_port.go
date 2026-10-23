package tenant

import (
	"context"

	"jimu/internal/capabilities/tenant/domain"
	"jimu/internal/contract"

	"gorm.io/gorm"
)

type tenantInfoSource struct {
	db *gorm.DB
}

func newTenantInfoSource(db *gorm.DB) contract.TenantInfoSource {
	return &tenantInfoSource{db: db}
}

func (s *tenantInfoSource) GetByIDs(ctx context.Context, ids []uint64) (map[uint64]contract.TenantInfo, error) {
	out := make(map[uint64]contract.TenantInfo, len(ids))
	if s.db == nil || len(ids) == 0 {
		return out, nil
	}
	var rows []domain.Tenant
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = contract.TenantInfo{ID: row.ID, Name: row.Name}
	}
	return out, nil
}
