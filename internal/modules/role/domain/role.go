package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID          uint64         `gorm:"primaryKey" json:"id"`
	TenantID    uint64         `gorm:"column:tenant_id;default:0;uniqueIndex:idx_roles_tenant_name,priority:1" json:"tenant_id"` // 所属租户 ID（0=未归属）
	Name        string         `gorm:"size:64;not null;uniqueIndex:idx_roles_tenant_name,priority:2" json:"name"`                // 租户内唯一
	Description string         `gorm:"size:255;default:''" json:"description"`
	Status      int8           `gorm:"default:1" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Role) TableName() string { return "roles" }

type RoleRepository interface {
	FindByID(ctx context.Context, id uint64) (*Role, error)
	List(ctx context.Context, tenantID uint64, offset, limit int, sort, order string) ([]Role, int64, error)
	Create(ctx context.Context, role *Role) error
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uint64) error
	AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error
	GetPermissions(ctx context.Context, roleID uint64) ([]Permission, error)
}
