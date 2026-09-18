package infrastructure

import (
	"context"

	"jimu/internal/capabilities/tenant/domain"

	"gorm.io/gorm"
)

type mysqlPlanRepository struct {
	db *gorm.DB
}

// NewMysqlPlanRepository 创建套餐仓储
func NewMysqlPlanRepository(db *gorm.DB) domain.PlanRepository {
	return &mysqlPlanRepository{db: db}
}

func (r *mysqlPlanRepository) FindByID(ctx context.Context, id uint64) (*domain.Plan, error) {
	var plan domain.Plan
	if err := r.db.WithContext(ctx).First(&plan, id).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *mysqlPlanRepository) FindByCode(ctx context.Context, code string) (*domain.Plan, error) {
	var plan domain.Plan
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&plan).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *mysqlPlanRepository) List(ctx context.Context, offset, limit int) ([]domain.Plan, int64, error) {
	var plans []domain.Plan
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.Plan{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order("id ASC").Offset(offset).Limit(limit).Find(&plans).Error
	return plans, total, err
}

func (r *mysqlPlanRepository) Create(ctx context.Context, plan *domain.Plan) error {
	return r.db.WithContext(ctx).Create(plan).Error
}

func (r *mysqlPlanRepository) Update(ctx context.Context, plan *domain.Plan) error {
	return r.db.WithContext(ctx).Model(&domain.Plan{}).Where("id = ?", plan.ID).Updates(map[string]interface{}{
		"name":         plan.Name,
		"max_users":    plan.MaxUsers,
		"max_roles":    plan.MaxRoles,
		"max_api_keys": plan.MaxAPIKeys,
	}).Error
}

func (r *mysqlPlanRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.Plan{}, id).Error
}

func (r *mysqlPlanRepository) CountTenants(ctx context.Context, planID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("tenants").Where("plan_id = ? AND deleted_at IS NULL", planID).Count(&n).Error
	return n, err
}

type mysqlQuotaRepository struct {
	db *gorm.DB
}

// NewMysqlQuotaRepository 创建配额查询仓储
func NewMysqlQuotaRepository(db *gorm.DB) domain.QuotaRepository {
	return &mysqlQuotaRepository{db: db}
}

// FindPlanByTenant 返回租户套餐；未分配（plan_id=0）或套餐已被删除时返回 nil（视为不限）
func (r *mysqlQuotaRepository) FindPlanByTenant(ctx context.Context, tenantID uint64) (*domain.Plan, error) {
	var tenant domain.Tenant
	if err := r.db.WithContext(ctx).Select("id", "plan_id").First(&tenant, tenantID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	if tenant.PlanID == 0 {
		return nil, nil
	}
	var plan domain.Plan
	if err := r.db.WithContext(ctx).First(&plan, tenant.PlanID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

// Count 统计资源用量；users/roles 排除软删除记录，api_keys 为硬删除直接计数
func (r *mysqlQuotaRepository) Count(ctx context.Context, tenantID uint64, resource domain.QuotaResource) (int64, error) {
	var n int64
	query := r.db.WithContext(ctx).Table(string(resource)).Where("tenant_id = ?", tenantID)
	switch resource {
	case domain.QuotaResourceUsers, domain.QuotaResourceRoles:
		query = query.Where("deleted_at IS NULL")
	case domain.QuotaResourceAPIKeys:
	default:
		return 0, gorm.ErrInvalidField
	}
	err := query.Count(&n).Error
	return n, err
}

func (r *mysqlQuotaRepository) AssignPlan(ctx context.Context, tenantID, planID uint64) error {
	return r.db.WithContext(ctx).Model(&domain.Tenant{}).Where("id = ?", tenantID).
		Updates(map[string]interface{}{
			"plan_id": planID,
			"version": gorm.Expr("version + 1"), // 与名称/状态更新共用乐观锁版本号
		}).Error
}
