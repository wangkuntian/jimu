package domain

import (
	"context"
	"time"
)

// Plan 租户套餐：一组资源上限，0 表示该项不限。
type Plan struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	Code       string    `gorm:"size:32;uniqueIndex;not null" json:"code"`
	Name       string    `gorm:"size:64;not null" json:"name"`
	MaxUsers   int       `gorm:"not null;default:0" json:"max_users"`
	MaxRoles   int       `gorm:"not null;default:0" json:"max_roles"`
	MaxAPIKeys int       `gorm:"not null;default:0" json:"max_api_keys"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Plan) TableName() string { return "tenant_plans" }

// QuotaResource 受配额约束的资源
type QuotaResource string

const (
	QuotaResourceUsers   QuotaResource = "users"
	QuotaResourceRoles   QuotaResource = "roles"
	QuotaResourceAPIKeys QuotaResource = "api_keys"
)

// PlanRepository 套餐仓储接口
type PlanRepository interface {
	FindByID(ctx context.Context, id uint64) (*Plan, error)
	FindByCode(ctx context.Context, code string) (*Plan, error)
	List(ctx context.Context, offset, limit int) ([]Plan, int64, error)
	Create(ctx context.Context, plan *Plan) error
	// Update 全量更新套餐（code 不可修改）
	Update(ctx context.Context, plan *Plan) error
	Delete(ctx context.Context, id uint64) error
	// CountTenants 统计使用该套餐的租户数量（删除前校验）
	CountTenants(ctx context.Context, planID uint64) (int64, error)
}

// QuotaRepository 配额与用量查询接口
type QuotaRepository interface {
	// FindPlanByTenant 返回租户的套餐；未分配套餐时返回 nil
	FindPlanByTenant(ctx context.Context, tenantID uint64) (*Plan, error)
	// Count 统计租户在某种资源上的当前用量（已软删除的记录不计入）
	Count(ctx context.Context, tenantID uint64, resource QuotaResource) (int64, error)
	// AssignPlan 为租户分配套餐（planID=0 表示取消套餐）
	AssignPlan(ctx context.Context, tenantID, planID uint64) error
}

// ResourceUsage 单项资源的用量与上限
type ResourceUsage struct {
	Used  int64 `json:"used"`
	Limit int   `json:"limit"` // 0=不限
}
