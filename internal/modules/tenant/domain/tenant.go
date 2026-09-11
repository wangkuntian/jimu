package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Tenant 租户实体
type Tenant struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	Code      string         `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Status    int8           `gorm:"default:1" json:"status"`
	Version   int64          `gorm:"not null;default:0" json:"-"` // 乐观锁版本号（每次更新自增）
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Tenant) TableName() string { return "tenants" }

// TenantRepository 租户仓储接口
type TenantRepository interface {
	FindByID(ctx context.Context, id uint64) (*Tenant, error)
	FindByCode(ctx context.Context, code string) (*Tenant, error)
	List(ctx context.Context, offset, limit int, sort, order string) ([]Tenant, int64, error)
	Create(ctx context.Context, tenant *Tenant) error
	Update(ctx context.Context, tenant *Tenant) error
	Delete(ctx context.Context, id uint64) error
}
